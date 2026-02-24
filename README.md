# Emby Users Panel (Go)

Go 版 Emby 用户管理面板，统一了后台管理、用户查询、日志、自动任务、备份恢复等能力。

## 1. 项目定位

- 管理端：`http://<host>:8086`
- 用户查询端：`http://<host>:8085/user/user.html`
- 固定端口语义：`8086` 管理，`8085` 查询

## 2. 核心能力

- 多服务器管理：新增、编辑、删除、切换当前服务器
- 用户管理：创建、充值、编辑、启用、禁用、删除、批量操作
- 自动任务：按时间执行到期检查，支持到期禁用或删除
- 通知能力：SMTP 测试邮件、到期提醒、操作通知
- 日志中心：结构化日志浏览、关键字过滤、日志下载
- 备份恢复：本地用户数据备份导出、JSON 恢复导入
- 查询接口：可选 token 校验 + IP 频率限制

## 3. 技术栈与运行要求

- 后端：Go `1.24`
- 前端：原生 JavaScript + HTML + CSS
- 数据：SQLite + JSON
- 容器：Docker / Docker Compose

## 4. 快速启动

### 4.1 Docker Compose（推荐）

```bash
docker compose up -d --build
```

访问地址：

- 管理端：`http://127.0.0.1:8086`
- 查询端：`http://127.0.0.1:8085/user/user.html`

### 4.2 Docker 命令

```bash
docker build -t emby-users-panel:latest .
docker run -d \
  --name embyuserspanel \
  -p 8086:8086 -p 8085:8085 \
  -e TZ=Asia/Shanghai \
  -e APP_DATA_DIR=/data \
  -v $(pwd)/data:/data \
  emby-users-panel:latest
```

### 4.3 本地运行

要求：Go `1.24+`

```bash
go mod tidy
go run main.go
```

注意：本地运行时请在项目根目录执行，否则无法加载 `templates/` 和 `public/`。

## 5. 目录结构

```text
Emby Users Panel/
├─ main.go
├─ go.mod
├─ Dockerfile
├─ docker-compose.yml
├─ templates/
│  ├─ login.html
│  └─ dashboard.html
├─ public/
│  ├─ assets/
│  │  ├─ style.css
│  │  └─ script.js
│  └─ user/
│     └─ user.html
└─ data/                               # 运行期数据（务必定期备份）
   ├─ config.json
   ├─ users/
   │  └─ <server_name_or_default>/
   │     ├─ users.db
   │     ├─ users.json
   │     └─ emby_users_cache_*.json
   ├─ log/
   │  └─ <server_name_or_default>/
   │     ├─ operation_log.txt
   │     └─ operation_log.txt.1..5
   └─ rate_limit/
      └─ *.txt
```

## 6. 环境变量

| 变量名 | 默认值 | 说明 |
|---|---|---|
| `APP_DATA_DIR` | `./data` | 数据目录根路径 |
| `TZ` | `Asia/Shanghai` | 时区，影响日期计算、自动任务与日志时间 |

## 7. 配置文件 `data/config.json`

### 7.1 全局字段（含默认值）

