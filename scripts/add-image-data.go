package main

import (
	"context"
	"fmt"
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

// ImageModel 图片模型
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

func addImageData() {
	// 连接MongoDB
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.Background())

	// 选择数据库和集合
	db := client.Database("foreignscan")
	collection := db.Collection("images")

	// 清空现有数据
	_, err = collection.DeleteMany(context.Background(), bson.M{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("已清空现有图片数据")

	// 图片目录路径
	imagesDir := "/Users/uu/Desktop/dnui-foreignscan/foreignscan-backend/uploads/images"

	// 遍历目录
	dirs, err := os.ReadDir(imagesDir)
	if err != nil {
		log.Fatal(err)
	}

	sequenceNumber := 1
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}

		// 解析场景ID
		sceneIDStr := dir.Name()
		sceneID, err := primitive.ObjectIDFromHex(sceneIDStr)
		if err != nil {
			fmt.Printf("无效的场景ID: %s, 跳过\n", sceneIDStr)
			continue
		}

		// 读取该场景下的所有图片
		scenePath := filepath.Join(imagesDir, sceneIDStr)
		files, err := os.ReadDir(scenePath)
		if err != nil {
			log.Printf("读取目录失败: %s, 错误: %v\n", scenePath, err)
			continue
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			filename := file.Name()
			// 检查是否是图片文件
			if !strings.HasSuffix(strings.ToLower(filename), ".jpg") &&
				!strings.HasSuffix(strings.ToLower(filename), ".jpeg") &&
				!strings.HasSuffix(strings.ToLower(filename), ".png") {
				continue
			}

			// 从文件名解析日期信息
			dateParts := strings.Split(filename, "_")
			var timestamp time.Time
			if len(dateParts) > 0 {
				dateStr := dateParts[0]
				if len(dateStr) >= 8 {
					year, _ := strconv.Atoi("20" + dateStr[0:2])
					month, _ := strconv.Atoi(dateStr[2:4])
					day, _ := strconv.Atoi(dateStr[4:6])
					timestamp = time.Date(year, time.Month(month), day, 12, 0, 0, 0, time.Local)
				} else {
					timestamp = time.Now()
				}
			} else {
				timestamp = time.Now()
			}

			// 创建图片记录
			image := ImageModel{
				ID:             primitive.NewObjectID(),
				SequenceNumber: sequenceNumber,
				SceneID:        sceneID,
				Timestamp:      timestamp,
				Location:       "测试位置",
				Filename:       filename,
				Path:           filepath.Join("/uploads/images", sceneIDStr, filename),
				IsDetected:     false,
				HasIssue:       false,
				IssueType:      "",
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}

			// 随机设置一些图片为已检测状态
			if sequenceNumber%3 == 0 {
				image.IsDetected = true
				// 随机设置一些已检测的图片有问题
				if sequenceNumber%6 == 0 {
					image.HasIssue = true
					image.IssueType = "异物"
				}
			}

			// 插入数据库
			_, err = collection.InsertOne(context.Background(), image)
			if err != nil {
				log.Printf("插入图片记录失败: %v\n", err)
				continue
			}

			fmt.Printf("已添加图片记录: %s, 序号: %d\n", filename, sequenceNumber)
			sequenceNumber++
		}
	}

	fmt.Printf("成功添加 %d 条图片记录\n", sequenceNumber-1)
}

func main() {
	addImageData()
}