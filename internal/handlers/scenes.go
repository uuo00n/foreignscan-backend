package handlers

import (
	"net/http"
	"time"

	"foreignscan/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetScenes godoc
// @Summary 获取所有场景
// @Description 获取系统中所有可用的场景列表
// @Tags scenes
// @Accept json
// @Produce json
// @Success 200 {array} models.Scene
// @Failure 500 {object} handlers.ErrorResponse
// @Router /scenes [get]
func GetScenes(c *gin.Context) {
	// 获取所有场景
	scenes, err := models.FindAllScenes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取场景失败: " + err.Error(),
		})
		return
	}

	// 返回JSON响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"scenes":  scenes,
	})
}

// GetScene godoc
// @Summary 获取单个场景详情
// @Description 根据ID获取单个场景的详细信息
// @Tags scenes
// @Accept json
// @Produce json
// @Param id path string true "场景ID"
// @Success 200 {object} map[string]interface{} "成功获取场景详情"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "场景不存在"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /scenes/{id} [get]
func GetScene(c *gin.Context) {
	// 从URL获取场景ID
	id := c.Param("id")

	// 查找场景
	scene, err := models.FindSceneByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "场景不存在: " + err.Error(),
		})
		return
	}

	// 返回JSON响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"scene":   scene,
	})
}

// CreateScene godoc
// @Summary 创建新场景
// @Description 创建一个新的场景
// @Tags scenes
// @Accept json
// @Produce json
// @Param scene body map[string]interface{} true "场景信息"
// @Success 201 {object} map[string]interface{} "成功创建场景"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /scenes [post]
func CreateScene(c *gin.Context) {
	// 解析请求体
	var scene models.Scene
	if err := c.ShouldBindJSON(&scene); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的请求数据: " + err.Error(),
		})
		return
	}

	// 设置创建时间和更新时间
	scene.CreatedAt = time.Now()
	scene.UpdatedAt = time.Now()

	// 保存场景
	err := scene.Save()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建场景失败: " + err.Error(),
		})
		return
	}

	// 返回JSON响应
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"scene":   scene,
	})
}

// UpdateScene 更新场景
func UpdateScene(c *gin.Context) {
	// 从URL获取场景ID
	id := c.Param("id")

	// 将ID转换为ObjectID
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的场景ID",
		})
		return
	}

	// 解析请求体
	var updatedScene models.Scene
	if err := c.ShouldBindJSON(&updatedScene); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的请求数据: " + err.Error(),
		})
		return
	}

	// 查找现有场景
	existingScene, err := models.FindSceneByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "场景不存在: " + err.Error(),
		})
		return
	}

	// 更新场景字段
	updatedScene.ID = objID
	updatedScene.CreatedAt = existingScene.CreatedAt
	updatedScene.UpdatedAt = time.Now()

	// 保存更新
	err = updatedScene.Update()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新场景失败: " + err.Error(),
		})
		return
	}

	// 返回JSON响应
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"scene":   updatedScene,
	})
}

// DeleteScene 删除场景
func DeleteScene(c *gin.Context) {
	// 从URL获取场景ID
	id := c.Param("id")

	// 查找场景
	scene, err := models.FindSceneByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "场景不存在: " + err.Error(),
		})
		return
	}

	// 删除场景
	err = scene.Delete()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除场景失败: " + err.Error(),
		})
		return
	}

	// 返回成功响应
	c.Status(http.StatusNoContent)
}