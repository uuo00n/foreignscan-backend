package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"foreignscan/internal/config"
	"foreignscan/internal/database"
	"foreignscan/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

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

func ensureImageIndexes() error {
	col := database.GetCollection("images")
	ctx := context.Background()
	models := []mongo.IndexModel{
		{Keys: bson.D{{Key: "sceneId", Value: 1}}},
		{Keys: bson.D{{Key: "createdAt", Value: -1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "isDetected", Value: 1}}},
		{Keys: bson.D{{Key: "hasIssue", Value: 1}}},
	}
	_, err := col.Indexes().CreateMany(ctx, models)
	return err
}

func ensureSceneIndexes() error {
	col := database.GetCollection("scenes")
	ctx := context.Background()
	models := []mongo.IndexModel{
		{Keys: bson.D{{Key: "createdAt", Value: -1}}},
	}
	_, err := col.Indexes().CreateMany(ctx, models)
	return err
}

func ensureStyleImageIndexes() error {
	col := database.GetCollection("styleImages")
	ctx := context.Background()
	models := []mongo.IndexModel{
		{Keys: bson.D{{Key: "sceneId", Value: 1}}},
		{Keys: bson.D{{Key: "createdAt", Value: -1}}},
	}
	_, err := col.Indexes().CreateMany(ctx, models)
	return err
}

func main() {
	var (
		interactive bool
		mongoURI    string
		dbName      string
	)

	flag.BoolVar(&interactive, "interactive", true, "是否使用交互模式")
	flag.StringVar(&mongoURI, "mongo-uri", "mongodb://localhost:27017", "MongoDB连接URI")
	flag.StringVar(&dbName, "db-name", "foreignscan", "数据库名称")
	flag.Parse()

	if interactive {
		fmt.Println("=== 数据库结构初始化 ===")
		fmt.Println("注意：此操作仅创建集合和索引，不会写入任何初始数据。")
		mongoURI = getUserInput("MongoDB连接URI", mongoURI)
		dbName = getUserInput("数据库名称", dbName)
		fmt.Println("\n=== 初始化配置 ===")
		fmt.Printf("MongoDB URI: %s\n", mongoURI)
		fmt.Printf("数据库名称: %s\n", dbName)
		if !getUserConfirmation("\n确认以上配置并继续？", true) {
			fmt.Println("操作已取消")
			return
		}
	}

	database.SetConfig(&config.Config{MongoURI: mongoURI, DatabaseName: dbName})
	if err := database.Connect(); err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer database.Close()

	ctx := context.Background()

	// 1. 创建集合
	collections := []string{"scenes", "styleImages", "images", "detections"}
	for _, coll := range collections {
		if err := database.GetDatabase().CreateCollection(ctx, coll); err != nil {
			// 忽略已存在的错误
			if !strings.Contains(err.Error(), "already exists") {
				log.Printf("创建集合 %s 提示: %v", coll, err)
			} else {
				log.Printf("集合已存在: %s", coll)
			}
		} else {
			log.Printf("已创建集合: %s", coll)
		}
	}

	// 2. 创建索引
	fmt.Println("\n=== 开始创建/更新索引 ===")
	if err := models.EnsureDetectionIndexes(); err != nil {
		log.Fatalf("创建 Detection 索引失败: %v", err)
	}
	if err := ensureImageIndexes(); err != nil {
		log.Fatalf("创建 Images 索引失败: %v", err)
	}
	if err := ensureSceneIndexes(); err != nil {
		log.Fatalf("创建 Scenes 索引失败: %v", err)
	}
	if err := ensureStyleImageIndexes(); err != nil {
		log.Fatalf("创建 StyleImages 索引失败: %v", err)
	}

	fmt.Println("\n数据库结构初始化完成！")
}
