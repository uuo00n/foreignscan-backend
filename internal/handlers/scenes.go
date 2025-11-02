package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"foreignscan/internal/models"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetScenes 获取所有场景
func GetScenes(w http.ResponseWriter, r *http.Request) {
	// 获取所有场景
	scenes, err := models.FindAllScenes()
	if err != nil {
		http.Error(w, "获取场景失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scenes)
}

// GetScene 获取单个场景
func GetScene(w http.ResponseWriter, r *http.Request) {
	// 从URL获取场景ID
	vars := mux.Vars(r)
	id := vars["id"]

	// 查找场景
	scene, err := models.FindSceneByID(id)
	if err != nil {
		http.Error(w, "场景不存在: "+err.Error(), http.StatusNotFound)
		return
	}

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scene)
}

// CreateScene 创建新场景
func CreateScene(w http.ResponseWriter, r *http.Request) {
	// 解析请求体
	var scene models.Scene
	err := json.NewDecoder(r.Body).Decode(&scene)
	if err != nil {
		http.Error(w, "无效的请求数据: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 设置创建时间和更新时间
	scene.CreatedAt = time.Now()
	scene.UpdatedAt = time.Now()

	// 保存场景
	err = scene.Save()
	if err != nil {
		http.Error(w, "创建场景失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(scene)
}

// UpdateScene 更新场景
func UpdateScene(w http.ResponseWriter, r *http.Request) {
	// 从URL获取场景ID
	vars := mux.Vars(r)
	id := vars["id"]

	// 将ID转换为ObjectID
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "无效的场景ID", http.StatusBadRequest)
		return
	}

	// 解析请求体
	var updatedScene models.Scene
	err = json.NewDecoder(r.Body).Decode(&updatedScene)
	if err != nil {
		http.Error(w, "无效的请求数据: "+err.Error(), http.StatusBadRequest)
		return
	}

	// 查找现有场景
	existingScene, err := models.FindSceneByID(id)
	if err != nil {
		http.Error(w, "场景不存在: "+err.Error(), http.StatusNotFound)
		return
	}

	// 更新场景字段
	updatedScene.ID = objID
	updatedScene.CreatedAt = existingScene.CreatedAt
	updatedScene.UpdatedAt = time.Now()

	// 保存更新
	err = updatedScene.Update()
	if err != nil {
		http.Error(w, "更新场景失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回JSON响应
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedScene)
}

// DeleteScene 删除场景
func DeleteScene(w http.ResponseWriter, r *http.Request) {
	// 从URL获取场景ID
	vars := mux.Vars(r)
	id := vars["id"]

	// 查找场景
	scene, err := models.FindSceneByID(id)
	if err != nil {
		http.Error(w, "场景不存在: "+err.Error(), http.StatusNotFound)
		return
	}

	// 删除场景
	err = scene.Delete()
	if err != nil {
		http.Error(w, "删除场景失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回成功响应
	w.WriteHeader(http.StatusNoContent)
}