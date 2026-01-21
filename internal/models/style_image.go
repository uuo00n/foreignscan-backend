package models

import (
	"context"
	"time"

	"foreignscan/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// StyleImage 样式图模型
type StyleImage struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SceneID     primitive.ObjectID `bson:"sceneId" json:"sceneId"`         // 关联的场景ID
	Name        string             `bson:"name" json:"name"`               // 样式图名称
	Description string             `bson:"description" json:"description"` // 样式图描述
	Filename    string             `bson:"filename" json:"filename"`       // 文件名
	Path        string             `bson:"path" json:"path"`               // 文件路径
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`     // 创建时间
	UpdatedAt   time.Time          `bson:"updatedAt" json:"updatedAt"`     // 更新时间
}

// FindAllStyleImages 获取所有样式图
func FindAllStyleImages() ([]StyleImage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := database.GetCollection("styleImages")

	// 查询所有样式图，按创建时间降序排列
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := collection.Find(ctx, bson.D{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// 解析结果
	var styleImages []StyleImage
	if err := cursor.All(ctx, &styleImages); err != nil {
		return nil, err
	}

	return styleImages, nil
}

// FindStyleImagesBySceneID 根据场景ID查找样式图
func FindStyleImagesBySceneID(sceneID string) ([]StyleImage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := database.GetCollection("styleImages")

	objID, err := primitive.ObjectIDFromHex(sceneID)
	if err != nil {
		return nil, err
	}

	// 查询指定场景的所有样式图
	filter := bson.M{"sceneId": objID}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	// 解析结果
	var styleImages []StyleImage
	if err := cursor.All(ctx, &styleImages); err != nil {
		return nil, err
	}

	return styleImages, nil
}

// FindStyleImageByID 根据ID查找样式图
func FindStyleImageByID(id string) (*StyleImage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := database.GetCollection("styleImages")

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var styleImage StyleImage
	err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&styleImage)
	if err != nil {
		return nil, err
	}

	return &styleImage, nil
}

// Save 保存样式图
func (s *StyleImage) Save() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := database.GetCollection("styleImages")

	if s.ID.IsZero() {
		s.ID = primitive.NewObjectID()
		s.CreatedAt = time.Now()
	}
	s.UpdatedAt = time.Now()

	_, err := collection.InsertOne(ctx, s)
	return err
}

// Update 更新样式图
func (s *StyleImage) Update() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := database.GetCollection("styleImages")

	s.UpdatedAt = time.Now()

	filter := bson.M{"_id": s.ID}
	update := bson.M{"$set": s}

	_, err := collection.UpdateOne(ctx, filter, update)
	return err
}

// Delete 删除样式图
func (s *StyleImage) Delete() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := database.GetCollection("styleImages")

	filter := bson.M{"_id": s.ID}

	_, err := collection.DeleteOne(ctx, filter)
	return err
}
