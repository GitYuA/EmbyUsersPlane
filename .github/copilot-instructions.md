# Emby Users Panel - Copilot Instructions (Go)

## 1. Scope

This project is now a **single-binary Go application** (not PHP).

- Admin service: `:8086`
- User query service: `:8085`
- Runtime data directory: `/data` (config + sqlite + logs)
- Main entry: `main.go`

Any suggestion that adds back `src/*.php`, `public/index.php` PHP flow, or PHP-only dependencies is incorrect.

## 2. Runtime Architecture

### 2.1 Process model

`main.go` starts two HTTP servers in one process:

- `registerAdminRoutes(...)` for admin panel
- `registerQueryRoutes(...)` for user self-query

### 2.2 Route map

Admin side:

- `GET/POST /`
- `GET/POST /index.php` (compat path, still handled by Go)
- Static assets: `/assets/*`

Query side:

- `GET /` -> redirect to `/user/user.html`
- `GET /user/user.html`
- `POST /query.php`
- `POST /user/query.php`
- `/index.php` is blocked on query service

### 2.3 Data and state

- Config file: `/data/config.json` via `ConfigStore`
- SQLite: per-server DB under `/data/users/<safe_server_name>/users.db`
- Logs: `/data/operation_log.txt` and per-server logs under `/data/log/...`
- Rate limit data: `/data/rate_limit`

`users` table schema is created in Go (`DBManager.get`) with fields:
`id,name,open_date,last_recharge,expire_date,days_left,status,group_name,email,last_notify_date,charge_history`.

### 2.4 Scheduler

`startAutoCheckScheduler()` runs every minute and triggers per-server auto-check at configured `checkTime`.
No external PHP cron script is required.

## 3. Request Handling Conventions

### 3.1 Admin action dispatch

`dispatchAdminAction(...)` accepts:

- `charge`
- `create`
- `save_edit`
- `delete`
- `batch`
- `refresh_cache`
- `server_op`
- `settings_op`
- `test_email`
- `restore`
- `get_users`
- `get_logs`
- `run_auto_check`

JSON response shape should stay stable:

- success: boolean
- message: string
- optional payload fields (`user`, `users`, `files`, etc.)

### 3.2 Security

- Session + CSRF token are used on admin side.
- Query API can require token (`query_require_token`, `query_token` in config).
- Keep existing security headers wrapper (`secure`, `secureFunc`) when adding routes.

## 4. Build and Dependency Rules

- Required Go version: `go 1.24` (see `go.mod`).
- Docker multi-stage build compiles static binary from `main.go`.
- If dependencies change, update module metadata with:
  - `go mod tidy`
  - commit both `go.mod` and `go.sum`

## 5. Editing Guardrails for AI

1. Do not reintroduce PHP architecture references.
2. Prefer extending existing functions in `main.go` instead of creating parallel patterns.
3. Keep route compatibility (`/index.php`, `/query.php`) unless migration plan is explicit.
4. Preserve data compatibility in `/data/config.json` and existing SQLite schema.
5. Template files in `templates/` and `public/` must remain valid HTML:
   - no escaped HTML like `\"` inside attribute names
   - avoid injecting JS strings directly into HTML attributes
6. Use UTF-8 **without BOM** for `go.mod`, Go files, and templates to avoid parser/build issues.

## 6. Quick Verification Checklist

After changes, run at least:

```bash
go mod tidy
go build ./...
docker compose build
```

Then verify:

- Admin panel: `http://localhost:8086`
- Query page: `http://localhost:8085/user/user.html`
- API query: `POST /query.php` works as expected
