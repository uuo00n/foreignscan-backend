package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// 图片模型
type Image struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SequenceNumber   int                `bson:"sequenceNumber" json:"sequenceNumber"`
	SceneID          string             `bson:"sceneId" json:"sceneId"`
	Timestamp        time.Time          `bson:"timestamp" json:"timestamp"`
	Location         string             `bson:"location" json:"location"`
	Filename         string             `bson:"filename" json:"filename"`
	Path             string             `bson:"path" json:"path"`
	IsDetected       bool               `bson:"isDetected" json:"isDetected"`
	HasIssue         bool               `bson:"hasIssue" json:"hasIssue"`
	IssueType        string             `bson:"issueType" json:"issueType"`
	DetectionResults []interface{}      `bson:"detectionResults" json:"detectionResults"`
}

// 从文件名解析场景ID和时间
func parseFileInfo(filename string) (string, time.Time) {
	// 从文件名中提取日期信息 (格式: YYYYMMDD)
	parts := strings.Split(filename, "_")
	if len(parts) < 2 {
		// 默认值
		return "unknown", time.Now()
	}
	
	dateStr := parts[0]
	if len(dateStr) != 8 {
		return "unknown", time.Now()
	}
	
	year, _ := strconv.Atoi(dateStr[0:4])
	month, _ := strconv.Atoi(dateStr[4:6])
	day, _ := strconv.Atoi(dateStr[6:8])
	
	// 生成场景ID (基于日期)
	sceneID := fmt.Sprintf("scene-%s", dateStr[4:8])
	
	// 创建时间戳 (使用文件序号作为小时)
	seqNum, _ := strconv.Atoi(parts[1])
	hour := 8 + (seqNum % 12) // 8点到20点之间
	
	timestamp := time.Date(year, time.Month(month), day, hour, 0, 0, 0, time.Local)
	
	return sceneID, timestamp
}

// 获取地点信息
func getLocation(sceneID string) string {
	// 基于场景ID分配不同的地点
	if strings.Contains(sceneID, "1027") {
		return "北区-A栋"
	} else if strings.Contains(sceneID, "1028") {
		return "南区-B栋"
	} else if strings.Contains(sceneID, "1031") {
		return "东区-C栋"
	}
	return "西区-D栋"
}

func main() {
	// 连接MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 默认连接本地MongoDB
	mongoURI := "mongodb://localhost:27017"
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("连接MongoDB失败: %v", err)
	}
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			log.Fatalf("断开MongoDB连接失败: %v", err)
		}
	}()

	// 检查连接
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("MongoDB连接测试失败: %v", err)
	}
	fmt.Println("成功连接到MongoDB")

	// 获取数据库和集合
	db := client.Database("foreignscan")
	imagesCollection := db.Collection("images")

	// 删除现有集合（如果存在）
	err = imagesCollection.Drop(ctx)
	if err != nil {
		log.Printf("删除现有集合时出错: %v", err)
	} else {
		fmt.Println("已删除现有images集合")
	}

	// 创建索引
	indexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "sequenceNumber", Value: -1}},
		Options: options.Index().SetUnique(true),
	}
	_, err = imagesCollection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		log.Fatalf("创建索引失败: %v", err)
	}
	fmt.Println("成功创建sequenceNumber索引")

	// 读取uploads目录中的真实图片文件
	uploadsDir := "./uploads"
	files, err := ioutil.ReadDir(uploadsDir)
	if err != nil {
		log.Fatalf("读取uploads目录失败: %v", err)
	}

	// 准备真实图片数据
	var realImages []Image
	var seqNum int = 1
	
	// 问题类型列表
	issueTypes := []string{"裂缝", "磨损", "变形", ""}
	
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".jpg") {
			// 解析文件信息
			sceneID, timestamp := parseFileInfo(file.Name())
			location := getLocation(sceneID)
			
			// 随机决定是否已检测和是否有问题
			isDetected := seqNum%3 != 0 // 2/3的图片已检测
			hasIssue := isDetected && seqNum%2 == 0 // 已检测的图片中一半有问题
			
			// 选择问题类型
			issueType := ""
			var detectionResults []interface{}
			
			if hasIssue {
				issueIndex := (seqNum % 3)
				issueType = issueTypes[issueIndex]
				
				// 创建检测结果
				detectionResults = []interface{}{
					bson.M{
						"x":          100 + (seqNum * 20),
						"y":          150 + (seqNum * 15),
						"width":      40 + (seqNum % 30),
						"height":     30 + (seqNum % 20),
						"type":       issueType,
						"confidence": 0.75 + float64(seqNum%20)/100.0,
					},
				}
			} else {
				detectionResults = []interface{}{}
			}
			
			// 创建图片记录
			img := Image{
				SequenceNumber:   seqNum,
				SceneID:          sceneID,
				Timestamp:        timestamp,
				Location:         location,
				Filename:         file.Name(),
				Path:             "uploads/" + file.Name(),
				IsDetected:       isDetected,
				HasIssue:         hasIssue,
				IssueType:        issueType,
				DetectionResults: detectionResults,
			}
			
			realImages = append(realImages, img)
			seqNum++
		}
	}

	// 插入真实图片数据
	if len(realImages) > 0 {
		var documents []interface{}
		for _, img := range realImages {
			documents = append(documents, img)
		}

		insertResult, err := imagesCollection.InsertMany(ctx, documents)
		if err != nil {
			log.Fatalf("插入真实图片数据失败: %v", err)
		}

		fmt.Printf("成功插入 %d 条真实图片数据\n", len(insertResult.InsertedIDs))
	} else {
		fmt.Println("未找到任何图片文件")
	}
	
	fmt.Println("数据库初始化完成!")
}