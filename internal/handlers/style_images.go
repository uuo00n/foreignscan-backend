package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"foreignscan/internal/models"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetStyleImages 获取所有样式图
func GetStyleImages(w http.ResponseWriter, r *http.Request) {
	// 获取所有样式图
	styleImages, err := models.FindAllStyleImages()
	if err != nil {
		http.Error(w, "获取样式图失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(styleImages)
}

// GetStyleImagesByScene 获取指定场景的所有样式图
func GetStyleImagesByScene(w http.ResponseWriter, r *http.Request) {
	// 从URL获取场景ID
	vars := mux.Vars(r)
	sceneID := vars["sceneId"]

	// 查找样式图
	styleImages, err := models.FindStyleImagesBySceneID(sceneID)
	if err != nil {
		http.Error(w, "获取样式图失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(styleImages)
}

// GetStyleImage 获取单个样式图
func GetStyleImage(w http.ResponseWriter, r *http.Request) {
	// 从URL获取样式图ID
	vars := mux.Vars(r)
	id := vars["id"]

	// 查找样式图
	styleImage, err := models.FindStyleImageByID(id)
	if err != nil {
		http.Error(w, "样式图不存在: "+err.Error(), http.StatusNotFound)
		return
	}

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(styleImage)
}

// UploadStyleImage 上传样式图
func UploadStyleImage(w http.ResponseWriter, r *http.Request) {
	// 解析表单数据
	err := r.ParseMultipartForm(10 << 20) // 限制上传文件大小为10MB
	if err != nil {
		http.Error(w, "无法解析表单: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 获取场景ID
	sceneIDStr := r.FormValue("sceneId")
	if sceneIDStr == "" {
		http.Error(w, "缺少场景ID", http.StatusBadRequest)
		return
	}

	// 将场景ID转换为ObjectID
	sceneID, err := primitive.ObjectIDFromHex(sceneIDStr)
	if err != nil {
		http.Error(w, "无效的场景ID", http.StatusBadRequest)
		return
	}

	// 获取上传的文件
	file, handler, err := r.FormFile("styleImage")
	if err != nil {
		http.Error(w, "获取上传文件失败: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 创建样式图目录
	styleDir := filepath.Join("./uploads/styles", sceneID.Hex())
	if err := os.MkdirAll(styleDir, os.ModePerm); err != nil {
		http.Error(w, "创建样式图目录失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 生成唯一文件名
	ext := filepath.Ext(handler.Filename)
	filename := "style_" + time.Now().Format("20060102150405") + ext
	filePath := filepath.Join(styleDir, filename)
	
	// 创建文件
	dst, err := os.Create(filePath)
	if err != nil {
		http.Error(w, "创建文件失败: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// 保存文件
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "保存文件失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 创建样式图记录
	styleImage := models.StyleImage{
		SceneID:     sceneID,
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		Filename:    filename,
		Path:        filePath,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 保存样式图记录
	err = styleImage.Save()
	if err != nil {
		http.Error(w, "保存样式图记录失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(styleImage)
}

// UpdateStyleImage 更新样式图
func UpdateStyleImage(w http.ResponseWriter, r *http.Request) {
	// 从URL获取样式图ID
	vars := mux.Vars(r)
	id := vars["id"]

	// 将ID转换为ObjectID
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "无效的样式图ID", http.StatusBadRequest)
		return
	}

	// 解析请求体
	var updatedStyleImage models.StyleImage
	err = json.NewDecoder(r.Body).Decode(&updatedStyleImage)
	if err != nil {
		http.Error(w, "无效的请求数据: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 查找现有样式图
	existingStyleImage, err := models.FindStyleImageByID(id)
	if err != nil {
		http.Error(w, "样式图不存在: "+err.Error(), http.StatusNotFound)
		return
	}

	// 更新样式图字段
	updatedStyleImage.ID = objID
	updatedStyleImage.Filename = existingStyleImage.Filename
	updatedStyleImage.Path = existingStyleImage.Path
	updatedStyleImage.CreatedAt = existingStyleImage.CreatedAt
	updatedStyleImage.UpdatedAt = time.Now()

	// 保存更新
	err = updatedStyleImage.Update()
	if err != nil {
		http.Error(w, "更新样式图失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedStyleImage)
}

// DeleteStyleImage 删除样式图
func DeleteStyleImage(w http.ResponseWriter, r *http.Request) {
	// 从URL获取样式图ID
	vars := mux.Vars(r)
	id := vars["id"]

	// 查找样式图
	styleImage, err := models.FindStyleImageByID(id)
	if err != nil {
		http.Error(w, "样式图不存在: "+err.Error(), http.StatusNotFound)
		return
	}

	// 删除文件
	if err := os.Remove(styleImage.Path); err != nil && !os.IsNotExist(err) {
		http.Error(w, "删除文件失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 删除样式图记录
	err = styleImage.Delete()
	if err != nil {
		http.Error(w, "删除样式图记录失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回成功响应
	w.WriteHeader(http.StatusNoContent)
}