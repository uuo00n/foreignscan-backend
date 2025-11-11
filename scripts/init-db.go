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

    "foreignscan/internal/config"
    "foreignscan/internal/database"
    "foreignscan/internal/models"

    "go.mongodb.org/mongo-driver/bson"
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

// seedExtraData 填充 issues 和 comparisons 的测试数据
func seedExtraData() {
    log.Println("开始填充 issues 和 comparisons 测试数据...")

	images, err := models.FindAll()
	if err != nil {
		log.Fatalf("查询图片失败: %v", err)
	}
	if len(images) == 0 {
		log.Println("数据库中没有图片，跳过测试数据填充。")
		return
	}

	n := 3
	if len(images) < n {
		n = len(images)
	}

	types := []string{"TEST-类型A", "TEST-类型B", "TEST-类型C"}
	descs := []string{"TEST-设备皮带磨损说明", "TEST-护罩缺失说明", "TEST-油污泄漏说明"}

	for i := 0; i < n; i++ {
		img := images[i]

		// 插入问题记录
		issue := &models.Issue{
			ImageID:     img.ID,
			SceneID:     img.SceneID,
			Type:        types[i%len(types)],
			Description: descs[i%len(descs)],
		}
		issueID, err := models.InsertIssue(issue)
		if err != nil {
			log.Fatalf("插入问题记录失败 (imageId=%s): %v", img.ID.Hex(), err)
		}

		// 插入对比记录
		processedFilename := fmt.Sprintf("processed_%s", img.Filename)
		afterPath := filepath.Join("uploads", "images", img.SceneID.Hex(), processedFilename)
		comp := &models.Comparison{
			ImageID:    img.ID,
			BeforePath: img.Path,
			AfterPath:  afterPath,
			DiffInfo:   map[string]interface{}{"note": "TEST-示例对比，无真实文件"},
			Remark:     fmt.Sprintf("TEST-对比记录-%d", i+1),
		}
		compID, err := models.InsertComparison(comp)
		if err != nil {
			log.Fatalf("插入对比记录失败 (imageId=%s): %v", img.ID.Hex(), err)
		}

		log.Printf("已插入 Issue: %s, Comparison: %s, 对应图片: %s", issueID.Hex(), compID.Hex(), img.ID.Hex())
	}

	log.Printf("完成：issues/comparisons 各插入 %d 条测试数据。", n)
}

// augmentExisting 针对已存在的 images，补充缺失的 issues/comparisons 数据
// - dryRun: 仅打印将要执行的操作，不实际写入数据库
// - limit: 限制处理的图片数量（0 或负数表示不限制）
func augmentExisting(ctx context.Context, dryRun bool, limit int) {
    log.Println("开始增补现有图片的 issues/comparisons 数据...")

    // 读取所有图片
    images, err := models.FindAll()
    if err != nil {
        log.Fatalf("查询图片失败: %v", err)
    }
    if len(images) == 0 {
        log.Println("数据库中没有图片，增补操作结束。")
        return
    }

    // 如果设置了 limit，则截取前 limit 条
    if limit > 0 && limit < len(images) {
        images = images[:limit]
    }

    // 计数器
    var addedIssues, addedComparisons, skipped int

    // 遍历图片，检查并增补
    for _, img := range images {
        // 检查是否已有 issue
        issueCount, err := database.GetCollection("issues").CountDocuments(ctx, bson.M{"imageId": img.ID})
        if err != nil {
            log.Fatalf("统计 issues 失败 (imageId=%s): %v", img.ID.Hex(), err)
        }

        // 检查是否已有 comparison
        compCount, err := database.GetCollection("comparisons").CountDocuments(ctx, bson.M{"imageId": img.ID})
        if err != nil {
            log.Fatalf("统计 comparisons 失败 (imageId=%s): %v", img.ID.Hex(), err)
        }

        // 如果两者都已存在，则跳过
        if issueCount > 0 && compCount > 0 {
            skipped++
            continue
        }

        // 构造测试 issue 数据
        issue := &models.Issue{
            ImageID:     img.ID,
            SceneID:     img.SceneID,
            Type:        "TEST-自动增补类型",
            Description: "TEST-自动为缺失图片补充的问题记录",
        }

        // 构造测试 comparison 数据
        processedFilename := fmt.Sprintf("processed_%s", img.Filename)
        afterPath := filepath.Join("uploads", "images", img.SceneID.Hex(), processedFilename)
        comp := &models.Comparison{
            ImageID:    img.ID,
            BeforePath: img.Path,
            AfterPath:  afterPath,
            DiffInfo:   map[string]interface{}{"note": "TEST-自动增补的对比信息，无真实文件"},
            Remark:     "TEST-自动增补",
        }

        // dry-run 模式仅打印，不写库
        if dryRun {
            if issueCount == 0 {
                log.Printf("[dry-run] 将为图片 %s 插入 Issue", img.ID.Hex())
            }
            if compCount == 0 {
                log.Printf("[dry-run] 将为图片 %s 插入 Comparison", img.ID.Hex())
            }
            continue
        }

        // 实际写库
        if issueCount == 0 {
            if _, err := models.InsertIssue(issue); err != nil {
                log.Fatalf("插入 Issue 失败 (imageId=%s): %v", img.ID.Hex(), err)
            }
            addedIssues++
        }
        if compCount == 0 {
            if _, err := models.InsertComparison(comp); err != nil {
                log.Fatalf("插入 Comparison 失败 (imageId=%s): %v", img.ID.Hex(), err)
            }
            addedComparisons++
        }
    }

    if dryRun {
        log.Printf("[dry-run] 预估增补：Issues=%d, Comparisons=%d, 已存在跳过=%d", addedIssues, addedComparisons, skipped)
    } else {
        log.Printf("增补完成：新增 Issues=%d, 新增 Comparisons=%d, 已存在跳过=%d", addedIssues, addedComparisons, skipped)
    }
}

