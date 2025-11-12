package services

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "sync"
    "time"

    "foreignscan/internal/models"
    "foreignscan/internal/utils"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

// DetectConfig 模型与推理配置
type DetectConfig struct {
    Weights     string  // 模型权重文件，如 yolov8s.pt
    ModelName   string  // 模型名称，展示用途
    ModelVersion string // 模型版本，可选
    Device      string  // 设备：cpu/cuda:0/mps 等
    Conf        float64 // 置信度阈值
    IoU         float64 // IoU阈值
    ServiceURL  string
}

// DetectJob 批量推理任务
type DetectJob struct {
    ID        string
    SceneID   primitive.ObjectID
    Status    string // pending/running/parsing/completed/failed
    Progress  int
    Total     int
    Message   string
    Error     string
    StartedAt time.Time
    EndedAt   *time.Time
    Canceled  bool `json:"canceled"` // 是否已被取消
    ctx       context.Context `json:"-"` // 任务上下文，用于取消
    cancel    context.CancelFunc `json:"-"` // 取消函数
}

// JobManager 管理与查询任务状态
type JobManager struct {
    mu   sync.RWMutex
    jobs map[string]*DetectJob
    // watchers: 订阅任务状态的SSE消费者
    watchers map[string][]chan DetectJob
    // sceneLocks: 同一场景的并发限制，值为持有锁的jobID
    sceneLocks map[string]string
}

var defaultJobManager = &JobManager{jobs: make(map[string]*DetectJob), watchers: make(map[string][]chan DetectJob), sceneLocks: make(map[string]string)}

// GetJobManager 获取默认任务管理器
func GetJobManager() *JobManager { return defaultJobManager }

// GetJob 查询任务
func (m *JobManager) GetJob(id string) (*DetectJob, bool) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    j, ok := m.jobs[id]
    return j, ok
}

// SetJob 设置/更新任务
func (m *JobManager) SetJob(j *DetectJob) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.jobs[j.ID] = j
    // 广播最新状态给SSE订阅者
    if subs, ok := m.watchers[j.ID]; ok {
        for _, ch := range subs {
            // 非阻塞发送：如果通道满则跳过，避免阻塞服务
            select { case ch <- *j: default: }
        }
    }
}

// Subscribe 订阅某个任务的状态更新（SSE）
// 返回：更新通道与取消订阅函数
func (m *JobManager) Subscribe(jobID string) (chan DetectJob, func()) {
    m.mu.Lock()
    defer m.mu.Unlock()
    ch := make(chan DetectJob, 8)
    m.watchers[jobID] = append(m.watchers[jobID], ch)
    // 取消订阅函数
    unsub := func() {
        m.mu.Lock(); defer m.mu.Unlock()
        subs := m.watchers[jobID]
        idx := -1
        for i, c := range subs { if c == ch { idx = i; break } }
        if idx >= 0 {
            // 移除指定订阅者
            m.watchers[jobID] = append(subs[:idx], subs[idx+1:]...)
        }
        close(ch)
    }
    return ch, unsub
}

// AcquireScene 尝试为场景加锁（防并发），成功返回true
func (m *JobManager) AcquireScene(sceneID primitive.ObjectID, jobID string) bool {
    m.mu.Lock()
    defer m.mu.Unlock()
    key := sceneID.Hex()
    if holder, ok := m.sceneLocks[key]; ok && holder != "" {
        return false
    }
    m.sceneLocks[key] = jobID
    return true
}

// ReleaseScene 释放场景锁
func (m *JobManager) ReleaseScene(sceneID primitive.ObjectID, jobID string) {
    m.mu.Lock(); defer m.mu.Unlock()
    key := sceneID.Hex()
    if holder, ok := m.sceneLocks[key]; ok && holder == jobID {
        delete(m.sceneLocks, key)
    }
}

