package models

import (
	"time"

	"foreignscan/internal/database"
	"foreignscan/pkg/utils"

	"gorm.io/gorm"
)

// Scene 场景模型
type Scene struct {
	ID          string    `gorm:"primaryKey;type:varchar(24)" json:"id"`
	Name        string    `gorm:"type:varchar(255)" json:"name"`     // 场景名称
	Description string    `gorm:"type:text" json:"description"`      // 场景描述
	Location    string    `gorm:"type:varchar(255)" json:"location"` // 场景位置
	CreatedAt   time.Time `json:"createdAt"`                         // 创建时间
	UpdatedAt   time.Time `json:"updatedAt"`                         // 更新时间
	Status      string    `gorm:"type:varchar(50)" json:"status"`    // 场景状态
}

// BeforeCreate GORM hook
func (s *Scene) BeforeCreate(tx *gorm.DB) (err error) {
	if s.ID == "" {
		s.ID = utils.GenerateObjectID()
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}
	s.UpdatedAt = time.Now()
	return
}

// FindAllScenes 获取所有场景
func FindAllScenes() ([]Scene, error) {
	db := database.GetDB()
	var scenes []Scene
	err := db.Order("created_at DESC").Find(&scenes).Error
	if scenes == nil {
		scenes = []Scene{}
	}
	return scenes, err
}

// FindSceneByID 根据ID查找场景
func FindSceneByID(id string) (*Scene, error) {
	db := database.GetDB()
	var scene Scene
	err := db.First(&scene, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &scene, nil
}

// Save 保存场景
func (s *Scene) Save() error {
	db := database.GetDB()
	return db.Create(s).Error
}

// Update 更新场景
func (s *Scene) Update() error {
	db := database.GetDB()
	s.UpdatedAt = time.Now()
	return db.Save(s).Error
}

// DeleteScene 删除场景
func DeleteScene(id string) error {
	db := database.GetDB()
	return db.Delete(&Scene{}, "id = ?", id).Error
}
