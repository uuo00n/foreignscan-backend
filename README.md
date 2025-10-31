# ForeignScan 后端服务 (Go版本)

这是ForeignScan项目的Go语言后端实现，提供图片上传、存储和检测功能。

## 项目结构

```
backend/ 
├── cmd/ 
│   └── server/ 
│       └── main.go         # 服务器入口点 
├── internal/ 
│   ├── config/ 
│   │   └── config.go       # 配置管理 
│   ├── handlers/ 
│   │   ├── images.go       # 图片相关API处理 
│   │   └── upload.go       # 上传处理 
│   ├── middleware/ 
│   │   └── middleware.go   # 中间件 
│   ├── models/ 
│   │   └── image.go        # 图片模型 
│   └── database/ 
│       └── mongodb.go      # 数据库连接 
├── pkg/ 
│   └── utils/ 
│       └── utils.go        # 通用工具函数 
└── go.mod                  # Go模块定义 
```

## 功能特性

- 图片上传和存储
- 图片元数据管理
- 图片检测API（模拟实现）
- RESTful API接口

## 技术栈

- Go 1.21+
- Gin Web框架
- MongoDB数据库
- 模块化架构设计

## 快速开始

### 前置条件

- 安装Go 1.21或更高版本
- 安装并运行MongoDB服务

### 安装依赖

```bash
go mod download
```

### 运行服务

```bash
cd cmd/server
go run main.go
```

服务将在 http://localhost:3000 上启动。

## API接口

### 健康检查

```
GET /ping
```

### 获取图片列表

```
GET /api/images
```

### 上传图片

```
POST /api/upload
POST /api/upload-image
```

参数:
- `image`: 图片文件
- `sceneId`: 场景ID
- `location`: 位置信息

### 检测图片

```
POST /api/detect
```

参数:
- `imageId`: 图片ID