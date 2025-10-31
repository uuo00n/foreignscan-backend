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
	collection *mongo.Collection
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
	
	// 获取数据库和集合
	database = client.Database(cfg.DatabaseName)
	collection = database.Collection("images")
	
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
	if name == "images" {
		return collection
	}
	return database.Collection(name)
}

// GetDatabase 获取数据库实例
func GetDatabase() *mongo.Database {
	return database
}