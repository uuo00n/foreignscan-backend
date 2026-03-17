package handlers

import (
	"net/http"
	"time"

	"foreignscan/internal/models"

	"github.com/gin-gonic/gin"
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
	images, err := models.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "获取图片列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "images": images})
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
	dateStr := c.Query("date")
	if dateStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "日期参数不能为空"})
		return
	}

	if _, err := time.Parse("2006-01-02", dateStr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "日期格式错误，请使用YYYY-MM-DD格式"})
		return
	}

	images, err := models.FindImagesByDate(dateStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "获取图片失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "date": dateStr, "count": len(images), "images": images})
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
	imageIDStr := c.Param("id")
	image, err := models.FindByID(imageIDStr)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "未找到图片: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "image": image})
}

// GetImagesByStatusTime godoc
// @Summary 根据多条件筛选图片（状态/时间/房间/点位）
// @Description 支持按状态、房间ID、点位ID、时间范围（start/end）筛选图片。
// @Tags images
// @Accept json
// @Produce json
// @Param status query string false "状态（已检测/未检测）"
// @Param hasIssue query string false "是否存在问题（true/false，仅在status=已检测时生效）"
// @Param roomId query string false "房间ID"
// @Param pointId query string false "点位ID"
// @Param start query string false "起始时间（YYYY-MM-DD 或 RFC3339）"
// @Param end query string false "结束时间（YYYY-MM-DD 或 RFC3339）"
// @Success 200 {object} map[string]interface{} "成功获取筛选结果"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Failure 500 {object} map[string]interface{} "服务器错误"
// @Router /images/filter [get]
func GetImagesByStatusTime(c *gin.Context) {
	status := c.Query("status")
	roomIDStr := c.Query("roomId")
	pointIDStr := c.Query("pointId")
	startStr := c.Query("start")
	endStr := c.Query("end")
	hasIssueStr := c.Query("hasIssue")

	var startDate, endDate *time.Time

	if hasIssueStr != "" {
		if hasIssueStr == "true" || hasIssueStr == "1" {
			status = "has_issue"
		} else {
			status = "no_issue"
		}
	}

	parseTime := func(s string, isStart bool) (*time.Time, error) {
		if len(s) == 10 {
			d, err := time.Parse("2006-01-02", s)
			if err != nil {
				return nil, err
			}
			var t time.Time
			if isStart {
				t = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())
			} else {
				t = time.Date(d.Year(), d.Month(), d.Day(), 23, 59, 59, int(time.Second-time.Nanosecond), d.Location())
			}
			return &t, nil
		}
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, err
		}
		return &t, nil
	}

	if startStr != "" {
		var err error
		startDate, err = parseTime(startStr, true)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "start时间格式错误"})
			return
		}
	}
	if endStr != "" {
		var err error
		endDate, err = parseTime(endStr, false)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "end时间格式错误"})
			return
		}
	}

	images, total, err := models.FindImagesByFilter(status, roomIDStr, pointIDStr, startDate, endDate, 1, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "查询失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"filters": gin.H{
			"status":   status,
			"roomId":   roomIDStr,
			"pointId":  pointIDStr,
			"start":    startStr,
			"end":      endStr,
			"hasIssue": hasIssueStr,
		},
		"count":  total,
		"images": images,
	})
}
