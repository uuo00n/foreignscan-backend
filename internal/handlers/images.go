package handlers

import (
	"net/http"

	"foreignscan/internal/models"

	"github.com/gin-gonic/gin"
)

// GetImages 获取所有图片列表
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

// DetectImage 处理图片检测
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