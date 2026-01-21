# ForeignScan 后端服务 (Go版本)

这是ForeignScan项目的Go语言后端实现，提供图片上传、存储和检测功能。

## 项目结构

```
foreignscan-backend/ 
├── cmd/ 
│   └── server/ 
│       ├── main.go         # 服务器入口点 
│       └── uploads/        # 默认上传目录（可通过 UPLOAD_DIR 覆盖）
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
├── uploads/                # 旧版上传目录（当前代码默认使用 cmd/server/uploads）
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

### 配置说明

后端配置通过三层来源决定（优先级从高到低）：

1. **系统环境变量**（如在 shell 或服务管理器中设置）
2. 项目根目录下的 **`.env` 文件**（当前仓库已提供示例 [.env](./.env)）
3. 代码中的默认值（见 [internal/config/config.go](./internal/config/config.go)）

`.env` 中支持的关键字段包括：

- `PORT`：服务监听端口，默认 `3000`
- `MONGO_URI`：MongoDB 连接串，默认 `mongodb://localhost:27017`
- `DB_NAME`：数据库名称，默认 `foreignscan`
- `ALLOWED_ORIGINS`：CORS 允许的源，默认 `*`
- `DETECT_SERVICE_URL`：YOLO 检测服务地址，默认 `http://127.0.0.1:8077`
- `UPLOAD_DIR`：上传目录

> 注意：若系统中已存在同名环境变量，`.env` 不会覆盖该变量。

#### 上传目录策略

最终使用的上传目录为 `config.Get().UploadDir`，其来源为：

1. 若设置了环境变量或 `.env` 中的 `UPLOAD_DIR`：
   - 若为绝对路径（如 `/data/foreignscan/uploads`），将直接使用
   - 若为相对路径（如 `cmd/server/uploads`），会基于当前工作目录转为绝对路径
2. 否则使用代码默认策略：
   - 若当前工作目录在仓库内，则为 `<repoRoot>/cmd/server/uploads`
   - 否则为 `<可执行文件所在目录>/uploads`

本地开发时通常可以：

- 不设置 `UPLOAD_DIR`，使用默认的 `cmd/server/uploads`

生产部署时建议显式设置绝对路径，例如：

```env
UPLOAD_DIR=/data/foreignscan/uploads
```

### 安装依赖

```bash
go mod download
```

### 初始化数据库

scripts/init-db.go 现支持“仅结构初始化”（不导入任何数据）：

- 功能：创建集合并建立基础索引，不删除、不导入、不增补数据
- 创建集合：`scenes`、`styleImages`、`images`、`detections`
- 创建索引：
  - `detections`：`imageId`、`sceneId+createdAt(desc)`、`summary.hasIssue`、`items.class`、`runId(unique)`
  - `images`：`sceneId`、`createdAt(desc)`、`status`、`isDetected`、`hasIssue`
  - `scenes`：`createdAt(desc)`
  - `styleImages`：`sceneId`、`createdAt(desc)`

使用示例：

```bash
# 非交互执行（推荐生产环境）
go run scripts/init-db.go -interactive=false -mongo-uri="mongodb://localhost:27017" -db-name="foreignscan"

# 交互执行（按提示输入 MongoDB URI / 数据库名）
go run scripts/init-db.go
```

参数说明：

- `-interactive`：是否交互模式，默认 `true`；设为 `false` 使用参数直接运行
- `-mongo-uri`：MongoDB 连接串，默认 `mongodb://localhost:27017`
- `-db-name`：数据库名称，默认 `foreignscan`

注意事项：

- 多次运行是幂等的：集合已存在时仅提示，不会影响现有数据
- 仅结构初始化不会导入文件系统数据；如需数据导入，请使用 `mongorestore` 或自定义脚本

### 数据导入/恢复（使用 mongorestore）

#### macOS（本地开发环境）

如果你已通过 `mongodump` 将数据库导出到 `scripts/data/foreignscan/`，可以使用下列命令进行恢复：

前置：安装 MongoDB 数据库工具（如未安装）

```bash
brew tap mongodb/brew && brew install mongodb-database-tools
```

完整恢复到 foreignscan（不删除现有数据）

```bash
mongorestore --uri="mongodb://localhost:27017" --db=foreignscan \
  "/Users/uu/Desktop/dnui-foreignscan/foreignscan-backend/scripts/data/foreignscan"
```

完整恢复到 foreignscan（覆盖现有集合数据，谨慎使用）

```bash
mongorestore --uri="mongodb://localhost:27017" --db=foreignscan --drop \
  "/Users/uu/Desktop/dnui-foreignscan/foreignscan-backend/scripts/data/foreignscan"
```

仅恢复指定集合（示例：只恢复 images 与 scenes）

