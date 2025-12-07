package handlers

import (
	"foreignscan/internal/models"
	"foreignscan/internal/utils"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
	imageIDStr := c.Param("id")
	oid, err := primitive.ObjectIDFromHex(imageIDStr)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "图片ID格式错误"})
		return
	}

	// 校验图片是否存在（避免返回误导数据）
	_, err = models.FindByID(imageIDStr)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "未找到图片: " + err.Error()})
		return
	}

    runs, err := models.FindDetectionsByImageID(oid)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "查询检测结果失败: " + err.Error()})
        return
    }

    type detectionRunView struct {
        ID                  primitive.ObjectID      `json:"id"`
        RunID               string                 `json:"runId,omitempty"`
        ImageID             primitive.ObjectID      `json:"imageId"`
        SceneID             primitive.ObjectID      `json:"sceneId"`
        SourceFilename      string                 `json:"sourceFilename"`
        SourcePath          string                 `json:"sourcePath"`
        ProcessedFilename   string                 `json:"processedFilename"`
        ProcessedPath       string                 `json:"processedPath"`
        ModelName           string                 `json:"modelName"`
        ModelVersion        string                 `json:"modelVersion"`
        Device              string                 `json:"device,omitempty"`
        IoUThreshold        float64                `json:"iouThreshold"`
        ConfidenceThreshold float64                `json:"confidenceThreshold"`
        InferenceTimeMs     int64                  `json:"inferenceTimeMs"`
        Items               []models.DetectionItem `json:"items"`
        Summary             models.DetectionSummary `json:"summary"`
        CreatedAt           time.Time              `json:"createdAt"`
        UpdatedAt           time.Time              `json:"updatedAt"`
        SourceURL           string                 `json:"sourceUrl"`
        ProcessedURL        string                 `json:"processedUrl"`
    }

    webPath := func(p string) string {
        if p == "" { return "" }
        s := strings.ReplaceAll(p, "\\", "/")
        if !strings.HasPrefix(s, "/") { s = "/" + s }
        return s
    }

    views := make([]detectionRunView, 0, len(runs))
    for _, r := range runs {
        v := detectionRunView{
            ID: r.ID,
            RunID: r.RunID,
            ImageID: r.ImageID,
            SceneID: r.SceneID,
            SourceFilename: r.SourceFilename,
            SourcePath: r.SourcePath,
            ProcessedFilename: r.ProcessedFilename,
            ProcessedPath: r.ProcessedPath,
            ModelName: r.ModelName,
            ModelVersion: r.ModelVersion,
            Device: r.Device,
            IoUThreshold: r.IoUThreshold,
            ConfidenceThreshold: r.ConfidenceThreshold,
            InferenceTimeMs: r.InferenceTimeMs,
            Items: r.Items,
            Summary: r.Summary,
            CreatedAt: r.CreatedAt,
            UpdatedAt: r.UpdatedAt,
            SourceURL: webPath(r.SourcePath),
            ProcessedURL: webPath(r.ProcessedPath),
        }
        views = append(views, v)
    }

    c.JSON(http.StatusOK, gin.H{"success": true, "count": len(views), "detections": views})
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
	imageIDStr := c.Param("id")
	image, err := models.FindByID(imageIDStr)
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
		// 约定：处理后的图片与标签txt存放在同一目录（uploads/labels/<sceneId>/），
		// 标签文件与图片同名，仅扩展名不同（.txt）

		// 1) 取处理后图片所在目录
		processedDir := filepath.Dir(req.ProcessedPath)
		// 2) 基名（不含扩展名）
		baseName := strings.TrimSuffix(filepath.Base(req.ProcessedPath), filepath.Ext(req.ProcessedPath))
		// 3) 标签文件路径（优先与图片同目录；若不存在则回退到子目录 labels/）
		labelPathPrimary := filepath.Join(processedDir, baseName+".txt")
		labelAbsPath := utils.NormalizeUploadsLocalPath(labelPathPrimary)
		if _, err := os.Stat(labelAbsPath); os.IsNotExist(err) {
			// 回退到 ".../labels/<name>.txt"，兼容Ultralytics默认输出结构
			labelPathFallback := filepath.Join(processedDir, "labels", baseName+".txt")
			labelAbsPath = utils.NormalizeUploadsLocalPath(labelPathFallback)
		}
		imgAbsPath := utils.NormalizeUploadsLocalPath(req.SourcePath)

		// 解析标签生成检测项
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
            if strings.EqualFold(it.Class, "hole") { hasHole = true }
            if !strings.EqualFold(it.Class, "Bolts") { allBolts = false }
        }
        avg := 0.0
        if len(req.Items) > 0 { avg = sum / float64(len(req.Items)) }
        hi := (len(req.Items) == 0) || hasHole || !allBolts
        itype := "auto"
        if len(req.Items) == 0 { itype = "no_object" }
        if hasHole { itype = "hole" }
        req.Summary = models.DetectionSummary{HasIssue: hi, IssueType: itype, ObjectCount: len(req.Items), AvgScore: avg}
    }

	run := &models.DetectionRun{
		RunID:               req.RunID,
		ImageID:             image.ID,
		SceneID:             image.SceneID,
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
		Items:               req.Items,
		Summary:             req.Summary,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

    id, err := models.InsertDetectionRun(run)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "写入检测结果失败: " + err.Error()})
        return
    }

	c.JSON(http.StatusCreated, gin.H{"success": true, "id": id.Hex(), "message": "检测结果写入成功"})
}

