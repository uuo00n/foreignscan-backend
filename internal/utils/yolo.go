package utils

import (
	"bufio"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"foreignscan/internal/config"
	"foreignscan/internal/models"
)

// ParseYOLOLabelsToItems 从 YOLO 的 txt 标签与图片尺寸解析为检测项列表
// 说明：
// - labelPath: YOLO生成的标签文件路径（同名txt）
// - imgPath:   原始图片文件路径，用于读取宽高并将归一化坐标转换为像素坐标
// 返回：[]DetectionItem 列表，包含类别ID、名称、置信度和像素坐标的BBox
func ParseYOLOLabelsToItems(labelPath, imgPath string) ([]models.DetectionItem, error) {
	// 读取图片尺寸（只读元数据，不完整解码）
	imgFile, err := os.Open(imgPath)
	if err != nil {
		return nil, err
	}
	defer imgFile.Close()
	cfg, _, err := image.DecodeConfig(imgFile)
	if err != nil {
		return nil, err
	}
	W, H := float64(cfg.Width), float64(cfg.Height)

	// 逐行解析 YOLO txt
	f, err := os.Open(labelPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)

	var items []models.DetectionItem
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) < 5 {
			// 基本字段不足，跳过
			continue
		}
		classID, _ := strconv.Atoi(parts[0])
		xN, _ := strconv.ParseFloat(parts[1], 64)
		yN, _ := strconv.ParseFloat(parts[2], 64)
		wN, _ := strconv.ParseFloat(parts[3], 64)
		hN, _ := strconv.ParseFloat(parts[4], 64)

		// 转为像素坐标
		w := wN * W
		h := hN * H
		x := xN*W - w/2
		y := yN*H - h/2

		// 置信度（可选）
		conf := 0.0
		if len(parts) >= 6 {
			conf, _ = strconv.ParseFloat(parts[5], 64)
		}

		items = append(items, models.DetectionItem{
			ClassID:    classID,
			Class:      ClassNameFromID(classID), // 类别映射，建议后续从配置/数据库读取
			Confidence: conf,
			BBox:       models.BoundingBox{X: x, Y: y, Width: w, Height: h},
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// NormalizeUploadsLocalPath 将可能的HTTP路径或相对路径归一化为本地文件系统可读的路径
// 示例：
// - "/uploads/labels/scene/xxx.jpg" -> "uploads/labels/scene/xxx.jpg"
// - "labels/scene/xxx.jpg"          -> "uploads/labels/scene/xxx.jpg"
// - "uploads/labels/scene/xxx.txt"  -> "uploads/labels/scene/xxx.txt"（保持不变）
func NormalizeUploadsLocalPath(p string) string {
	raw := filepath.Clean(p)
	if filepath.IsAbs(raw) {
		return raw
	}
	cleaned := filepath.Clean(strings.TrimPrefix(raw, "/"))
	cleaned = strings.ReplaceAll(cleaned, "/", string(os.PathSeparator))
	cleaned = strings.TrimPrefix(cleaned, "."+string(os.PathSeparator))
	if cleaned == "uploads" {
		cleaned = ""
	} else if strings.HasPrefix(cleaned, "uploads"+string(os.PathSeparator)) {
		cleaned = strings.TrimPrefix(cleaned, "uploads"+string(os.PathSeparator))
	}
	return filepath.Join(config.Get().UploadDir, cleaned)
}

// ClassNameFromID 简单的类别ID到名称映射（示例）
// 后续建议从配置或数据库读取，避免“神秘命名”坏味道
func ClassNameFromID(id int) string {
	return "class_" + strconv.Itoa(id)
}
