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

// Issue 问题表模型
// 说明：用于记录检测发现的具体问题，满足“问题ID/类型/说明”的最小需求
type Issue struct {
    ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`                    // 问题ID
    ImageID        primitive.ObjectID `bson:"imageId" json:"imageId"`                    // 关联图片ID
    SceneID        primitive.ObjectID `bson:"sceneId" json:"sceneId"`                    // 关联场景ID（便于筛选）
    DetectionRunID primitive.ObjectID `bson:"detectionRunId,omitempty" json:"detectionRunId,omitempty"` // 可选：关联检测运行
    Type           string             `bson:"type" json:"type"`                          // 问题类型
    Description    string             `bson:"description" json:"description"`            // 问题说明
    CreatedAt      time.Time          `bson:"createdAt" json:"createdAt"`                // 创建时间
    UpdatedAt      time.Time          `bson:"updatedAt" json:"updatedAt"`                // 更新时间
}

// InsertIssue 插入问题记录
func InsertIssue(issue *Issue) (primitive.ObjectID, error) {
    col := database.GetCollection("issues")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if issue.ID.IsZero() {
        issue.ID = primitive.NewObjectID()
    }
    now := time.Now()
    if issue.CreatedAt.IsZero() {
        issue.CreatedAt = now
    }
    issue.UpdatedAt = now

    res, err := col.InsertOne(ctx, issue)
    if err != nil {
        return primitive.NilObjectID, err
    }
    if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
        return oid, nil
    }
    return primitive.NilObjectID, mongo.ErrInvalidIndexValue
}

// FindIssues 通用查询问题列表
// 支持按场景/图片/类型筛选
func FindIssues(filter bson.M, sort bson.D, limit int64) ([]Issue, error) {
    col := database.GetCollection("issues")
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

    var issues []Issue
    if err := cur.All(ctx, &issues); err != nil {
        return nil, err
    }
    return issues, nil
}

// EnsureIssueIndexes 创建常用索引
func EnsureIssueIndexes() error {
    col := database.GetCollection("issues")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    models := []mongo.IndexModel{
        {Keys: bson.D{{Key: "imageId", Value: 1}}},
        {Keys: bson.D{{Key: "sceneId", Value: 1}, {Key: "createdAt", Value: -1}}},
        {Keys: bson.D{{Key: "type", Value: 1}}},
    }
    _, err := col.Indexes().CreateMany(ctx, models)
    return err
}