# API Gateway Manager

基于 Kong 的 API 网关管理系统：空间 / 用户 / 网关 / API 分组 / API 发布与版本切换。

## 技术栈

| 层 | 技术 |
|----|------|
| 前端 | Vue 3 + Vite + TypeScript + Element Plus + Pinia + Vue Router |
| 后端 | Go + Gin + Zap + GORM + JWT |
| 数据 | PostgreSQL |
| 网关 | Kong（通过 go-kong 管理 Admin API） |

## 快速启动

### 1. 启动依赖（PostgreSQL + Kong）

```bash
docker compose up -d
```

等待 Kong 健康后，Admin API 为 `http://localhost:18001`，代理端口 `18000`。

> 若本机已有 Kong 占用 `8001`，本项目默认映射到 `18001` 以避免冲突。也可在网关管理中直接填写已有 Kong 的 Admin API（如 `http://localhost:8001`）。

### 2. 启动后端

```bash
cd backend
go run ./cmd/server -config configs/config.yaml
```

默认监听 `http://localhost:8088`。

### 3. 启动前端

```bash
cd frontend
npm install
npm run dev
```

浏览器打开 `http://localhost:5173`。

## 默认约定

- **首个注册用户**自动成为 `system_admin`（系统管理员）
- 后续注册用户角色为 `member`，可申请空间或加入已有空间
- **申请空间即创建**，申请人成为该空间 `space_admin`
- 空间创建成功后，**所有系统管理员**自动加入该空间（空间角色为 `space_admin`）
- 角色层级：`system_admin` > `space_admin` > `member`

## 功能说明

### 空间管理

- 申请 / 更新 / 删除空间
- 加入空间（用户可属于多个空间）
- 成员列表与空间内角色调整（`space_admin` / `member`）

### 网关管理（仅系统管理员）

- 字段：网关名、Admin API、Domain（`IP:端口` 或 `域名:端口`）、网络区域
- 示例：`http://localhost:18001` / 网络区域 `内网` 或 `DMZ`

### API 分组

- 隶属于某个空间
- 创建时必须指定所属网关

### API 管理

- 增删改查：名称、路径、方法、上游地址、Strip Path
- **发布**：通过 go-kong 创建/更新 Kong Service + Route，并生成版本（`v1`、`v2`…）
- **下线**：删除 Kong 上对应 Service/Route
- **版本切换**：下线当前配置，按所选版本快照重新发布

## 配置

后端配置见 [backend/configs/config.yaml](backend/configs/config.yaml)：

- 数据库默认：`agm / agm123 @ localhost:5432 / api_gateway_manager`
- JWT secret 与过期时间可按需修改
- 环境变量前缀 `AGM_`（如 `AGM_SERVER_PORT=8088`）

## 本地验证账号

首次启动后自行注册即可。冒烟测试示例：

- 用户名：`admin` / 密码：`admin123`（首个注册用户为系统管理员）

## 目录结构

```
├── backend/           # Go 后端
│   ├── cmd/server/
│   ├── configs/
│   └── internal/
├── frontend/          # Vue3 前端
├── docker-compose.yml # PostgreSQL + Kong
└── Makefile
```

## API 前缀

后端 REST 接口统一前缀：`/api/v1`。前端开发服务器已将 `/api` 代理到 `http://localhost:8088`。
