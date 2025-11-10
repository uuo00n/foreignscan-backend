package handlers

import (
    "net/http"
    "time"

    "foreignscan/internal/models"

    "github.com/gin-gonic/gin"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

// RegisterIssueRoutes 在主路由中注册问题相关接口
// 使用 gin.IRouter 以兼容 Engine 与 RouterGroup
func RegisterIssueRoutes(r gin.IRouter) {
    // 注册问题创建与查询
    r.POST("/issues", CreateIssue)
    r.GET("/issues", QueryIssues)
    // 注册图片的所有问题查询
    r.GET("/images/:id/issues", GetImageIssues)
}

// CreateIssueRequest 创建问题的请求体
type CreateIssueRequest struct {
    ImageID        string `json:"imageId"`                 // 图片ID（必填）
    DetectionRunID string `json:"detectionRunId,omitempty"` // 可选：关联的检测运行ID
    Type           string `json:"type"`                    // 问题类型（必填）
    Description    string `json:"description"`             // 问题说明（可选）
}

// CreateIssue godoc
// @Summary 创建问题记录
// @Description 新建问题，包含问题类型和说明；自动补充sceneId
// @Tags issues
// @Accept json
// @Produce json
// @Param body body CreateIssueRequest true "问题信息"
// @Success 201 {object} map[string]interface{} "创建成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 404 {object} map[string]interface{} "图片不存在"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /issues [post]
func CreateIssue(c *gin.Context) {
    var req CreateIssueRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求体解析失败: " + err.Error()})
        return
    }
    if req.ImageID == "" || req.Type == "" {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "imageId与type为必填"})
        return
    }

    // 校验图片存在并提取sceneId
    img, err := models.FindByID(req.ImageID)
    if err != nil || img == nil {
        c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "未找到图片"})
        return
    }

    var detRunOID primitive.ObjectID
    if req.DetectionRunID != "" {
        oid, err := primitive.ObjectIDFromHex(req.DetectionRunID)
        if err == nil {
            detRunOID = oid
        }
        // 如果格式错误则忽略，保持空值
    }

    issue := &models.Issue{
        ID:             primitive.NewObjectID(),
        ImageID:        img.ID,
        SceneID:        img.SceneID,
        DetectionRunID: detRunOID,
        Type:           req.Type,
        Description:    req.Description,
        CreatedAt:      time.Now(),
        UpdatedAt:      time.Now(),
    }

    oid, err := models.InsertIssue(issue)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "创建问题失败: " + err.Error()})
        return
    }
    c.JSON(http.StatusCreated, gin.H{"success": true, "id": oid.Hex()})
}

// QueryIssues godoc
// @Summary 查询问题列表
// @Description 支持按场景、图片、问题类型筛选
// @Tags issues
// @Accept json
// @Produce json
// @Param sceneId query string false "场景ID"
// @Param imageId query string false "图片ID"
// @Param type query string false "问题类型"
// @Success 200 {object} map[string]interface{} "查询成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /issues [get]
func QueryIssues(c *gin.Context) {
    filter := bson.M{}

    if sid := c.Query("sceneId"); sid != "" {
        if oid, err := primitive.ObjectIDFromHex(sid); err == nil {
            filter["sceneId"] = oid
        } else {
            c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "sceneId格式错误"})
            return
        }
    }
    if iid := c.Query("imageId"); iid != "" {
        if oid, err := primitive.ObjectIDFromHex(iid); err == nil {
            filter["imageId"] = oid
        } else {
            c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "imageId格式错误"})
            return
        }
    }
    if t := c.Query("type"); t != "" {
        filter["type"] = t
    }

    issues, err := models.FindIssues(filter, nil, 0)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "查询问题失败: " + err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"success": true, "count": len(issues), "issues": issues})
}

// GetImageIssues godoc
// @Summary 获取单个图片的所有问题
// @Description 根据图片ID获取对应问题列表
// @Tags issues
// @Accept json
// @Produce json
// @Param id path string true "图片ID"
// @Success 200 {object} map[string]interface{} "查询成功"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Router /images/{id}/issues [get]
func GetImageIssues(c *gin.Context) {
    imageID := c.Param("id")
    oid, err := primitive.ObjectIDFromHex(imageID)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "图片ID格式错误"})
        return
    }
    issues, err := models.FindIssues(bson.M{"imageId": oid}, nil, 0)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "查询问题失败: " + err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"success": true, "count": len(issues), "issues": issues})
}