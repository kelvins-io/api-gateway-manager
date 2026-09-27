# API Gateway Manager

基于 Kong 的 API 网关管理系统：空间与成员审批、共享/私有网关授权、API 分组、Upstream、Consumer、Plugin、API 发布与版本切换、OpenAPI 导入与批量运维，以及 API 市场分享与跨空间 Consumer 关联。

[English](README_EN.md)

## 技术栈

| 层 | 技术 |
|----|------|
| 前端 | Vue 3 + Vite + TypeScript + Element Plus + Pinia + Vue Router |
| 后端 | Go + Gin + Zap + GORM + JWT |
| 数据 | PostgreSQL |
| 网关 | Kong（通过 go-kong 管理 Admin API，插件目录对齐 Kong Gateway 3.4.2 OSS） |

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
| 健康检查 | http://localhost:10000/health |
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
# 或
make backend
```

默认监听 `http://localhost:10000`。

#### 3. 启动前端

```bash
cd frontend
npm install
npm run dev
# 或
make frontend
```

浏览器打开 `http://localhost:5373`。开发服务器把 `/api` 代理到 `http://localhost:10000`。

## 默认约定

- **首个注册用户**自动成为 `system_admin`（系统管理员）
- 后续注册用户角色为 `member`
- **申请空间**：普通用户创建后状态为 `pending`，需系统管理员审批通过后变为 `active`；系统管理员创建的空间直接 `active`
- 申请人成为该空间 `space_admin`；空间激活后，**所有系统管理员**自动加入该空间（空间角色为 `space_admin`）
- **加入空间**：普通用户申请后为 `pending`，需空间管理员确认；系统管理员加入时直接生效且为 `space_admin`
- 角色层级：`system_admin` > `space_admin` > `member`
- 命名唯一性：空间名全局唯一；分组名在同一空间内唯一；API 名在同一分组内唯一；Upstream / Consumer / Plugin 名在同一空间内唯一
- 空间、API 分组、API、Upstream、Consumer、Plugin、API 市场列表默认每页 10 条，可切换为 20、50、100 条
- 各列表支持按名称等条件搜索过滤（API 还可按状态、分享筛选；API 市场可按认证类型筛选）

## 功能说明

### 空间管理

- 申请 / 更新 / 删除空间；空间下仍有分组时不能删除
- 创建时指定路径前缀（如 `/order`），创建后不可修改；发布 API 时会拼到接入路径前面
- 系统管理员可审批或拒绝待审批空间（拒绝即删除申请）
- 加入空间（用户可属于多个空间）；空间管理员可审批 / 拒绝加入申请
- 空间管理员可从候选用户中直接添加成员，并调整空间内角色（`space_admin` / `member`）
- 空间所有者不可被移除
- 列表支持按空间名称搜索

### 网关管理（仅系统管理员）

- 字段：网关名、Admin API、Domain（`IP:端口` 或 `域名:端口`）、网络区域、是否共享
- 创建时探测 Admin API 是否可达；Admin API 创建后不可修改
- **共享网关**（默认）：所有空间创建 API 分组时均可选用
- **私有网关**：仅被授权的空间可选；系统管理员可将未授权的 active 空间批量授权给该网关
- 分组绑定时只暴露网关名和网络区域，不暴露 Admin API
- API 列表与 API 市场会结合 Domain + 空间前缀展示完整访问地址

### API 分组

- 隶属于某个空间
- 创建时必须指定所属网关（可选范围为共享网关 + 已授权给该空间的私有网关），创建后不可修改
- 分组下仍有 API 时不能删除
- 列表支持按分组名称搜索

### Upstream

- 隶属于某个空间，可配置负载算法（round-robin、least-connections、consistent-hashing、latency）、Target 权重和健康检查
- 名称在同一空间内唯一；写入 Kong 时带空间前缀，避免多个空间共用同一网关时冲突
- API 的后端主机可以选择「直接地址」或某个 Upstream
- 列表支持按名称搜索

### Consumers

- 隶属于某个空间，字段为用户名、Custom ID，以及凭证（key-auth、basic-auth、jwt、hmac-auth、acl）
- 可关联开启了对应认证插件的 API（含本空间及市场跨空间关联的 API）
- 列表展示关联 API 数量，点击可分页查看详情（API 名、所属空间/分组、协议、完整路径、方法、认证、状态、版本等）
- 保存后同步到该空间分组已绑定的网关；也可手动「同步到网关」。分组增删时会重新同步
- 写入 Kong 的用户名带空间前缀
- 列表支持按名称搜索

