package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"foreignscan/internal/config"
	"foreignscan/internal/database"
	"foreignscan/internal/handlers"
	"foreignscan/internal/middleware"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	
	_ "foreignscan/docs" // 导入生成的docs包
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

// @title ForeignScan API
// @version 1.0
// @description ForeignScan后端API服务
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.foreignscan.com/support
// @contact.email support@foreignscan.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:3000
// @BasePath /api
// @schemes http

// 设置路由
func setupRoutes(r *gin.Engine) {
	// Swagger文档路由
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	
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
		
		// 根据日期获取图片
		api.GET("/images/by-date", handlers.GetImagesByDate)
		
		// 根据日期和场景ID获取图片
		api.GET("/images/by-date-scene", handlers.GetImagesByDateAndScene)
		
		// 上传图片
		api.POST("/upload", handlers.UploadImage)
		api.POST("/upload-image", handlers.UploadImage) // 兼容客户端的路由
		
		// 场景相关API - 使用迁移后的Gin处理器
		api.GET("/scenes", handlers.GetScenes)
		api.GET("/scenes/:id", handlers.GetScene)
		api.POST("/scenes", handlers.CreateScene)
		api.PUT("/scenes/:id", handlers.UpdateScene)
		api.DELETE("/scenes/:id", handlers.DeleteScene)
		// 获取特定场景下的图片列表
		api.GET("/scenes/:id/images", handlers.GetSceneImages)
		// 获取特定场景下的第一张图片
		api.GET("/scenes/:id/first-image", handlers.GetSceneFirstImage)	
		// 获取所有场景的第一张图片
		api.GET("/scenes/all/first-images", handlers.GetAllScenesFirstImage)
		
		// 样式图片相关API - 使用迁移后的Gin处理器
		api.GET("/style-images", handlers.GetStyleImages)
		api.GET("/style-images/scene/:sceneId", handlers.GetStyleImagesByScene)
		api.GET("/style-images/:id", handlers.GetStyleImage)
		api.POST("/style-images", handlers.UploadStyleImage)
		api.PUT("/style-images/:id", handlers.UpdateStyleImage)
		api.DELETE("/style-images/:id", handlers.DeleteStyleImage)

	}
}