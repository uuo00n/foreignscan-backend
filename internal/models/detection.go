package models

import (
    "context"
    "errors"
    "time"

    "foreignscan/internal/database"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

// BoundingBox 表示边界框，建议使用像素坐标，必要时也可存归一化坐标
type BoundingBox struct {
    X      float64 `bson:"x" json:"x"`             // 左上角X（像素）
    Y      float64 `bson:"y" json:"y"`             // 左上角Y（像素）
    Width  float64 `bson:"width" json:"width"`     // 宽度（像素）
    Height float64 `bson:"height" json:"height"`   // 高度（像素）
}

// PolygonPoint 可选的分割或多边形坐标
type PolygonPoint struct {
    X float64 `bson:"x" json:"x"`
    Y float64 `bson:"y" json:"y"`
}

// DetectionItem 单个检测框/目标的结果
type DetectionItem struct {
    Class       string        `bson:"class" json:"class"`                 // 目标类别名称
    ClassID     int           `bson:"classId" json:"classId"`             // 目标类别ID（如YOLO的类别索引）
    Confidence  float64       `bson:"confidence" json:"confidence"`       // 置信度
    BBox        BoundingBox   `bson:"bbox" json:"bbox"`                   // 边界框
    Polygon     []PolygonPoint `bson:"polygon,omitempty" json:"polygon,omitempty"` // 可选的分割多边形
    Note        string        `bson:"note,omitempty" json:"note,omitempty"`       // 可选备注（比如规则命中说明）
}

// DetectionSummary 用于查询与展示的汇总字段
type DetectionSummary struct {
    HasIssue    bool    `bson:"hasIssue" json:"hasIssue"`           // 是否存在问题
    IssueType   string  `bson:"issueType" json:"issueType"`         // 问题类型（业务定义）
    ObjectCount int     `bson:"objectCount" json:"objectCount"`     // 检测到的目标数量
    AvgScore    float64 `bson:"avgScore" json:"avgScore"`           // 平均置信度（便于排序/筛选）
}

// DetectionRun 一次推理运行的结果（支持同一张图片多次推理）
type DetectionRun struct {
    ID                 primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    RunID              string             `bson:"runId,omitempty" json:"runId,omitempty"` // 幂等插入的运行ID，可由YOLO服务生成
    ImageID            primitive.ObjectID `bson:"imageId" json:"imageId"`                 // 关联图片ID
    SceneID            primitive.ObjectID `bson:"sceneId" json:"sceneId"`                 // 反范式存储，便于按场景查询
    SourceFilename     string             `bson:"sourceFilename" json:"sourceFilename"`   // 原图文件名
    SourcePath         string             `bson:"sourcePath" json:"sourcePath"`           // 原图相对路径
    ProcessedFilename  string             `bson:"processedFilename" json:"processedFilename"` // 处理后图片文件名（带框/标注）
    ProcessedPath      string             `bson:"processedPath" json:"processedPath"`         // 处理后图片相对路径
    ModelName          string             `bson:"modelName" json:"modelName"`               // 模型名称（例如：yolov5s）
    ModelVersion       string             `bson:"modelVersion" json:"modelVersion"`         // 模型版本
    Device             string             `bson:"device,omitempty" json:"device,omitempty"` // 推理设备（cpu、cuda:0等）
    IoUThreshold       float64            `bson:"iouThreshold" json:"iouThreshold"`         // IoU阈值
    ConfidenceThreshold float64           `bson:"confidenceThreshold" json:"confidenceThreshold"` // 置信度阈值
    InferenceTimeMs    int64              `bson:"inferenceTimeMs" json:"inferenceTimeMs"`  // 推理耗时（毫秒）
    Items              []DetectionItem    `bson:"items" json:"items"`                       // 具体检测项列表
    Summary            DetectionSummary   `bson:"summary" json:"summary"`                   // 汇总字段
    CreatedAt          time.Time          `bson:"createdAt" json:"createdAt"`               // 创建时间
    UpdatedAt          time.Time          `bson:"updatedAt" json:"updatedAt"`               // 更新时间
}

// InsertDetectionRun 插入一次检测结果，并更新图片的检测摘要字段
// 注意：该方法保证小步且幂等（如果传入RunID），避免重复插入
func InsertDetectionRun(run *DetectionRun) (primitive.ObjectID, error) {
    if run == nil {
        return primitive.NilObjectID, errors.New("nil detection run")
    }

    col := database.GetCollection("detections")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // 基本字段填充
    now := time.Now()
    if run.CreatedAt.IsZero() {
        run.CreatedAt = now
    }
    run.UpdatedAt = now

    // 如果提供RunID，尝试使用upsert保证幂等
    if run.RunID != "" {
        filter := bson.M{"runId": run.RunID}
        update := bson.M{"$setOnInsert": run}
        opts := options.Update().SetUpsert(true)
        res, err := col.UpdateOne(ctx, filter, update, opts)
        if err != nil {
            return primitive.NilObjectID, err
        }
        // 如果是插入新文档，返回新ID；否则查询已有文档ID
        if res.UpsertedID != nil {
            if oid, ok := res.UpsertedID.(primitive.ObjectID); ok {
                if err2 := UpdateImageDetectionSummary(run.ImageID, run.Summary.HasIssue, run.Summary.IssueType, true); err2 != nil {
                    return primitive.NilObjectID, err2
                }
                return oid, nil
            }
        }
        // 幂等：已存在则查出ID
        var existing DetectionRun
        err = col.FindOne(ctx, filter).Decode(&existing)
        if err != nil {
            return primitive.NilObjectID, err
        }
        if err2 := UpdateImageDetectionSummary(run.ImageID, run.Summary.HasIssue, run.Summary.IssueType, true); err2 != nil {
            return primitive.NilObjectID, err2
        }
        return existing.ID, nil
    }

    // 未提供RunID，直接插入
    res, err := col.InsertOne(ctx, run)
    if err != nil {
        return primitive.NilObjectID, err
    }
    oid, ok := res.InsertedID.(primitive.ObjectID)
    if ok {
        if err2 := UpdateImageDetectionSummary(run.ImageID, run.Summary.HasIssue, run.Summary.IssueType, true); err2 != nil {
            return primitive.NilObjectID, err2
        }
        return oid, nil
    }
    return primitive.NilObjectID, errors.New("insertedID is not ObjectID")
}

// UpdateImageDetectionSummary 将图片的检测摘要字段更新为最新状态
// 仅更新必要字段，遵循小步改动
func UpdateImageDetectionSummary(imageID primitive.ObjectID, hasIssue bool, issueType string, isDetected bool) error {
    col := database.GetCollection("images")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // 根据检测结果映射图片状态
    status := ImageStatusQualified
    if hasIssue {
        status = ImageStatusDefective
    }
    if !isDetected {
        status = ImageStatusUndetected
    }

    update := bson.M{
        "$set": bson.M{
            "isDetected": isDetected,
            "hasIssue":   hasIssue,
            "issueType":  issueType,
            "status":     status,     // 同步更新状态字段
            "updatedAt":  time.Now(),
        },
    }
    _, err := col.UpdateByID(ctx, imageID, update)
    return err
}

// FindDetectionsByImageID 根据图片ID查询所有检测运行
func FindDetectionsByImageID(imageID primitive.ObjectID) ([]DetectionRun, error) {
    col := database.GetCollection("detections")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    cur, err := col.Find(ctx, bson.M{"imageId": imageID}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}))
    if err != nil {
        return nil, err
    }
    defer cur.Close(ctx)

    var runs []DetectionRun
    if err := cur.All(ctx, &runs); err != nil {
        return nil, err
    }
    return runs, nil
}

// FindDetections 通用查询接口：支持按场景、时间范围、是否有问题、类别过滤
func FindDetections(filter bson.M, sort bson.D, limit int64) ([]DetectionRun, error) {
    col := database.GetCollection("detections")
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

    var runs []DetectionRun
    if err := cur.All(ctx, &runs); err != nil {
        return nil, err
    }
    return runs, nil
}

// EnsureDetectionIndexes 可选：创建常用索引，建议在应用启动时调用
func EnsureDetectionIndexes() error {
    col := database.GetCollection("detections")
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    models := []mongo.IndexModel{
        {Keys: bson.D{{Key: "imageId", Value: 1}}},
        {Keys: bson.D{{Key: "sceneId", Value: 1}, {Key: "createdAt", Value: -1}}},
        {Keys: bson.D{{Key: "summary.hasIssue", Value: 1}}},
        {Keys: bson.D{{Key: "items.class", Value: 1}}},
        {Keys: bson.D{{Key: "runId", Value: 1}}, Options: options.Index().SetUnique(true)}, // 幂等运行ID唯一索引
    }
    _, err := col.Indexes().CreateMany(ctx, models)
    return err
}