### Plugins

- 隶属于某个空间；可选类型对齐 Kong Gateway **3.4.2 OSS** 自带插件，新建时按分类展示（中英双语）：
  - 认证 / Authentication：basic-auth、hmac-auth、jwt、key-auth、ldap-auth、oauth2、session
  - 安全 / Security：acme、bot-detection、cors、ip-restriction
  - 流量控制 / Traffic Control：acl、proxy-cache、rate-limiting、request-size-limiting、request-termination、response-ratelimiting
  - 无服务器 / Serverless：aws-lambda、azure-functions、pre-function、post-function
  - 分析与监控 / Analytics & Monitoring：datadog、opentelemetry、prometheus、statsd、zipkin
  - 转换 / Transformations：correlation-id、grpc-gateway、grpc-web、request-transformer、response-transformer
  - 日志 / Logging：file-log、http-log、loggly、syslog、tcp-log、udp-log
- 常用插件提供表单；枚举类配置使用下拉选择；`request-transformer` / `response-transformer` 提供可视化编辑器；其余插件按 Kong 3.4.2 schema 提供配置表单（嵌套结构用 JSON 字段），发布时由 Kong 校验
- API 可关联多个 Plugin；发布或更新关联后，同步到该 API 对应的 Kong Service
- 修改或删除 Plugin 时，会更新仍在发布状态的关联 API
- 可查看某个 Plugin 当前关联的 API；版本详情中可查看该版本绑定的 Plugin 快照
- 列表支持按名称搜索

### API 管理

- 接入：协议、路径、方法、Host、Header、Strip Path、Request/Response Buffering
- 接入路径支持 Kong 正则：路径以 `~` 开头，或含 `*` 时发布到 Kong 会自动加上 `~`；空间前缀拼接后仍保留 `~` 在路径最前
- 后端服务：协议、直接地址或 Upstream、端口、路径、重试和超时
- 认证：可启用 key-auth、basic-auth、jwt、hmac-auth、acl，并绑定空间内 Plugin 与 Consumer
- **OpenAPI 导入**：支持 OpenAPI 3 / Swagger 2 的 JSON、YAML（最大 8MB）；可覆盖后端协议/主机/端口/Path；导入前可预览；接入路径自动拼接空间前缀；同名 API 在当前分组内更新而非新建
- **发布**：通过 go-kong 创建/更新 Kong Service + Route（Route 名为 `agm-{space}-{group}-{api}`），并生成版本（`v1`、`v2`…）
- **下线**：删除 Kong 上对应 Service/Route；已关联 Consumer 的 API 不允许下线；下线时自动取消市场分享
- **删除**：已发布或已关联 Consumer 的 API 不允许删除
- **版本切换**：下线当前配置，按所选版本快照重新发布
- **分享 / 取消分享**：仅已发布的 API 可分享到 API 市场
- **批量操作**：多选后可批量发布、下线、删除（自动跳过不符合条件的项）
- **复制 Curl**：按网关 Domain、接入路径与认证信息生成 curl 命令并复制到剪贴板
- 列表支持按名称、状态、是否分享搜索

### API 市场

- 展示所有已分享的已发布 API
- 可见所属空间、接入协议、完整访问地址、请求方法、认证类型与当前版本
- 登录用户均可浏览；分享与取消分享由对应空间管理员在 API 管理中操作
- 空间管理员可将本空间 Consumer 关联到已启用认证的市场 API（跨空间消费），未启用认证的 API 不可关联
- 列表支持按 API 名称、认证类型搜索

## 配置

后端配置见 [backend/configs/config.yaml](backend/configs/config.yaml)：

- 服务端口默认 `10000`
- 数据库默认：`agm / agm123 @ localhost:15444 / api_gateway_manager`
- JWT secret 与过期时间可按需修改
- 环境变量前缀 `AGM_`（如 `AGM_SERVER_PORT=10000`、`AGM_JWT_SECRET=...`）

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
├── Makefile
├── README.md          # 中文说明
└── README_EN.md       # English
```

## API 前缀

后端 REST 接口统一前缀：`/api/v1`。前端开发服务器已将 `/api` 代理到 `http://localhost:10000`。
