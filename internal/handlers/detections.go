package handlers

import (
	"foreignscan/internal/models"
	"foreignscan/internal/utils"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

// CreateDetectionRequest 用于接收YOLO推理服务提交的检测结果请求体
type CreateDetectionRequest struct {
	RunID               string                  `json:"runId"` // 可选：运行ID，用于幂等
	ModelName           string                  `json:"modelName"`
	ModelVersion        string                  `json:"modelVersion"`
	Device              string                  `json:"device"`
	IoUThreshold        float64                 `json:"iouThreshold"`
	ConfidenceThreshold float64                 `json:"confidenceThreshold"`
	InferenceTimeMs     int64                   `json:"inferenceTimeMs"`
	SourceFilename      string                  `json:"sourceFilename"`
	SourcePath          string                  `json:"sourcePath"`
	ProcessedFilename   string                  `json:"processedFilename"`
	ProcessedPath       string                  `json:"processedPath"`
	Items               []models.DetectionItem  `json:"items"`   // 检测项
	Summary             models.DetectionSummary `json:"summary"` // 汇总信息（是否有问题等）
}

// GetImageDetections godoc
// @Summary 获取图片的检测结果列表
// @Description 根据图片ID获取所有检测运行结果（按创建时间倒序）
// @Tags detections
// @Accept json
// @Produce json
// @Param id path string true "图片ID"
// @Success 200 {object} map[string]interface{} "成功获取检测结果"
// @Failure 404 {object} map[string]interface{} "图片不存在"
// @Router /images/{id}/detections [get]
func GetImageDetections(c *gin.Context) {
	imageID := c.Param("id")

	// 校验图片是否存在（避免返回误导数据）
	_, err := models.FindByID(imageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "未找到图片: " + err.Error()})
		return
	}

	runs, total, err := models.QueryDetections(1, 100, imageID, "", "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "查询检测结果失败: " + err.Error()})
		return
	}

	type detectionRunView struct {
		ID                  string                  `json:"id"`
		RunID               string                  `json:"runId,omitempty"`
		ImageID             string                  `json:"imageId"`
		RoomID              string                  `json:"roomId"`
		PointID             string                  `json:"pointId"`
		SourceFilename      string                  `json:"sourceFilename"`
		SourcePath          string                  `json:"sourcePath"`
		ProcessedFilename   string                  `json:"processedFilename"`
		ProcessedPath       string                  `json:"processedPath"`
		ModelName           string                  `json:"modelName"`
		ModelVersion        string                  `json:"modelVersion"`
		Device              string                  `json:"device,omitempty"`
		IoUThreshold        float64                 `json:"iouThreshold"`
		ConfidenceThreshold float64                 `json:"confidenceThreshold"`
		InferenceTimeMs     int64                   `json:"inferenceTimeMs"`
		Items               []models.DetectionItem  `json:"items"`
		Summary             models.DetectionSummary `json:"summary"`
		CreatedAt           time.Time               `json:"createdAt"`
		UpdatedAt           time.Time               `json:"updatedAt"`
		SourceURL           string                  `json:"sourceUrl"`
		ProcessedURL        string                  `json:"processedUrl"`
	}

	views := make([]detectionRunView, 0, len(runs))
	for _, r := range runs {
		v := detectionRunView{
			ID:                  r.ID,
			RunID:               r.RunID,
			ImageID:             r.ImageID,
			RoomID:              r.RoomID,
			PointID:             r.PointID,
			SourceFilename:      r.SourceFilename,
			SourcePath:          r.SourcePath,
			ProcessedFilename:   r.ProcessedFilename,
			ProcessedPath:       r.ProcessedPath,
			ModelName:           r.ModelName,
			ModelVersion:        r.ModelVersion,
			Device:              r.Device,
			IoUThreshold:        r.IoUThreshold,
			ConfidenceThreshold: r.ConfidenceThreshold,
			InferenceTimeMs:     r.InferenceTimeMs,
			Items:               r.Items,
			Summary:             r.Summary,
			CreatedAt:           r.CreatedAt,
			UpdatedAt:           r.UpdatedAt,
			SourceURL:           utils.NormalizeToUploadsWebPath(r.SourcePath),
			ProcessedURL:        utils.NormalizeToUploadsWebPath(r.ProcessedPath),
		}
		views = append(views, v)
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "count": total, "detections": views})
}

