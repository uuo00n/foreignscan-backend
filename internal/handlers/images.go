package handlers

import (
	"net/http"
	"time"

	"foreignscan/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetImages godoc
// @Summary 获取所有图片列表
// @Description 获取系统中所有图片的列表
// @Tags images
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "成功获取图片列表"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /images [get]
func GetImages(c *gin.Context) {
	// 获取所有图片
	images, err := models.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取图片列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"images":  images,
	})
}

// GetImagesByDate godoc
// @Summary 根据日期获取图片
// @Description 获取指定日期上传的所有图片
// @Tags images
// @Accept json
// @Produce json
// @Param date query string true "日期 (格式: YYYY-MM-DD)"
// @Success 200 {object} map[string]interface{} "成功获取图片列表"
// @Failure 400 {object} map[string]interface{} "日期格式错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /images/by-date [get]
func GetImagesByDate(c *gin.Context) {
	// 从查询参数获取日期
	dateStr := c.Query("date")
	if dateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "日期参数不能为空",
		})
		return
	}

	// 解析日期字符串 (YYYY-MM-DD)
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "日期格式错误，请使用YYYY-MM-DD格式",
		})
		return
	}

	// 查询指定日期的图片
	images, err := models.FindByDate(date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取图片失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"date":    dateStr,
		"count":   len(images),
		"images":  images,
	})
}

// GetImagesByDateAndScene godoc
// @Summary 根据日期和场景ID获取图片
// @Description 获取指定日期和场景ID的所有图片
// @Tags images
// @Accept json
// @Produce json
// @Param date query string true "日期 (格式: YYYY-MM-DD)"
// @Param scene_id query string true "场景ID"
// @Success 200 {object} map[string]interface{} "成功获取图片列表"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /images/by-date-scene [get]
func GetImagesByDateAndScene(c *gin.Context) {
	// 从查询参数获取日期
	dateStr := c.Query("date")
	if dateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "日期参数不能为空",
		})
		return
	}

	// 解析日期字符串 (YYYY-MM-DD)
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "日期格式错误，请使用YYYY-MM-DD格式",
		})
		return
	}
	
	// 从查询参数获取场景ID
	sceneIDStr := c.Query("scene_id")
	if sceneIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "场景ID参数不能为空",
		})
		return
	}
	
	// 将场景ID转换为ObjectID
	sceneID, err := primitive.ObjectIDFromHex(sceneIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的场景ID: " + err.Error(),
		})
		return
	}

	// 查询指定日期和场景的图片
	images, err := models.FindByDateAndSceneID(date, sceneID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取图片失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"date":    dateStr,
		"sceneId": sceneIDStr,
		"count":   len(images),
		"images":  images,
	})
}

// GetSceneImages godoc
// @Summary 获取场景下的所有图片
// @Description 获取特定场景下的所有图片列表
// @Tags scenes,images
// @Accept json
// @Produce json
// @Param id path string true "场景ID"
// @Success 200 {object} map[string]interface{} "成功获取场景图片"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /scenes/{id}/images [get]
func GetSceneImages(c *gin.Context) {
	// 从URL获取场景ID
	sceneIDStr := c.Param("id")
	
	// 将场景ID转换为ObjectID
	sceneID, err := primitive.ObjectIDFromHex(sceneIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的场景ID: " + err.Error(),
		})
		return
	}
	
	// 查找该场景下的所有图片
	images, err := models.FindBySceneID(sceneID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取场景图片失败: " + err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"images":  images,
	})
}

// GetSceneFirstImage godoc
// @Summary 获取场景的第一张图片
// @Description 根据场景ID获取该场景下的第一张图片
// @Tags scenes,images
// @Accept json
// @Produce json
// @Param scene_id path string true "场景ID"
// @Success 200 {object} map[string]interface{} "成功获取场景第一张图片"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "场景不存在或没有图片"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /scenes/{scene_id}/first-image [get]
func GetSceneFirstImage(c *gin.Context) {
	// 从URL获取场景ID
	sceneIDStr := c.Param("id")
	
	// 将场景ID转换为ObjectID
	sceneID, err := primitive.ObjectIDFromHex(sceneIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的场景ID: " + err.Error(),
		})
		return
	}
	
	// 查找该场景下的第一张图片（按序列号排序）
	image, err := models.FindFirstBySceneID(sceneID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取场景第一张图片失败: " + err.Error(),
		})
		return
	}
	
	// 如果没有找到图片
	if image == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "该场景下没有图片",
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"image":   image,
	})
}

