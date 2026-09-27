# API Gateway Manager

A Kong-based API gateway management system covering spaces and membership approval, shared/private gateway authorization, API groups, upstreams, consumers, plugins, API publish/version switching, OpenAPI import and batch operations, plus an API market for sharing and cross-space consumer linking.

[中文](README.md)

## Tech Stack

| Layer | Stack |
|-------|-------|
| Frontend | Vue 3 + Vite + TypeScript + Element Plus + Pinia + Vue Router |
| Backend | Go + Gin + Zap + GORM + JWT |
| Data | PostgreSQL |
| Gateway | Kong (Admin API via go-kong; plugin catalog aligned with Kong Gateway 3.4.2 OSS) |

## Quick Start

### Docker Compose (recommended)

```bash
docker compose up -d --build
# or
make docker-up
```

After startup:

| Service | URL |
|---------|-----|
| Frontend | http://localhost:5373 |
| Backend API | http://localhost:10000 |
| Health check | http://localhost:10000/health |
| PostgreSQL | localhost:15444 |

If port `10000` is already in use: `AGM_BACKEND_PORT=11000 docker compose up -d --build`.

Frontend Nginx reverse-proxies `/api` to the backend. For production, override the JWT secret:

```bash
AGM_JWT_SECRET=your-strong-secret docker compose up -d --build
```

Stop:

```bash
docker compose down
# or
make docker-down
```

The Kong service in `docker-compose.yml` is commented out by default. Use an existing Kong and set its Admin API in Gateway Management (e.g. `http://host.docker.internal:8001`).

### Local development

#### 1. PostgreSQL only

```bash
docker compose up -d postgres
# or
make deps
```

Postgres is mapped to `localhost:15444`.

#### 2. Backend

```bash
cd backend
go run ./cmd/server -config configs/config.yaml
# or
make backend
```

Listens on `http://localhost:10000` by default.

#### 3. Frontend

```bash
cd frontend
npm install
npm run dev
# or
make frontend
```

Open `http://localhost:5373`. The Vite dev server proxies `/api` to `http://localhost:10000`.

## Conventions

- The **first registered user** becomes `system_admin`
- Later users get the `member` role
- **Space applications**: non-admin creates a space in `pending` status until a system admin approves it (`active`); spaces created by system admins are active immediately
- The applicant becomes `space_admin` of that space; once active, **all system admins** are auto-added as `space_admin`
- **Join requests**: non-admin joins as `pending` until a space admin approves; system admins join as active `space_admin` immediately
- Role hierarchy: `system_admin` > `space_admin` > `member`
- Name uniqueness: space names are globally unique; group names unique within a space; API names unique within a group; upstream / consumer / plugin names unique within a space
- Lists (spaces, groups, APIs, upstreams, consumers, plugins, API market) default to 10 items per page (20 / 50 / 100 available)
- Lists support name (and related) search filters (APIs also by status/share; market also by auth type)

## Features

### Spaces

- Apply / update / delete spaces; a space with groups cannot be deleted
- Path prefix (e.g. `/order`) is set at creation and immutable; it is prepended to access paths when publishing APIs
- System admins can approve or reject pending spaces (reject deletes the application)
- Users may belong to multiple spaces; space admins approve / reject join requests
- Space admins can add members from candidate users and change space roles (`space_admin` / `member`)
- The space owner cannot be removed
- List supports search by space name

### Gateways (system admin only)

- Fields: name, Admin API, Domain (`IP:port` or `hostname:port`), network zone, shared flag
- Admin API reachability is probed on create; Admin API is immutable afterward
- **Shared gateways** (default): available to all spaces when creating API groups
- **Private gateways**: only authorized spaces can select them; system admins can authorize active spaces to a private gateway in batch
- Group binding exposes only gateway name and network zone (not Admin API)
- API list and API market show full access URLs using Domain + space prefix

### API Groups

- Belong to a space
- Must bind a gateway at creation (shared gateways plus private gateways authorized for that space); gateway binding is immutable
- A group with APIs cannot be deleted
- List supports search by group name

### Upstreams

- Space-scoped; load balancing (round-robin, least-connections, consistent-hashing, latency), target weights, and health checks
- Names are unique within a space; Kong names are prefixed by space to avoid collisions across spaces on the same gateway
- An API backend host can be a direct address or an upstream
- List supports search by name

### Consumers

