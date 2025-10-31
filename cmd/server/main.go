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

	// 静态文件服务
	r.Static("/uploads", "../../uploads")
	r.Static("/public", "./public")

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
		
		// 上传图片
		api.POST("/upload", handlers.UploadImage)
		api.POST("/upload-image", handlers.UploadImage) // 兼容客户端的路由
		
		// 检测图片
		api.POST("/detect", handlers.DetectImage)
	}
}