func main() {
    var (
        interactive bool
        seedExtra   bool
        mongoURI    string
        dbName      string
        imagesDir   string
        stylesDir   string
        mode        string // full-init 或 augment-existing
        dryRun      bool   // 仅用于 augment-existing 模式
        limit       int    // 仅用于 augment-existing 模式
    )

    flag.BoolVar(&interactive, "interactive", true, "是否使用交互模式")
    flag.BoolVar(&seedExtra, "seed-extra", true, "是否填充 issues 和 comparisons 的测试数据（仅 full-init 模式生效）")
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
            seedExtra = getUserConfirmation("是否填充 issues 和 comparisons 的测试数据?", seedExtra)

            fmt.Println("\n=== 初始化配置 ===")
            fmt.Printf("MongoDB URI: %s\n", mongoURI)
            fmt.Printf("数据库名称: %s\n", dbName)
            fmt.Printf("运行模式: %s\n", mode)
            fmt.Printf("图片目录: %s\n", imagesDir)
            fmt.Printf("样式图目录: %s\n", stylesDir)
            fmt.Printf("填充测试数据: %v\n", seedExtra)
        } else if mode == "augment-existing" {
            dryRun = getUserConfirmation("增补模式：是否使用 dry-run（仅打印，不写库）?", dryRun)
            // limit 输入为字符串，再转换为整数
            limitStr := getUserInput("增补模式：处理的图片数量上限（0 表示不限制）", fmt.Sprintf("%d", limit))
            if v, err := strconv.Atoi(strings.TrimSpace(limitStr)); err == nil {
                limit = v
            }
            fmt.Println("\n=== 初始化配置 ===")
            fmt.Printf("MongoDB URI: %s\n", mongoURI)
            fmt.Printf("数据库名称: %s\n", dbName)
            fmt.Printf("运行模式: %s\n", mode)
            fmt.Printf("dry-run: %v\n", dryRun)
            fmt.Printf("limit: %d\n", limit)
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
    if err := models.EnsureIssueIndexes(); err != nil {
        log.Fatalf("创建 Issue 索引失败: %v", err)
    }
    if err := models.EnsureComparisonIndexes(); err != nil {
        log.Fatalf("创建 Comparison 索引失败: %v", err)
    }
    if err := models.EnsureDetectionIndexes(); err != nil {
        log.Fatalf("创建 Detection 索引失败: %v", err)
    }
    fmt.Println("已确保 issues/comparisons/detections 索引")

    if mode == "structure-only" {
        // 仅初始化集合，不进行任何数据导入或增补
        collections := []string{"scenes", "styleImages", "images", "issues", "comparisons", "detections"}
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

    if mode == "augment-existing" {
        // 增补模式：不清理集合，不读文件，仅为现有图片补数据
        augmentExisting(ctx, dryRun, limit)
        fmt.Println("增补操作完成！")
        return
    }

    // full-init 模式：清库并按文件系统导入
    collectionsToDrop := []string{"scenes", "styleImages", "images", "issues", "comparisons"}
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

    // 如果需要，填充额外测试数据
    if seedExtra {
        seedExtraData()
    }

    fmt.Println("数据库初始化完成!")
}