// QueryDetections godoc
// @Summary 查询检测结果
// @Description 支持按场景、时间范围、是否有问题、类别等条件筛选检测运行
// @Tags detections
// @Accept json
// @Produce json
// @Param sceneId query string false "场景ID"
// @Param hasIssue query bool false "是否存在问题"
// @Param class query string false "目标类别名称筛选"
// @Param start query string false "开始日期（YYYY-MM-DD）"
// @Param end query string false "结束日期（YYYY-MM-DD）"
// @Success 200 {object} map[string]interface{} "成功查询检测结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /detections [get]
func QueryDetections(c *gin.Context) {
	filter := bson.M{}

	// 场景筛选
	if sid := c.Query("sceneId"); sid != "" {
		if oid, err := primitive.ObjectIDFromHex(sid); err == nil {
			filter["sceneId"] = oid
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "sceneId格式错误"})
			return
		}
	}

	// 是否有问题筛选
	if hi := c.Query("hasIssue"); hi != "" {
		if hi == "true" || hi == "false" {
			filter["summary.hasIssue"] = (hi == "true")
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "hasIssue应为true或false"})
			return
		}
	}

	// 类别筛选（数组中包含某类别）
	if cls := c.Query("class"); cls != "" {
		filter["items.class"] = cls
	}

	// 时间范围筛选
	startStr := c.Query("start")
	endStr := c.Query("end")
	if startStr != "" || endStr != "" {
		// 解析日期字符串 (YYYY-MM-DD)
		layout := "2006-01-02"
		var start, end time.Time
		var err error
		if startStr != "" {
			start, err = time.Parse(layout, startStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "开始日期格式错误"})
				return
			}
		}
		if endStr != "" {
			end, err = time.Parse(layout, endStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "结束日期格式错误"})
				return
			}
			// 将结束日期设为当天23:59:59，包含整天
			end = end.Add(24 * time.Hour)
		}

		// 构造时间范围过滤
		timeFilter := bson.M{}
		if !start.IsZero() {
			timeFilter["$gte"] = start
		}
		if !end.IsZero() {
			timeFilter["$lt"] = end
		}
		filter["createdAt"] = timeFilter
	}

	runs, err := models.FindDetections(filter, nil, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "查询失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "count": len(runs), "detections": runs})
}
