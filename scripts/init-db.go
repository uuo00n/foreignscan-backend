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
    "strings"
    "time"

    "foreignscan/internal/config"
    "foreignscan/internal/database"
    "foreignscan/internal/models"

    "go.mongodb.org/mongo-driver/bson/primitive"
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



func main() {
    var (
        interactive bool
        mongoURI    string
        dbName      string
        imagesDir   string
        stylesDir   string
        mode        string // full-init 或 structure-only
        dryRun      bool   // 保留参数占位（无需使用）
        limit       int    // 保留参数占位（无需使用）
    )

    flag.BoolVar(&interactive, "interactive", true, "是否使用交互模式")
    // 已移除 issues/comparisons 相关测试数据填充
    flag.StringVar(&mongoURI, "mongo-uri", "mongodb://localhost:27017", "MongoDB连接URI")
    flag.StringVar(&dbName, "db-name", "foreignscan", "数据库名称")
    flag.StringVar(&imagesDir, "images-dir", "./uploads/images", "图片目录路径（仅 full-init 模式使用）")
    flag.StringVar(&stylesDir, "styles-dir", "./uploads/styles", "样式图目录路径（仅 full-init 模式使用）")
    flag.StringVar(&mode, "mode", "full-init", "运行模式：full-init（重建并导入）或 augment-existing（增补缺失数据）或 structure-only（仅初始化集合与索引）")
    flag.BoolVar(&dryRun, "dry-run", false, "增补模式下仅打印计划，不写入数据库")
    flag.IntVar(&limit, "limit", 0, "增补模式下处理的图片数量上限（0 表示不限制）")
    flag.Parse()

    if interactive {
        fmt.Println("=== 数据库初始化工具 ===")
        mongoURI = getUserInput("MongoDB连接URI", mongoURI)
        dbName = getUserInput("数据库名称", dbName)
        mode = strings.ToLower(getUserInput("运行模式 (full-init / augment-existing / structure-only)", mode))

        // 根据模式分别收集参数
        if mode == "full-init" {
            imagesDir = getUserInput("图片目录路径", imagesDir)
            stylesDir = getUserInput("样式图目录路径", stylesDir)
            // 不再填充 issues/comparisons 测试数据

            fmt.Println("\n=== 初始化配置 ===")
            fmt.Printf("MongoDB URI: %s\n", mongoURI)
            fmt.Printf("数据库名称: %s\n", dbName)
            fmt.Printf("运行模式: %s\n", mode)
            fmt.Printf("图片目录: %s\n", imagesDir)
            fmt.Printf("样式图目录: %s\n", stylesDir)
            // 不再显示 issues/comparisons 测试数据配置
        } else if mode == "structure-only" {
            fmt.Println("\n=== 初始化配置 ===")
            fmt.Printf("MongoDB URI: %s\n", mongoURI)
            fmt.Printf("数据库名称: %s\n", dbName)
            fmt.Printf("运行模式: %s\n", mode)
        } else {
            log.Fatalf("无效的运行模式: %s。请使用 full-init / augment-existing / structure-only", mode)
        }

        if !getUserConfirmation("\n确认以上配置并继续？", true) {
            fmt.Println("操作已取消")
            return
        }
    }

    // 设置数据库配置并连接
    database.SetConfig(&config.Config{MongoURI: mongoURI, DatabaseName: dbName})
    if err := database.Connect(); err != nil {
        log.Fatalf("连接数据库失败: %v", err)
    }
    defer database.Close()

    ctx := context.Background()

    // 创建索引（所有模式都需要）
    if err := models.EnsureDetectionIndexes(); err != nil {
        log.Fatalf("创建 Detection 索引失败: %v", err)
    }
    fmt.Println("已确保 detections 索引")

    if mode == "structure-only" {
        // 仅初始化集合，不进行任何数据导入或增补
        collections := []string{"scenes", "styleImages", "images", "detections"}
        for _, coll := range collections {
            if err := database.GetDatabase().CreateCollection(ctx, coll); err != nil {
                // 如果集合已存在，CreateCollection 会报错，此处仅提示
                log.Printf("创建集合 %s 提示: %v", coll, err)
            } else {
                log.Printf("已创建集合: %s", coll)
            }
        }
        fmt.Println("仅结构初始化完成（集合与基本索引）。")
        return
    }

    // 已移除 augment-existing 模式

    // full-init 模式：清库并按文件系统导入
    collectionsToDrop := []string{"scenes", "styleImages", "images"}
    for _, coll := range collectionsToDrop {
        if err := database.GetCollection(coll).Drop(ctx); err != nil {
            log.Printf("删除集合 %s 时出错 (可能不存在): %v", coll, err)
        }
    }
    fmt.Println("已删除现有集合")

    fmt.Println("开始按文件系统导入场景/样式图/图片...")
    // 定义文件夹映射关系
    folderMappings := []FolderMapping{
        {"001", "001-machine", "机器设备场景", "北区-A栋"},
        {"002", "002-drum", "鼓形设备场景", "南区-B栋"},
        {"003", "003-excavator", "挖掘机场景", "东区-C栋"},
    }

    var seqNum = 1
    for _, mapping := range folderMappings {
        // 创建场景
        scene := createSceneRecord(mapping)
        if _, err := database.GetCollection("scenes").InsertOne(ctx, scene); err != nil {
            log.Fatalf("插入场景失败: %v", err)
        }
        fmt.Printf("创建场景: %s, ID: %s\n", scene.Name, scene.ID.Hex())

        // 创建样式图
        styleImage := createStyleImageRecord(scene.ID, mapping.StyleFolder, mapping.Name+"样式")
        if _, err := database.GetCollection("styleImages").InsertOne(ctx, styleImage); err != nil {
            log.Fatalf("插入样式图失败: %v", err)
        }
        fmt.Printf("创建样式图: %s\n", styleImage.Name)

        // 处理图片
        imagesFolderPath := filepath.Join(imagesDir, mapping.ImageFolder)
        files, err := ioutil.ReadDir(imagesFolderPath)
        if err != nil {
            log.Printf("读取图片目录失败: %v，跳过此文件夹", err)
            continue
        }

        var folderImages []interface{}
        for _, file := range files {
            if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".jpg") {
                img := createImageRecord(file, seqNum, scene.ID, mapping.ImageFolder)
                img.Location = mapping.Location
                folderImages = append(folderImages, img)
                seqNum++
            }
        }

        if len(folderImages) > 0 {
            if _, err := database.GetCollection("images").InsertMany(ctx, folderImages); err != nil {
                log.Fatalf("插入图片数据失败: %v", err)
            }
            fmt.Printf("成功插入 %d 条图片数据到场景 %s\n", len(folderImages), scene.Name)
        }
    }

    // 不再填充 issues/comparisons 测试数据

    fmt.Println("数据库初始化完成!")
}
