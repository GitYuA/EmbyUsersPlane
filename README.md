# Emby Users Panel

Emby 用户管理面板 (Go 实现)，专为 Emby 媒体服务器设计的轻量级用户管理系统。集成了多服务器管理、用户状态监控、自动过期处理、充值续费、SMTP 通知以及日志审计等功能。

## 🌟 核心特性

- **多服务器支持**：在统一面板管理多个 Emby 服务器实例，支持一键切换。
- **用户生命周期管理**：
  - 创建、编辑、删除 Emby 用户。
  - 支持充值/续费操作，自动计算过期时间。
  - 一键启用/禁用用户。
  - 批量操作支持。
- **自动化任务**：
  - 定时检查用户过期状态。
  - 支持过期自动禁用或自动删除策略。
  - 自定义保留期与通知时间。
- **通知系统**：
  - SMTP 邮件通知集成。
  - 支持到期提醒、操作通知。
- **数据安全与审计**：
  - 完整的操作日志记录与浏览。
  - 结构化日志支持关键字过滤。
  - 用户数据备份（导出 JSON）与恢复。
- **独立查询门户**：
  - 提供无需登录的公开查询页面，供用户自助查询账号状态。
  - 支持 IP 频率限制与 Token 校验保护。

## 🚀 快速开始

### 方式一：Docker Compose（推荐）

1. 创建 `docker-compose.yml` 文件：

```yaml
version: '3'
services:
  emby-users-panel:
    image: emby-users-panel:latest
    build: .
    container_name: embyuserspanel
    restart: always
    ports:
      - "8086:8086" # 管理面板端口
      - "8085:8085" # 用户查询端口
    volumes:
      - ./data:/data # 数据持久化目录
    environment:
      - TZ=Asia/Shanghai
      - APP_DATA_DIR=/data
```

2. 启动服务：

```bash
docker compose up -d
```

### 方式二：Docker CLI

```bash
docker build -t emby-users-panel:latest .
docker run -d \
  --name embyuserspanel \
  -p 8086:8086 -p 8085:8085 \
  -v $(pwd)/data:/data \
  -e TZ=Asia/Shanghai \
  -e APP_DATA_DIR=/data \
  emby-users-panel:latest
```

### 方式三：本地编译运行

**环境要求**：Go 1.24+

1. 克隆项目并进入目录。
2. 安装依赖并运行：

```bash
go mod tidy
go run main.go
```

> **注意**：本地运行时请确保在项目根目录下执行命令，否则程序可能无法正确加载 `templates/` 和 `public/` 静态资源。

## 📖 使用指南

### 访问地址

- **管理后台**：`http://localhost:8086`
  - 首次访问可能需要配置面板密码（如果在 `config.json` 中配置）。
- **用户自助查询**：`http://localhost:8085/user/user.html`

### 端口说明

| 端口 | 用途 | 说明 |
|---|---|---|
| `8086` | 管理端 | 管理员使用的后台，包含所有管理功能。 |
| `8085` | 查询端 | 对外开放，仅提供用户状态查询接口。 |

### 环境变量

| 变量名 | 默认值 | 说明 |
|---|---|---|
| `TZ` | `Asia/Shanghai` | 容器时区设置，影响日志和计划任务时间。 |
| `APP_DATA_DIR` | `/data` | 配置文件、数据库和日志的存储路径。 |

## 📂 目录结构

```text
Emby Users Panel/
├── main.go                # 程序入口
├── go.mod                 # Go 依赖定义
├── Dockerfile             # Docker 构建文件
├── docker-compose.yml     # Docker Compose 编排文件
├── templates/             # 后端渲染模板 (HTML)
│   ├── login.html
│   └── dashboard.html
├── public/                # 静态资源文件
│   ├── assets/            # JS/CSS 资源
│   └── user/              # 用户查询页面
└── data/                  # [持久化] 运行数据目录
    ├── config.json        # 主配置文件
    ├── users/             # 用户数据库 (SQLite + JSON)
    ├── log/               # 操作日志
    └── rate_limit/        # 频率限制缓存
```

## 🛠️ 配置说明

大部分配置可以通过管理后台的 **"设置"** 页面进行修改，包括：
- **服务器配置**：Emby 地址、API Key。
- **过期策略**：过期动作（禁用/删除）、从何处读取过期时间。
- **邮件通知**：SMTP 服务器设置、发件人信息。
- **面板安全**：修改面板访问密码。

---
*Powered by Golang & SQLite*


## 🔧 进阶运维

### 反向代理配置

建议使用 Nginx 或 Caddy 进行反向代理，并开启 HTTPS。
请确保转发以下 Header 以获取正确的客户端 IP 和协议状态：

- `X-Forwarded-For`
- `X-Forwarded-Proto`

### 数据备份

建议定期备份 `data/` 目录。
该目录包含了：
- `config.json`: 面板配置
- `users/`: 用户数据库
- `log/`: 操作日志

### 故障排查

**Q: 启动报错 `go.mod: unexpected input character '\ufeff'`**
A: `go.mod` 文件编码若包含 BOM，请转换为 `UTF-8 (无 BOM)`。

**Q: 提示 `module ... requires go >= 1.24`**
A: 本项目使用了 Go 1.24 新特性，请升级 Go 环境或使用 Docker 部署。

**Q: 页面无法访问 / 502**
A: 检查容器是否运行：`docker logs embyuserspanel`。确认防火墙放行了 8086/8085 端口。

**Q: 查询接口提示 Token 无效**
A: 若并在设置中开启了 `query_require_token`，请确保查询请求携带了正确的 `token` 参数或 `X-Query-Token` 请求头。


