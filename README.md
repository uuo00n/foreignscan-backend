# ForeignScan 后端服务 (Go版本)

这是ForeignScan项目的Go语言后端实现，提供图片上传、存储和检测功能。

## 项目结构

```
foreignscan-backend/ 
├── cmd/ 
│   └── server/ 
│       ├── main.go         # 服务器入口点 
│       └── uploads/        # 上传文件临时存储目录
├── docs/                   # Swagger自动生成的API文档
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
├── internal/ 
│   ├── config/ 
│   │   └── config.go       # 配置管理 
│   ├── database/ 
│   │   └── mongodb.go      # 数据库连接 
│   ├── handlers/ 
│   │   ├── common.go       # 通用处理函数和结构
│   │   ├── images.go       # 图片相关API处理 
│   │   ├── scenes.go       # 场景相关API处理
│   │   ├── style_images.go # 样式图片相关API处理
│   │   └── upload.go       # 上传处理 
│   ├── middleware/ 
│   │   └── middleware.go   # 中间件 
│   └── models/ 
│       ├── image.go        # 图片模型 
│       ├── scene.go        # 场景模型
│       └── style_image.go  # 样式图片模型
├── pkg/ 
│   └── utils/ 
│       └── utils.go        # 通用工具函数 
├── scripts/
│   └── init-db.go          # 数据库初始化脚本
├── uploads/                # 上传文件存储目录
└── go.mod                  # Go模块定义 
```

## 功能特性

- 图片上传和存储管理
- 场景和样式图片管理
- 图片元数据管理
- 图片检测API
- 完整的RESTful API接口
- Swagger API文档

## 技术栈

- Go 1.21+
- Gin Web框架
- MongoDB数据库
- Swagger文档生成
- 模块化架构设计

## 快速开始

### 前置条件

- 安装Go 1.21或更高版本
- 安装并运行MongoDB服务

### 安装依赖

```bash
go mod download
```

### 初始化数据库

项目提供了数据库初始化脚本，可以通过以下命令运行：

```bash
go run scripts/init-db.go
```

脚本支持以下命令行参数：

| 参数            | 说明             | 默认值                    |
| --------------- | ---------------- | ------------------------- |
| `--test-data`   | 是否插入测试数据 | false                     |
| `--interactive` | 是否使用交互模式 | true                      |
| `--mongo-uri`   | MongoDB连接URI   | mongodb://localhost:27017 |
| `--db-name`     | 数据库名称       | foreignscan               |
| `--uploads-dir` | 上传目录路径     | ./uploads                 |

示例：

```bash
# 不使用交互模式，插入测试数据
go run scripts/init-db.go --interactive=false --test-data=true

# 不使用交互模式，不插入测试数据
go run scripts/init-db.go --interactive=false --test-data=false

# 使用自定义数据库名称
go run scripts/init-db.go --db-name=mydb
```

### 运行服务

```bash
cd cmd/server
go run main.go
```

服务将在 http://localhost:3000 上启动。

## API接口

### Swagger文档

项目集成了Swagger文档，可以通过以下URL访问API文档：

```
http://localhost:3000/swagger/index.html
```

### 如何更新Swagger文档

当修改API或添加新API后，需要重新生成Swagger文档：

```bash
# 安装swag工具（如果尚未安装）
go install github.com/swaggo/swag/cmd/swag@latest

# 生成Swagger文档
$(go env GOPATH)/bin/swag init -g cmd/server/main.go
```

### API概览

项目提供以下主要API：

#### 图片相关API

- `GET /api/images` - 获取所有图片列表
- `GET /api/images/:id` - 获取单个图片详情
- `GET /api/images/:id/detect` - 检测图片内容

#### 场景相关API

- `GET /api/scenes` - 获取所有场景
- `GET /api/scenes/:id` - 获取单个场景详情
- `POST /api/scenes` - 创建新场景
- `PUT /api/scenes/:id` - 更新场景
- `DELETE /api/scenes/:id` - 删除场景
- `GET /api/scenes/:id/images` - 获取特定场景下的图片列表
- `GET /api/scenes/:id/first-image` - 获取特定场景下的第一张图片
- `GET /api/scenes/first-images` - 获取所有场景的第一张图片

#### 样式图片相关API

- `GET /api/style-images` - 获取所有样式图片
- `GET /api/style-images/scene/:sceneId` - 获取特定场景下的样式图片
- `GET /api/style-images/:id` - 获取单个样式图片详情
- `POST /api/style-images` - 上传样式图片
- `PUT /api/style-images/:id` - 更新样式图片
- `DELETE /api/style-images/:id` - 删除样式图片

#### 上传相关API

- `POST /api/upload` - 上传图片
