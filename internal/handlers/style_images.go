package handlers

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"foreignscan/internal/config"
	"foreignscan/internal/models"
	internalutils "foreignscan/internal/utils"
	"foreignscan/pkg/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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
	styleImages, err := models.FindAllStyleImages()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取样式图失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"styleImages": styleImages,
	})
}

// GetStyleImageByPoint godoc
// @Summary 获取指定点位的样式图
// @Description 每个点位仅允许绑定一张样式图
// @Tags style-images
// @Accept json
// @Produce json
// @Param pointId path string true "点位ID"
// @Success 200 {object} map[string]interface{} "成功获取样式图片"
// @Failure 404 {object} map[string]interface{} "未绑定样式图片"
// @Router /style-images/point/{pointId} [get]
func GetStyleImageByPoint(c *gin.Context) {
	pointID := c.Param("pointId")
	styleImage, err := models.FindStyleImageByPointID(pointID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "点位未绑定对照图"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "获取样式图失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "styleImage": styleImage})
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
	id := c.Param("id")
	styleImage, err := models.FindStyleImageByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "样式图不存在: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "styleImage": styleImage})
}

// UploadStyleImage godoc
// @Summary 上传点位对照图
// @Description 上传或替换点位对照图（point 一对一绑定）
// @Tags style-images
// @Accept multipart/form-data
// @Produce json
// @Param pointId formData string true "点位ID"
// @Param file formData file true "样式图文件"
// @Param name formData string false "样式图名称"
// @Success 201 {object} map[string]interface{} "成功上传样式图"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /style-images [post]
func UploadStyleImage(c *gin.Context) {
	pointID := c.PostForm("pointId")
	if pointID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "缺少点位ID"})
		return
	}

	point, err := models.FindPointByID(pointID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "点位不存在"})
		return
	}

	header, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "获取上传文件失败: " + err.Error()})
		return
	}

	uploadsRoot := config.Get().UploadDir
	styleDir := filepath.Join(uploadsRoot, "styles", point.RoomID, pointID)
	if err := utils.EnsureDir(styleDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "创建样式图目录失败: " + err.Error()})
		return
	}

	ext := filepath.Ext(header.Filename)
	filename := "style_" + time.Now().Format("20060102150405") + ext
	filePathFS := filepath.Join(styleDir, filename)
	if err := c.SaveUploadedFile(header, filePathFS); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "保存文件失败: " + err.Error()})
		return
	}

	path := filepath.ToSlash(filepath.Join("uploads", "styles", point.RoomID, pointID, filename))
	styleImage, findErr := models.FindStyleImageByPointID(pointID)
	if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "查询现有对照图失败: " + findErr.Error()})
		return
	}

	if styleImage != nil {
		if styleImage.Path != "" {
			_ = os.Remove(internalutils.NormalizeUploadsLocalPath(styleImage.Path))
		}
		styleImage.Name = c.PostForm("name")
		styleImage.Description = c.PostForm("description")
		styleImage.Filename = filename
		styleImage.Path = path
		styleImage.UpdatedAt = time.Now()
		if err := styleImage.Update(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "更新样式图记录失败: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "styleImage": styleImage})
		return
	}

	created := models.StyleImage{
		PointID:     pointID,
		Name:        c.PostForm("name"),
		Description: c.PostForm("description"),
		Filename:    filename,
		Path:        path,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if err := created.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "保存样式图记录失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "styleImage": created})
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
	id := c.Param("id")

	var updatedStyleImage models.StyleImage
	if err := c.ShouldBindJSON(&updatedStyleImage); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的请求数据: " + err.Error()})
		return
	}

	existingStyleImage, err := models.FindStyleImageByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "样式图不存在: " + err.Error()})
		return
	}

	updatedStyleImage.ID = id
	updatedStyleImage.PointID = existingStyleImage.PointID
	updatedStyleImage.Filename = existingStyleImage.Filename
	updatedStyleImage.Path = existingStyleImage.Path
	updatedStyleImage.CreatedAt = existingStyleImage.CreatedAt
	updatedStyleImage.UpdatedAt = time.Now()

	if err := updatedStyleImage.Update(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "更新样式图失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "styleImage": updatedStyleImage})
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
	id := c.Param("id")
	styleImage, err := models.FindStyleImageByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "样式图不存在: " + err.Error()})
		return
	}

	if err := os.Remove(internalutils.NormalizeUploadsLocalPath(styleImage.Path)); err != nil && !os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "删除文件失败: " + err.Error()})
		return
	}

	if err := models.DeleteStyleImage(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "删除样式图记录失败: " + err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