| 字段 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| `servers` | `[]object` | `[]` | Emby 服务器列表（`id/name/url/key`） |
| `currentServerId` | `string` | `""` | 当前服务器 ID |
| `expireField` | `string` | `RemoteClientBitrateLimit` | 兼容字段，保留 |
| `permanentDate` | `string` | `2099-12-31` | 永久日期判定值 |
| `checkTime` | `string` | `00:00` | 自动任务执行时间（`HH:mm`） |
| `logRetentionDays` | `int` | `30` | 日志保留天数 |
| `panelPass` | `string` | `""` | 管理端密码（保存 bcrypt 哈希） |
| `notify_on_operation` | `bool` | `true` | 操作后是否邮件通知 |
| `query_api_fallback` | `bool` | `false` | 查询接口本地无数据时是否回源 Emby API |
| `query_require_token` | `bool` | `false` | 查询接口是否必须 token |
| `query_token` | `string` | `""` | 查询接口 token |
| `logFile` | `string` | `<APP_DATA_DIR>/operation_log.txt` | 日志基准文件（实际按服务器拆分） |
| `hiddenUsers` | `any` | `{}` | 隐藏用户配置（支持 map 或旧数组格式） |
| `server_settings` | `map` | `{}` | 按服务器覆盖设置 |
| `smtp_host` | `string` | `""` | SMTP 主机 |
| `smtp_port` | `int` | `0` | SMTP 端口（0 时按协议推断） |
| `smtp_user` | `string` | `""` | SMTP 用户名 |
| `smtp_pass` | `string` | `""` | SMTP 密码/授权码 |
| `smtp_from` | `string` | `""` | 发件人地址（空则用 `smtp_user`） |
| `smtp_secure` | `string` | `ssl` | `ssl` 或 `tls`（STARTTLS） |
| `notify_before_days` | `int` | `3` | 到期前提醒天数 |
| `expireAction` | `string` | `disable` | 到期动作：`disable`/`delete` |
| `restoreTemplateUser` | `string` | `""` | 恢复时模板用户 |
| `autoTaskEnabled` | `*bool` | `nil` | 全局自动任务开关（`nil` 视为启用） |

### 7.2 `server_settings` 字段

单个服务器可覆盖以下字段（优先级高于全局）：

- `checkTime`
- `autoTaskEnabled`
- `logRetentionDays`
- `expireAction`
- `restoreTemplateUser`
- `defaultTemplateUser`
- `smtp_host`
- `smtp_port`
- `smtp_user`
- `smtp_pass`
- `smtp_from`
- `smtp_secure`
- `notify_before_days`
- `notify_on_operation`

### 7.3 配置优先级

`全局配置` -> `server_settings[当前服务器]` 覆盖 -> `运行时兜底默认值`

### 7.4 `hiddenUsers` 格式说明

推荐格式（按服务器 ID 隐藏）：

```json
{
  "hiddenUsers": {
    "server_id_1": ["userA", "userB"],
    "server_id_2": ["userX"]
  }
}
```

兼容旧格式（数组），会映射到空键 `""`：

```json
{
  "hiddenUsers": ["legacyUser1", "legacyUser2"]
}
```

### 7.5 最小可用示例

```json
{
  "servers": [
    {
      "id": "main",
      "name": "Main Emby",
      "url": "http://127.0.0.1:8096",
      "key": "your_api_key"
    }
  ],
  "currentServerId": "main",
  "checkTime": "00:00",
  "logRetentionDays": 30,
  "query_require_token": false,
  "query_token": "",
  "query_api_fallback": false,
  "notify_on_operation": true,
  "smtp_secure": "ssl",
  "notify_before_days": 3,
  "expireAction": "disable",
  "server_settings": {}
}
```

## 8. 数据模型（核心）

`LocalUser` 主要字段如下：

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | `string` | 用户 ID |
| `name` | `string` | 用户名 |
| `openDate` | `string` | 开通日期（`YYYY-MM-DD`） |
| `lastRecharge` | `string` | 最近充值日期（`YYYY-MM-DD`） |
| `expireDate` | `string` | 到期日期（`YYYY-MM-DD`） |
| `daysLeft` | `string` | 剩余天数（字符串形式，永久为“永久”） |
| `status` | `string` | 状态（启用/禁用/管理员） |
| `group` | `string` | 分组 |
| `email` | `string` | 邮箱 |
| `lastNotifyDate` | `string` | 最后通知标记（日期或 `expired`） |
| `chargeHistory` | `[]ChargeRecord` | 充值历史 |

SQLite 表：`users`（主键 `id`，`charge_history` 以 JSON 存储）。

## 9. HTTP 路由总览

### 9.1 管理端（`:8086`）

| 路由 | 方法 | 说明 |
|---|---|---|
| `/` | `GET` | 登录页或控制台 |
| `/index.php` | `GET/POST` | 兼容入口，行为同 `/` |
| `/?action=backup&token=...` | `GET` | 下载用户备份 JSON |
| `/?action=download_log&file=...&token=...` | `GET` | 下载日志文件 |
| `/assets/*` | `GET` | 前端静态资源 |

