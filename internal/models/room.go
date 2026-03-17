package models

import (
	"time"

	"foreignscan/internal/database"
	"foreignscan/pkg/utils"

	"gorm.io/gorm"
)

// Room 机房/房间模型（绑定一个推理模型）
type Room struct {
	ID          string    `gorm:"primaryKey;type:varchar(24)" json:"id"`
	Name        string    `gorm:"type:varchar(255);not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	ModelPath   string    `gorm:"type:text;not null" json:"modelPath"`
	Status      string    `gorm:"type:varchar(50)" json:"status"`
	IsActive    bool      `gorm:"default:true" json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (r *Room) BeforeCreate(tx *gorm.DB) (err error) {
	if r.ID == "" {
		r.ID = utils.GenerateObjectID()
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now()
	}
	r.UpdatedAt = time.Now()
	return
}

func FindAllRooms() ([]Room, error) {
	db := database.GetDB()
	var rooms []Room
	err := db.Order("created_at DESC").Find(&rooms).Error
	if rooms == nil {
		rooms = []Room{}
	}
	return rooms, err
}

func FindRoomByID(id string) (*Room, error) {
	db := database.GetDB()
	var room Room
	if err := db.First(&room, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *Room) Save() error {
	db := database.GetDB()
	return db.Create(r).Error
}

func (r *Room) Update() error {
	db := database.GetDB()
	r.UpdatedAt = time.Now()
	return db.Save(r).Error
}

func DeleteRoom(id string) error {
	db := database.GetDB()
	return db.Delete(&Room{}, "id = ?", id).Error
}
