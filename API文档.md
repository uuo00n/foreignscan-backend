# ForeignScan 后端 API 文档

## 基础信息

- 基础URL: `http://localhost:端口号/api`
- 响应格式: JSON
- 认证方式: 无（当前版本）

## 通用响应格式

成功响应:
```json
{
  "success": true,
  "数据字段": 数据值
}
```

错误响应:
```json
{
  "success": false,
  "message": "错误信息"
}
```

## API 接口列表

### 1. 健康检查

- **URL**: `/ping`
- **方法**: GET
- **描述**: 检查服务器是否正常运行
- **响应示例**:
  ```json
  {
    "status": "ok",
    "message": "Server is running"
  }
  ```

### 2. 图片管理

#### 2.1 获取图片列表

- **URL**: `/api/images`
- **方法**: GET
- **描述**: 获取所有已上传的图片列表
- **响应示例**:
  ```json
  {
    "success": true,
    "images": [
      {
        "id": "图片ID",
        "filename": "图片文件名",
        "path": "图片路径",
        "sceneId": "场景ID",
        "createdAt": "创建时间",
        "updatedAt": "更新时间"
      }
    ]
  }
  ```

#### 2.2 上传图片

- **URL**: `/api/upload` 或 `/api/upload-image`
- **方法**: POST
- **描述**: 上传新图片
- **请求格式**: multipart/form-data
- **参数**:
  - `file`: 图片文件
  - `sceneId`: 场景ID
- **响应示例**:
  ```json
  {
    "success": true,
    "image": {
      "id": "新图片ID",
      "filename": "图片文件名",
      "path": "图片路径",
      "sceneId": "场景ID",
      "createdAt": "创建时间",
      "updatedAt": "更新时间"
    }
  }
  ```

#### 2.3 检测图片

- **URL**: `/api/detect`
- **方法**: POST
- **描述**: 对指定图片进行检测分析
- **请求格式**: JSON
- **参数**:
  ```json
  {
    "imageId": "图片ID"
  }
  ```
- **响应示例**:
  ```json
  {
    "success": true,
    "result": {
      "检测结果字段": "检测结果值"
    }
  }
  ```

### 3. 场景管理

#### 3.1 获取所有场景

- **URL**: `/api/scenes`
- **方法**: GET
- **描述**: 获取所有场景列表
- **响应示例**:
  ```json
  {
    "success": true,
    "scenes": [
      {
        "id": "场景ID",
        "name": "场景名称",
        "description": "场景描述",
        "createdAt": "创建时间",
        "updatedAt": "更新时间"
      }
    ]
  }
  ```

#### 3.2 获取单个场景

- **URL**: `/api/scenes/:id`
- **方法**: GET
- **描述**: 获取指定ID的场景详情
- **参数**:
  - `id`: 场景ID (URL参数)
- **响应示例**:
  ```json
  {
    "success": true,
    "scene": {
      "id": "场景ID",
      "name": "场景名称",
      "description": "场景描述",
      "createdAt": "创建时间",
      "updatedAt": "更新时间"
    }
  }
  ```

#### 3.3 创建场景

- **URL**: `/api/scenes`
- **方法**: POST
- **描述**: 创建新场景
- **请求格式**: JSON
- **参数**:
  ```json
  {
    "name": "场景名称",
    "description": "场景描述"
  }
  ```
- **响应示例**:
  ```json
  {
    "success": true,
    "scene": {
      "id": "新场景ID",
      "name": "场景名称",
      "description": "场景描述",
      "createdAt": "创建时间",
      "updatedAt": "更新时间"
    }
  }
  ```

### 4. 样式图片管理

#### 4.1 获取所有样式图片

- **URL**: `/api/style-images`
- **方法**: GET
- **描述**: 获取所有样式图片列表
- **响应示例**:
  ```json
  {
    "success": true,
    "styleImages": [
      {
        "id": "样式图片ID",
        "filename": "样式图片文件名",
        "path": "样式图片路径",
        "sceneId": "场景ID",
        "createdAt": "创建时间",
        "updatedAt": "更新时间"
      }
    ]
  }
  ```

#### 4.2 获取指定场景的样式图片

- **URL**: `/api/style-images/scene/:sceneId`
- **方法**: GET
- **描述**: 获取指定场景ID的所有样式图片
- **参数**:
  - `sceneId`: 场景ID (URL参数)
- **响应示例**:
  ```json
  {
    "success": true,
    "styleImages": [
      {
        "id": "样式图片ID",
        "filename": "样式图片文件名",
        "path": "样式图片路径",
        "sceneId": "场景ID",
        "createdAt": "创建时间",
        "updatedAt": "更新时间"
      }
    ]
  }
  ```

#### 4.3 上传样式图片

- **URL**: `/api/style-images`
- **方法**: POST
- **描述**: 上传新的样式图片
- **请求格式**: multipart/form-data
- **参数**:
  - `file`: 样式图片文件
  - `sceneId`: 场景ID
- **响应示例**:
  ```json
  {
    "success": true,
    "styleImage": {
      "id": "新样式图片ID",
      "filename": "样式图片文件名",
      "path": "样式图片路径",
      "sceneId": "场景ID",
      "createdAt": "创建时间",
      "updatedAt": "更新时间"
    }
  }
  ```

## 静态资源访问

### 图片访问

- **URL格式**: `/uploads/images/{图片ID}/{文件名}`
- **方法**: GET
- **描述**: 直接访问上传的图片文件

### 样式图片访问

- **URL格式**: `/uploads/styles/{样式图片ID}/{文件名}`
- **方法**: GET
- **描述**: 直接访问上传的样式图片文件

## 错误码说明

- 200: 请求成功
- 400: 请求参数错误
- 404: 资源不存在
- 500: 服务器内部错误