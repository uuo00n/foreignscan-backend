package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"foreignscan/internal/models"
	"foreignscan/pkg/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UploadImage 处理图片上传
func UploadImage(c *gin.Context) {
	// 获取上传的文件
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "没有上传文件",
		})
		return
	}

	// 获取请求中的元数据
	sceneIDStr := c.PostForm("sceneId")
	var sceneID primitive.ObjectID
	
	// 如果提供了场景ID，尝试转换为ObjectID
	if sceneIDStr != "" {
		var err error
		sceneID, err = primitive.ObjectIDFromHex(sceneIDStr)
		if err != nil {
			// 如果转换失败，创建一个新的ObjectID
			sceneID = primitive.NewObjectID()
		}
	} else {
		// 如果没有提供场景ID，创建一个新的ObjectID
		sceneID = primitive.NewObjectID()
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
	imagesDir := filepath.Join("uploads/images", sceneID.Hex())
	if err := utils.EnsureDir(imagesDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建图片目录失败: " + err.Error(),
		})
		return
	}
	
	// 保存文件到图片目录
	dst := filepath.Join(imagesDir, filename)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "保存文件失败: " + err.Error(),
		})
		return
	}

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
		ID:               primitive.NewObjectID(),
		SequenceNumber:   sequenceNumber,
		SceneID:          sceneID,
		Timestamp:        now,
		Location:         location,
		Filename:         filename,
		Path:             dst,
		IsDetected:       false,
		HasIssue:         false,
		IssueType:        "",
		DetectionResults: []interface{}{},
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

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"file":           filename,
		"path":           fmt.Sprintf("/uploads/%s", filename),
		"imageId":        newImage.ID.Hex(),
		"sequenceNumber": sequenceNumber,
	})
}