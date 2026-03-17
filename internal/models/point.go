package models

import (
	"time"

	"foreignscan/internal/database"
	"foreignscan/pkg/utils"

	"gorm.io/gorm"
)

// Point 点位模型（隶属一个房间）
type Point struct {
	ID        string    `gorm:"primaryKey;type:varchar(24)" json:"id"`
	RoomID    string    `gorm:"index;type:varchar(24);not null" json:"roomId"`
	Name      string    `gorm:"type:varchar(255);not null" json:"name"`
	Code      string    `gorm:"type:varchar(64)" json:"code"`
	Location  string    `gorm:"type:varchar(255)" json:"location"`
	IsActive  bool      `gorm:"default:true" json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (p *Point) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == "" {
		p.ID = utils.GenerateObjectID()
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now()
	}
	p.UpdatedAt = time.Now()
	return
}

func FindPointsByRoomID(roomID string) ([]Point, error) {
	db := database.GetDB()
	var points []Point
	err := db.Where("room_id = ?", roomID).Order("created_at DESC").Find(&points).Error
	if points == nil {
		points = []Point{}
	}
	return points, err
}

func FindPointByID(id string) (*Point, error) {
	db := database.GetDB()
	var point Point
	if err := db.First(&point, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &point, nil
}

func FindPointByIDAndRoom(pointID, roomID string) (*Point, error) {
	db := database.GetDB()
	var point Point
	if err := db.First(&point, "id = ? AND room_id = ?", pointID, roomID).Error; err != nil {
		return nil, err
	}
	return &point, nil
}

func (p *Point) Save() error {
	db := database.GetDB()
	return db.Create(p).Error
}

func (p *Point) Update() error {
	db := database.GetDB()
	p.UpdatedAt = time.Now()
	return db.Save(p).Error
}

func DeletePoint(id string) error {
	db := database.GetDB()
	return db.Delete(&Point{}, "id = ?", id).Error
}

type PointAssociationCounts struct {
	StyleImages int64 `json:"styleImages"`
	Images      int64 `json:"images"`
	Detections  int64 `json:"detections"`
}

func CountPointAssociations(pointID string) (PointAssociationCounts, error) {
	db := database.GetDB()
	counts := PointAssociationCounts{}

	if err := db.Model(&StyleImage{}).Where("point_id = ?", pointID).Count(&counts.StyleImages).Error; err != nil {
		return counts, err
	}
	if err := db.Model(&Image{}).Where("point_id = ?", pointID).Count(&counts.Images).Error; err != nil {
		return counts, err
	}
	if err := db.Model(&DetectionRun{}).Where("point_id = ?", pointID).Count(&counts.Detections).Error; err != nil {
		return counts, err
	}
	return counts, nil
}
