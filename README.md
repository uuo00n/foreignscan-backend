# ForeignScan Backend

![Private](https://img.shields.io/badge/Repository-Private-red)
![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)
![Gin](https://img.shields.io/badge/Framework-Gin-00ADD8?style=flat&logo=go)
![PostgreSQL](https://img.shields.io/badge/Database-PostgreSQL-336791?style=flat&logo=postgresql)
![Swagger](https://img.shields.io/badge/Docs-Swagger-85EA2D?style=flat&logo=swagger)
![Copyright](https://img.shields.io/badge/Copyright-2026_uuo00n-blue)

**ForeignScan Backend** 是一款高性能的工业异物检测系统后端服务。该项目基于 Go 语言构建，提供了 RESTful API 接口，用于管理检测场景、处理图像数据流以及与 AI 推理服务的核心交互。

---

## 项目简介

本项目作为 ForeignScan 系统的核心枢纽，连接了前端桌面应用与底层的 AI 检测引擎。它负责处理图像上传与存储、调度检测任务，并确保在工业环境中对缺陷进行精确记录和可追溯管理。

## 主要功能

- **场景管理**：定义和管理不同的检测场景，支持参考样本维护。
- **图像处理**：高效的图像上传、存储及元数据管理，支持按时间与状态的多维度检索。
- **AI 调度**：与外部 YOLO 检测服务无缝集成，支持实时单图检测与批量离线任务。
- **任务追踪**：全链路追踪检测任务状态，支持进度查询与结果持久化。
- **API 文档**：内置 Swagger UI，方便进行交互式 API 调试。

## 技术栈

- **编程语言**: Go (1.24+)
- **Web 框架**: Gin
- **数据库**: PostgreSQL
- **ORM 框架**: GORM
- **接口文档**: Swagger

## 快速开始

### 前置要求

- **Go**: 1.24 或更高版本
- **PostgreSQL**: 13 或更高版本
- **AI 检测服务**: 必须已部署并运行 (默认地址: `http://127.0.0.1:8077`)

### 1. 获取代码

```bash
git clone <repository-url>
cd foreignscan-backend
```

### 2. 安装依赖

```bash
go mod download
```

### 3. 配置环境

在项目根目录下创建一个 `.env` 文件（可选，用于覆盖环境变量）：

```env
PORT=8080
POSTGRES_DSN="host=127.0.0.1 user=postgres password=your_password dbname=foreignscan port=5432 sslmode=disable TimeZone=Asia/Shanghai"
UPLOAD_DIR="cmd/server/uploads"
DETECT_SERVICE_URL="http://127.0.0.1:8077"
```

### 4. 启动服务

```bash
cd cmd/server
go run main.go
```

服务默认监听 `http://localhost:8080`。

## API 文档

服务启动后，访问以下 URL 查看完整的 API 文档：

**http://localhost:8080/swagger/index.html**

## 项目结构

```text
foreignscan-backend/
├── cmd/server/          # 应用程序入口
├── internal/
│   ├── config/          # 配置管理
│   ├── database/        # 数据库连接
│   ├── handlers/        # HTTP 处理函数
│   ├── models/          # 数据模型
│   ├── services/        # 业务逻辑
│   └── utils/           # 工具函数
├── pkg/                 # 公共包
└── docs/                # Swagger 文档
```

## 版权与许可

本项目为专有软件。**非开源**。

Copyright © 2026 uuo00n. 保留所有权利。

严禁通过任何媒介未经授权复制、修改、分发或使用本软件。
