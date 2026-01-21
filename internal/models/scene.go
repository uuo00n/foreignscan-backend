package models

import (
	"context"
	"time"

	"foreignscan/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Scene 场景模型
type Scene struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`               // 场景名称
	Description string             `bson:"description" json:"description"` // 场景描述
	Location    string             `bson:"location" json:"location"`       // 场景位置
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`     // 创建时间
	UpdatedAt   time.Time          `bson:"updatedAt" json:"updatedAt"`     // 更新时间
	Status      string             `bson:"status" json:"status"`           // 场景状态
}

// FindAllScenes 获取所有场景
func FindAllScenes() ([]Scene, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := database.GetCollection("scenes")

	// 查询所有场景，按创建时间降序排列
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := collection.Find(ctx, bson.D{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// 解析结果
	var scenes []Scene
	if err := cursor.All(ctx, &scenes); err != nil {
		return nil, err
	}

	return scenes, nil
}

// FindSceneByID 根据ID查找场景
func FindSceneByID(id string) (*Scene, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := database.GetCollection("scenes")

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var scene Scene
	err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&scene)
	if err != nil {
		return nil, err
	}

	return &scene, nil
}

// Save 保存场景
func (s *Scene) Save() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := database.GetCollection("scenes")

	if s.ID.IsZero() {
		s.ID = primitive.NewObjectID()
		s.CreatedAt = time.Now()
	}
	s.UpdatedAt = time.Now()

	_, err := collection.InsertOne(ctx, s)
	return err
}

// Update 更新场景
func (s *Scene) Update() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := database.GetCollection("scenes")

	s.UpdatedAt = time.Now()

	filter := bson.M{"_id": s.ID}
	update := bson.M{"$set": s}

	_, err := collection.UpdateOne(ctx, filter, update)
	return err
}

// Delete 删除场景
func (s *Scene) Delete() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := database.GetCollection("scenes")

	filter := bson.M{"_id": s.ID}

	_, err := collection.DeleteOne(ctx, filter)
	return err
}
