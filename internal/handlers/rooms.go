package handlers

import (
	"errors"
	"net/http"
	"time"

	"foreignscan/internal/database"
	"foreignscan/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type pointView struct {
	ID        string `json:"id"`
	RoomID    string `json:"roomId"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	Location  string `json:"location"`
	IsActive  bool   `json:"isActive"`
	HasStyle  bool   `json:"hasStyleImage"`
	StyleID   string `json:"styleImageId,omitempty"`
	StylePath string `json:"styleImagePath,omitempty"`
}

type roomTreeView struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	ModelPath   string      `json:"modelPath"`
	Status      string      `json:"status"`
	IsActive    bool        `json:"isActive"`
	Points      []pointView `json:"points"`
}

// GetRoomsTree godoc
// @Summary 获取房间-点位树
// @Tags rooms
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /rooms/tree [get]
func GetRoomsTree(c *gin.Context) {
	rooms, err := models.FindAllRooms()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "获取房间失败: " + err.Error()})
		return
	}

	result := make([]roomTreeView, 0, len(rooms))
	for _, room := range rooms {
		points, err := models.FindPointsByRoomID(room.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "获取点位失败: " + err.Error()})
			return
		}
		pointViews := make([]pointView, 0, len(points))
		for _, p := range points {
			v := pointView{ID: p.ID, RoomID: p.RoomID, Name: p.Name, Code: p.Code, Location: p.Location, IsActive: p.IsActive}
			if style, e := models.FindStyleImageByPointID(p.ID); e == nil && style != nil {
				v.HasStyle = true
				v.StyleID = style.ID
				v.StylePath = style.Path
			} else if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "读取点位对照图失败: " + e.Error()})
				return
			}
			pointViews = append(pointViews, v)
		}
		result = append(result, roomTreeView{
			ID:          room.ID,
			Name:        room.Name,
			Description: room.Description,
			ModelPath:   room.ModelPath,
			Status:      room.Status,
			IsActive:    room.IsActive,
			Points:      pointViews,
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "rooms": result})
}

type importRoomsRequest struct {
	Rooms []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		ModelPath   string `json:"model_path"`
		Status      string `json:"status"`
		Points      []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Code     string `json:"code"`
			Location string `json:"location"`
		} `json:"points"`
	} `json:"rooms"`
}

// ImportRooms godoc
// @Summary 导入房间点位配置
// @Tags rooms
// @Accept json
// @Produce json
// @Param body body importRoomsRequest true "房间配置"
// @Success 200 {object} map[string]interface{}
// @Router /rooms/import [post]
func ImportRooms(c *gin.Context) {
	var req importRoomsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求数据无效: " + err.Error()})
		return
	}
	if len(req.Rooms) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "rooms 不能为空"})
		return
	}

	db := database.GetDB()
	if err := db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		if err := tx.Exec("DELETE FROM detection_runs").Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM images").Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM style_images").Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM points").Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM rooms").Error; err != nil {
			return err
		}

		for _, inRoom := range req.Rooms {
			room := models.Room{ID: inRoom.ID, Name: inRoom.Name, Description: inRoom.Description, ModelPath: inRoom.ModelPath, Status: inRoom.Status, IsActive: true, CreatedAt: now, UpdatedAt: now}
			if room.Status == "" {
				room.Status = "enabled"
			}
			if err := tx.Create(&room).Error; err != nil {
				return err
			}
			for _, inPoint := range inRoom.Points {
				point := models.Point{ID: inPoint.ID, RoomID: room.ID, Name: inPoint.Name, Code: inPoint.Code, Location: inPoint.Location, IsActive: true, CreatedAt: now, UpdatedAt: now}
				if err := tx.Create(&point).Error; err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "导入失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "导入成功（已重建 rooms/points 并清空历史检测数据）"})
}
