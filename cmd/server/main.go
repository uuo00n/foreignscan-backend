package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"foreignscan/internal/config"
	"foreignscan/internal/database"
	"foreignscan/internal/handlers"
	"foreignscan/internal/middleware"
	"foreignscan/internal/models"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "foreignscan/docs" // 导入生成的docs包
	"foreignscan/pkg/utils"
)

func main() {
	// 初始化日志
	utils.InitLogger()
	defer utils.GetLogger().Sync()

	// 加载配置
	cfg := config.Get()

	// 创建Gin引擎
	r := gin.Default()

	// 应用中间件
	middleware.Setup(r)

	// 确保上传目录存在
	if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		log.Fatalf("创建上传目录失败: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(cfg.UploadDir, "labels"), 0o755); err != nil {
		log.Fatalf("创建 labels 目录失败: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(cfg.UploadDir, "images"), 0o755); err != nil {
		log.Fatalf("创建 images 目录失败: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(cfg.UploadDir, "styles"), 0o755); err != nil {
		log.Fatalf("创建 styles 目录失败: %v", err)
	}

	// 静态文件服务
	r.Static("/uploads", cfg.UploadDir)
	r.Static("/public", "./public")

	// 添加调试日志
	fmt.Printf("上传目录路径: %s\n", cfg.UploadDir)

	// 初始化数据库连接
	if err := database.Connect(); err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	// 自动迁移核心表结构
	if err := database.AutoMigrate(&models.Room{}, &models.Point{}, &models.Image{}, &models.StyleImage{}, &models.DetectionRun{}, &models.PadBindingKey{}); err != nil {
		log.Fatalf("数据库自动迁移失败: %v", err)
	}
	if err := database.GetDB().Exec(`
DO $$
BEGIN
  IF to_regclass('public.rooms') IS NOT NULL THEN
    ALTER TABLE rooms DROP COLUMN IF EXISTS model_path;
  END IF;
END $$;
`).Error; err != nil {
		log.Fatalf("数据库字段迁移失败（删除 rooms.model_path）: %v", err)
	}
	defer database.Close()

	// 已移除 issues/comparisons 索引初始化（未使用）

	// 注册路由
	setupRoutes(r)
	if err := validateCriticalRoutes(r); err != nil {
		log.Fatalf("关键路由契约自检失败: %v", err)
	}
	log.Printf("关键路由契约自检通过: %s", strings.Join(requiredRouteContracts(), ", "))

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
	redirectToSwagger := func(c *gin.Context) {
		c.Redirect(http.StatusTemporaryRedirect, "/swagger/index.html")
	}

	r.GET("/", redirectToSwagger)
	r.GET("/docs", redirectToSwagger)
	r.GET("/docs/", redirectToSwagger)

	// Swagger文档路由
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 健康检查路由
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "Server is running",
		})
	})

	// 标准健康检查
	r.GET("/health", handlers.HealthCheck)
	r.GET("/ready", handlers.ReadinessCheck)

	// API路由组 (v1)
	api := r.Group("/api")
	{
		// 获取图片列表
		api.GET("/images", handlers.GetImages)

		// 获取单个图片详细信息
		api.GET("/images/:id", handlers.GetImageDetail)

		// 根据日期获取图片
		api.GET("/images/by-date", handlers.GetImagesByDate)

		// 新增：根据状态或状态+时间范围筛选图片
		api.GET("/images/filter", handlers.GetImagesByStatusTime)

		// 上传图片
		api.POST("/upload", handlers.UploadImage)
		api.POST("/upload-image", handlers.UploadImage) // 兼容客户端的路由

		// 房间-点位配置
		api.GET("/rooms/tree", handlers.GetRoomsTree)
		api.GET("/pad/room-context", handlers.GetPadRoomContext)
		api.POST("/pad/bind", handlers.BindPadWithKey)
		api.POST("/rooms/import", handlers.ImportRooms)
		api.POST("/rooms/:roomId/pad-binding-keys", handlers.CreateRoomPadBindingKey)
		api.POST("/rooms/:roomId/points", handlers.CreatePoint)
		api.DELETE("/rooms/:roomId/points/:pointId", handlers.DeletePoint)

		// 样式图片相关API - 使用迁移后的Gin处理器
		api.GET("/style-images", handlers.GetStyleImages)
		api.GET("/style-images/point/:pointId", handlers.GetStyleImageByPoint)
		api.GET("/style-images/:id", handlers.GetStyleImage)
		api.POST("/style-images", handlers.UploadStyleImage)
		api.PUT("/style-images/:id", handlers.UpdateStyleImage)
		api.DELETE("/style-images/:id", handlers.DeleteStyleImage)
		// 检测结果相关API
		api.GET("/images/:id/detections", handlers.GetImageDetections)
		api.POST("/images/:id/detections", handlers.CreateImageDetection)
		api.GET("/detections", handlers.QueryDetections)

		// 单图异步推理
		api.POST("/images/:id/detect", handlers.StartImageDetect)
		// 兼容前端直接调用 /api/detect，传 imageId
		api.POST("/detect", handlers.DetectEntry)
		// 同步推理入口（房间+点位+文件）
		api.POST("/predict", handlers.Predict)
		// 房间-模型绑定（代理到 YOLO 服务）
		api.GET("/room-models", handlers.GetRoomModels)
		api.PUT("/room-models/:roomId", handlers.PutRoomModel)
		api.DELETE("/room-models/:roomId", handlers.DeleteRoomModel)
		// 任务管理：取消与实时进度
		api.DELETE("/detect/jobs/:id", handlers.CancelDetectJob)
		api.GET("/detect/jobs/:id/stream", handlers.GetDetectJobStream)
		api.GET("/detect/jobs/:id", handlers.GetDetectJob)

		// 已移除问题与对比相关API（未使用）
	}
}

func requiredRouteContracts() []string {
	return []string{
		http.MethodGet + " /api/rooms/tree",
		http.MethodGet + " /api/style-images",
		http.MethodGet + " /api/room-models",
	}
}

func validateCriticalRoutes(r *gin.Engine) error {
	required := requiredRouteContracts()
	registered := make(map[string]struct{}, len(r.Routes()))
	for _, route := range r.Routes() {
		registered[route.Method+" "+route.Path] = struct{}{}
	}

	missing := make([]string, 0)
	for _, route := range required {
		if _, ok := registered[route]; !ok {
			missing = append(missing, route)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(missing)
	return fmt.Errorf("缺少关键路由: %s", strings.Join(missing, ", "))
}
