package models

import (
	"context"
	"time"

	"foreignscan/internal/database"
	"foreignscan/pkg/utils"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Image 图片模型
type Image struct {
	ID               string                             `gorm:"primaryKey;type:varchar(24)" json:"id"`
	SequenceNumber   int                                `gorm:"index" json:"sequenceNumber"`
	SceneID          string                             `gorm:"index;type:varchar(24)" json:"sceneId"` // 关联的场景ID
	Timestamp        time.Time                          `json:"timestamp"`
	Location         string                             `gorm:"type:varchar(255)" json:"location"`
	Filename         string                             `gorm:"type:varchar(255)" json:"filename"`
	Path             string                             `gorm:"type:text" json:"path"`
	IsDetected       bool                               `json:"isDetected"`
	HasIssue         bool                               `json:"hasIssue"`
	IssueType        string                             `gorm:"type:varchar(50)" json:"issueType"`
	Status           string                             `gorm:"type:varchar(50)" json:"status"` // 图片检测状态：未检测/已检测
	DetectionResults datatypes.JSONSlice[DetectionItem] `gorm:"type:jsonb" json:"detectionResults"`
	CreatedAt        time.Time                          `json:"createdAt"` // 创建时间
	UpdatedAt        time.Time                          `json:"updatedAt"` // 更新时间
}

// 定义图片状态常量，避免魔法字符串
const (
	ImageStatusUndetected = "未检测"
	ImageStatusDetected   = "已检测"
	// 兼容旧数据（读取时可转换为已检测）
	ImageStatusQualified = "合格"
	ImageStatusDefective = "缺陷"
)

// BeforeCreate GORM hook to generate ID
func (i *Image) BeforeCreate(tx *gorm.DB) (err error) {
	if i.ID == "" {
		i.ID = utils.GenerateObjectID()
	}
	if i.CreatedAt.IsZero() {
		i.CreatedAt = time.Now()
	}
	i.UpdatedAt = time.Now()
	if i.DetectionResults == nil {
		i.DetectionResults = datatypes.JSONSlice[DetectionItem]{}
	}
	return
}

// AfterFind GORM hook to ensure slices are not nil
func (i *Image) AfterFind(tx *gorm.DB) (err error) {
	if i.DetectionResults == nil {
		i.DetectionResults = datatypes.JSONSlice[DetectionItem]{}
	}
	return
}

// Save 保存图片
func (i *Image) Save() error {
	db := database.GetDB()
	return db.Save(i).Error
}

// GetNextSequence 获取下一个序列号
func GetNextSequence() (int, error) {
	db := database.GetDB()
	var maxSeq int
	// 使用Coalesce处理空表情况
	err := db.Model(&Image{}).Select("COALESCE(MAX(sequence_number), 0)").Scan(&maxSeq).Error
	if err != nil {
		return 0, err
	}
	return maxSeq + 1, nil
}

// FindAll 获取所有图片
func FindAll() ([]Image, error) {
	db := database.GetDB()
	var images []Image
	err := db.Order("sequence_number DESC").Find(&images).Error
	if images == nil {
		images = []Image{}
	}
	return images, err
}

// FindByID 根据ID查找图片
func FindByID(id string) (*Image, error) {
	db := database.GetDB()
	var image Image
	err := db.First(&image, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &image, nil
}

// FindBySceneID 根据场景ID查找图片
func FindBySceneID(sceneID string) ([]Image, error) {
	db := database.GetDB()
	var images []Image
	err := db.Where("scene_id = ?", sceneID).Order("sequence_number DESC").Find(&images).Error
	if images == nil {
		images = []Image{}
	}
	return images, err
}

// UpdateImageDetectionSummary 更新图片检测结果摘要
func UpdateImageDetectionSummary(imageID string, hasIssue bool, issueType string, isDetected bool) error {
	db := database.GetDB()

	updates := map[string]interface{}{
		"has_issue":  hasIssue,
		"issue_type": issueType,
		"updated_at": time.Now(),
	}

	if isDetected {
		updates["is_detected"] = true
		updates["status"] = ImageStatusDetected

		// 同时更新 DetectionResults 为最新的一次检测记录
		// 这里需要先查出来最新的 run
		run, err := FindLatestDetectionByImageID(imageID)
		if err == nil && run != nil {
			updates["detection_results"] = run.Items
		}
	}

	return db.Model(&Image{}).Where("id = ?", imageID).Updates(updates).Error
}

// FindImagesByDate 根据日期查找图片
func FindImagesByDate(dateStr string) ([]Image, error) {
	db := database.GetDB()

	// 解析日期，假设格式为 YYYY-MM-DD
	startTime, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, err
	}
	endTime := startTime.Add(24 * time.Hour)

	var images []Image
	err = db.Where("created_at >= ? AND created_at < ?", startTime, endTime).
		Order("sequence_number DESC").
		Find(&images).Error

	if images == nil {
		images = []Image{}
	}
	return images, err
}

// FindImagesByDateAndScene 根据日期和场景查找图片
func FindImagesByDateAndScene(dateStr string, sceneID string) ([]Image, error) {
	db := database.GetDB()

	query := db.Model(&Image{})

	if dateStr != "" {
		startTime, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, err
		}
		endTime := startTime.Add(24 * time.Hour)
		query = query.Where("created_at >= ? AND created_at < ?", startTime, endTime)
	}

	if sceneID != "" {
		query = query.Where("scene_id = ?", sceneID)
	}

	var images []Image
	err := query.Order("sequence_number DESC").Find(&images).Error
	if images == nil {
		images = []Image{}
	}
	return images, err
}

