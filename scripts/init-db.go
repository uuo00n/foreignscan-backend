package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// 场景模型
type Scene struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description" json:"description"`
	Location    string             `bson:"location" json:"location"`
	Status      string             `bson:"status" json:"status"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// 样式图模型
type StyleImage struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SceneID     primitive.ObjectID `bson:"sceneId" json:"sceneId"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description" json:"description"`
	Filename    string             `bson:"filename" json:"filename"`
	Path        string             `bson:"path" json:"path"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// 图片模型
type ImageModel struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SequenceNumber   int                `bson:"sequenceNumber" json:"sequenceNumber"`
	SceneID          primitive.ObjectID `bson:"sceneId" json:"sceneId"`
	Timestamp        time.Time          `bson:"timestamp" json:"timestamp"`
	Location         string             `bson:"location" json:"location"`
	Filename         string             `bson:"filename" json:"filename"`
	Path             string             `bson:"path" json:"path"`
	IsDetected       bool               `bson:"isDetected" json:"isDetected"`
	HasIssue         bool               `bson:"hasIssue" json:"hasIssue"`
	IssueType        string             `bson:"issueType" json:"issueType"`
	DetectionResults []interface{}      `bson:"detectionResults" json:"detectionResults"`
	CreatedAt        time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt        time.Time          `bson:"updatedAt" json:"updatedAt"`
}

// 文件夹映射关系
type FolderMapping struct {
	ImageFolder string
	StyleFolder string
	Name        string
	Location    string
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

// 从用户获取输入
func getUserInput(prompt string, defaultValue string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [%s]: ", prompt, defaultValue)
	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}

	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}

	return input
}

// 从用户获取是/否输入
func getUserConfirmation(prompt string, defaultValue bool) bool {
	defaultStr := "y"
	if !defaultValue {
		defaultStr = "n"
	}

	for {
		input := getUserInput(prompt+" (y/n)", defaultStr)
		input = strings.ToLower(input)

		if input == "y" || input == "yes" {
			return true
		} else if input == "n" || input == "no" {
			return false
		} else if input == defaultStr {
			return defaultValue
		}

		fmt.Println("请输入 y 或 n")
	}
}

// 创建场景记录
func createSceneRecord(folderMapping FolderMapping) Scene {
	now := time.Now()
	return Scene{
		ID:          primitive.NewObjectID(),
		Name:        folderMapping.Name,
		Description: "根据文件夹映射创建的场景",
		Location:    folderMapping.Location,
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// 创建样式图记录
func createStyleImageRecord(sceneID primitive.ObjectID, styleFolder string, styleName string) StyleImage {
	now := time.Now()
	// 使用样式文件夹中的图片
	stylePath := filepath.Join("uploads/styles", styleFolder, "example.jpg")
	return StyleImage{
		ID:          primitive.NewObjectID(),
		SceneID:     sceneID,
		Name:        styleName,
		Description: "场景的样式图",
		Filename:    "example.jpg",
		Path:        stylePath,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// 创建图片记录
func createImageRecord(file os.FileInfo, seqNum int, sceneID primitive.ObjectID, imageFolder string) ImageModel {
	// 解析文件信息
	_, timestamp := parseFileInfo(file.Name())
	now := time.Now()

	// 创建基本图片记录
	img := ImageModel{
		SequenceNumber:   seqNum,
		SceneID:          sceneID,
		Timestamp:        timestamp,
		Location:         "", // 将在后面设置
		Filename:         file.Name(),
		Path:             filepath.Join("uploads/images", imageFolder, file.Name()),
		IsDetected:       false,
		HasIssue:         false,
		IssueType:        "",
		DetectionResults: []interface{}{},
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	return img
}

func main() {
	// 定义命令行参数
	var interactive bool
	var mongoURI string
	var dbName string
	var imagesDir string
	var stylesDir string

	flag.BoolVar(&interactive, "interactive", true, "是否使用交互模式")
	flag.StringVar(&mongoURI, "mongo-uri", "mongodb://localhost:27017", "MongoDB连接URI")
	flag.StringVar(&dbName, "db-name", "foreignscan", "数据库名称")
	flag.StringVar(&imagesDir, "images-dir", "./uploads/images", "图片目录路径")
	flag.StringVar(&stylesDir, "styles-dir", "./uploads/styles", "样式图目录路径")

	flag.Parse()

	// 如果是交互模式，询问用户选项
	if interactive {
		fmt.Println("=== 数据库初始化工具 ===")
		fmt.Println("该工具将根据文件夹对应关系初始化数据库。")

		// 询问MongoDB连接信息
		mongoURI = getUserInput("MongoDB连接URI", mongoURI)
		dbName = getUserInput("数据库名称", dbName)
		imagesDir = getUserInput("图片目录路径", imagesDir)
		stylesDir = getUserInput("样式图目录路径", stylesDir)

		fmt.Println("\n=== 初始化配置 ===")
		fmt.Printf("MongoDB URI: %s\n", mongoURI)
		fmt.Printf("数据库名称: %s\n", dbName)
		fmt.Printf("图片目录: %s\n", imagesDir)
		fmt.Printf("样式图目录: %s\n", stylesDir)

		// 最终确认
		confirm := getUserConfirmation("\n确认以上配置并继续？", true)
		if !confirm {
			fmt.Println("操作已取消")
			return
		}
	}

	// 连接MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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
	db := client.Database(dbName)
	scenesCollection := db.Collection("scenes")
	styleImagesCollection := db.Collection("styleImages")
	imagesCollection := db.Collection("images")

	// 删除现有集合（如果存在）
	err = scenesCollection.Drop(ctx)
	if err != nil {
		log.Printf("删除现有scenes集合时出错: %v", err)
	}
	err = styleImagesCollection.Drop(ctx)
	if err != nil {
		log.Printf("删除现有styleImages集合时出错: %v", err)
	}
	err = imagesCollection.Drop(ctx)
	if err != nil {
		log.Printf("删除现有images集合时出错: %v", err)
	}
	fmt.Println("已删除现有集合")

	// 创建索引
	seqIndexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "sequenceNumber", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	sceneIdIndexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "sceneId", Value: 1}},
		Options: options.Index().SetBackground(true),
	}
	_, err = imagesCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{seqIndexModel, sceneIdIndexModel})
	if err != nil {
		log.Fatalf("创建图片索引失败: %v", err)
	}

	// 为styleImages集合创建索引
	styleSceneIdIndexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "sceneId", Value: 1}},
		Options: options.Index().SetBackground(true),
	}
	_, err = styleImagesCollection.Indexes().CreateOne(ctx, styleSceneIdIndexModel)
	if err != nil {
		log.Fatalf("创建样式图索引失败: %v", err)
	}

	fmt.Println("成功创建索引")

	// 定义文件夹映射关系
	folderMappings := []FolderMapping{
		{
			ImageFolder: "001",
			StyleFolder: "001-machine",
			Name:        "机器设备场景",
			Location:    "北区-A栋",
		},
		{
			ImageFolder: "002",
			StyleFolder: "002-drum",
			Name:        "鼓形设备场景",
			Location:    "南区-B栋",
		},
		{
			ImageFolder: "003",
			StyleFolder: "003-excavator",
			Name:        "挖掘机场景",
			Location:    "东区-C栋",
		},
	}

	// 创建场景映射表，用于跟踪已创建的场景
	sceneMap := make(map[string]primitive.ObjectID)
	var seqNum int = 1

	// 处理每个文件夹映射
	for _, mapping := range folderMappings {
		// 创建场景记录
		scene := createSceneRecord(mapping)
		result, err := scenesCollection.InsertOne(ctx, scene)
		if err != nil {
			log.Fatalf("插入场景失败: %v", err)
		}

		// 存储场景ID
		sceneMap[mapping.ImageFolder] = scene.ID
		fmt.Printf("创建场景: %s, ID: %s\n", scene.Name, result.InsertedID)

		// 创建样式图记录
		styleImage := createStyleImageRecord(scene.ID, mapping.StyleFolder, mapping.Name+"样式")
		_, err = styleImagesCollection.InsertOne(ctx, styleImage)
		if err != nil {
			log.Fatalf("插入样式图失败: %v", err)
		}
		fmt.Printf("创建样式图: %s\n", styleImage.Name)

		// 读取该文件夹中的图片
		imagesFolderPath := filepath.Join(imagesDir, mapping.ImageFolder)
		files, err := ioutil.ReadDir(imagesFolderPath)
		if err != nil {
			log.Printf("读取图片目录失败: %v，跳过此文件夹", err)
			continue
		}

		// 处理该文件夹中的每张图片
		var folderImages []ImageModel
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".jpg") {
				// 创建图片记录
				img := createImageRecord(file, seqNum, scene.ID, mapping.ImageFolder)
				img.Location = mapping.Location // 设置位置信息
				folderImages = append(folderImages, img)
				seqNum++
			}
		}

		// 插入该文件夹的图片数据
		if len(folderImages) > 0 {
			var documents []interface{}
			for _, img := range folderImages {
				documents = append(documents, img)
			}

			insertResult, err := imagesCollection.InsertMany(ctx, documents)
			if err != nil {
				log.Fatalf("插入图片数据失败: %v", err)
			}

			fmt.Printf("成功插入 %d 条图片数据到场景 %s\n", len(insertResult.InsertedIDs), scene.Name)
		} else {
			fmt.Printf("在文件夹 %s 中未找到任何图片文件\n", mapping.ImageFolder)
		}
	}

	fmt.Println("数据库初始化完成!")
}
