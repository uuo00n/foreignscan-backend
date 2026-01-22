# ForeignScan 后端服务

ForeignScan 后端服务是一个使用 Go 实现的 RESTful API 服务，用于管理检验场景、图片及其检测结果，并对接外部检测服务完成缺陷检测。

## 简介

主要能力包括：

- 管理检验场景与样式图片
- 图片上传、存储以及元数据管理
- 与外部检测服务协作触发检测并保存检测结果
- 提供面向前端的查询接口和检测任务管理接口

## 功能特性

- 场景管理：创建、更新、删除场景，查询场景列表和首张图片
- 图片管理：上传图片、按场景和时间过滤查询、查看详细信息
- 样式图片管理：为场景维护参考/样式图片
- 检测任务：触发单图或整场景检测、查询检测结果、管理检测任务
- 数据持久化：使用 PostgreSQL 存储所有业务数据
- API 文档：内置 Swagger 文档，便于调试与对接

## 技术栈

- 语言：Go 1.24+
- Web 框架：Gin
- 数据访问：GORM + PostgreSQL
- 文档：Swagger（swaggo）

## 项目结构

```text
foreignscan-backend/
├── cmd/
│   └── server/
│       ├── main.go          # 服务器入口
│       └── uploads/         # 默认上传目录（可通过 UPLOAD_DIR 覆盖）
├── docs/                    # Swagger 自动生成的 API 文档
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
├── internal/
│   ├── config/
│   │   └── config.go        # 配置管理（读取 .env 和环境变量）
│   ├── database/
│   │   └── postgres.go      # PostgreSQL 连接（GORM）
│   ├── handlers/            # HTTP 处理器（Gin）
│   │   ├── common.go
│   │   ├── detect.go        # 检测入口和任务接口
│   │   ├── detections.go    # 检测结果查询
│   │   ├── images.go        # 图片相关 API
│   │   ├── scenes.go        # 场景相关 API
│   │   ├── style_images.go  # 样式图片相关 API
│   │   └── upload.go        # 上传处理
│   ├── middleware/
│   │   └── middleware.go    # CORS、日志等中间件
│   ├── models/
│   │   ├── image.go         # 图片模型
│   │   ├── scene.go         # 场景模型
│   │   ├── style_image.go   # 样式图片模型
│   │   └── detection.go     # 检测结果与检测运行记录
│   ├── services/
│   │   └── detect_job.go    # 检测任务管理与状态跟踪
│   └── utils/
│       └── yolo.go          # 与外部检测服务交互的工具方法
├── pkg/
│   └── utils/
│       └── utils.go         # 通用工具函数
├── scripts/
│   └── data/foreignscan/    # 示例数据导出（JSON，仅供参考）
└── go.mod                   # Go 模块定义
```

## 配置

配置来源按优先级从高到低依次为：

1. 进程环境变量
2. 项目根目录的 `.env` 文件
3. 代码中的默认值（见 `internal/config/config.go`）

### 环境变量

- `PORT`
  - 说明：HTTP 服务监听端口
  - 默认：`3000`

- `POSTGRES_DSN`
  - 说明：PostgreSQL 连接串（GORM DSN）
  - 默认：

    ```text
    host=localhost user=postgres password=postgres dbname=foreignscan port=5432 sslmode=disable TimeZone=Asia/Shanghai
    ```

- `UPLOAD_DIR`
  - 说明：上传根目录（用于存放 images/labels/styles 等子目录）
  - 默认：根据当前工作目录或可执行文件位置推断，一般为 `<repoRoot>/cmd/server/uploads`

- `ALLOWED_ORIGINS`
  - 说明：CORS 允许的前端 Origin
  - 默认：`*`

- `DETECT_SERVICE_URL`
  - 说明：外部检测服务的 HTTP 地址
  - 默认：`http://127.0.0.1:8077`

> 若进程环境中已设置某个变量，则 `.env` 中的同名配置不会覆盖它。

### `.env` 示例

在项目根目录 `foreignscan-backend` 下可以创建 `.env` 文件：

```env
PORT=3000
POSTGRES_DSN=host=127.0.0.1 user=postgres password=your_password dbname=foreignscan port=5432 sslmode=disable TimeZone=Asia/Shanghai
UPLOAD_DIR=cmd/server/uploads
ALLOWED_ORIGINS=*
DETECT_SERVICE_URL=http://127.0.0.1:8077
```

### 上传目录策略

最终使用的上传目录为 `config.Get().UploadDir`，其取值规则：

1. 若设置了环境变量或 `.env` 中的 `UPLOAD_DIR`：
   - 若为绝对路径（例如 `D:\data\foreignscan\uploads`），直接使用
   - 若为相对路径（例如 `cmd/server/uploads`），基于当前工作目录转换为绝对路径
2. 若未显式配置，则使用默认规则：
   - 当前工作目录在仓库内时，使用 `<repoRoot>/cmd/server/uploads`
   - 否则使用 `<可执行文件所在目录>/uploads`

## 快速开始

### 前置条件

- 已安装 Go 1.24 或更高版本
- 已安装并启动 PostgreSQL（推荐 13 及以上版本）
  - 建议准备：数据库 `foreignscan`，账号 `postgres`，并在 `POSTGRES_DSN` 中配置正确的密码
- 已部署并可访问的外部检测服务（对应 `DETECT_SERVICE_URL`）

### 安装依赖

在项目根目录执行：

```bash
go mod download
```

### 启动服务

在项目根目录执行：

```bash
cd cmd/server
go run main.go
```

默认情况下服务监听：

- `http://localhost:3000`

## 数据存储

本服务使用 PostgreSQL 存储所有业务数据，核心表由 GORM 模型自动迁移生成，包括：

- 场景（`scene.go`）
- 图片（`image.go`）
- 样式图片（`style_image.go`）
- 检测结果及检测运行记录（`detection.go`）

连接初始化逻辑位于 `internal/database/postgres.go`，根据 `POSTGRES_DSN` 建立连接并配置连接池。

## API 文档

服务内置 Swagger 文档，启动后可通过浏览器访问：

```text
http://localhost:3000/swagger/index.html
```

可在页面中查看所有可用接口、入参和返回结构，并进行在线调试调用。

## 开发说明

### 更新 Swagger 文档

在修改或新增 API 后，可通过 swag 工具重新生成 Swagger 文档：

```bash
go install github.com/swaggo/swag/cmd/swag@latest
$(go env GOPATH)/bin/swag init -g cmd/server/main.go
```

生成结果会写入 `docs/` 目录，并在服务启动时被导入用于提供 `/swagger` 接口。
