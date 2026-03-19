package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"foreignscan/internal/config"
	"foreignscan/internal/models"
	"foreignscan/pkg/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

type detectServiceResponse struct {
	Success bool `json:"success"`
	Items   []struct {
		ClassId    int     `json:"classId"`
		Class_     string  `json:"class_"`
		Confidence float64 `json:"confidence"`
		Bbox       struct {
			X      float64 `json:"x"`
			Y      float64 `json:"y"`
			Width  float64 `json:"width"`
			Height float64 `json:"height"`
		} `json:"bbox"`
	} `json:"items"`
	Summary struct {
		HasIssue    bool    `json:"hasIssue"`
		IssueType   string  `json:"issueType"`
		ObjectCount int     `json:"objectCount"`
		AvgScore    float64 `json:"avgScore"`
	} `json:"summary"`
	LabeledPath string `json:"labeledPath"`
}

// Predict godoc
// @Summary 同步推理（房间+点位）
// @Tags detections
// @Accept multipart/form-data
// @Produce json
// @Param X-Pad-Id header string true "Pad ID（必填）"
// @Param X-Pad-Key header string true "Pad 密钥（必填）"
// @Param roomId formData string false "房间ID（可选，若传入需与Pad绑定房间一致）"
// @Param pointId formData string true "点位ID"
// @Param file formData file true "图片文件"
// @Param conf formData number false "置信度"
// @Param iou formData number false "IoU"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Router /predict [post]
func Predict(c *gin.Context) {
	roomFromPad, status, msg := resolveRoomByPadHeadersRequired(c)
	if status != 0 {
		c.JSON(status, gin.H{"success": false, "message": msg})
		return
	}

	legacyRoomID := strings.TrimSpace(c.PostForm("roomId"))
	pointID := strings.TrimSpace(c.PostForm("pointId"))
	if pointID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "pointId 不能为空"})
		return
	}

	roomID := roomFromPad.ID
	if legacyRoomID != "" && legacyRoomID != roomID {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "roomId 与 pad 绑定房间不一致"})
		return
	}

	room, err := models.FindRoomByID(roomID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "房间不存在"})
		return
	}
	if strings.TrimSpace(room.ModelPath) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "房间未配置模型路径"})
		return
	}

	if _, err := models.FindPointByIDAndRoom(pointID, roomID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "点位不属于该房间"})
		return
	}
	if _, err := models.FindStyleImageByPointID(pointID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "该点位未绑定对照图，禁止检测"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "缺少上传文件"})
		return
	}

	conf := 0.25
	iou := 0.45
	if v := strings.TrimSpace(c.PostForm("conf")); v != "" {
		if f, e := strconv.ParseFloat(v, 64); e == nil && f > 0 {
			conf = f
		}
	}
	if v := strings.TrimSpace(c.PostForm("iou")); v != "" {
		if f, e := strconv.ParseFloat(v, 64); e == nil && f > 0 {
			iou = f
		}
	}

	ext := filepath.Ext(file.Filename)
	if ext == "" {
		ext = ".jpg"
	}
	filename := utils.GenerateUUID() + ext

	uploadsRoot := config.Get().UploadDir
	imagesDir := filepath.Join(uploadsRoot, "images", roomID, pointID)
	if err := utils.EnsureDir(imagesDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "创建上传目录失败: " + err.Error()})
		return
	}
	fullPath := filepath.Join(imagesDir, filename)
	if err := c.SaveUploadedFile(file, fullPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "保存文件失败: " + err.Error()})
		return
	}

	webPath := filepath.ToSlash(filepath.Join("uploads", "images", roomID, pointID, filename))
	seq, err := models.GetNextSequence()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "生成序列号失败"})
		return
	}
	now := time.Now()
	img := models.Image{
		ID:               utils.GenerateObjectID(),
		SequenceNumber:   seq,
		RoomID:           roomID,
		PointID:          pointID,
		Timestamp:        now,
		Filename:         filename,
		Path:             webPath,
		IsDetected:       false,
		HasIssue:         false,
		IssueType:        "",
		Status:           models.ImageStatusUndetected,
		DetectionResults: datatypes.JSONSlice[models.DetectionItem]{},
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := img.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "保存图片记录失败: " + err.Error()})
		return
	}

	reqBody := map[string]interface{}{
		"image_path": webPath,
		"model_path": room.ModelPath,
		"conf":       conf,
		"iou":        iou,
	}
	b, _ := json.Marshal(reqBody)
	start := time.Now()
	resp, err := http.Post(strings.TrimRight(config.Get().DetectServiceURL, "/")+"/api/detect", "application/json", bytes.NewReader(b))
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "调用检测服务失败: " + err.Error()})
		return
	}
	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if readErr != nil {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "读取检测服务响应失败: " + readErr.Error()})
		return
	}
	if resp.StatusCode >= 400 {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "检测服务返回错误", "detail": string(body)})
		return
	}

	var dr detectServiceResponse
	if err := json.Unmarshal(body, &dr); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "解析检测服务响应失败: " + err.Error()})
		return
	}
	if !dr.Success {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": "检测服务返回失败", "detail": string(body)})
		return
	}

	items := make([]models.DetectionItem, 0, len(dr.Items))
	for _, it := range dr.Items {
		items = append(items, models.DetectionItem{
			Class:      it.Class_,
			ClassID:    it.ClassId,
			Confidence: it.Confidence,
			BBox:       models.BoundingBox{X: it.Bbox.X, Y: it.Bbox.Y, Width: it.Bbox.Width, Height: it.Bbox.Height},
		})
	}

	processedPath := webPath
	processedFilename := filename
	if strings.TrimSpace(dr.LabeledPath) != "" {
		processedPath = dr.LabeledPath
		processedFilename = path.Base(dr.LabeledPath)
	}

	run := &models.DetectionRun{
		RunID:               fmt.Sprintf("predict-%s-%d", img.ID, now.UnixNano()),
		ImageID:             img.ID,
		RoomID:              roomID,
		PointID:             pointID,
		SourceFilename:      img.Filename,
		SourcePath:          webPath,
		ProcessedFilename:   processedFilename,
		ProcessedPath:       processedPath,
		ModelName:           room.Name,
		ModelVersion:        "",
		IoUThreshold:        iou,
		ConfidenceThreshold: conf,
		InferenceTimeMs:     time.Since(start).Milliseconds(),
		Items:               items,
		Summary: models.DetectionSummary{
			HasIssue:    dr.Summary.HasIssue,
			IssueType:   dr.Summary.IssueType,
			ObjectCount: dr.Summary.ObjectCount,
			AvgScore:    dr.Summary.AvgScore,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if _, err := models.InsertDetectionRun(run); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "保存检测结果失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"imageId":     img.ID,
		"roomId":      roomID,
		"pointId":     pointID,
		"detections":  items,
		"summary":     run.Summary,
		"labeledPath": processedPath,
	})
}