// CancelJob 取消指定任务（支持pending/running/parsing阶段）
// 返回是否取消成功（false表示未找到或任务已结束）
func (m *JobManager) CancelJob(jobID string) bool {
    m.mu.Lock()
    defer m.mu.Unlock()
    job, ok := m.jobs[jobID]
    if !ok { return false }
    if job.Status == "completed" || job.Status == "failed" || job.Status == "canceled" { return false }
    // 触发上下文取消（如果可用）
    if job.cancel != nil {
        job.cancel()
    }
    // 标记为取消并记录结束时间
    job.Status = "canceled"
    job.Canceled = true
    t := time.Now(); job.EndedAt = &t
    m.jobs[jobID] = job
    // 广播状态更新
    if subs, ok := m.watchers[jobID]; ok {
        for _, ch := range subs {
            select { case ch <- *job: default: }
        }
    }
    // 释放场景锁
    key := job.SceneID.Hex()
    if holder, ok := m.sceneLocks[key]; ok && holder == job.ID {
        delete(m.sceneLocks, key)
    }
    return true
}

// StartSceneDetect 启动指定场景的批量推理，异步执行
func StartSceneDetect(sceneID primitive.ObjectID, cfg DetectConfig) (string, error) {
    // 生成任务ID（使用时间戳+sceneID简化，后续可改为UUID）
    jobID := fmt.Sprintf("detect-%s-%d", sceneID.Hex(), time.Now().UnixNano())
    job := &DetectJob{
        ID:        jobID,
        SceneID:   sceneID,
        Status:    "pending",
        Progress:  0,
        Total:     0,
        Message:   "初始化",
        StartedAt: time.Now(),
    }
    GetJobManager().SetJob(job)

    // 同一场景并发限制
    if !GetJobManager().AcquireScene(sceneID, jobID) {
        job.Status = "failed"
        job.Error = "当前场景已有进行中的任务，已拒绝新任务"
        t := time.Now(); job.EndedAt = &t
        GetJobManager().SetJob(job)
        return jobID, fmt.Errorf("scene %s busy", sceneID.Hex())
    }

    // 异步执行
    go func() {
        // 初始化可取消上下文
        job.ctx, job.cancel = context.WithCancel(context.Background())
        GetJobManager().SetJob(job)

        // 步骤1：准备路径
        sceneHex := sceneID.Hex()
        sourceDir := filepath.Join("uploads", "images", sceneHex)
        projectDir := filepath.Join("uploads", "labels")
        jobDir := filepath.Join(projectDir, sceneHex)
        predictDir := filepath.Join(jobDir, "predict")

        // 创建输出目录（后续YOLO也会创建，但这里先确保父目录存在）
        _ = os.MkdirAll(jobDir, 0o755)

        if cfg.ServiceURL != "" {
            job.Status = "running"
            job.Message = "正在调用服务推理"
            GetJobManager().SetJob(job)

            entries, err := os.ReadDir(sourceDir)
            if err != nil {
                job.Status = "failed"
                job.Error = fmt.Sprintf("读取源目录失败: %v", err)
                t := time.Now(); job.EndedAt = &t
                GetJobManager().SetJob(job)
                GetJobManager().ReleaseScene(sceneID, jobID)
                return
            }
            job.Total = len(entries)
            GetJobManager().SetJob(job)

            images, err := models.FindBySceneID(sceneID)
            if err != nil {
                job.Status = "failed"
                job.Error = fmt.Sprintf("查询图片失败: %v", err)
                t := time.Now(); job.EndedAt = &t
                GetJobManager().SetJob(job)
                GetJobManager().ReleaseScene(sceneID, jobID)
                return
            }
            nameToImage := make(map[string]models.Image, len(images))
            for _, im := range images {
                nameToImage[strings.TrimSpace(im.Filename)] = im
            }

            for _, e := range entries {
                if job.ctx.Err() != nil {
                    job.Status = "canceled"
                    job.Canceled = true
                    t := time.Now(); job.EndedAt = &t
                    GetJobManager().SetJob(job)
                    GetJobManager().ReleaseScene(sceneID, jobID)
                    return
                }
                if e.IsDir() { continue }
                ext := strings.ToLower(filepath.Ext(e.Name()))
                if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".bmp" { continue }

                im, ok := nameToImage[e.Name()]
                if !ok {
                    base := strings.TrimSuffix(e.Name(), ext)
                    if img, ok2 := nameToImage[base+".jpg"]; ok2 { im = img; ok = true }
                    if !ok { if img, ok3 := nameToImage[base+".png"]; ok3 { im = img; ok = true } }
                    if !ok { if img, ok4 := nameToImage[base+".jpeg"]; ok4 { im = img; ok = true } }
                    if !ok { if img, ok5 := nameToImage[base+".bmp"]; ok5 { im = img; ok = true } }
                }
                if !ok {
                    job.Message = fmt.Sprintf("跳过未匹配文件: %s", e.Name())
                    GetJobManager().SetJob(job)
                    continue
                }

                sourcePath := filepath.Join("uploads", "images", sceneHex, im.Filename)

                reqBody := map[string]interface{}{
                    "image_path": utils.NormalizeUploadsLocalPath(sourcePath),
                    "model_path": cfg.Weights,
                    "conf": cfg.Conf,
                    "iou": cfg.IoU,
                }
                b, _ := json.Marshal(reqBody)
                start := time.Now()
                resp, err := http.Post(strings.TrimRight(cfg.ServiceURL, "/")+"/detect", "application/json", bytes.NewReader(b))
                if err != nil {
                    job.Message = fmt.Sprintf("服务调用失败 %s: %v", e.Name(), err)
                    GetJobManager().SetJob(job)
                    continue
                }
                var dr struct{
                    Success bool `json:"success"`
                    Items []struct{
                        ClassId int `json:"classId"`
                        Class_ string `json:"class_"`
                        Confidence float64 `json:"confidence"`
                        Bbox struct{ X float64 `json:"x"`; Y float64 `json:"y"`; Width float64 `json:"width"`; Height float64 `json:"height"` } `json:"bbox"`
                    } `json:"items"`
                    Summary struct{ HasIssue bool `json:"hasIssue"`; IssueType string `json:"issueType"`; ObjectCount int `json:"objectCount"`; AvgScore float64 `json:"avgScore"` } `json:"summary"`
                }
                _ = json.NewDecoder(resp.Body).Decode(&dr)
                _ = resp.Body.Close()
                if !dr.Success {
                    job.Message = "服务返回失败"
                    GetJobManager().SetJob(job)
                    continue
                }

                items := make([]models.DetectionItem, 0, len(dr.Items))
                for _, it := range dr.Items {
                    items = append(items, models.DetectionItem{
                        Class: it.Class_,
                        ClassID: it.ClassId,
                        Confidence: it.Confidence,
                        BBox: models.BoundingBox{X: it.Bbox.X, Y: it.Bbox.Y, Width: it.Bbox.Width, Height: it.Bbox.Height},
                    })
                }
                run := &models.DetectionRun{
                    RunID:               jobID+":"+im.Filename,
                    ImageID:             im.ID,
                    SceneID:             sceneID,
                    SourceFilename:      im.Filename,
                    SourcePath:          sourcePath,
                    ProcessedFilename:   im.Filename,
                    ProcessedPath:       sourcePath,
                    ModelName:           cfg.ModelName,
                    ModelVersion:        cfg.ModelVersion,
                    Device:              cfg.Device,
                    IoUThreshold:        cfg.IoU,
                    ConfidenceThreshold: cfg.Conf,
                    InferenceTimeMs:     time.Since(start).Milliseconds(),
                    Items:               items,
                    Summary:             models.DetectionSummary{HasIssue: dr.Summary.HasIssue, IssueType: dr.Summary.IssueType, ObjectCount: dr.Summary.ObjectCount, AvgScore: dr.Summary.AvgScore},
                    CreatedAt:           time.Now(),
                    UpdatedAt:           time.Now(),
                }
                if _, err := models.InsertDetectionRun(run); err != nil {
                    job.Message = fmt.Sprintf("写库失败 %s: %v", e.Name(), err)
                    GetJobManager().SetJob(job)
                    continue
                }
                job.Progress++
                GetJobManager().SetJob(job)
            }

            job.Status = "completed"
            job.Message = "任务完成"
            t := time.Now(); job.EndedAt = &t
            GetJobManager().SetJob(job)
            GetJobManager().ReleaseScene(sceneID, jobID)
            return
        }

        job.Status = "running"
        job.Message = "正在运行YOLO推理"
        GetJobManager().SetJob(job)

        args := []string{"detect", "predict",
            fmt.Sprintf("model=%s", cfg.Weights),
            fmt.Sprintf("source=%s", sourceDir),
            fmt.Sprintf("project=%s", projectDir),
            fmt.Sprintf("name=%s", sceneHex),
            "save=True", "save_txt=True", "save_conf=True", "exist_ok=True",
            fmt.Sprintf("conf=%.4f", cfg.Conf),
            fmt.Sprintf("iou=%.4f", cfg.IoU),
        }
        if cfg.Device != "" {
            args = append(args, fmt.Sprintf("device=%s", cfg.Device))
        }
        cmd := exec.CommandContext(job.ctx, "yolo", args...)
        if out, err := cmd.CombinedOutput(); err != nil {
            if job.ctx.Err() != nil {
                job.Status = "canceled"
                job.Canceled = true
                job.Message = shortOut(string(out))
            } else {
                job.Status = "failed"
                job.Error = fmt.Sprintf("YOLO执行失败: %v", err)
                job.Message = shortOut(string(out))
            }
            t := time.Now(); job.EndedAt = &t
            GetJobManager().SetJob(job)
            GetJobManager().ReleaseScene(sceneID, jobID)
            return
        }

        job.Status = "parsing"
        job.Message = "解析标签并写入数据库"
        GetJobManager().SetJob(job)

        // 读取场景下的图片列表，构建 filename -> image 映射
        images, err := models.FindBySceneID(sceneID)
        if err != nil {
            job.Status = "failed"
            job.Error = fmt.Sprintf("查询图片失败: %v", err)
            t := time.Now(); job.EndedAt = &t
            GetJobManager().SetJob(job)
            return
        }
        nameToImage := make(map[string]models.Image, len(images))
        for _, im := range images {
            nameToImage[strings.TrimSpace(im.Filename)] = im
        }

        // 遍历处理后图片目录
        // Ultralytics 默认将推理后的图片放在 predictDir，标签在 predictDir/labels 下
        entries, err := os.ReadDir(predictDir)
        if err != nil {
            job.Status = "failed"
            job.Error = fmt.Sprintf("读取推理输出目录失败: %v", err)
            t := time.Now(); job.EndedAt = &t
            GetJobManager().SetJob(job)
            return
        }
        job.Total = len(entries)
        GetJobManager().SetJob(job)

        for _, e := range entries {
            // 支持取消：在解析循环中检查上下文
            if job.ctx.Err() != nil {
                job.Status = "canceled"
                job.Canceled = true
                t := time.Now(); job.EndedAt = &t
                GetJobManager().SetJob(job)
                // 释放场景锁
                GetJobManager().ReleaseScene(sceneID, jobID)
                return
            }
            if e.IsDir() { continue }
            // 仅处理常见图片扩展名
            ext := strings.ToLower(filepath.Ext(e.Name()))
            if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".bmp" { continue }

            base := strings.TrimSuffix(e.Name(), ext)
            processedPath := filepath.Join("uploads", "labels", sceneHex, "predict", e.Name())
            // 标签优先与图片同目录（base.txt），否则回退 predict/labels/base.txt
            labelPath := filepath.Join("uploads", "labels", sceneHex, "predict", base+".txt")
            if _, err := os.Stat(labelPath); os.IsNotExist(err) {
                labelPath = filepath.Join("uploads", "labels", sceneHex, "predict", "labels", base+".txt")
            }

            // 查找对应图片ID
            im, ok := nameToImage[e.Name()]
            if !ok {
                // 有些场景原图扩展可能不同，此处尝试以base匹配（不含扩展）
                if img, ok2 := nameToImage[base+".jpg"]; ok2 { im = img; ok = true }
                if !ok { if img, ok3 := nameToImage[base+".png"]; ok3 { im = img; ok = true } }
                if !ok { if img, ok4 := nameToImage[base+".jpeg"]; ok4 { im = img; ok = true } }
                if !ok { if img, ok5 := nameToImage[base+".bmp"]; ok5 { im = img; ok = true } }
            }
            if !ok {
                // 找不到对应图片，跳过但记录信息
                job.Message = fmt.Sprintf("跳过未匹配文件: %s", e.Name())
                GetJobManager().SetJob(job)
                continue
            }

            // 解析标签 -> items
            sourcePath := filepath.Join("uploads", "images", sceneHex, im.Filename)
            items, err := utils.ParseYOLOLabelsToItems(utils.NormalizeUploadsLocalPath(labelPath), utils.NormalizeUploadsLocalPath(sourcePath))
            if err != nil {
                job.Message = fmt.Sprintf("解析标签失败 %s: %v", e.Name(), err)
                GetJobManager().SetJob(job)
                continue
            }

            // 汇总（简单示例：有任意目标视为存在问题）
            summary := models.DetectionSummary{
                HasIssue:    len(items) > 0,
                IssueType:   "auto",
                ObjectCount: len(items),
                AvgScore:    avgConfidence(items),
            }

            run := &models.DetectionRun{
                RunID:               jobID + ":" + im.Filename,
                ImageID:             im.ID,
                SceneID:             sceneID,
                SourceFilename:      im.Filename,
                SourcePath:          sourcePath,
                ProcessedFilename:   e.Name(),
                ProcessedPath:       processedPath,
                ModelName:           cfg.ModelName,
                ModelVersion:        cfg.ModelVersion,
                Device:              cfg.Device,
                IoUThreshold:        cfg.IoU,
                ConfidenceThreshold: cfg.Conf,
                InferenceTimeMs:     0, // 可选：后续可从YOLO输出解析
                Items:               items,
                Summary:             summary,
                CreatedAt:           time.Now(),
                UpdatedAt:           time.Now(),
            }
            if _, err := models.InsertDetectionRun(run); err != nil {
                job.Message = fmt.Sprintf("写库失败 %s: %v", e.Name(), err)
                GetJobManager().SetJob(job)
                continue
            }

            job.Progress++
            GetJobManager().SetJob(job)
        }

        job.Status = "completed"
        job.Message = "任务完成"
        t := time.Now(); job.EndedAt = &t
        GetJobManager().SetJob(job)
        // 释放场景锁
        GetJobManager().ReleaseScene(sceneID, jobID)
    }()

    return jobID, nil
}

