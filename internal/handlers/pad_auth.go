package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"foreignscan/internal/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	padIDHeader  = "X-Pad-Id"
	padKeyHeader = "X-Pad-Key"
)

func resolveRoomByPadHeaders(c padHeaderReader) (*models.Room, bool, int, string) {
	padID := strings.TrimSpace(c.GetHeader(padIDHeader))
	padKey := strings.TrimSpace(c.GetHeader(padKeyHeader))

	if padID == "" && padKey == "" {
		return nil, false, 0, ""
	}
	if padID == "" || padKey == "" {
		return nil, true, http.StatusUnauthorized, "X-Pad-Id 与 X-Pad-Key 必须同时提供"
	}

	room, err := models.FindRoomByPadID(padID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, true, http.StatusUnauthorized, "pad 未绑定房间"
		}
		return nil, true, http.StatusInternalServerError, "查询 pad 绑定关系失败: " + err.Error()
	}

	if strings.TrimSpace(room.PadKeyHash) == "" {
		return nil, true, http.StatusUnauthorized, "pad 密钥未配置"
	}
	if err := bcrypt.CompareHashAndPassword([]byte(room.PadKeyHash), []byte(padKey)); err != nil {
		return nil, true, http.StatusUnauthorized, "pad 鉴权失败"
	}

	_ = models.TouchRoomPadLastSeen(room.ID, time.Now())
	return room, true, 0, ""
}

func resolveRoomByPadHeadersRequired(c padHeaderReader) (*models.Room, int, string) {
	room, usePadAuth, status, msg := resolveRoomByPadHeaders(c)
	if status != 0 {
		return nil, status, msg
	}
	if !usePadAuth || room == nil {
		return nil, http.StatusUnauthorized, "缺少 X-Pad-Id 或 X-Pad-Key"
	}
	return room, 0, ""
}

func hashPadKey(raw string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

type padHeaderReader interface {
	GetHeader(string) string
}
