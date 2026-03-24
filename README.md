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
- **容器编排**: Docker Compose
- **ORM 框架**: GORM
- **接口文档**: Swagger

## Docker 启动

推荐优先使用 Docker 版本开发和运行，数据库会随编排自动启动，不再依赖本机单独安装和维护 PostgreSQL。

### 启动方式总览

Linux/macOS:

```bash
# 开发环境启动（按顺序：postgres -> healthy -> api）
./scripts/linux/dev-up.sh

# 开发环境重建（先 down，再 build+up，保留 volumes）
./scripts/linux/dev-rebuild.sh

# 开发环境停止
./scripts/linux/dev-down.sh

# 生产环境启动（按顺序：postgres -> healthy -> api）
./scripts/linux/prod-up.sh

# 生产环境重建（先 down，再 build+up，保留 volumes）
./scripts/linux/prod-rebuild.sh

# 生产环境停止
./scripts/linux/prod-down.sh
```

### 前置要求

- **Docker**: 29+
- **Docker Compose**: v2+
- **YOLO 检测服务**: 仍为外部依赖，默认地址为 `http://host.docker.internal:8077`

### 1. 准备 Docker 环境变量

```bash
cp .env.docker.example .env.docker
```

默认会创建以下数据库配置：

- `POSTGRES_DB=foreignscan`
- `POSTGRES_USER=postgres`
- `POSTGRES_PASSWORD=postgres`

### 2. 推荐：使用顺序化脚本启动

脚本会固定执行顺序：

1. 先启动 `postgres`
2. 等待 `postgres` 变为 `healthy`
3. 再启动 `api`

开发版（前台日志可改为 `-d` 后自行看 logs）：

```bash
./scripts/linux/dev-up.sh
```

如果你改了 Dockerfile、Compose 配置，或者想整套容器重建但保留数据卷，直接执行：

```bash
./scripts/linux/dev-rebuild.sh
```

生产版：

```bash
./scripts/linux/prod-up.sh
```

生产版整套重建：

```bash
./scripts/linux/prod-rebuild.sh
```

停止服务：

```bash
./scripts/linux/dev-down.sh
# 或
./scripts/linux/prod-down.sh
```

`./scripts/linux/dev-rebuild.sh` / `./scripts/linux/prod-rebuild.sh` 会先执行 `down --remove-orphans`，再执行 `up --build`，默认保留数据库和上传文件 volumes。

### 3. Windows 手动容器管理命令（docker compose）

以下命令适用于 Windows 环境（PowerShell/cmd），需在 `foreignscan-backend` 根目录执行。

开发环境（`compose.dev.yml`）：

```bash
# 启动容器
docker compose --env-file .env.docker -f compose.yml -f compose.dev.yml up -d postgres api

# 重新构建容器
docker compose --env-file .env.docker -f compose.yml -f compose.dev.yml down --remove-orphans
docker compose --env-file .env.docker -f compose.yml -f compose.dev.yml up --build -d postgres api

# 重启容器
docker compose --env-file .env.docker -f compose.yml -f compose.dev.yml restart postgres api

# 删除容器（保留数据卷）
docker compose --env-file .env.docker -f compose.yml -f compose.dev.yml down --remove-orphans

# 如需同时删除数据卷
docker compose --env-file .env.docker -f compose.yml -f compose.dev.yml down --remove-orphans --volumes
```

生产环境（`compose.prod.yml`）：

```bash
# 启动容器
docker compose --env-file .env.docker -f compose.yml -f compose.prod.yml up -d postgres api

# 重新构建容器
docker compose --env-file .env.docker -f compose.yml -f compose.prod.yml down --remove-orphans
docker compose --env-file .env.docker -f compose.yml -f compose.prod.yml up --build -d postgres api

# 重启容器
docker compose --env-file .env.docker -f compose.yml -f compose.prod.yml restart postgres api

# 删除容器（保留数据卷）
docker compose --env-file .env.docker -f compose.yml -f compose.prod.yml down --remove-orphans

# 如需同时删除数据卷
docker compose --env-file .env.docker -f compose.yml -f compose.prod.yml down --remove-orphans --volumes
```

### 4. 启动后校验

```bash
curl http://localhost:3000/health
curl http://localhost:3000/ready
```

`/ready` 返回 `{"status":"ready"}` 说明数据库依赖已就绪。

### 5. 数据与文件持久化

- 开发版数据库数据保存在 Docker volume `postgres_data`
- 开发版上传文件保存在仓库根目录 `uploads/`
- 生产版数据库数据保存在 Docker volume `postgres_data`
- 生产版上传文件保存在 Docker volume `uploads_data`

### 6. 外部检测服务说明

当前仓库未容器化 YOLO 检测服务，后端继续通过 `FS_DETECT_URL` 调用它。

- 默认 Docker 地址：`http://host.docker.internal:8077`
- 如果你的检测服务不在宿主机，请修改 `.env.docker`

## 裸机启动

如果你仍然需要本机直接运行 Go 服务，可以使用下面的方式。

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
FS_API_PORT=3000
FS_POSTGRES_DSN="host=127.0.0.1 user=postgres password=your_password dbname=foreignscan port=5432 sslmode=disable TimeZone=Asia/Shanghai"
FS_UPLOAD_DIR="cmd/server/uploads"
FS_DETECT_URL="http://127.0.0.1:8077"
FS_ALLOWED_ORIGINS="http://localhost:8080,http://127.0.0.1:8080"
```

### 4. 启动服务

```bash
cd cmd/server
go run main.go
```

服务默认监听 `http://localhost:3000`。

## API 文档

服务启动后，访问以下 URL 查看完整的 API 文档：

**http://localhost:3000/swagger/index.html**

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
