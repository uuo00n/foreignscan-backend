package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"foreignscan/internal/config"
	"foreignscan/internal/models"
	"foreignscan/pkg/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

// UploadImage godoc
// @Summary 上传图片
// @Description 上传新图片到系统
// @Tags upload
// @Accept multipart/form-data
// @Produce json
// @Param roomId formData string true "房间ID"
// @Param pointId formData string true "点位ID"
// @Param file formData file true "要上传的图片文件"
// @Success 201 {object} map[string]interface{} "成功上传图片"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /upload [post]
func UploadImage(c *gin.Context) {
	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "没有上传文件",
		})
		return
	}

	// 获取请求中的元数据
	roomID := c.PostForm("roomId")
	pointID := c.PostForm("pointId")
	if roomID == "" || pointID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "roomId 与 pointId 均为必填",
		})
		return
	}
	if _, err := models.FindPointByIDAndRoom(pointID, roomID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "点位不属于该房间",
		})
		return
	}
	if _, err := models.FindStyleImageByPointID(pointID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "点位未绑定对照图，禁止上传检测图",
		})
		return
	}

	location := c.PostForm("location")

	// 生成文件名
	var filename string
	customFilename := c.Query("filename")
	if customFilename != "" {
		filename = customFilename
	} else {
		// 使用UUID生成唯一文件名
		ext := filepath.Ext(file.Filename)
		filename = utils.GenerateUUID() + ext
	}

	// 创建图片专用目录
	uploadsRoot := config.Get().UploadDir
	imagesDir := filepath.Join(uploadsRoot, "images", roomID, pointID)
	if err := utils.EnsureDir(imagesDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建图片目录失败: " + err.Error(),
		})
		return
	}

	// 保存文件到图片目录
	dstFS := filepath.Join(imagesDir, filename)
	if err := c.SaveUploadedFile(file, dstFS); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "保存文件失败: " + err.Error(),
		})
		return
	}

	dstWeb := filepath.ToSlash(filepath.Join("uploads", "images", roomID, pointID, filename))

	// 获取下一个序列号
	sequenceNumber, err := models.GetNextSequence()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取序列号失败",
		})
		return
	}

	// 创建新的图片记录
	now := time.Now()
	newImage := models.Image{
		ID:               utils.GenerateObjectID(),
		SequenceNumber:   sequenceNumber,
		RoomID:           roomID,
		PointID:          pointID,
		Timestamp:        now,
		Location:         location,
		Filename:         filename,
		Path:             dstWeb,
		IsDetected:       false,
		HasIssue:         false,
		IssueType:        "",
		Status:           models.ImageStatusUndetected, // 新上传图片默认状态为“未检测”
		DetectionResults: datatypes.JSONSlice[models.DetectionItem]{},
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// 保存到数据库
	if err := newImage.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "保存图片信息失败",
		})
		return
	}

	// 构建正确的访问路径
	accessPath := fmt.Sprintf("/uploads/images/%s/%s/%s", roomID, pointID, filename)

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"file":           filename,
		"path":           accessPath,
		"imageId":        newImage.ID,
		"sequenceNumber": sequenceNumber,
	})
}
