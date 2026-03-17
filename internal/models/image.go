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
	RoomID           string                             `gorm:"index;type:varchar(24)" json:"roomId"`
	PointID          string                             `gorm:"index;type:varchar(24)" json:"pointId"`
	Timestamp        time.Time                          `json:"timestamp"`
	Location         string                             `gorm:"type:varchar(255)" json:"location"`
	Filename         string                             `gorm:"type:varchar(255)" json:"filename"`
	Path             string                             `gorm:"type:text" json:"path"`
	IsDetected       bool                               `json:"isDetected"`
	HasIssue         bool                               `json:"hasIssue"`
	IssueType        string                             `gorm:"type:varchar(50)" json:"issueType"`
	Status           string                             `gorm:"type:varchar(50)" json:"status"`
	DetectionResults datatypes.JSONSlice[DetectionItem] `gorm:"type:jsonb" json:"detectionResults"`
	CreatedAt        time.Time                          `json:"createdAt"`
	UpdatedAt        time.Time                          `json:"updatedAt"`
}

const (
	ImageStatusUndetected = "未检测"
	ImageStatusDetected   = "已检测"
	ImageStatusQualified  = "合格"
	ImageStatusDefective  = "缺陷"
)

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

func (i *Image) AfterFind(tx *gorm.DB) (err error) {
	if i.DetectionResults == nil {
		i.DetectionResults = datatypes.JSONSlice[DetectionItem]{}
	}
	return
}

func (i *Image) Save() error {
	db := database.GetDB()
	return db.Save(i).Error
}

func GetNextSequence() (int, error) {
	db := database.GetDB()
	var maxSeq int
	err := db.Model(&Image{}).Select("COALESCE(MAX(sequence_number), 0)").Scan(&maxSeq).Error
	if err != nil {
		return 0, err
	}
	return maxSeq + 1, nil
}

func FindAll() ([]Image, error) {
	db := database.GetDB()
	var images []Image
	err := db.Order("sequence_number DESC").Find(&images).Error
	if images == nil {
		images = []Image{}
	}
	return images, err
}

func FindByID(id string) (*Image, error) {
	db := database.GetDB()
	var image Image
	err := db.First(&image, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &image, nil
}

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
		run, err := FindLatestDetectionByImageID(imageID)
		if err == nil && run != nil {
			updates["detection_results"] = run.Items
		}
	}

	return db.Model(&Image{}).Where("id = ?", imageID).Updates(updates).Error
}

func FindImagesByDate(dateStr string) ([]Image, error) {
	db := database.GetDB()
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

func FindImagesByFilter(status, roomID, pointID string, startTime, endTime *time.Time, page, pageSize int) ([]Image, int64, error) {
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
	if roomID != "" {
		query = query.Where("room_id = ?", roomID)
	}
	if pointID != "" {
		query = query.Where("point_id = ?", pointID)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&images).Error

	if images == nil {
		images = []Image{}
	}
	return images, total, err
}

func CountImagesByStatus(ctx context.Context) (map[string]int64, error) {
	db := database.GetDB()
	var results []struct {
		IsDetected bool
		HasIssue   bool
		Count      int64
	}

	err := db.Model(&Image{}).Select("is_detected, has_issue, count(*) as count").Group("is_detected, has_issue").Scan(&results).Error
	if err != nil {
		return nil, err
	}

	stats := map[string]int64{"total": 0, "undetected": 0, "has_issue": 0, "no_issue": 0}
	for _, r := range results {
		stats["total"] += r.Count
		if !r.IsDetected {
			stats["undetected"] += r.Count
		} else if r.HasIssue {
			stats["has_issue"] += r.Count
		} else {
			stats["no_issue"] += r.Count
		}
	}

	return stats, nil
}