```bash
mongorestore --uri="mongodb://localhost:27017" \
  --nsInclude foreignscan.images --nsInclude foreignscan.scenes \
  "/Users/uu/Desktop/dnui-foreignscan/foreignscan-backend/scripts/data/foreignscan"
```

恢复到其他数据库名（例如 mydb）

```bash
mongorestore --uri="mongodb://localhost:27017" --db=mydb \
  "/Users/uu/Desktop/dnui-foreignscan/foreignscan-backend/scripts/data/foreignscan"
```

说明与注意：

- 导出目录中同时包含 `*.bson` 与 `*.metadata.json`，其中 metadata 中保存了索引定义；默认会恢复索引，若不希望恢复索引，可加 `--noIndexRestore`
- 如果你的 MongoDB 需要认证，可加上 `--username <user> --password <pass> --authenticationDatabase admin`
- 恢复后可使用 `mongosh` 验证：
  - `mongosh` 进入 shell 后执行：
  - `use foreignscan`
  - `db.images.countDocuments()`、`db.scenes.countDocuments()` 等命令查看数量

---

提示：Windows 平台的恢复命令请参见下方《Windows 生产环境数据导出/导入方法》章节（含 PowerShell 与 CMD 示例）。

### Windows 生产环境数据导出/导入方法

说明：以下命令同时给出 PowerShell 与 CMD 两种用法。请根据你的环境选择一种即可。

1) 安装 MongoDB Database Tools（包含 mongodump/mongorestore）

- 方案A（推荐，需已安装 Chocolatey）：
  - 以管理员身份打开 PowerShell 或 CMD，执行：
  - choco install mongodb-database-tools
- 方案B（手动）：
  - 从 MongoDB 官方下载 Database Tools 的 Windows 压缩包（zip），解压后将解压目录下的 bin 路径加入系统 PATH 环境变量；之后新开终端即可使用 mongodump/mongorestore。

2) 导出整库（foreignscan）到项目 scripts/data 目录

- PowerShell 示例：
  - $OutDir = "C:\Users\<你的用户名>\Desktop\dnui-foreignscan\foreignscan-backend\scripts\data"
  - New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
  - mongodump --uri="mongodb://localhost:27017" --db=foreignscan --out="$OutDir"

- CMD 示例：
  - mkdir C:\Users\<你的用户名>\Desktop\dnui-foreignscan\foreignscan-backend\scripts\data
  - mongodump --uri="mongodb://localhost:27017" --db=foreignscan --out="C:\Users\<你的用户名>\Desktop\dnui-foreignscan\foreignscan-backend\scripts\data"

3) 恢复/导入到数据库

- PowerShell：
  - $DumpDir = "C:\Users\<你的用户名>\Desktop\dnui-foreignscan\foreignscan-backend\scripts\data\foreignscan"
  - 不覆盖现有数据：
    - mongorestore --uri="mongodb://localhost:27017" --db=foreignscan "$DumpDir"
  - 覆盖现有集合数据（谨慎）：
    - mongorestore --uri="mongodb://localhost:27017" --db=foreignscan --drop "$DumpDir"

- CMD：
  - mongorestore --uri="mongodb://localhost:27017" --db=foreignscan "C:\Users\<你的用户名>\Desktop\dnui-foreignscan\foreignscan-backend\scripts\data\foreignscan"
  - 覆盖现有集合：
    - mongorestore --uri="mongodb://localhost:27017" --db=foreignscan --drop "C:\Users\<你的用户名>\Desktop\dnui-foreignscan\foreignscan-backend\scripts\data\foreignscan"

4) 仅恢复指定集合（例如 images 与 scenes）

- PowerShell：
  - mongorestore --uri="mongodb://localhost:27017" \
    --nsInclude foreignscan.images --nsInclude foreignscan.scenes "$DumpDir"

- CMD：
  - mongorestore --uri="mongodb://localhost:27017" --nsInclude foreignscan.images --nsInclude foreignscan.scenes "C:\Users\<你的用户名>\Desktop\dnui-foreignscan\foreignscan-backend\scripts\data\foreignscan"

5) 恢复到其它数据库名（例如 mydb）

- mongorestore --uri="mongodb://localhost:27017" --db=mydb "C:\Users\<你的用户名>\Desktop\dnui-foreignscan\foreignscan-backend\scripts\data\foreignscan"

6) 认证与索引相关说明

- 如果需要认证：在命令后追加 --username <user> --password <pass> --authenticationDatabase admin
- 默认会恢复索引；如不希望恢复索引：追加 --noIndexRestore

7) 常见问题与注意事项

- 路径中包含空格时，请务必使用双引号包裹路径
- 安装后若命令未找到，确认 Database Tools 的 bin 目录已加入 PATH 并重启终端
- 如果连接的是远端或 Atlas，可能需要使用 SRV 连接串： --uri="mongodb+srv://<cluster-url>" 并按需追加认证参数

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
