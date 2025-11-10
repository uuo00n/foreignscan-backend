package handlers

import (
    "net/http"
    "time"

    "foreignscan/internal/models"

    "github.com/gin-gonic/gin"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

// RegisterComparisonRoutes 在主路由中注册对比相关接口
// 使用 gin.IRouter 以兼容 Engine 与 RouterGroup
func RegisterComparisonRoutes(r gin.IRouter) {
    r.POST("/comparisons", CreateComparison)
    r.GET("/comparisons", QueryComparisons)
}

// CreateComparisonRequest 创建对比记录的请求体
type CreateComparisonRequest struct {
    ImageID        string      `json:"imageId"`                 // 图片ID（必填）
    DetectionRunID string      `json:"detectionRunId,omitempty"` // 可选：检测运行ID
    BeforePath     string      `json:"beforePath"`             // 处理前图片路径（必填）
    AfterPath      string      `json:"afterPath"`              // 处理后图片路径（必填）
    DiffInfo       interface{} `json:"diffInfo,omitempty"`      // 差异信息（可选）
    Remark         string      `json:"remark,omitempty"`        // 备注（可选）
}

// CreateComparison godoc
// @Summary 创建处理前后对比记录
// @Description 保存源图与处理后图路径，并可选保存差异信息
// @Tags comparisons
// @Accept json
// @Produce json
// @Param body body CreateComparisonRequest true "对比信息"
// @Success 201 {object} map[string]interface{} "创建成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 404 {object} map[string]interface{} "图片不存在"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /comparisons [post]
func CreateComparison(c *gin.Context) {
    var req CreateComparisonRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求体解析失败: " + err.Error()})
        return
    }
    if req.ImageID == "" || req.BeforePath == "" || req.AfterPath == "" {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "imageId、beforePath、afterPath为必填"})
        return
    }

    // 校验图片存在
    img, err := models.FindByID(req.ImageID)
    if err != nil || img == nil {
        c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "未找到图片"})
        return
    }

    var detRunOID primitive.ObjectID
    if req.DetectionRunID != "" {
        if oid, err := primitive.ObjectIDFromHex(req.DetectionRunID); err == nil {
            detRunOID = oid
        }
    }

    comp := &models.Comparison{
        ID:             primitive.NewObjectID(),
        ImageID:        img.ID,
        DetectionRunID: detRunOID,
        BeforePath:     req.BeforePath,
        AfterPath:      req.AfterPath,
        DiffInfo:       req.DiffInfo,
        Remark:         req.Remark,
        CreatedAt:      time.Now(),
    }

    oid, err := models.InsertComparison(comp)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "创建对比记录失败: " + err.Error()})
        return
    }
    c.JSON(http.StatusCreated, gin.H{"success": true, "id": oid.Hex()})
}

// QueryComparisons godoc
// @Summary 查询对比记录
// @Description 支持按图片ID或检测运行ID筛选
// @Tags comparisons
// @Accept json
// @Produce json
// @Param imageId query string false "图片ID"
// @Param detectionRunId query string false "检测运行ID"
// @Success 200 {object} map[string]interface{} "查询成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /comparisons [get]
func QueryComparisons(c *gin.Context) {
    filter := bson.M{}
    if iid := c.Query("imageId"); iid != "" {
        if oid, err := primitive.ObjectIDFromHex(iid); err == nil {
            filter["imageId"] = oid
        } else {
            c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "imageId格式错误"})
            return
        }
    }
    if dr := c.Query("detectionRunId"); dr != "" {
        if oid, err := primitive.ObjectIDFromHex(dr); err == nil {
            filter["detectionRunId"] = oid
        } else {
            c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "detectionRunId格式错误"})
            return
        }
    }

    comps, err := models.FindComparisons(filter, nil, 0)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "查询对比记录失败: " + err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"success": true, "count": len(comps), "comparisons": comps})
}