// FindImagesByFilter 综合筛选图片
func FindImagesByFilter(status string, startTime, endTime *time.Time, page, pageSize int) ([]Image, int64, error) {
	db := database.GetDB()
	var images []Image
	var total int64

	query := db.Model(&Image{})

	if status != "" {
		if status == "has_issue" {
			query = query.Where("has_issue = ?", true)
		} else if status == "no_issue" {
			query = query.Where("has_issue = ? AND is_detected = ?", false, true)
		} else if status == "undetected" {
			query = query.Where("is_detected = ?", false)
		}
	}

	if startTime != nil {
		query = query.Where("created_at >= ?", *startTime)
	}
	if endTime != nil {
		query = query.Where("created_at <= ?", *endTime)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&images).Error

	if images == nil {
		images = []Image{}
	}
	return images, total, err
}

// GetFirstImageBySceneID 获取场景的第一张图片
func GetFirstImageBySceneID(sceneID string) (*Image, error) {
	db := database.GetDB()
	var image Image
	// 按序列号最小的（最早的）
	err := db.Where("scene_id = ?", sceneID).Order("sequence_number ASC").First(&image).Error
	if err != nil {
		return nil, err
	}
	return &image, nil
}

// BatchGetFirstImagesBySceneIDs 批量获取场景首图
func BatchGetFirstImagesBySceneIDs(sceneIDs []string) (map[string]Image, error) {
	if len(sceneIDs) == 0 {
		return map[string]Image{}, nil
	}

	db := database.GetDB()

	// 使用窗口函数获取每个分组的第一条
	// SQL: SELECT * FROM (SELECT *, ROW_NUMBER() OVER (PARTITION BY scene_id ORDER BY sequence_number ASC) as rn FROM images WHERE scene_id IN (?)) sub WHERE rn = 1

	var images []Image
	// GORM 不直接支持窗口函数的简单构造，可以用原生SQL
	// 或者简单点：如果场景不多，循环查；如果多，用 DISTINCT ON (Postgres特性)

	err := db.Raw(`
		SELECT DISTINCT ON (scene_id) * 
		FROM images 
		WHERE scene_id IN ? 
		ORDER BY scene_id, sequence_number ASC
	`, sceneIDs).Scan(&images).Error

	if err != nil {
		return nil, err
	}

	result := make(map[string]Image)
	for _, img := range images {
		result[img.SceneID] = img
	}
	return result, nil
}

// CountImagesByStatus 统计不同状态的图片数量
func CountImagesByStatus(ctx context.Context) (map[string]int64, error) {
	db := database.GetDB()
	var results []struct {
		IsDetected bool
		HasIssue   bool
		Count      int64
	}

	// 聚合查询
	err := db.Model(&Image{}).Select("is_detected, has_issue, count(*) as count").Group("is_detected, has_issue").Scan(&results).Error
	if err != nil {
		return nil, err
	}

	stats := map[string]int64{
		"total":      0,
		"undetected": 0,
		"has_issue":  0,
		"no_issue":   0,
	}

	for _, r := range results {
		stats["total"] += r.Count
		if !r.IsDetected {
			stats["undetected"] += r.Count
		} else {
			if r.HasIssue {
				stats["has_issue"] += r.Count
			} else {
				stats["no_issue"] += r.Count
			}
		}
	}

	return stats, nil
}
