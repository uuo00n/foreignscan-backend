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
    Status           string             `bson:"status" json:"status"`               // 图片检测状态：未检测/已检测
    DetectionResults []interface{}      `bson:"detectionResults" json:"detectionResults"`
    CreatedAt        time.Time          `bson:"createdAt" json:"createdAt"`       // 创建时间
    UpdatedAt        time.Time          `bson:"updatedAt" json:"updatedAt"`       // 更新时间
}

// 定义图片状态常量，避免魔法字符串
const (
    ImageStatusUndetected = "未检测"
    ImageStatusDetected   = "已检测"
    // 兼容旧数据（读取时可转换为已检测）
    ImageStatusQualified  = "合格"
    ImageStatusDefective  = "缺陷"
)

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

// FindBySceneID 根据场景ID查找图片
func FindBySceneID(sceneID primitive.ObjectID) ([]Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	collection := database.GetCollection("images")
	
	// 查询指定场景下的所有图片，按序列号升序排列
	opts := options.Find().SetSort(bson.D{{Key: "sequenceNumber", Value: 1}})
	cursor, err := collection.Find(ctx, bson.M{"sceneId": sceneID}, opts)
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

// FindFirstBySceneID 根据场景ID查找最新一张图片（按时间）
// 说明：为了避免sequenceNumber异常导致排序不准确，这里改为按createdAt降序取第一条
func FindFirstBySceneID(sceneID primitive.ObjectID) (*Image, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    collection := database.GetCollection("images")
    
    // 查询指定场景下的最新一张图片（按createdAt降序排列）
    opts := options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}})
    var image Image
    err := collection.FindOne(ctx, bson.M{"sceneId": sceneID}, opts).Decode(&image)
    
    if err != nil {
        if err.Error() == "mongo: no documents in result" {
            // 没有找到图片，返回nil而不是错误
            return nil, nil
        }
        return nil, err
    }
    
    return &image, nil
}

// FindByDate 根据日期查找图片
func FindByDate(date time.Time) ([]Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	collection := database.GetCollection("images")
	
	// 计算日期范围（当天00:00:00到23:59:59）
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Nanosecond)
	
	// 查询在指定日期范围内创建的图片
	filter := bson.M{
		"createdAt": bson.M{
			"$gte": startOfDay,
			"$lte": endOfDay,
		},
	}
	
	// 按创建时间降序排列
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := collection.Find(ctx, filter, opts)
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

// FindByDateAndSceneID 根据日期和场景ID查找图片
func FindByDateAndSceneID(date time.Time, sceneID primitive.ObjectID) ([]Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	collection := database.GetCollection("images")
	
	// 计算日期范围（当天00:00:00到23:59:59）
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour).Add(-time.Nanosecond)
	
	// 查询在指定日期范围内且属于指定场景的图片
	filter := bson.M{
		"createdAt": bson.M{
			"$gte": startOfDay,
			"$lte": endOfDay,
		},
		"sceneId": sceneID,
	}
	
	// 按创建时间降序排列
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	cursor, err := collection.Find(ctx, filter, opts)
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

// FindByStatus 根据状态筛选图片
// 参数：
// - status: 图片状态（合格/缺陷/未检测）
// 返回：满足条件的图片列表，按创建时间降序
func FindByStatus(status string) ([]Image, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    collection := database.GetCollection("images")

    // 按创建时间降序排列
    opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
    cursor, err := collection.Find(ctx, bson.M{"status": status}, opts)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var images []Image
    if err := cursor.All(ctx, &images); err != nil {
        return nil, err
    }
    return images, nil
}

// FindByStatusAndTimeRange 根据状态与时间范围筛选图片
// 参数：
// - status: 图片状态（合格/缺陷/未检测）
// - start: 起始时间（包含）
// - end: 结束时间（包含）
// 返回：满足条件的图片列表，按创建时间降序
func FindByStatusAndTimeRange(status string, start, end time.Time) ([]Image, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    collection := database.GetCollection("images")

    filter := bson.M{
        "status": status,
        "createdAt": bson.M{
            "$gte": start,
            "$lte": end,
        },
    }
    opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
    cursor, err := collection.Find(ctx, filter, opts)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var images []Image
    if err := cursor.All(ctx, &images); err != nil {
        return nil, err
    }
    return images, nil
}

// FindByFlags 根据 isDetected 与可选的 hasIssue 筛选图片
// hasIssueParamProvided 为 true 时按 hasIssue 精确筛选；否则仅按 isDetected
func FindByFlags(isDetected bool, hasIssueParamProvided bool, hasIssue bool) ([]Image, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    collection := database.GetCollection("images")

    filter := bson.M{
        "isDetected": isDetected,
    }
    if hasIssueParamProvided {
        filter["hasIssue"] = hasIssue
    }
    opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
    cursor, err := collection.Find(ctx, filter, opts)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var images []Image
    if err := cursor.All(ctx, &images); err != nil {
        return nil, err
    }
    return images, nil
}

// FindByFlagsAndTimeRange 根据 isDetected 与可选 hasIssue 及时间范围筛选图片
func FindByFlagsAndTimeRange(isDetected bool, hasIssueParamProvided bool, hasIssue bool, start, end time.Time) ([]Image, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    collection := database.GetCollection("images")

    filter := bson.M{
        "isDetected": isDetected,
        "createdAt": bson.M{
            "$gte": start,
            "$lte": end,
        },
    }
    if hasIssueParamProvided {
        filter["hasIssue"] = hasIssue
    }
    opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
    cursor, err := collection.Find(ctx, filter, opts)
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var images []Image
    if err := cursor.All(ctx, &images); err != nil {
        return nil, err
    }
    return images, nil
}

// FindImagesByFilterInput 筛选参数结构体
type FindImagesByFilterInput struct {
	Status      string             // "已检测"/"未检测"，为空则不筛选状态
	HasIssue    *bool              // true/false，为nil则不筛选
	SceneID     primitive.ObjectID // Zero ObjectID则不筛选
	StartDate   time.Time          // Zero time则不筛选
	EndDate     time.Time          // Zero time则不筛选
}

// FindImagesByFilter 通用筛选函数
func FindImagesByFilter(input FindImagesByFilterInput) ([]Image, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := database.GetCollection("images")
	filter := bson.M{}

	// 1. 状态筛选 (兼容 status 字段和 isDetected 字段)
	if input.Status != "" {
		if input.Status == ImageStatusUndetected {
			filter["isDetected"] = false
		} else if input.Status == ImageStatusDetected {
			filter["isDetected"] = true
		} else {
			// 如果是其他状态字符串，尝试匹配 status 字段
			filter["status"] = input.Status
		}
	}

	// 2. 是否有问题筛选
	if input.HasIssue != nil {
		filter["hasIssue"] = *input.HasIssue
	}

	// 3. 场景筛选
	if !input.SceneID.IsZero() {
		filter["sceneId"] = input.SceneID
	}

	// 4. 时间范围筛选
	if !input.StartDate.IsZero() || !input.EndDate.IsZero() {
		dateFilter := bson.M{}
		if !input.StartDate.IsZero() {
			dateFilter["$gte"] = input.StartDate
		}
		if !input.EndDate.IsZero() {
			dateFilter["$lte"] = input.EndDate
		}
		filter["createdAt"] = dateFilter
	}

	// 按创建时间降序排列
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}})
	
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var images []Image
	if err := cursor.All(ctx, &images); err != nil {
		return nil, err
	}
	return images, nil
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