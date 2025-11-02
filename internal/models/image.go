package models

import (
	"context"
	"time"

	"foreignscan/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Image 图片模型
type Image struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SequenceNumber   int                `bson:"sequenceNumber" json:"sequenceNumber"`
	SceneID          primitive.ObjectID `bson:"sceneId" json:"sceneId"`           // 关联的场景ID
	Timestamp        time.Time          `bson:"timestamp" json:"timestamp"`
	Location         string             `bson:"location" json:"location"`
	Filename         string             `bson:"filename" json:"filename"`
	Path             string             `bson:"path" json:"path"`
	IsDetected       bool               `bson:"isDetected" json:"isDetected"`
	HasIssue         bool               `bson:"hasIssue" json:"hasIssue"`
	IssueType        string             `bson:"issueType" json:"issueType"`
	DetectionResults []interface{}      `bson:"detectionResults" json:"detectionResults"`
	CreatedAt        time.Time          `bson:"createdAt" json:"createdAt"`       // 创建时间
	UpdatedAt        time.Time          `bson:"updatedAt" json:"updatedAt"`       // 更新时间
}

// GetNextSequence 获取下一个序列号
func GetNextSequence() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	collection := database.GetCollection("images")
	
	// 查找最大序号
	opts := options.FindOne().SetSort(bson.D{{Key: "sequenceNumber", Value: -1}})
	var maxImage Image
	err := collection.FindOne(ctx, bson.D{}, opts).Decode(&maxImage)
	
	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			// 如果没有记录，从1开始
			return 1, nil
		}
		return 0, err
	}
	
	// 返回最大序号+1
	return maxImage.SequenceNumber + 1, nil
}

// FindAll 获取所有图片
func FindAll() ([]Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	collection := database.GetCollection("images")
	
	// 查询所有图片，按序号降序排列
	opts := options.Find().SetSort(bson.D{{Key: "sequenceNumber", Value: -1}})
	cursor, err := collection.Find(ctx, bson.D{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	
	// 解析结果
	var images []Image
	if err := cursor.All(ctx, &images); err != nil {
		return nil, err
	}
	
	return images, nil
}

// FindByID 根据ID查找图片
func FindByID(id string) (*Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	collection := database.GetCollection("images")
	
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}
	
	var image Image
	err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&image)
	if err != nil {
		return nil, err
	}
	
	return &image, nil
}

// Save 保存图片
func (i *Image) Save() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	collection := database.GetCollection("images")
	
	if i.ID.IsZero() {
		i.ID = primitive.NewObjectID()
	}
	
	_, err := collection.InsertOne(ctx, i)
	return err
}

// Update 更新图片
func (i *Image) Update() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	collection := database.GetCollection("images")
	
	filter := bson.M{"_id": i.ID}
	update := bson.M{"$set": i}
	
	_, err := collection.UpdateOne(ctx, filter, update)
	return err
}