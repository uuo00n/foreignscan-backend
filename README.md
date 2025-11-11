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

scripts/init-db.go 提供三种模式以适配不同场景：

- full-init：清空并初始化数据库结构，从文件系统导入 scenes/styleImages/images，且可选在导入后补充缺失的 issues/comparisons 测试数据
- augment-existing：不清库，仅为现有 images 增补缺失的 issues/comparisons，可选择 dry-run 预览
- structure-only：只创建集合与基础索引，不导入或增补任何数据，适合在新机器上“只搭结构”

常用示例（非交互模式）：

```bash
# 1) 仅初始化结构（不导数据）
go run scripts/init-db.go -mode=structure-only -interactive=false -mongo-uri="mongodb://localhost:27017" -db-name="foreignscan"

# 2) 全量初始化（导入文件系统数据 + 可选补充测试数据）
# 根据你的实际目录选择 images-dir/styles-dir，默认使用 ./uploads/images 与 ./uploads/styles
go run scripts/init-db.go -mode=full-init -interactive=false \
  -mongo-uri="mongodb://localhost:27017" -db-name="foreignscan" \
  -images-dir="./uploads/images" -styles-dir="./uploads/styles" \
  -seed-extra=true

# 3) 增补现有数据（只为缺失的 issues/comparisons 补数据）
go run scripts/init-db.go -mode=augment-existing -interactive=false \
  -mongo-uri="mongodb://localhost:27017" -db-name="foreignscan" \
  -dry-run=true -limit=100
```

交互模式：

```bash
go run scripts/init-db.go
```

- 运行后按提示选择模式与参数；当选择 structure-only 时，只进行集合与索引初始化

可用参数说明：

- -mode：运行模式，可选 full-init | augment-existing | structure-only（必选，交互模式下会提示选择）
- -interactive：是否交互模式，默认 true；设为 false 使用命令行参数直接运行
- -mongo-uri：MongoDB 连接 URI，默认 mongodb://localhost:27017
- -db-name：数据库名称，默认 foreignscan
- -images-dir：图片数据目录（仅 full-init 使用），默认 ./uploads/images
- -styles-dir：样式图目录（仅 full-init 使用），默认 ./uploads/styles
- -seed-extra：是否在 full-init 导入后补充缺失的 issues/comparisons 测试数据（仅 full-init 使用），默认 false
- -dry-run：仅打印计划操作而不写库（仅 augment-existing 使用），默认 false
- -limit：最多处理多少条 images（仅 augment-existing 使用），默认 0 表示无限制

注意事项：

- structure-only 模式不会删除任何数据，也不会导入/增补数据，只保证基础集合与索引存在
- full-init 模式会清理相关集合后再导入，请谨慎在生产环境使用
- 如果你的数据目录在 cmd/server/uploads 下，请将 -images-dir/-styles-dir 指向对应子目录；也可以使用绝对路径

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