// CreateImageDetection godoc
// @Summary 写入图片的检测结果
// @Description YOLO推理完成后，向后端写入一次检测运行结果；支持RunID幂等插入，自动更新图片摘要字段
// @Tags detections
// @Accept json
// @Produce json
// @Param id path string true "图片ID"
// @Param body body CreateDetectionRequest true "检测结果请求体"
// @Success 201 {object} map[string]interface{} "成功写入检测结果"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "图片不存在"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /images/{id}/detections [post]
func CreateImageDetection(c *gin.Context) {
	imageID := c.Param("id")
	image, err := models.FindByID(imageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "未找到图片: " + err.Error()})
		return
	}

	var req CreateDetectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求体解析失败: " + err.Error()})
		return
	}

	// 如果请求中没有检测项，则尝试从 YOLO 标签文件解析
	if len(req.Items) == 0 && req.ProcessedPath != "" {
		processedAbsPath := utils.NormalizeUploadsLocalPath(req.ProcessedPath)
		processedDir := filepath.Dir(processedAbsPath)
		baseName := strings.TrimSuffix(filepath.Base(processedAbsPath), filepath.Ext(processedAbsPath))
		labelAbsPath := filepath.Join(processedDir, baseName+".txt")
		if _, err := os.Stat(labelAbsPath); os.IsNotExist(err) {
			labelAbsPath = filepath.Join(processedDir, "labels", baseName+".txt")
		}
		imgAbsPath := utils.NormalizeUploadsLocalPath(req.SourcePath)
		if items, err := utils.ParseYOLOLabelsToItems(labelAbsPath, imgAbsPath); err == nil {
			req.Items = items
		}
	}

	// 兜底生成 Summary（当请求未提供或为空且 items 非空时）
	if req.Summary.IssueType == "" && req.Summary.ObjectCount == 0 && req.Summary.AvgScore == 0 {
		sum := 0.0
		hasHole := false
		allBolts := len(req.Items) > 0
		for _, it := range req.Items {
			sum += it.Confidence
			if strings.EqualFold(it.Class, "hole") {
				hasHole = true
			}
			if !strings.EqualFold(it.Class, "Bolts") {
				allBolts = false
			}
		}
		avg := 0.0
		if len(req.Items) > 0 {
			avg = sum / float64(len(req.Items))
		}
		hi := (len(req.Items) == 0) || hasHole || !allBolts
		itype := "auto"
		if len(req.Items) == 0 {
			itype = "no_object"
		}
		if hasHole {
			itype = "hole"
		}
		req.Summary = models.DetectionSummary{HasIssue: hi, IssueType: itype, ObjectCount: len(req.Items), AvgScore: avg}
	}

	run := &models.DetectionRun{
		RunID:               req.RunID,
		ImageID:             image.ID,
		RoomID:              image.RoomID,
		PointID:             image.PointID,
		SourceFilename:      req.SourceFilename,
		SourcePath:          req.SourcePath,
		ProcessedFilename:   req.ProcessedFilename,
		ProcessedPath:       req.ProcessedPath,
		ModelName:           req.ModelName,
		ModelVersion:        req.ModelVersion,
		Device:              req.Device,
		IoUThreshold:        req.IoUThreshold,
		ConfidenceThreshold: req.ConfidenceThreshold,
		InferenceTimeMs:     req.InferenceTimeMs,
		Items:               datatypes.JSONSlice[models.DetectionItem](req.Items),
		Summary:             req.Summary,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	id, err := models.InsertDetectionRun(run)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "写入检测结果失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "id": id, "message": "检测结果写入成功"})
}

// QueryDetections godoc
// @Summary 查询检测结果
// @Description 支持按房间、点位、时间范围、是否有问题、类别等条件筛选检测运行
// @Tags detections
// @Accept json
// @Produce json
// @Param roomId query string false "房间ID"
// @Param pointId query string false "点位ID"
// @Param hasIssue query bool false "是否存在问题"
// @Param class query string false "目标类别名称筛选"
// @Param start query string false "开始日期（YYYY-MM-DD）"
// @Param end query string false "结束日期（YYYY-MM-DD）"
// @Success 200 {object} map[string]interface{} "成功查询检测结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /detections [get]
func QueryDetections(c *gin.Context) {
	roomID := c.Query("roomId")
	pointID := c.Query("pointId")

	// 分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	// 调用 GORM 查询方法
	runs, total, err := models.QueryDetections(page, pageSize, "", roomID, pointID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "查询失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"count":      len(runs), // 当前页数量
		"total":      total,     // 总数量
		"detections": runs,
	})
}
