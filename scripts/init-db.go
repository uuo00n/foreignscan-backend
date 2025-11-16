package main

import (
    "bufio"
    "context"
    "flag"
    "fmt"
    "log"
    "os"
    "path/filepath"
    "strings"
    "time"

    "foreignscan/internal/config"
    "foreignscan/internal/database"
    "foreignscan/internal/models"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"
)

// FolderMapping 文件夹映射关系
type FolderMapping struct {
	ImageFolder string
	StyleFolder string
	Name        string
	Location    string
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

// createSceneRecord 创建场景记录
func createSceneRecord(mapping FolderMapping) models.Scene {
	now := time.Now()
	return models.Scene{
		ID:          primitive.NewObjectID(),
		Name:        mapping.Name,
		Description: "根据文件夹映射创建的场景",
		Location:    mapping.Location,
		Status:      "active",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// createStyleImageRecord 创建样式图记录
func createStyleImageRecord(sceneID primitive.ObjectID, styleFolder, styleName string) models.StyleImage {
	now := time.Now()
	stylePath := filepath.Join("uploads", "styles", styleFolder, "example.jpg")
	return models.StyleImage{
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

// createImageRecord 创建图片记录
func createImageRecord(file os.FileInfo, seqNum int, sceneID primitive.ObjectID, imageFolder string) models.Image {
	timestamp, _ := time.Parse("20060102", strings.Split(file.Name(), "_")[0])
	now := time.Now()

	return models.Image{
		ID:               primitive.NewObjectID(),
		SequenceNumber:   seqNum,
		SceneID:          sceneID,
		Timestamp:        timestamp,
		Location:         "", // 将在后面设置
		Filename:         file.Name(),
		Path:             filepath.Join("uploads", "images", imageFolder, file.Name()),
		IsDetected:       false,
		HasIssue:         false,
		IssueType:        "",
		Status:           models.ImageStatusUndetected,
		DetectionResults: []interface{}{},
		CreatedAt:        now,
		UpdatedAt:        now,
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

    collections := []string{"scenes", "styleImages", "images", "detections"}
    for _, coll := range collections {
        if err := database.GetDatabase().CreateCollection(ctx, coll); err != nil {
            log.Printf("创建集合 %s 提示: %v", coll, err)
        } else {
            log.Printf("已创建集合: %s", coll)
        }
    }

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

    fmt.Println("仅结构初始化完成")
}
