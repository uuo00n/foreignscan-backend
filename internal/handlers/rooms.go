package handlers

import (
	"errors"
	"net/http"
	"strings"
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
	ID            string      `json:"id"`
	Name          string      `json:"name"`
	Description   string      `json:"description"`
	ModelPath     string      `json:"modelPath"`
	Status        string      `json:"status"`
	PadID         string      `json:"padId,omitempty"`
	PadLastSeenAt *time.Time  `json:"padLastSeenAt,omitempty"`
	IsActive      bool        `json:"isActive"`
	Points        []pointView `json:"points"`
}

func stringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func buildPointViews(roomID string) ([]pointView, error) {
	points, err := models.FindPointsByRoomID(roomID)
	if err != nil {
		return nil, err
	}

	pointViews := make([]pointView, 0, len(points))
	for _, p := range points {
		v := pointView{ID: p.ID, RoomID: p.RoomID, Name: p.Name, Code: p.Code, Location: p.Location, IsActive: p.IsActive}
		if style, e := models.FindStyleImageByPointID(p.ID); e == nil && style != nil {
			v.HasStyle = true
			v.StyleID = style.ID
			v.StylePath = style.Path
		} else if e != nil && !errors.Is(e, gorm.ErrRecordNotFound) {
			return nil, e
		}
		pointViews = append(pointViews, v)
	}
	return pointViews, nil
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
		pointViews, e := buildPointViews(room.ID)
		if e != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "读取点位数据失败: " + e.Error()})
			return
		}
		result = append(result, roomTreeView{
			ID:            room.ID,
			Name:          room.Name,
			Description:   room.Description,
			ModelPath:     room.ModelPath,
			Status:        room.Status,
			PadID:         stringValue(room.PadID),
			PadLastSeenAt: room.PadLastSeenAt,
			IsActive:      room.IsActive,
			Points:        pointViews,
		})
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "rooms": result})
}

// GetPadRoomContext godoc
// @Summary 获取Pad绑定房间上下文
// @Description 通过 Pad 鉴权返回绑定房间及其点位列表
// @Tags rooms
// @Produce json
// @Param X-Pad-Id header string true "Pad ID（必填）"
// @Param X-Pad-Key header string true "Pad 密钥（必填）"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /pad/room-context [get]
func GetPadRoomContext(c *gin.Context) {
	room, status, msg := resolveRoomByPadHeadersRequired(c)
	if status != 0 {
		c.JSON(status, gin.H{"success": false, "message": msg})
		return
	}

	points, err := buildPointViews(room.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "读取点位数据失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"room": gin.H{
			"id":            room.ID,
			"name":          room.Name,
			"description":   room.Description,
			"modelPath":     room.ModelPath,
			"status":        room.Status,
			"padId":         stringValue(room.PadID),
			"padLastSeenAt": room.PadLastSeenAt,
			"isActive":      room.IsActive,
		},
		"points": points,
	})
}

type importRoomsRequest struct {
	Rooms []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		ModelPath   string `json:"model_path"`
		PadID       string `json:"pad_id"`
		PadIDCamel  string `json:"padId"`
		Status      string `json:"status"`
		Points      []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Code     string `json:"code"`
			Location string `json:"location"`
		} `json:"points"`
	} `json:"rooms"`
}

type createPointRequest struct {
	Name     string `json:"name"`
	Code     string `json:"code"`
	Location string `json:"location"`
}

