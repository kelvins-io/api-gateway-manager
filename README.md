# API Gateway Manager

基于 Kong 的 API 网关管理系统：空间、用户、网关、API 分组、Upstream、Consumer、Plugin，以及 API 发布与版本切换。

## 技术栈

| 层 | 技术 |
|----|------|
| 前端 | Vue 3 + Vite + TypeScript + Element Plus + Pinia + Vue Router |
| 后端 | Go + Gin + Zap + GORM + JWT |
| 数据 | PostgreSQL |
| 网关 | Kong（通过 go-kong 管理 Admin API） |

## 快速启动

### Docker Compose 一键部署（推荐）

```bash
docker compose up -d --build
# 或
make docker-up
```

启动后访问：

| 服务 | 地址 |
|------|------|
| 前端 | http://localhost:5373 |
| 后端 API | http://localhost:10000 |
| PostgreSQL | localhost:15444 |

若本机 `10000` 已被占用，可改映射：`AGM_BACKEND_PORT=11000 docker compose up -d --build`。

前端 Nginx 会把 `/api` 反向代理到后端。生产环境建议通过环境变量覆盖 JWT：

```bash
AGM_JWT_SECRET=your-strong-secret docker compose up -d --build
```

停止：

```bash
docker compose down
# 或
make docker-down
```

`docker-compose.yml` 里的 Kong 服务默认注释掉，网关使用已有 Kong，在「网关管理」中填写其 Admin API（如 `http://host.docker.internal:8001`）。

### 本地开发

#### 1. 仅启动 PostgreSQL

```bash
docker compose up -d postgres
# 或
make deps
```

数据库映射到本机 `localhost:15444`。

#### 2. 启动后端

```bash
cd backend
go run ./cmd/server -config configs/config.yaml
```

默认监听 `http://localhost:10000`。

#### 3. 启动前端

```bash
cd frontend
npm install
npm run dev
```

浏览器打开 `http://localhost:5373`。开发服务器把 `/api` 代理到 `http://localhost:10000`。

## 默认约定

- **首个注册用户**自动成为 `system_admin`（系统管理员）
- 后续注册用户角色为 `member`，可申请空间或加入已有空间
- **申请空间即创建**，申请人成为该空间 `space_admin`
- 空间创建成功后，**所有系统管理员**自动加入该空间（空间角色为 `space_admin`）
- 角色层级：`system_admin` > `space_admin` > `member`
- 空间、API 分组、API、Upstream、Consumer、Plugin 列表默认每页 10 条，可切换为 20、50、100 条

## 功能说明

### 空间管理

- 申请 / 更新 / 删除空间；空间下仍有分组时不能删除
- 创建时指定路径前缀（如 `/order`），创建后不可修改；发布 API 时会拼到接入路径前面
- 加入空间（用户可属于多个空间）
- 成员列表与空间内角色调整（`space_admin` / `member`）

### 网关管理（仅系统管理员）

- 字段：网关名、Admin API、Domain（`IP:端口` 或 `域名:端口`）、网络区域
- 创建时探测 Admin API 是否可达；Admin API 创建后不可修改
- 分组绑定时只暴露网关名和网络区域，不暴露 Admin API

### API 分组

- 隶属于某个空间
- 创建时必须指定所属网关，创建后不可修改
- 分组下仍有 API 时不能删除

### Upstream

- 隶属于某个空间，可配置负载算法（round-robin、least-connections、consistent-hashing、latency）、Target 权重和健康检查
- 名称在同一空间内唯一；写入 Kong 时带空间前缀，避免多个空间共用同一网关时冲突
- API 的后端主机可以选择「直接地址」或某个 Upstream

### Consumers

- 隶属于某个空间，字段为用户名、Custom ID，以及凭证（key-auth、basic-auth、jwt、hmac-auth、acl）
- 可关联开启了对应认证插件的 API
- 保存后同步到该空间分组已绑定的网关；也可手动「同步到网关」。分组增删时会重新同步
- 写入 Kong 的用户名带空间前缀

### Plugins

- 隶属于某个空间；可选类型对齐 Kong Gateway **3.4.2 OSS** 自带插件，新建时按分类展示（中英双语）：
  - 认证 / Authentication：basic-auth、hmac-auth、jwt、key-auth、ldap-auth、oauth2、session
  - 安全 / Security：acme、bot-detection、cors、ip-restriction
  - 流量控制 / Traffic Control：acl、proxy-cache、rate-limiting、request-size-limiting、request-termination、response-ratelimiting
  - 无服务器 / Serverless：aws-lambda、azure-functions、pre-function、post-function
  - 分析与监控 / Analytics & Monitoring：datadog、opentelemetry、prometheus、statsd、zipkin
  - 转换 / Transformations：correlation-id、grpc-gateway、grpc-web、request-transformer、response-transformer
  - 日志 / Logging：file-log、http-log、loggly、syslog、tcp-log、udp-log
- 常用插件提供表单；其余插件按 Kong 3.4.2 schema 提供配置表单（嵌套结构用 JSON 字段），发布时由 Kong 校验
- API 可关联多个 Plugin；发布或更新关联后，同步到该 API 对应的 Kong Service
- 修改或删除 Plugin 时，会更新仍在发布状态的关联 API
- 可查看某个 Plugin 当前关联的 API

### API 管理

- 接入：协议、路径、方法、Host、Header、Strip Path
- 后端服务：协议、直接地址或 Upstream、端口、路径、重试和超时
- 认证：可启用 key-auth、basic-auth、jwt、hmac-auth、acl，并绑定空间内 Plugin 与 Consumer
- **发布**：通过 go-kong 创建/更新 Kong Service + Route，并生成版本（`v1`、`v2`…）
- **下线**：删除 Kong 上对应 Service/Route；已关联 Consumer 的 API 不允许下线
- **删除**：已发布或已关联 Consumer 的 API 不允许删除
- **版本切换**：下线当前配置，按所选版本快照重新发布

## 配置

后端配置见 [backend/configs/config.yaml](backend/configs/config.yaml)：

- 服务端口默认 `10000`
- 数据库默认：`agm / agm123 @ localhost:15444 / api_gateway_manager`
- JWT secret 与过期时间可按需修改
- 环境变量前缀 `AGM_`（如 `AGM_SERVER_PORT=10000`）

## 本地验证账号

首次启动后自行注册即可。冒烟测试示例：

- 用户名：`admin` / 密码：`admin123`（首个注册用户为系统管理员）

## 目录结构

```
├── backend/           # Go 后端
│   ├── cmd/server/
│   ├── configs/       # config.yaml（本地）/ config.docker.yaml（容器）
│   ├── Dockerfile
│   └── internal/
├── frontend/          # Vue3 前端
│   ├── Dockerfile
│   └── nginx.conf     # 静态资源 + /api 反代
├── docker-compose.yml # postgres + backend + frontend（Kong 已注释）
└── Makefile
```

## API 前缀

后端 REST 接口统一前缀：`/api/v1`。前端开发服务器已将 `/api` 代理到 `http://localhost:10000`。