- Space-scoped; username, Custom ID, and credentials (key-auth, basic-auth, jwt, hmac-auth, acl)
- Can be linked to APIs that enable matching auth plugins (including market APIs linked across spaces)
- List shows associated API count; click to open a paged detail dialog (API name, space/group, protocols, full path, methods, auth, status, version, etc.)
- Synced to gateways bound by the space’s groups on save; manual “sync to gateway” is also available; re-synced when groups change
- Kong usernames are prefixed by space
- List supports search by name

### Plugins

- Space-scoped; types match Kong Gateway **3.4.2 OSS** built-ins, grouped bilingually when creating:
  - Authentication: basic-auth, hmac-auth, jwt, key-auth, ldap-auth, oauth2, session
  - Security: acme, bot-detection, cors, ip-restriction
  - Traffic Control: acl, proxy-cache, rate-limiting, request-size-limiting, request-termination, response-ratelimiting
  - Serverless: aws-lambda, azure-functions, pre-function, post-function
  - Analytics & Monitoring: datadog, opentelemetry, prometheus, statsd, zipkin
  - Transformations: correlation-id, grpc-gateway, grpc-web, request-transformer, response-transformer
  - Logging: file-log, http-log, loggly, syslog, tcp-log, udp-log
- Common plugins have dedicated forms; enum fields use selects; `request-transformer` / `response-transformer` have visual editors; others follow Kong 3.4.2 schema (nested fields as JSON), validated by Kong on publish
- APIs can attach multiple plugins; bindings sync to the Kong Service on publish / update
- Updating or deleting a plugin refreshes still-published linked APIs
- View APIs linked to a plugin; version details show the plugin snapshot for that version
- List supports search by name

### APIs

- Access: protocols, paths, methods, hosts, headers, strip path, request/response buffering
- Access paths support Kong regex: a leading `~`, or paths containing `*` get `~` prefixed when published; after space-prefix join, `~` stays at the front
- Upstream service: protocol, direct host or upstream, port, path, retries, timeouts
- Auth: key-auth, basic-auth, jwt, hmac-auth, acl, with space plugins and consumers
- **OpenAPI import**: OpenAPI 3 / Swagger 2 JSON or YAML (max 8MB); optional backend protocol/host/port/path overrides; preview before import; access paths get the space prefix; same-named APIs in the group are updated instead of created
- **Publish**: create/update Kong Service + Route via go-kong (route name `agm-{space}-{group}-{api}`) and record versions (`v1`, `v2`, …)
- **Offline**: remove Kong Service/Route; APIs linked to consumers cannot go offline; market share is cleared on offline
- **Delete**: published APIs or APIs linked to consumers cannot be deleted
- **Switch version**: offline current config and republish from the selected snapshot
- **Share / unshare**: only published APIs can be shared to the API market
- **Batch operations**: multi-select to publish, offline, or delete (ineligible items are skipped)
- **Copy as curl**: build a curl command from gateway Domain, access path, and auth, then copy to clipboard
- List supports search by name, status, and share flag

### API Market

- Lists all shared published APIs
- Shows space, access protocols, full access URL, methods, auth type, and current version
- Visible to all logged-in users; share/unshare is managed by space admins in API Management
- Space admins can link their space’s consumers to market APIs that have auth enabled (cross-space consumption); APIs without auth cannot be linked
- List supports search by API name and auth type

## Configuration

See [backend/configs/config.yaml](backend/configs/config.yaml):

- Server port defaults to `10000`
- Database defaults: `agm / agm123 @ localhost:15444 / api_gateway_manager`
- JWT secret and expiry are configurable
- Env var prefix `AGM_` (e.g. `AGM_SERVER_PORT=10000`, `AGM_JWT_SECRET=...`)

## Smoke Test Account

Register after first startup. Example:

- Username: `admin` / password: `admin123` (first user becomes system admin)

## Layout

```
├── backend/           # Go backend
│   ├── cmd/server/
│   ├── configs/       # config.yaml (local) / config.docker.yaml (container)
│   ├── Dockerfile
│   └── internal/
├── frontend/          # Vue 3 frontend
│   ├── Dockerfile
│   └── nginx.conf     # static assets + /api reverse proxy
├── docker-compose.yml # postgres + backend + frontend (Kong commented out)
├── Makefile
├── README.md          # Chinese
└── README_EN.md       # English
```

## API Prefix

Backend REST APIs use the `/api/v1` prefix. The frontend dev server proxies `/api` to `http://localhost:10000`.
