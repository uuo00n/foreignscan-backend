package models

import (
    "context"
    "time"

    "foreignscan/internal/database"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

// Comparison 对比表模型
// 说明：用于记录处理前后对比（源图/处理后图），可扩展存储差异信息
type Comparison struct {
    ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`                    // 对比记录ID
    ImageID        primitive.ObjectID `bson:"imageId" json:"imageId"`                    // 关联图片ID
    DetectionRunID primitive.ObjectID `bson:"detectionRunId,omitempty" json:"detectionRunId,omitempty"` // 可选：关联检测运行
    BeforePath     string             `bson:"beforePath" json:"beforePath"`              // 处理前图片路径
    AfterPath      string             `bson:"afterPath" json:"afterPath"`                // 处理后图片路径
    DiffInfo       interface{}        `bson:"diffInfo,omitempty" json:"diffInfo,omitempty"` // 可选：差异信息（如BBox变化、标注差异）
    Remark         string             `bson:"remark,omitempty" json:"remark,omitempty"`  // 可选：备注
    CreatedAt      time.Time          `bson:"createdAt" json:"createdAt"`                // 创建时间
}

// InsertComparison 插入对比记录
func InsertComparison(comp *Comparison) (primitive.ObjectID, error) {
    col := database.GetCollection("comparisons")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if comp.ID.IsZero() {
        comp.ID = primitive.NewObjectID()
    }
    if comp.CreatedAt.IsZero() {
        comp.CreatedAt = time.Now()
    }

    res, err := col.InsertOne(ctx, comp)
    if err != nil {
        return primitive.NilObjectID, err
    }
    if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
        return oid, nil
    }
    return primitive.NilObjectID, mongo.ErrInvalidIndexValue
}

// FindComparisons 查询对比记录
// 支持按图片ID或检测运行ID筛选
func FindComparisons(filter bson.M, sort bson.D, limit int64) ([]Comparison, error) {
    col := database.GetCollection("comparisons")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    opts := options.Find()
    if sort != nil {
        opts.SetSort(sort)
    } else {
        opts.SetSort(bson.D{{Key: "createdAt", Value: -1}})
    }
    if limit > 0 {
        opts.SetLimit(limit)
    }

    cur, err := col.Find(ctx, filter, opts)
    if err != nil {
        return nil, err
    }
    defer cur.Close(ctx)

    var comps []Comparison
    if err := cur.All(ctx, &comps); err != nil {
        return nil, err
    }
    return comps, nil
}

// EnsureComparisonIndexes 创建常用索引
func EnsureComparisonIndexes() error {
    col := database.GetCollection("comparisons")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    models := []mongo.IndexModel{
        {Keys: bson.D{{Key: "imageId", Value: 1}, {Key: "createdAt", Value: -1}}},
        {Keys: bson.D{{Key: "detectionRunId", Value: 1}}},
    }
    _, err := col.Indexes().CreateMany(ctx, models)
    return err
}