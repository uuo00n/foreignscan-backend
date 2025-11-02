package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"foreignscan/internal/models"
	"foreignscan/pkg/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetStyleImages 获取所有样式图
func GetStyleImages(c *gin.Context) {
	// 获取所有样式图
	styleImages, err := models.FindAllStyleImages()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取样式图失败: " + err.Error(),
		})
		return
	}

	// 返回JSON响应
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"styleImages": styleImages,
	})
}

// GetStyleImagesByScene 获取指定场景的所有样式图
func GetStyleImagesByScene(c *gin.Context) {
	// 从URL获取场景ID
	sceneID := c.Param("sceneId")

	// 查找样式图
	styleImages, err := models.FindStyleImagesBySceneID(sceneID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取样式图失败: " + err.Error(),
		})
		return
	}

	// 返回JSON响应
	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"styleImages": styleImages,
	})
}

// GetStyleImage 获取单个样式图
func GetStyleImage(c *gin.Context) {
	// 从URL获取样式图ID
	id := c.Param("id")

	// 查找样式图
	styleImage, err := models.FindStyleImageByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "样式图不存在: " + err.Error(),
		})
		return
	}

	// 返回JSON响应
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"styleImage": styleImage,
	})
}

// UploadStyleImage 上传样式图
func UploadStyleImage(c *gin.Context) {
	// 获取场景ID
	sceneIDStr := c.PostForm("sceneId")
	if sceneIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "缺少场景ID",
		})
		return
	}

	// 将场景ID转换为ObjectID
	sceneID, err := primitive.ObjectIDFromHex(sceneIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的场景ID",
		})
		return
	}

	// 获取上传的文件
	file, header, err := c.Request.FormFile("styleImage")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "获取上传文件失败: " + err.Error(),
		})
		return
	}
	defer file.Close()

	// 创建样式图目录
	styleDir := filepath.Join("./uploads/styles", sceneID.Hex())
	if err := utils.EnsureDir(styleDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建样式图目录失败: " + err.Error(),
		})
		return
	}

	// 生成唯一文件名
	ext := filepath.Ext(header.Filename)
	filename := "style_" + time.Now().Format("20060102150405") + ext
	filePath := filepath.Join(styleDir, filename)
	
	// 保存文件
	if err := c.SaveUploadedFile(header, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "保存文件失败: " + err.Error(),
		})
		return
	}

	// 创建样式图记录
	styleImage := models.StyleImage{
		SceneID:     sceneID,
		Name:        c.PostForm("name"),
		Description: c.PostForm("description"),
		Filename:    filename,
		Path:        filePath,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 保存样式图记录
	err = styleImage.Save()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "保存样式图记录失败: " + err.Error(),
		})
		return
	}

	// 返回JSON响应
	c.JSON(http.StatusCreated, gin.H{
		"success":    true,
		"styleImage": styleImage,
	})
}

// UpdateStyleImage 更新样式图
func UpdateStyleImage(c *gin.Context) {
	// 从URL获取样式图ID
	id := c.Param("id")

	// 将ID转换为ObjectID
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的样式图ID",
		})
		return
	}

	// 解析请求体
	var updatedStyleImage models.StyleImage
	if err := c.ShouldBindJSON(&updatedStyleImage); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的请求数据: " + err.Error(),
		})
		return
	}

	// 查找现有样式图
	existingStyleImage, err := models.FindStyleImageByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "样式图不存在: " + err.Error(),
		})
		return
	}

	// 更新样式图字段
	updatedStyleImage.ID = objID
	updatedStyleImage.Filename = existingStyleImage.Filename
	updatedStyleImage.Path = existingStyleImage.Path
	updatedStyleImage.CreatedAt = existingStyleImage.CreatedAt
	updatedStyleImage.UpdatedAt = time.Now()

	// 保存更新
	err = updatedStyleImage.Update()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "更新样式图失败: " + err.Error(),
		})
		return
	}

	// 返回JSON响应
	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"styleImage": updatedStyleImage,
	})
}

// DeleteStyleImage 删除样式图
func DeleteStyleImage(c *gin.Context) {
	// 从URL获取样式图ID
	id := c.Param("id")

	// 查找样式图
	styleImage, err := models.FindStyleImageByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "样式图不存在: " + err.Error(),
		})
		return
	}

	// 删除文件
	if err := os.Remove(styleImage.Path); err != nil && !os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除文件失败: " + err.Error(),
		})
		return
	}

	// 删除样式图记录
	err = styleImage.Delete()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除样式图记录失败: " + err.Error(),
		})
		return
	}

	// 返回成功响应
	c.Status(http.StatusNoContent)
}