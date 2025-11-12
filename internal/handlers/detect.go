package handlers

import (
    "net/http"
    "time"

    "foreignscan/internal/services"
    "github.com/gin-gonic/gin"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "encoding/json"
    "strings"
)

// StartDetectRequest 前端触发推理的请求体
// 说明：所有字段均可选，采用合理默认值；前端可根据模型配置传入
type StartDetectRequest struct {
    Weights      string  `json:"weights"`       // 模型权重文件路径或名称，默认 yolov8s.pt
    ModelName    string  `json:"modelName"`    // 模型名称，默认 "yolov8"
    ModelVersion string  `json:"modelVersion"` // 模型版本，可选
    Device       string  `json:"device"`       // 设备：cpu/cuda:0/mps，默认空（由YOLO自动选择）
    Conf         float64 `json:"conf"`         // 置信度阈值，默认 0.25
    IoU          float64 `json:"iou"`          // IoU 阈值，默认 0.45
}

// StartSceneDetect godoc
// @Summary 前端一键触发场景批量推理
// @Description 由前端点击触发，后端启动异步任务，运行YOLO并写入数据库
// @Tags detections
// @Accept json
// @Produce json
// @Param id path string true "场景ID"
// @Param body body StartDetectRequest false "推理配置，可选"
// @Success 202 {object} map[string]interface{} "任务已启动，返回jobId与初始状态"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Router /scenes/{id}/detect [post]
func StartSceneDetect(c *gin.Context) {
    sceneIDStr := c.Param("id")
    sceneID, err := primitive.ObjectIDFromHex(sceneIDStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的场景ID: " + err.Error()})
        return
    }

    var req StartDetectRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        // 允许空请求体，按默认值处理
        req = StartDetectRequest{}
    }
    if req.Weights == "" { req.Weights = "yolov8s.pt" }
    if req.ModelName == "" { req.ModelName = "yolov8" }
    if req.Conf <= 0 { req.Conf = 0.25 }
    if req.IoU <= 0 { req.IoU = 0.45 }

    jobID, err := services.StartSceneDetect(sceneID, services.DetectConfig{
        Weights:      req.Weights,
        ModelName:    req.ModelName,
        ModelVersion: req.ModelVersion,
        Device:       req.Device,
        Conf:         req.Conf,
        IoU:          req.IoU,
    })
    if err != nil {
        if strings.Contains(err.Error(), "busy") {
            c.JSON(http.StatusConflict, gin.H{"success": false, "message": "当前场景已有进行中的任务"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "启动任务失败: " + err.Error()})
        }
        return
    }

    c.JSON(http.StatusAccepted, gin.H{
        "success": true,
        "jobId":   jobID,
        "status":  "pending",
        "startedAt": time.Now(),
    })
}

// GetDetectJob godoc
// @Summary 查询推理任务状态
// @Description 前端轮询或SSE，可获取任务进度/状态
// @Tags detections
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} map[string]interface{} "任务状态"
// @Failure 404 {object} map[string]interface{} "未找到任务"
// @Router /detect/jobs/{id} [get]
func GetDetectJob(c *gin.Context) {
    jobID := c.Param("id")
    job, ok := services.GetJobManager().GetJob(jobID)
    if !ok {
        c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "未找到任务"})
        return
    }
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "job":     job,
    })
}

