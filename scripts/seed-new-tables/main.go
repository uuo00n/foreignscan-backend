package main

import (
    "fmt"
    "log"
    "path/filepath"
    "time"

    "foreignscan/internal/database"
    "foreignscan/internal/models"
)

// 说明：
// 该脚本用于为新建的 issues 与 comparisons 两个集合各插入三条测试数据，
// 不修改、删除任何现有数据，仅做“新增”操作。
//
// 使用方式：
// 1) 在项目根目录运行（任选其一）：
//    go run ./scripts/seed-new-tables
//    或
//    go run scripts/seed-new-tables/main.go
// 2) 运行后可通过接口验证：
//    GET  http://localhost:3000/api/issues
//    GET  http://localhost:3000/api/comparisons
//    GET  http://localhost:3000/api/images/{imageId}/issues
//
// 注意：
// - 如果当前数据库没有图片数据（images集合为空），脚本会提示并退出，不会强行创建图片。
// - 插入的数据使用 TEST 前缀，便于后续清理。

func main() {
    // 1. 连接数据库
    if err := database.Connect(); err != nil {
        log.Fatalf("连接数据库失败: %v", err)
    }
    defer func() {
        if err := database.Close(); err != nil {
            log.Printf("关闭数据库连接失败: %v", err)
        }
    }()

    // 2. 初始化新表索引（幂等，不影响既有数据）
    if err := models.EnsureIssueIndexes(); err != nil {
        log.Printf("初始化Issue索引失败: %v", err)
    }
    if err := models.EnsureComparisonIndexes(); err != nil {
        log.Printf("初始化Comparison索引失败: %v", err)
    }

    // 3. 获取现有图片，选择前3张作为测试对象
    images, err := models.FindAll()
    if err != nil {
        log.Fatalf("查询图片失败: %v", err)
    }
    if len(images) == 0 {
        log.Println("当前数据库中没有图片（images集合为空），已跳过测试数据插入。请先上传图片再运行本脚本。")
        return
    }

    n := 3
    if len(images) < n {
        n = len(images)
    }

    types := []string{"TEST-类型A", "TEST-类型B", "TEST-类型C"}
    descs := []string{"TEST-设备皮带磨损说明", "TEST-护罩缺失说明", "TEST-油污泄漏说明"}

    // 4. 依次为选取的图片插入问题与对比记录
    for i := 0; i < n; i++ {
        img := images[i]

        // 4.1 插入问题记录
        issue := &models.Issue{
            ImageID:     img.ID,
            SceneID:     img.SceneID,
            Type:        types[i%len(types)],
            Description: descs[i%len(descs)],
            CreatedAt:   time.Now(),
            UpdatedAt:   time.Now(),
        }
        issueID, err := models.InsertIssue(issue)
        if err != nil {
            log.Fatalf("插入问题记录失败(imageId=%s): %v", img.ID.Hex(), err)
        }

        // 4.2 插入对比记录
        // beforePath 使用图片原始路径，afterPath 使用同目录下的 processed_ 前缀文件名（仅做示例，不要求文件真实存在）
        processedFilename := fmt.Sprintf("processed_%s", img.Filename)
        afterPath := filepath.Join("uploads/images", img.SceneID.Hex(), processedFilename)

        comp := &models.Comparison{
            ImageID:    img.ID,
            BeforePath: img.Path,
            AfterPath:  afterPath,
            DiffInfo:   map[string]interface{}{"note": "TEST-示例对比，无真实文件"},
            Remark:     fmt.Sprintf("TEST-对比记录-%d", i+1),
            CreatedAt:  time.Now(),
        }
        compID, err := models.InsertComparison(comp)
        if err != nil {
            log.Fatalf("插入对比记录失败(imageId=%s): %v", img.ID.Hex(), err)
        }

        // 4.3 控制台提示
        log.Printf("已插入 Issue: %s, Comparison: %s, 对应图片: %s", issueID.Hex(), compID.Hex(), img.ID.Hex())
    }

    log.Printf("完成：issues/comparisons 各插入 %d 条测试数据（均以 TEST- 前缀标注）。", n)
}