### 9.2 查询端（`:8085`）

| 路由 | 方法 | 说明 |
|---|---|---|
| `/` | `GET` | 302 跳转到 `/user/user.html` |
| `/user/user.html` | `GET` | 用户查询页面 |
| `/query.php` | `POST` | 用户查询接口 |
| `/user/query.php` | `POST` | 同上（兼容路径） |
| `/index.php` | `GET` | 返回 `403` |
| `/assets/*` | `GET` | 前端静态资源 |

## 10. 管理端 API 参考（`POST /` 或 `POST /index.php`）

### 10.1 调用约束

- 所有管理端 POST 都要求 `X-CSRF-Token`（包括 `action=login`）
- 登录后动作要求会话已认证（`panelPass` 为空时默认直接放行）
- 请求格式支持：
  - `application/x-www-form-urlencoded`
  - `multipart/form-data`（恢复备份）
- 常见返回格式：`{ success: boolean, message?: string, ... }`

### 10.2 `action` 列表

| `action` | 主要参数 | 说明 |
|---|---|---|
| `login` | `password` | 后台登录 |
| `get_users` | 无 | 读取当前服务器用户列表 |
| `charge` | `charge_uid`, `charge_days`, `charge_note` | 单用户充值 |
| `save_edit` | `uid` + 编辑字段 | 单用户编辑/启用/禁用 |
| `delete` | `uid`, `note` | 删除单用户 |
| `batch` | `type`, `uids`, `note` + 类型字段 | 批量操作 |
| `refresh_cache` | 无 | 清理缓存并刷新 |
| `server_op` | `sub_action` + 子参数 | 服务器管理 |
| `settings_op` | 设置字段集合 | 保存设置（按当前服务器） |
| `test_email` | `test_to`, `smtp_*` | 发送 SMTP 测试邮件 |
| `restore` | `backup_file`(multipart) | 恢复备份 |
| `get_logs` | `file`(可选) | 获取日志文件列表与内容 |
| `run_auto_check` | 无 | 立即执行自动检查 |

### 10.3 关键动作参数

`save_edit` 可用字段：

- `newname`
- `newgroup`
- `newpass`
- `email`
- `opendate`（`YYYY-MM-DD`）
- `lastrecharge`（`YYYY-MM-DD`）
- `expdate`（`YYYY-MM-DD`）
- `expdate_action`（`permanent` / `clear`）
- `disabled`（`0/1`）
- `note`

`batch` 的 `type` 支持：

- `charge`：需要 `days`
- `edit`：可带 `newgroup/opendate/lastrecharge/expdate/expdate_action`
- `enable`
- `disable`
- `delete`

`server_op` 的 `sub_action`：

- `save`：`id,name,url,key`
- `delete`：`id`
- `switch`：`id`

### 10.4 下载型 GET 动作

| 路由 | 参数 | 说明 |
|---|---|---|
| `GET /?action=backup&token=...` | `token` | 下载备份文件（JSON） |
| `GET /?action=download_log&file=...&token=...` | `file`, `token` | 下载完整日志 |
| `GET /?action=download_log&file=...&tail=1&token=...` | `tail=1` | 下载最近 `128KB` |

## 11. 查询接口参考（`POST /query.php`）

请求参数：

- `username`（必填，最大 64 字符）
- `token`（可选；若开启 `query_require_token` 且配置了 `query_token` 则必填）

也可通过请求头传 token：

- `X-Query-Token: <token>`

返回示例（成功）：

```json
{
  "success": true,
  "data": [
    {
      "serverName": "Main Emby",
      "name": "alice",
      "status": "已启用",
      "openDate": "2026-02-01",
      "expireDate": "2026-03-01",
      "daysLeft": "6",
      "history": [
        { "date": "2026-02-10", "days": 30, "note": "续费" }
      ]
    }
  ]
}
```

返回示例（失败）：

```json
{
  "success": false,
  "message": "未找到该用户"
}
```

查询接口逻辑要点：