// StartImageDetect godoc
// @Summary 前端一键触发单图推理
// @Description 由前端点击触发，后端启动异步任务，运行YOLO并写入数据库（单张图片）
// @Tags detections
// @Accept json
// @Produce json
// @Param id path string true "图片ID"
// @Param body body StartDetectRequest false "推理配置，可选"
// @Success 202 {object} map[string]interface{} "任务已启动，返回jobId与初始状态"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Router /images/{id}/detect [post]
func StartImageDetect(c *gin.Context) {
    imageIDStr := c.Param("id")
    imageID, err := primitive.ObjectIDFromHex(imageIDStr)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的图片ID: " + err.Error()})
        return
    }

    var req StartDetectRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        // 允许空请求体，按默认值处理
        req = StartDetectRequest{}
    }
    if req.Weights == "" { req.Weights = "yolov8s.pt" }
    if req.ModelName == "" { req.ModelName = "yolov8" }
    if req.Conf <= 0 { req.Conf = 0.25 }
    if req.IoU <= 0 { req.IoU = 0.45 }

    jobID, err := services.StartImageDetect(imageID, services.DetectConfig{
        Weights:      req.Weights,
        ModelName:    req.ModelName,
        ModelVersion: req.ModelVersion,
        Device:       req.Device,
        Conf:         req.Conf,
        IoU:          req.IoU,
    })
    if err != nil {
        if strings.Contains(err.Error(), "busy") {
            c.JSON(http.StatusConflict, gin.H{"success": false, "message": "当前场景已有进行中的任务"})
        } else {
            c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "启动任务失败: " + err.Error()})
        }
        return
    }

    c.JSON(http.StatusAccepted, gin.H{
        "success": true,
        "jobId":   jobID,
        "status":  "pending",
        "startedAt": time.Now(),
    })
}

// CancelDetectJob godoc
// @Summary 取消正在运行的检测任务
// @Description 取消任务（支持pending/running/parsing阶段），任务将标记为canceled
// @Tags detections
// @Accept json
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} map[string]interface{} "取消成功"
// @Failure 404 {object} map[string]interface{} "未找到任务"
// @Failure 409 {object} map[string]interface{} "任务不可取消（已结束）"
// @Router /detect/jobs/{id} [delete]
func CancelDetectJob(c *gin.Context) {
    jobID := c.Param("id")
    jm := services.GetJobManager()
    job, ok := jm.GetJob(jobID)
    if !ok {
        c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "未找到任务"})
        return
    }
    if job.Status == "completed" || job.Status == "failed" || job.Status == "canceled" {
        c.JSON(http.StatusConflict, gin.H{"success": false, "message": "任务已结束，无法取消"})
        return
    }
    if ok := jm.CancelJob(jobID); !ok {
        c.JSON(http.StatusConflict, gin.H{"success": false, "message": "任务已结束或未找到"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"success": true, "message": "任务已取消"})
}

// GetDetectJobStream godoc
// @Summary 订阅检测任务实时进度（SSE）
// @Description 通过SSE实时获取任务状态更新，直到任务结束
// @Tags detections
// @Produce text/event-stream
// @Param id path string true "任务ID"
// @Success 200 {string} string "SSE事件流"
// @Failure 404 {object} map[string]interface{} "未找到任务"
// @Router /detect/jobs/{id}/stream [get]
func GetDetectJobStream(c *gin.Context) {
    jobID := c.Param("id")
    job, ok := services.GetJobManager().GetJob(jobID)
    if !ok {
        c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "未找到任务"})
        return
    }
    // 设置SSE头
    c.Writer.Header().Set("Content-Type", "text/event-stream")
    c.Writer.Header().Set("Cache-Control", "no-cache")
    c.Writer.Header().Set("Connection", "keep-alive")
    c.Writer.WriteHeader(http.StatusOK)

    // 订阅任务更新
    ch, unsub := services.GetJobManager().Subscribe(jobID)
    defer unsub()

    // 发送初始状态
    if b, err := json.Marshal(job); err == nil {
        c.Writer.Write([]byte("data: "))
        c.Writer.Write(b)
        c.Writer.Write([]byte("\n\n"))
        if flusher, ok := c.Writer.(http.Flusher); ok { flusher.Flush() }
    }

    // 循环接收更新直至任务结束或客户端断开
    for {
        select {
        case upd, ok := <-ch:
            if !ok { return }
            if b, err := json.Marshal(upd); err == nil {
                c.Writer.Write([]byte("data: "))
                c.Writer.Write(b)
                c.Writer.Write([]byte("\n\n"))
                if flusher, ok := c.Writer.(http.Flusher); ok { flusher.Flush() }
            }
            if upd.Status == "completed" || upd.Status == "failed" || upd.Status == "canceled" {
                return
            }
        case <-c.Request.Context().Done():
            return
        }
    }
}