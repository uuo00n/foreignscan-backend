package models

import (
	"time"

	"foreignscan/internal/database"
	"foreignscan/pkg/utils"

	"gorm.io/gorm"
)

// StyleImage 样式图模型
type StyleImage struct {
	ID          string    `gorm:"primaryKey;type:varchar(24)" json:"id"`
	SceneID     string    `gorm:"index;type:varchar(24)" json:"sceneId"` // 关联的场景ID
	Name        string    `gorm:"type:varchar(255)" json:"name"`         // 样式图名称
	Description string    `gorm:"type:text" json:"description"`          // 样式图描述
	Filename    string    `gorm:"type:varchar(255)" json:"filename"`     // 文件名
	Path        string    `gorm:"type:text" json:"path"`                 // 文件路径
	CreatedAt   time.Time `json:"createdAt"`                             // 创建时间
	UpdatedAt   time.Time `json:"updatedAt"`                             // 更新时间
}

// BeforeCreate GORM hook
func (s *StyleImage) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == "" {
		s.ID = utils.GenerateObjectID()
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}
	s.UpdatedAt = time.Now()
	return
}

// FindAllStyleImages 获取所有样式图
func FindAllStyleImages() ([]StyleImage, error) {
	db := database.GetDB()
	var styleImages []StyleImage
	err := db.Order("created_at DESC").Find(&styleImages).Error
	if styleImages == nil {
		styleImages = []StyleImage{}
	}
	return styleImages, err
}

// FindStyleImagesBySceneID 根据场景ID查找样式图
func FindStyleImagesBySceneID(sceneID string) ([]StyleImage, error) {
	db := database.GetDB()
	var styleImages []StyleImage
	err := db.Where("scene_id = ?", sceneID).Order("created_at DESC").Find(&styleImages).Error
	if styleImages == nil {
		styleImages = []StyleImage{}
	}
	return styleImages, err
}

// FindStyleImageByID 根据ID查找样式图
func FindStyleImageByID(id string) (*StyleImage, error) {
	db := database.GetDB()
	var styleImage StyleImage
	err := db.First(&styleImage, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &styleImage, nil
}

// Save 保存样式图
func (s *StyleImage) Save() error {
	db := database.GetDB()
	return db.Create(s).Error
}

// Update 更新样式图
func (s *StyleImage) Update() error {
	db := database.GetDB()
	s.UpdatedAt = time.Now()
	return db.Save(s).Error
}

// DeleteStyleImage 删除样式图
func DeleteStyleImage(id string) error {
	db := database.GetDB()
	return db.Delete(&StyleImage{}, "id = ?", id).Error
}
