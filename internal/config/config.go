package config

import (
	"os"
	"strconv"
)

// Config 应用配置结构
type Config struct {
	Port           int    // 服务器端口
	MongoURI       string // MongoDB连接URI
	DatabaseName   string // 数据库名称
	UploadDir      string // 上传目录
	AllowedOrigins string // 允许的CORS源
}

// Load 加载配置
func Load() *Config {
	// 默认配置
	cfg := &Config{
		Port:           3000,
		MongoURI:       "mongodb://localhost:27017",
		DatabaseName:   "foreignscan",
		UploadDir:      "uploads",
		AllowedOrigins: "*",
	}

	// 从环境变量加载配置（如果存在）
	if port := os.Getenv("PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Port = p
		}
	}

	if mongoURI := os.Getenv("MONGO_URI"); mongoURI != "" {
		cfg.MongoURI = mongoURI
	}

	if dbName := os.Getenv("DB_NAME"); dbName != "" {
		cfg.DatabaseName = dbName
	}

	if uploadDir := os.Getenv("UPLOAD_DIR"); uploadDir != "" {
		cfg.UploadDir = uploadDir
	}

	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		cfg.AllowedOrigins = origins
	}

	return cfg
}