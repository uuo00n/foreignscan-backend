package database

import (
	"context"
	"time"

	"foreignscan/internal/config"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	// 全局数据库连接
	client     *mongo.Client
	database   *mongo.Database
	collections map[string]*mongo.Collection
)

// Connect 连接到MongoDB数据库
func Connect() error {
	cfg := config.Load()
	
	// 设置MongoDB连接选项
	clientOptions := options.Client().ApplyURI(cfg.MongoURI)
	
	// 连接到MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	var err error
	client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		return err
	}
	
	// 检查连接
	err = client.Ping(ctx, nil)
	if err != nil {
		return err
	}
	
	// 获取数据库
	database = client.Database(cfg.DatabaseName)
	
	// 初始化集合映射
	collections = make(map[string]*mongo.Collection)

	// 添加所有需要的集合
	collections["images"] = database.Collection("images")
	collections["scenes"] = database.Collection("scenes")
	collections["styleImages"] = database.Collection("styleImages")
	// 新增YOLO检测结果集合
	collections["detections"] = database.Collection("detections")

	return nil
}

// Close 关闭数据库连接
func Close() error {
	if client == nil {
		return nil
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return client.Disconnect(ctx)
}

// GetCollection 获取指定的集合
func GetCollection(name string) *mongo.Collection {
	// 如果集合已经存在于映射中，直接返回
	if col, exists := collections[name]; exists {
		return col
	}
	
	// 如果集合不存在于映射中，创建并添加到映射
	collections[name] = database.Collection(name)
	return collections[name]
}

// GetDatabase 获取数据库实例
func GetDatabase() *mongo.Database {
	return database
}