type patchRoomPadBindingRequest struct {
	PadID  string `json:"padId"`
	PadKey string `json:"padKey"`
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
			padID := strings.TrimSpace(inRoom.PadID)
			if padID == "" {
				padID = strings.TrimSpace(inRoom.PadIDCamel)
			}
			room := models.Room{
				ID:          inRoom.ID,
				Name:        inRoom.Name,
				Description: inRoom.Description,
				ModelPath:   inRoom.ModelPath,
				Status:      inRoom.Status,
				IsActive:    true,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			if padID != "" {
				room.PadID = &padID
			}
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

// PatchRoomPadBinding godoc
// @Summary 更新房间与Pad绑定
// @Description 为房间设置或更新 padId 与 padKey（padKey仅存哈希）
// @Tags rooms
// @Accept json
// @Produce json
// @Param roomId path string true "房间ID"
// @Param body body patchRoomPadBindingRequest true "Pad绑定信息"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /rooms/{roomId}/pad-binding [patch]
func PatchRoomPadBinding(c *gin.Context) {
	roomID := strings.TrimSpace(c.Param("roomId"))
	if roomID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "roomId 不能为空"})
		return
	}

	var req patchRoomPadBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求数据无效: " + err.Error()})
		return
	}

	padID := strings.TrimSpace(req.PadID)
	padKey := strings.TrimSpace(req.PadKey)
	if padID == "" || padKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "padId 与 padKey 不能为空"})
		return
	}

	room, err := models.FindRoomByID(roomID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "房间不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "查询房间失败: " + err.Error()})
		return
	}

	existing, err := models.FindRoomByPadID(padID)
	if err == nil && existing != nil && existing.ID != room.ID {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "padId 已绑定其他房间"})
		return
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "校验 padId 失败: " + err.Error()})
		return
	}

	keyHash, err := hashPadKey(padKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "生成 padKey 哈希失败: " + err.Error()})
		return
	}

	room.PadID = &padID
	room.PadKeyHash = keyHash
	if err := room.Update(); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
			c.JSON(http.StatusConflict, gin.H{"success": false, "message": "padId 已绑定其他房间"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "更新房间绑定失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"room": gin.H{
			"id":            room.ID,
			"name":          room.Name,
			"padId":         padID,
			"padLastSeenAt": room.PadLastSeenAt,
		},
	})
}

// CreatePoint godoc
// @Summary 新增单个点位
// @Description 在指定房间下新增一个点位（点位ID自动生成）
// @Tags rooms
// @Accept json
// @Produce json
// @Param roomId path string true "房间ID"
// @Param body body createPointRequest true "点位信息"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /rooms/{roomId}/points [post]
func CreatePoint(c *gin.Context) {
	roomID := strings.TrimSpace(c.Param("roomId"))
	if roomID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "roomId 不能为空"})
		return
	}

	var req createPointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求数据无效: " + err.Error()})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "点位名称不能为空"})
		return
	}

	if _, err := models.FindRoomByID(roomID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "房间不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "查询房间失败: " + err.Error()})
		return
	}

	point := models.Point{
		RoomID:   roomID,
		Name:     name,
		Code:     strings.TrimSpace(req.Code),
		Location: strings.TrimSpace(req.Location),
		IsActive: true,
	}
	if err := point.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "新增点位失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "point": point})
}

// DeletePoint godoc
// @Summary 删除单个点位
// @Description 仅允许删除无关联数据（样式图/图片/检测记录）的点位
// @Tags rooms
// @Accept json
// @Produce json
// @Param roomId path string true "房间ID"
// @Param pointId path string true "点位ID"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /rooms/{roomId}/points/{pointId} [delete]
func DeletePoint(c *gin.Context) {
	roomID := strings.TrimSpace(c.Param("roomId"))
	pointID := strings.TrimSpace(c.Param("pointId"))
	if roomID == "" || pointID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "roomId 与 pointId 不能为空"})
		return
	}

	point, err := models.FindPointByIDAndRoom(pointID, roomID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "点位不存在或不属于该房间"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "查询点位失败: " + err.Error()})
		return
	}

	counts, err := models.CountPointAssociations(point.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "检查点位关联数据失败: " + err.Error()})
		return
	}

	if counts.StyleImages > 0 || counts.Images > 0 || counts.Detections > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": "点位存在关联数据，禁止删除",
			"counts":  counts,
		})
		return
	}

	if err := models.DeletePoint(point.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "删除点位失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "删除成功"})
}
