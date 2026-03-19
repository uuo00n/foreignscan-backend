package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"foreignscan/internal/config"

	"github.com/gin-gonic/gin"
)

type roomModelBindingRequest struct {
	ModelPath string `json:"modelPath"`
}

func detectServiceBaseURL() string {
	if env := strings.TrimSpace(os.Getenv("FS_DETECT_URL")); env != "" {
		return strings.TrimRight(env, "/")
	}
	return strings.TrimRight(strings.TrimSpace(config.Get().DetectServiceURL), "/")
}

func proxyDetectService(c *gin.Context, method, endpoint string, body []byte) {
	base := detectServiceBaseURL()
	if base == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "检测服务地址未配置"})
		return
	}

	req, err := http.NewRequest(method, base+endpoint, bytes.NewReader(body))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "构造检测服务请求失败: " + err.Error()})
		return
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "检测服务不可达: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	raw, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "读取检测服务响应失败: " + readErr.Error()})
		return
	}

	if len(raw) == 0 {
		if resp.StatusCode >= 400 {
			c.JSON(resp.StatusCode, gin.H{"success": false, "message": "检测服务返回空响应"})
			return
		}
		c.JSON(resp.StatusCode, gin.H{"success": true})
		return
	}

	if json.Valid(raw) {
		c.Data(resp.StatusCode, "application/json; charset=utf-8", raw)
		return
	}

	if resp.StatusCode >= 400 {
		c.JSON(resp.StatusCode, gin.H{"success": false, "message": "检测服务返回非JSON响应", "detail": string(raw)})
		return
	}
	c.Data(resp.StatusCode, "text/plain; charset=utf-8", raw)
}

// GetRoomModels godoc
// @Summary 获取房间模型映射
// @Description 通过后端代理从检测服务读取 room_id -> modelPath 映射
// @Tags detections
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 502 {object} map[string]interface{}
// @Router /room-models [get]
func GetRoomModels(c *gin.Context) {
	proxyDetectService(c, http.MethodGet, "/api/room-models", nil)
}

// PutRoomModel godoc
// @Summary 绑定房间模型
// @Description 为指定房间绑定 pt 模型路径（由检测服务执行唯一性校验）
// @Tags detections
// @Accept json
// @Produce json
// @Param roomId path string true "房间ID"
// @Param body body roomModelBindingRequest true "模型绑定请求"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 502 {object} map[string]interface{}
// @Router /room-models/{roomId} [put]
func PutRoomModel(c *gin.Context) {
	roomID := strings.TrimSpace(c.Param("roomId"))
	if roomID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "roomId 不能为空"})
		return
	}

	var req roomModelBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求体错误"})
		return
	}
	req.ModelPath = strings.TrimSpace(req.ModelPath)
	if req.ModelPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "modelPath 不能为空"})
		return
	}

	payload, _ := json.Marshal(req)
	proxyDetectService(c, http.MethodPut, "/api/room-models/"+url.PathEscape(roomID), payload)
}

// DeleteRoomModel godoc
// @Summary 解绑房间模型
// @Description 删除房间与模型映射
// @Tags detections
// @Produce json
// @Param roomId path string true "房间ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 502 {object} map[string]interface{}
// @Router /room-models/{roomId} [delete]
func DeleteRoomModel(c *gin.Context) {
	roomID := strings.TrimSpace(c.Param("roomId"))
	if roomID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "roomId 不能为空"})
		return
	}
	proxyDetectService(c, http.MethodDelete, "/api/room-models/"+url.PathEscape(roomID), nil)
}