// GetAllScenesFirstImage godoc
// @Summary 获取所有场景的第一张图片
// @Description 获取系统中所有场景的第一张图片，用于场景预览展示
// @Tags scenes,images
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "成功获取所有场景的第一张图片"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /scenes/first-images [get]
func GetAllScenesFirstImage(c *gin.Context) {
	// 获取所有场景
	scenes, err := models.FindAllScenes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取场景列表失败: " + err.Error(),
		})
		return
	}
	
	// 存储每个场景的第一张图片
	type SceneWithFirstImage struct {
		SceneID      primitive.ObjectID `json:"sceneId"`
		SceneName    string             `json:"sceneName"`
		FirstImage   *models.Image      `json:"firstImage"`
	}
	
	result := make([]SceneWithFirstImage, 0, len(scenes))
	
	// 遍历所有场景，获取每个场景的第一张图片
	for _, scene := range scenes {
		// 查找该场景下的第一张图片
		image, _ := models.FindFirstBySceneID(scene.ID)
		
		// 添加到结果中（即使没有图片）
		result = append(result, SceneWithFirstImage{
			SceneID:    scene.ID,
			SceneName:  scene.Name,
			FirstImage: image,
		})
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

// GetImageDetail godoc
// @Summary 获取单个图片详情
// @Description 根据ID获取特定图片的详细信息
// @Tags images
// @Accept json
// @Produce json
// @Param id path string true "图片ID"
// @Success 200 {object} map[string]interface{} "成功获取图片详情"
// @Failure 404 {object} map[string]interface{} "图片不存在"
// @Router /images/{id} [get]
func GetImageDetail(c *gin.Context) {
	// 从URL获取图片ID
	imageIDStr := c.Param("id")
	
	// 查找图片详细信息
	image, err := models.FindByID(imageIDStr)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "未找到图片: " + err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"image":   image,
	})
}

// DetectImage godoc
// @Summary 检测图片内容
// @Description 分析图片内容并返回检测结果
// @Tags images
// @Accept json
// @Produce json
// @Param id path string true "图片ID"
// @Success 200 {object} map[string]interface{} "成功检测图片"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "图片不存在"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /images/{id}/detect [get]
func DetectImage(c *gin.Context) {
	var req struct {
		ImageID string `json:"imageId"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的请求参数",
		})
		return
	}

	// 查找图片
	image, err := models.FindByID(req.ImageID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的图片ID",
		})
		return
	}

	// 模拟检测结果
	results := []map[string]interface{}{
		{
			"x":          200,
			"y":          150,
			"width":      50,
			"height":     50,
			"type":       "裂缝",
			"confidence": 0.92,
		},
		{
			"x":          350,
			"y":          150,
			"width":      50,
			"height":     50,
			"type":       "磨损",
			"confidence": 0.87,
		},
		{
			"x":          275,
			"y":          250,
			"width":      50,
			"height":     50,
			"type":       "变形",
			"confidence": 0.95,
		},
	}

	// 更新图片记录
	hasIssue := len(results) > 0
	var issueType string
	if hasIssue {
		for i, result := range results {
			if i > 0 {
				issueType += ","
			}
			issueType += result["type"].(string)
		}
	}

	// 更新图片信息
	image.IsDetected = true
	image.HasIssue = hasIssue
	image.IssueType = issueType
	
	// 将map转换为interface{}切片
	detectionResults := make([]interface{}, len(results))
	for i, result := range results {
		detectionResults[i] = result
	}
	image.DetectionResults = detectionResults

	if err := image.Update(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新图片信息失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"imageId":  req.ImageID,
		"hasIssue": hasIssue,
		"results":  results,
	})
}