package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"time"

	"foreignscan/internal/config"
	"foreignscan/internal/models"
	internalutils "foreignscan/internal/utils"
	"foreignscan/pkg/utils"

	"github.com/gin-gonic/gin"
)

// GetStyleImages godoc
// @Summary 获取所有样式图片
// @Description 获取系统中所有可用的样式图片列表
// @Tags style-images
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "成功获取样式图片列表"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /style-images [get]
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

// GetStyleImagesByScene godoc
// @Summary 获取指定场景的所有样式图
// @Description 获取特定场景下的所有样式图片列表
// @Tags style-images
// @Accept json
// @Produce json
// @Param sceneId path string true "场景ID"
// @Success 200 {object} map[string]interface{} "成功获取样式图片列表"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /style-images/scene/{sceneId} [get]
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

// GetStyleImage godoc
// @Summary 获取单个样式图
// @Description 根据ID获取特定样式图片的详细信息
// @Tags style-images
// @Accept json
// @Produce json
// @Param id path string true "样式图ID"
// @Success 200 {object} map[string]interface{} "成功获取样式图"
// @Failure 404 {object} map[string]interface{} "样式图不存在"
// @Router /style-images/{id} [get]
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

// UploadStyleImage godoc
// @Summary 上传样式图
// @Description 上传新的样式图片并关联到特定场景
// @Tags style-images
// @Accept multipart/form-data
// @Produce json
// @Param sceneId formData string true "场景ID"
// @Param file formData file true "样式图文件"
// @Param name formData string false "样式图名称"
// @Success 200 {object} map[string]interface{} "成功上传样式图"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /style-images [post]
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
	sceneID := sceneIDStr

	// 获取上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "获取上传文件失败: " + err.Error(),
		})
		return
	}
	defer file.Close()

	// 创建样式图目录
	uploadsRoot := config.Get().UploadDir
	styleDir := filepath.Join(uploadsRoot, "styles", sceneID)
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
	filePathFS := filepath.Join(styleDir, filename)

	// 保存文件
	if err := c.SaveUploadedFile(header, filePathFS); err != nil {
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
		Path:        filepath.ToSlash(filepath.Join("uploads", "styles", sceneID, filename)),
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

// UpdateStyleImage godoc
// @Summary 更新样式图
// @Description 更新已存在的样式图信息
// @Tags style-images
// @Accept json
// @Produce json
// @Param id path string true "样式图ID"
// @Param styleImage body models.StyleImage true "更新的样式图信息"
// @Success 200 {object} map[string]interface{} "成功更新样式图"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "样式图不存在"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /style-images/{id} [put]
func UpdateStyleImage(c *gin.Context) {
	// 从URL获取样式图ID
	id := c.Param("id")

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
	updatedStyleImage.ID = id
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

// DeleteStyleImage godoc
// @Summary 删除样式图
// @Description 删除指定的样式图及其文件
// @Tags style-images
// @Accept json
// @Produce json
// @Param id path string true "样式图ID"
// @Success 204 "成功删除样式图"
// @Failure 404 {object} map[string]interface{} "样式图不存在"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /style-images/{id} [delete]
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
	if err := os.Remove(internalutils.NormalizeUploadsLocalPath(styleImage.Path)); err != nil && !os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除文件失败: " + err.Error(),
		})
		return
	}

	// 删除样式图记录
	err = models.DeleteStyleImage(id)
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
