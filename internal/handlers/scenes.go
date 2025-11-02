package handlers

import (
	"net/http"
	"time"

	"foreignscan/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetScenes 获取所有场景
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

// GetScene 获取单个场景
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

// CreateScene 创建新场景
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