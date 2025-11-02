package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"foreignscan/internal/config"
	"foreignscan/internal/database"
	"foreignscan/internal/handlers"
	"foreignscan/internal/middleware"
	"foreignscan/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func main() {
	// 加载配置
	cfg := config.Load()

	// 创建Gin引擎
	r := gin.Default()

	// 应用中间件
	middleware.Setup(r)

	// 确保上传目录存在
	if _, err := os.Stat("uploads"); os.IsNotExist(err) {
		os.Mkdir("uploads", 0755)
	}

	// 静态文件服务 - 使用相对路径，适合交付到客户环境
	// 直接使用项目根目录下的uploads文件夹
	uploadsPath := "./uploads"
	r.Static("/uploads", uploadsPath)
	r.Static("/public", "./public")
	
	// 添加调试日志
	fmt.Printf("上传目录路径: %s\n", uploadsPath)

	// 初始化数据库连接
	if err := database.Connect(); err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer database.Close()

	// 注册路由
	setupRoutes(r)

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: r,
	}

	// 在goroutine中启动服务器
	go func() {
		fmt.Printf("服务器启动在 http://localhost:%d\n", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("监听失败: %s\n", err)
		}
	}()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("关闭服务器...")

	// 设置5秒的超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("服务器强制关闭:", err)
	}

	log.Println("服务器优雅退出")
}

// 设置路由
func setupRoutes(r *gin.Engine) {
	// 健康检查路由
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "Server is running",
		})
	})

	// API路由组
	api := r.Group("/api")
	{
		// 获取图片列表
		api.GET("/images", handlers.GetImages)
		
		// 获取单个图片详细信息
		api.GET("/images/:id", handlers.GetImageDetail)
		
		// 获取特定场景下的图片列表
		api.GET("/scenes/:id/images", handlers.GetSceneImages)
		
		// 获取特定场景下的第一张图片
		api.GET("/scenes/:id/first-image", handlers.GetSceneFirstImage)
		
		// 获取所有场景的第一张图片
		api.GET("/scenes/all/first-images", handlers.GetAllScenesFirstImage)
		
		// 上传图片
		api.POST("/upload", handlers.UploadImage)
		api.POST("/upload-image", handlers.UploadImage) // 兼容客户端的路由
		
		// 检测图片
		api.POST("/detect", handlers.DetectImage)
		
		// 场景相关API - 转换为Gin风格的处理函数
		api.GET("/scenes", func(c *gin.Context) {
			scenes, err := models.FindAllScenes()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": "获取场景失败: " + err.Error(),
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"scenes": scenes,
			})
		})
		api.GET("/scenes/:id", func(c *gin.Context) {
			id := c.Param("id")
			scene, err := models.FindSceneByID(id)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{
					"success": false,
					"message": "场景不存在: " + err.Error(),
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"scene": scene,
			})
		})
		api.POST("/scenes", func(c *gin.Context) {
			var scene models.Scene
			if err := c.ShouldBindJSON(&scene); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"message": "无效的请求数据: " + err.Error(),
				})
				return
			}
			scene.CreatedAt = time.Now()
			scene.UpdatedAt = time.Now()
			if err := scene.Save(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": "创建场景失败: " + err.Error(),
				})
				return
			}
			c.JSON(http.StatusCreated, gin.H{
				"success": true,
				"scene": scene,
			})
		})
		
		// 样式图片相关API - 转换为Gin风格的处理函数
		api.GET("/style-images", func(c *gin.Context) {
			styleImages, err := models.FindAllStyleImages()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": "获取样式图失败: " + err.Error(),
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"styleImages": styleImages,
			})
		})
		api.GET("/style-images/scene/:sceneId", func(c *gin.Context) {
			sceneId := c.Param("sceneId")
			styleImages, err := models.FindStyleImagesBySceneID(sceneId)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": "获取样式图失败: " + err.Error(),
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"styleImages": styleImages,
			})
		})
		
		// 上传样式图片
		api.POST("/style-images", func(c *gin.Context) {
			// 获取表单数据
			file, header, err := c.Request.FormFile("file")
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"message": "获取上传文件失败: " + err.Error(),
				})
				return
			}
			defer file.Close()
			
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
			
			// 创建样式图目录
			styleDir := filepath.Join("./uploads/styles", sceneIDStr)
			if err := os.MkdirAll(styleDir, os.ModePerm); err != nil {
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
				ID:          primitive.NewObjectID(),
				Name:        c.PostForm("name"),
				Description: c.PostForm("description"),
				SceneID:     sceneID,
				Filename:    filename,
				Path:        filePath,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			
			// 保存到数据库
			if err := styleImage.Save(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": "保存样式图片记录失败: " + err.Error(),
				})
				return
			}
			
			c.JSON(http.StatusCreated, gin.H{
				"success": true,
				"styleImage": styleImage,
			})
		})
	}
}