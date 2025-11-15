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
// @Summary 获取场景的最新图片
// @Description 根据场景ID获取该场景下的最新图片（按createdAt降序取第一条）
// @Tags scenes,images
// @Accept json
// @Produce json
// @Param id path string true "场景ID"
// @Success 200 {object} map[string]interface{} "成功获取场景最新图片"
// @Failure 400 {object} map[string]interface{} "请求参数错误"
// @Failure 404 {object} map[string]interface{} "场景不存在或没有图片"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /scenes/{id}/first-image [get]
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

    // 查找该场景下的最新一张图片（按createdAt降序）
    image, err := models.FindFirstBySceneID(sceneID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
        "message": "获取场景最新图片失败: " + err.Error(),
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
// @Summary 获取所有场景的最新图片
// @Description 获取系统中所有场景的最新图片（按createdAt降序取第一条），用于场景预览展示
// @Tags scenes,images
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "成功获取所有场景的最新图片"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /scenes/all/first-images [get]
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
		SceneID    primitive.ObjectID `json:"sceneId"`
		SceneName  string             `json:"sceneName"`
		FirstImage *models.Image      `json:"firstImage"`
	}

	result := make([]SceneWithFirstImage, 0, len(scenes))

    // 遍历所有场景，获取每个场景的最新一张图片（按createdAt降序）
    for _, scene := range scenes {
        // 查找该场景下的最新一张图片（按createdAt降序）
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

// GetImagesByStatusTime godoc
// @Summary 根据状态或状态+时间范围筛选图片
// @Description 按状态（必填）与可选的时间范围（start/end）筛选图片。时间格式支持 YYYY-MM-DD 或 RFC3339（如 2025-11-10T15:00:00Z）。
// @Tags images
// @Accept json
// @Produce json
// @Param status query string true "状态（已检测/未检测）"
// @Param hasIssue query string false "是否存在问题（true/false，仅在status=已检测时生效）"
// @Param start query string false "起始时间（YYYY-MM-DD 或 RFC3339）"
// @Param end query string false "结束时间（YYYY-MM-DD 或 RFC3339）"
// @Success 200 {object} map[string]interface{} "成功获取筛选结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /images/filter [get]
func GetImagesByStatusTime(c *gin.Context) {
    // 1) 读取并校验状态参数（必填）
    status := c.Query("status")
    if status == "" {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "status为必填参数"})
        return
    }
    // 允许的状态值（中文）
    valid := map[string]bool{
        models.ImageStatusUndetected: true,
        models.ImageStatusDetected:   true,
    }
    if !valid[status] {
        c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "status仅支持：已检测/未检测"})
        return
    }

    // 2) 解析时间参数（可选）。支持两种格式：
    // - 日期：YYYY-MM-DD（自动扩展为当天的00:00:00至23:59:59）
    // - RFC3339：如 2025-11-10T15:00:00Z
    startStr := c.Query("start")
    endStr := c.Query("end")
    hasIssueStr := c.Query("hasIssue")
    hasIssueParamProvided := false
    hasIssueVal := false
    if hasIssueStr != "" {
        hasIssueParamProvided = true
        if hasIssueStr == "true" || hasIssueStr == "1" { hasIssueVal = true }
        if hasIssueStr == "false" || hasIssueStr == "0" { /* remains false */ }
    }

    parseTime := func(s string, isStart bool) (time.Time, error) {
        if len(s) == 10 { // YYYY-MM-DD
            d, err := time.Parse("2006-01-02", s)
            if err != nil {
                return time.Time{}, err
            }
            if isStart {
                return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location()), nil
            }
            return time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), d.Location()), nil
        }
        // 尝试RFC3339
        return time.Parse(time.RFC3339, s)
    }

    var (
        images []models.Image
        err    error
    )

    // 3) 根据是否提供时间参数选择查询方法
    if startStr == "" && endStr == "" {
        // 仅按状态/flags筛选
        if status == models.ImageStatusUndetected {
            images, err = models.FindByFlags(false, false, false)
        } else {
            // 已检测，可选按 hasIssue 进一步筛选
            images, err = models.FindByFlags(true, hasIssueParamProvided, hasIssueVal)
        }
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "查询失败: " + err.Error()})
            return
        }
    } else {
        // 按状态+时间范围筛选
        var (
            start time.Time
            end   time.Time
        )
        if startStr != "" {
            start, err = parseTime(startStr, true)
            if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "start时间格式错误，支持YYYY-MM-DD或RFC3339"})
                return
            }
        } else {
            // 未提供start，则默认使用最早时间
            start = time.Unix(0, 0)
        }
        if endStr != "" {
            end, err = parseTime(endStr, false)
            if err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "end时间格式错误，支持YYYY-MM-DD或RFC3339"})
                return
            }
        } else {
            // 未提供end，则默认当前时间
            end = time.Now()
        }

        if status == models.ImageStatusUndetected {
            images, err = models.FindByFlagsAndTimeRange(false, false, false, start, end)
        } else {
            images, err = models.FindByFlagsAndTimeRange(true, hasIssueParamProvided, hasIssueVal, start, end)
        }
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "查询失败: " + err.Error()})
            return
        }
    }

    // 4) 返回结果
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "filters": gin.H{"status": status, "start": startStr, "end": endStr, "hasIssue": hasIssueStr},
        "count":   len(images),
        "images":  images,
    })
}
