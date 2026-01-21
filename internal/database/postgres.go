package database

import (
	"fmt"
	"log"
	"sync"
	"time"

	"foreignscan/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	db   *gorm.DB
	once sync.Once
)

// Connect 连接到PostgreSQL数据库
func Connect() error {
	var err error
	once.Do(func() {
		cfg := config.Get()

		// 配置GORM日志
		newLogger := logger.New(
			log.New(log.Writer(), "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		)

		db, err = gorm.Open(postgres.Open(cfg.PostgresDSN), &gorm.Config{
			Logger: newLogger,
		})
		if err != nil {
			return
		}

		// 获取底层sql.DB以设置连接池
		sqlDB, err := db.DB()
		if err != nil {
			return
		}

		// 设置连接池
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		sqlDB.SetConnMaxLifetime(time.Hour)
	})

	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	return nil
}

// GetDB 获取数据库连接单例
func GetDB() *gorm.DB {
	if db == nil {
		if err := Connect(); err != nil {
			log.Fatalf("Database not initialized: %v", err)
		}
	}
	return db
}

// Close 关闭数据库连接 (GORM通常不需要手动关闭，但在服务退出时可调用底层sql.DB关闭)
func Close() {
	if db != nil {
		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.Close()
		}
	}
}

// AutoMigrate 自动迁移表结构
func AutoMigrate(models ...interface{}) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}
	return db.AutoMigrate(models...)
}