func shortOut(s string) string {
    s = strings.TrimSpace(s)
    if len(s) > 200 {
        return s[:200] + "..."
    }
    return s
}

func avgConfidence(items []models.DetectionItem) float64 {
    if len(items) == 0 { return 0 }
    sum := 0.0
    for _, it := range items { sum += it.Confidence }
    return sum / float64(len(items))
}

// StartImageDetect 启动单张图片的推理任务（异步）
// 说明：
// - 根据 imageID 查询图片与场景，后端调用 YOLO CLI 对单张图片进行推理
// - 输出路径沿用 uploads/labels/<sceneId>/predict，标签优先与图片同目录或在 labels 子目录
// - 任务状态通过内存 JobManager 管理，前端可通过 /api/detect/jobs/:id 查询
func StartImageDetect(imageID primitive.ObjectID, cfg DetectConfig) (string, error) {
    // 生成任务ID（使用时间戳+imageID简化，后续可改为UUID）
    jobID := fmt.Sprintf("detect-image-%s-%d", imageID.Hex(), time.Now().UnixNano())
    job := &DetectJob{
        ID:        jobID,
        SceneID:   primitive.NilObjectID, // 稍后填充
        Status:    "pending",
        Progress:  0,
        Total:     1,
        Message:   "初始化",
        StartedAt: time.Now(),
    }
    GetJobManager().SetJob(job)

    // 查询图片信息以确定场景并尝试加锁
    im, err := models.FindByID(imageID.Hex())
    if err != nil || im == nil {
        job.Status = "failed"
        if err != nil { job.Error = fmt.Sprintf("查询图片失败: %v", err) } else { job.Error = "未找到图片" }
        t := time.Now(); job.EndedAt = &t
        GetJobManager().SetJob(job)
        return jobID, fmt.Errorf("image not found")
    }
    if !GetJobManager().AcquireScene(im.SceneID, jobID) {
        job.Status = "failed"
        job.Error = "当前场景已有进行中的任务，已拒绝新任务"
        t := time.Now(); job.EndedAt = &t
        GetJobManager().SetJob(job)
        return jobID, fmt.Errorf("scene %s busy", im.SceneID.Hex())
    }

    go func() {
        // 初始化可取消上下文
        job.ctx, job.cancel = context.WithCancel(context.Background())
        GetJobManager().SetJob(job)

        sceneHex := im.SceneID.Hex()
        job.SceneID = im.SceneID
        GetJobManager().SetJob(job)

        // 准备路径
        sourcePath := filepath.Join("uploads", "images", sceneHex, im.Filename)
        projectDir := filepath.Join("uploads", "labels")
        jobDir := filepath.Join(projectDir, sceneHex)
        _ = os.MkdirAll(jobDir, 0o755)

        if cfg.ServiceURL != "" {
            job.Status = "running"
            job.Message = "正在调用服务推理(单图)"
            GetJobManager().SetJob(job)

            reqBody := map[string]interface{}{
                "image_path": utils.NormalizeUploadsLocalPath(sourcePath),
                "model_path": cfg.Weights,
                "conf": cfg.Conf,
                "iou": cfg.IoU,
            }
            b, _ := json.Marshal(reqBody)
            start := time.Now()
            resp, err := http.Post(strings.TrimRight(cfg.ServiceURL, "/")+"/detect", "application/json", bytes.NewReader(b))
            if err != nil {
                job.Status = "failed"
                job.Error = fmt.Sprintf("服务调用失败: %v", err)
                t := time.Now(); job.EndedAt = &t
                GetJobManager().SetJob(job)
                GetJobManager().ReleaseScene(im.SceneID, jobID)
                return
            }
            var dr struct{
                Success bool `json:"success"`
                Items []struct{
                    ClassId int `json:"classId"`
                    Class_ string `json:"class_"`
                    Confidence float64 `json:"confidence"`
                    Bbox struct{ X float64 `json:"x"`; Y float64 `json:"y"`; Width float64 `json:"width"`; Height float64 `json:"height"` } `json:"bbox"`
                } `json:"items"`
                Summary struct{ HasIssue bool `json:"hasIssue"`; IssueType string `json:"issueType"`; ObjectCount int `json:"objectCount"`; AvgScore float64 `json:"avgScore"` } `json:"summary"`
            }
            _ = json.NewDecoder(resp.Body).Decode(&dr)
            _ = resp.Body.Close()
            if !dr.Success {
                job.Status = "failed"
                job.Error = "服务返回失败"
                t := time.Now(); job.EndedAt = &t
                GetJobManager().SetJob(job)
                GetJobManager().ReleaseScene(im.SceneID, jobID)
                return
            }

            items := make([]models.DetectionItem, 0, len(dr.Items))
            for _, it := range dr.Items {
                items = append(items, models.DetectionItem{Class: it.Class_, ClassID: it.ClassId, Confidence: it.Confidence, BBox: models.BoundingBox{X: it.Bbox.X, Y: it.Bbox.Y, Width: it.Bbox.Width, Height: it.Bbox.Height}})
            }
            summary := models.DetectionSummary{HasIssue: dr.Summary.HasIssue, IssueType: dr.Summary.IssueType, ObjectCount: dr.Summary.ObjectCount, AvgScore: dr.Summary.AvgScore}

            run := &models.DetectionRun{RunID: jobID+":"+im.Filename, ImageID: im.ID, SceneID: im.SceneID, SourceFilename: im.Filename, SourcePath: sourcePath, ProcessedFilename: im.Filename, ProcessedPath: sourcePath, ModelName: cfg.ModelName, ModelVersion: cfg.ModelVersion, Device: cfg.Device, IoUThreshold: cfg.IoU, ConfidenceThreshold: cfg.Conf, InferenceTimeMs: time.Since(start).Milliseconds(), Items: items, Summary: summary, CreatedAt: time.Now(), UpdatedAt: time.Now()}
            if _, err := models.InsertDetectionRun(run); err != nil {
                job.Status = "failed"
                job.Error = fmt.Sprintf("写库失败: %v", err)
                t := time.Now(); job.EndedAt = &t
                GetJobManager().SetJob(job)
                GetJobManager().ReleaseScene(im.SceneID, jobID)
                return
            }

            job.Progress = 1
            job.Status = "completed"
            job.Message = "任务完成(单图)"
            t := time.Now(); job.EndedAt = &t
            GetJobManager().SetJob(job)
            GetJobManager().ReleaseScene(im.SceneID, jobID)
            return
        }

        job.Status = "running"
        job.Message = "正在运行YOLO推理(单图)"
        GetJobManager().SetJob(job)

        args := []string{"detect", "predict",
            fmt.Sprintf("model=%s", cfg.Weights),
            fmt.Sprintf("source=%s", sourcePath),
            fmt.Sprintf("project=%s", projectDir),
            fmt.Sprintf("name=%s", sceneHex),
            "save=True", "save_txt=True", "save_conf=True", "exist_ok=True",
            fmt.Sprintf("conf=%.4f", cfg.Conf),
            fmt.Sprintf("iou=%.4f", cfg.IoU),
        }
        if cfg.Device != "" {
            args = append(args, fmt.Sprintf("device=%s", cfg.Device))
        }
        cmd := exec.CommandContext(job.ctx, "yolo", args...)
        if out, err := cmd.CombinedOutput(); err != nil {
            if job.ctx.Err() != nil {
                job.Status = "canceled"
                job.Canceled = true
                job.Message = shortOut(string(out))
            } else {
                job.Status = "failed"
                job.Error = fmt.Sprintf("YOLO执行失败: %v", err)
                job.Message = shortOut(string(out))
            }
            t := time.Now(); job.EndedAt = &t
            GetJobManager().SetJob(job)
            GetJobManager().ReleaseScene(im.SceneID, jobID)
            return
        }

        job.Status = "parsing"
        job.Message = "解析标签并写入数据库(单图)"
        GetJobManager().SetJob(job)

        // 处理后图片与标签路径
        base := strings.TrimSuffix(im.Filename, filepath.Ext(im.Filename))
        processedPath := filepath.Join("uploads", "labels", sceneHex, "predict", im.Filename)
        labelPath := filepath.Join("uploads", "labels", sceneHex, "predict", base+".txt")
        if _, err := os.Stat(labelPath); os.IsNotExist(err) {
            labelPath = filepath.Join("uploads", "labels", sceneHex, "predict", "labels", base+".txt")
        }

        // 支持取消：在解析阶段检查
        if job.ctx.Err() != nil {
            job.Status = "canceled"
            job.Canceled = true
            t := time.Now(); job.EndedAt = &t
            GetJobManager().SetJob(job)
            // 释放场景锁
            GetJobManager().ReleaseScene(im.SceneID, jobID)
            return
        }
        items, err := utils.ParseYOLOLabelsToItems(utils.NormalizeUploadsLocalPath(labelPath), utils.NormalizeUploadsLocalPath(sourcePath))
        if err != nil {
            job.Status = "failed"
            job.Error = fmt.Sprintf("解析标签失败: %v", err)
            t := time.Now(); job.EndedAt = &t
            GetJobManager().SetJob(job)
            // 释放场景锁
            GetJobManager().ReleaseScene(im.SceneID, jobID)
            return
        }

        summary := models.DetectionSummary{
            HasIssue:    len(items) > 0,
            IssueType:   "auto",
            ObjectCount: len(items),
            AvgScore:    avgConfidence(items),
        }

        run := &models.DetectionRun{
            RunID:               jobID + ":" + im.Filename,
            ImageID:             im.ID,
            SceneID:             im.SceneID,
            SourceFilename:      im.Filename,
            SourcePath:          sourcePath,
            ProcessedFilename:   im.Filename,
            ProcessedPath:       processedPath,
            ModelName:           cfg.ModelName,
            ModelVersion:        cfg.ModelVersion,
            Device:              cfg.Device,
            IoUThreshold:        cfg.IoU,
            ConfidenceThreshold: cfg.Conf,
            InferenceTimeMs:     0,
            Items:               items,
            Summary:             summary,
            CreatedAt:           time.Now(),
            UpdatedAt:           time.Now(),
        }
        if _, err := models.InsertDetectionRun(run); err != nil {
            job.Status = "failed"
            job.Error = fmt.Sprintf("写库失败: %v", err)
            t := time.Now(); job.EndedAt = &t
            GetJobManager().SetJob(job)
            GetJobManager().ReleaseScene(im.SceneID, jobID)
            return
        }

        job.Progress = 1
        job.Status = "completed"
        job.Message = "任务完成(单图)"
        t := time.Now(); job.EndedAt = &t
        GetJobManager().SetJob(job)
        // 释放场景锁
        GetJobManager().ReleaseScene(im.SceneID, jobID)
    }()

    return jobID, nil
}