- 单 IP 限流：`10` 次 / `60` 秒
- 本地存在用户时，会尽量与 Emby 校验一致性
- 若 `query_api_fallback=true`，本地无记录时可回源 Emby API 查询

## 12. 自动任务与到期逻辑

- 调度器每分钟触发一次
- 每个服务器在“当前时间 == `checkTime`（`HH:mm`）”时执行
- `autoTaskEnabled=false` 时不执行
- 永久用户判定：`expireDate` 为空或等于 `2099-12-31`
- 剩余天数按“到期日当天 23:59:59（本地时区）”计算
- `daysLeft == 0` 表示“今天到期”
- 到期处理策略：
  - `disable`：禁用账号
  - `delete`：删除账号
- 管理员账号不会被自动禁用/删除
- 到期提醒邮件按 `notify_before_days` 控制，且同一天只发一次

## 13. 日志系统

- 写入格式：`[YYYY-MM-DD HH:MM:SS] [IP/CLI] message`
- 单文件超过 `10MB` 自动轮转，最多 `5` 个历史文件
- 按 `logRetentionDays` 每日清理轮转文件
- 日志中心接口 `get_logs` 返回最近 `256KB` 内容（用于页面展示）
- 日志下载支持：
  - 完整文件
  - 最近 `128KB`

## 14. 备份与恢复

- 备份下载：`GET ?action=backup&token=...`
- 备份结构：

```json
{
  "users": {
    "<user_id>": { "name": "...", "expireDate": "...", "...": "..." }
  }
}
```

- 恢复上传：`action=restore` + `backup_file`（仅支持 `.json`）
- 恢复后会触发同步逻辑，将恢复数据写回 Emby（按可用能力执行）

## 15. 安全机制

- 会话：
  - Cookie：`HttpOnly` + `SameSite=Lax`
  - `Secure` 在 HTTPS 或 `X-Forwarded-Proto=https` 时启用
  - 会话空闲 `24h` 过期（内存会话）
- 登录防爆破：`10` 分钟内失败 `5` 次触发冷却
- CSRF：管理端 POST 要求 `X-CSRF-Token`
- 安全响应头：
  - `X-Frame-Options: DENY`
  - `X-Content-Type-Options: nosniff`
  - `Referrer-Policy: same-origin`
- 字符串比较使用常量时间比较，降低时序攻击风险

## 16. 开发与构建

```bash
# 依赖整理
go mod tidy

# 本地构建
go build ./...

# 直接运行
go run main.go

# 构建镜像
docker build -t emby-users-panel:latest .
```

`.dockerignore` 默认忽略：`.git`、`.vscode`、`.idea`、`data/`。

## 17. 反向代理注意事项

- 请正确转发以下头部：
  - `X-Forwarded-Proto`
  - `X-Forwarded-For`
- 如果查询接口启用 token 校验，建议仅在 HTTPS 下对外暴露。

## 18. 常见问题排障

### 18.1 `go.mod: unexpected input character '\ufeff'`

`go.mod` 存在 BOM，改为 `UTF-8 (无 BOM)` 即可。

### 18.2 `module ... requires go >= 1.24`

升级 Go 或 Docker 构建镜像版本到 `1.24+`。

### 18.3 `missing go.sum entry ...`

执行：

```bash
go mod tidy
```

然后重新构建。

### 18.4 页面 502 或无法访问

先检查容器和日志：

```bash
docker ps
docker logs -f embyuserspanel
```

再检查反代 upstream 是否正确指向 `8086/8085`。

### 18.5 查询总是提示 token 无效

- 确认 `query_require_token=true`
- 确认 `query_token` 非空且客户端传值一致
- 客户端可通过 `token` 表单字段或 `X-Query-Token` 头传递

### 18.6 “今天到期”显示判定

`daysLeft=0` 即“今天到期”，按本地时区在到期日 23:59:59 前都属于今天到期。

## 19. 运维建议

- 定期备份整个 `data/` 目录
- 生产环境放在反代后并启用 HTTPS
- 改动设置后执行一次“立即检查”验证行为
- 升级前先做 `data/` 快照并保留最近可回滚版本

