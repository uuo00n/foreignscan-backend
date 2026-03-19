package models

import (
	"time"

	"foreignscan/internal/database"
	"foreignscan/pkg/utils"

	"gorm.io/gorm"
)

const (
	PadBindingKeyStatusActive      = "active"
	PadBindingKeyStatusUsed        = "used"
	PadBindingKeyStatusInvalidated = "invalidated"
	PadBindingKeyStatusExpired     = "expired"
)

// PadBindingKey 房间一次性绑定码（仅用于绑定流程，不用于直接鉴权）
type PadBindingKey struct {
	ID         string     `gorm:"primaryKey;type:varchar(24)" json:"id"`
	RoomID     string     `gorm:"index;type:varchar(24);not null" json:"roomId"`
	LookupHash string     `gorm:"uniqueIndex;type:char(64);not null" json:"-"`
	ExpiresAt  time.Time  `gorm:"index;not null" json:"expiresAt"`
	UsedAt     *time.Time `json:"usedAt,omitempty"`
	UsedPadID  *string    `gorm:"type:varchar(128)" json:"usedPadId,omitempty"`
	Status     string     `gorm:"index;type:varchar(24);not null" json:"status"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

func (k *PadBindingKey) BeforeCreate(tx *gorm.DB) (err error) {
	if k.ID == "" {
		k.ID = utils.GenerateObjectID()
	}
	if k.Status == "" {
		k.Status = PadBindingKeyStatusActive
	}
	if k.CreatedAt.IsZero() {
		k.CreatedAt = time.Now()
	}
	k.UpdatedAt = time.Now()
	return
}

func FindPadBindingKeyByLookupHash(lookupHash string) (*PadBindingKey, error) {
	db := database.GetDB()
	var key PadBindingKey
	if err := db.First(&key, "lookup_hash = ?", lookupHash).Error; err != nil {
		return nil, err
	}
	return &key, nil
}

func InvalidateActivePadBindingKeysByRoom(roomID string) error {
	db := database.GetDB()
	now := time.Now()
	return db.Model(&PadBindingKey{}).
		Where("room_id = ? AND status = ?", roomID, PadBindingKeyStatusActive).
		Updates(map[string]interface{}{
			"status":     PadBindingKeyStatusInvalidated,
			"updated_at": now,
		}).Error
}

func MarkPadBindingKeyUsed(id, padID string, usedAt time.Time) error {
	db := database.GetDB()
	return db.Model(&PadBindingKey{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      PadBindingKeyStatusUsed,
			"used_at":     usedAt,
			"used_pad_id": padID,
			"updated_at":  usedAt,
		}).Error
}

func MarkPadBindingKeyExpired(id string) error {
	db := database.GetDB()
	now := time.Now()
	return db.Model(&PadBindingKey{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     PadBindingKeyStatusExpired,
			"updated_at": now,
		}).Error
}
