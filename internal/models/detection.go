package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"foreignscan/internal/database"
	"foreignscan/pkg/utils"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// BoundingBox 表示边界框
type BoundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// PolygonPoint 可选的分割或多边形坐标
type PolygonPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// DetectionItem 单个检测框/目标的结果
type DetectionItem struct {
	Class      string         `json:"class"`
	ClassID    int            `json:"classId"`
	Confidence float64        `json:"confidence"`
	BBox       BoundingBox    `json:"bbox"`
	Polygon    []PolygonPoint `json:"polygon,omitempty"`
	Note       string         `json:"note,omitempty"`
}

// DetectionSummary 用于查询与展示的汇总字段
type DetectionSummary struct {
	HasIssue    bool    `json:"hasIssue"`
	IssueType   string  `json:"issueType"`
	ObjectCount int     `json:"objectCount"`
	AvgScore    float64 `json:"avgScore"`
}

// Value 实现 driver.Valuer 接口，用于保存到数据库
func (s DetectionSummary) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan 实现 sql.Scanner 接口，用于从数据库读取
func (s *DetectionSummary) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}
	return json.Unmarshal(bytes, s)
}

// DetectionRun 一次推理运行的结果
type DetectionRun struct {
	ID                  string                             `gorm:"primaryKey;type:varchar(24)" json:"id"`
	RunID               string                             `gorm:"uniqueIndex;type:varchar(64)" json:"runId,omitempty"`
	ImageID             string                             `gorm:"index;type:varchar(24)" json:"imageId"`
	RoomID              string                             `gorm:"index;type:varchar(24)" json:"roomId"`
	PointID             string                             `gorm:"index;type:varchar(24)" json:"pointId"`
	SourceFilename      string                             `gorm:"type:varchar(255)" json:"sourceFilename"`
	SourcePath          string                             `gorm:"type:text" json:"sourcePath"`
	ProcessedFilename   string                             `gorm:"type:varchar(255)" json:"processedFilename"`
	ProcessedPath       string                             `gorm:"type:text" json:"processedPath"`
	ModelName           string                             `gorm:"type:varchar(64)" json:"modelName"`
	ModelVersion        string                             `gorm:"type:varchar(64)" json:"modelVersion"`
	Device              string                             `gorm:"type:varchar(32)" json:"device,omitempty"`
	IoUThreshold        float64                            `json:"iouThreshold"`
	ConfidenceThreshold float64                            `json:"confidenceThreshold"`
	InferenceTimeMs     int64                              `json:"inferenceTimeMs"`
	Items               datatypes.JSONSlice[DetectionItem] `gorm:"type:jsonb" json:"items"`
	Summary             DetectionSummary                   `gorm:"type:jsonb" json:"summary"`
	CreatedAt           time.Time                          `json:"createdAt"`
	UpdatedAt           time.Time                          `json:"updatedAt"`
}

// BeforeCreate GORM hook to generate ID
func (r *DetectionRun) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == "" {
		r.ID = utils.GenerateObjectID()
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now()
	}
	r.UpdatedAt = time.Now()
	return
}

// AfterFind GORM hook to ensure slices are not nil
func (r *DetectionRun) AfterFind(tx *gorm.DB) (err error) {
	if r.Items == nil {
		r.Items = datatypes.JSONSlice[DetectionItem]{}
	}
	return
}

// InsertDetectionRun 插入一次检测结果
func InsertDetectionRun(run *DetectionRun) (string, error) {
	if run == nil {
		return "", errors.New("nil detection run")
	}

	db := database.GetDB()

	// 使用Save（Upsert）如果 RunID 存在
	if run.RunID != "" {
		// 检查是否存在
		var count int64
		db.Model(&DetectionRun{}).Where("run_id = ?", run.RunID).Count(&count)
		if count > 0 {
			// 更新
			// 注意：这里需要先查出旧的ID，或者不更新ID
			var existing DetectionRun
			if err := db.Where("run_id = ?", run.RunID).First(&existing).Error; err != nil {
				return "", err
			}
			run.ID = existing.ID
			// 更新摘要
			if err := UpdateImageDetectionSummary(run.ImageID, run.Summary.HasIssue, run.Summary.IssueType, true); err != nil {
				return "", err
			}
			// 只更新必要字段
			return existing.ID, db.Model(&existing).Updates(run).Error
		}
	}

	if err := db.Create(run).Error; err != nil {
		return "", err
	}

	if err := UpdateImageDetectionSummary(run.ImageID, run.Summary.HasIssue, run.Summary.IssueType, true); err != nil {
		return "", err
	}

	return run.ID, nil
}

// QueryDetections 查询检测结果
func QueryDetections(page, pageSize int, imageID, roomID, pointID string) ([]DetectionRun, int64, error) {
	db := database.GetDB()
	var runs []DetectionRun
	var total int64

	query := db.Model(&DetectionRun{})
	if imageID != "" {
		query = query.Where("image_id = ?", imageID)
	}
	if roomID != "" {
		query = query.Where("room_id = ?", roomID)
	}
	if pointID != "" {
		query = query.Where("point_id = ?", pointID)
	}

	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&runs).Error

	// 确保切片不为nil
	if runs == nil {
		runs = []DetectionRun{}
	}

	return runs, total, err
}

// GetDetectionByID 获取单个检测记录
func GetDetectionByID(id string) (*DetectionRun, error) {
	db := database.GetDB()
	var run DetectionRun
	if err := db.First(&run, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

// FindLatestDetectionByImageID 获取图片的最新检测结果
func FindLatestDetectionByImageID(imageID string) (*DetectionRun, error) {
	db := database.GetDB()
	var run DetectionRun
	err := db.Where("image_id = ?", imageID).Order("created_at DESC").First(&run).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // 不存在时不报错，返回nil
		}
		return nil, err
	}
	return &run, nil
}
