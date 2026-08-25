<div align="center">
此项目为测试阶段，暂未发布
# EPanel

轻量、完整的 Emby 用户生命周期管理面板。

单二进制部署 · 多服务器管理 · Docker 支持 · 用户自助查询 · 到期自动化

![Go](https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white)
![SQLite](https://img.shields.io/badge/SQLite-Pure_Go-003B57?logo=sqlite&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?logo=docker&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-green)

</div>

---

## ✨ TL;DR

- 👥 管理 Emby 用户的创建、续费、禁用、删除和批量操作
- 🖥️ 支持多台 Emby 服务器统一管理
- ⏰ 支持用户到期检查、自动禁用或删除
- 📧 支持到期提醒和操作通知邮件
- 📊 支持播放统计、在线会话和媒体库概览
- 🔎 提供独立用户查询页，用户可自助查看到期时间
- 🐳 提供 Docker 镜像，一条命令即可部署

---

## 🔧 核心特性

### 管理面板

- 媒体库统计：电影、剧集、单集等内容概览
- 运营数据概览：用户数、播放次数、播放时长、最近入库、最近播放
- 在线会话监控：当前播放用户、进度、设备和播放内容
- 播放趋势与影视榜单：支持按 7 天、30 天、180 天切换统计范围

### 用户管理

- 创建、编辑、充值、续费、启用、禁用、删除 Emby 用户
- 支持批量充值、批量修改、批量启用、批量禁用、批量删除
- 支持从模板用户复制权限和配置
- 支持用户分组、备注、表格列显示偏好和排序偏好

### 多服务器

- 在同一个面板内管理多台 Emby 服务器
- 每台服务器拥有独立数据库、用户缓存和日志
- 支持按服务器覆盖定时任务、SMTP、到期策略、模板用户等配置

### 自动化任务

- 定时检查用户到期状态
- 可选择自动禁用或删除过期用户
- 支持到期前 N 天邮件提醒
- 支持充值、启用、禁用、删除等操作通知邮件
- 支持向管理员发送服务器状态邮件

### 用户自助查询

- 独立用户查询端口，用户可自行查看账号到期时间
- 可开启访问 Token 验证
- 内置 per-IP 和 per-username 限流

---

## 🐳 部署指南

默认端口：

```text
8086  管理面板
8085  用户查询
```

默认数据目录：

```text
/data
```

### Docker

```bash
docker run -d \
  --name epanel \
  --restart unless-stopped \
  -p 8086:8086 \
  -p 8085:8085 \
  -v ./data:/data \
  -e TZ=Asia/Shanghai \
  -e APP_DATA_DIR=/data \
  freeyua/epanel:latest
```

### Docker Compose

```yaml
services:
  epanel:
    image: freeyua/epanel:latest
    container_name: epanel
    restart: unless-stopped
    ports:
      - "8086:8086"
      - "8085:8085"
    volumes:
      - ./data:/data
    environment:
      - TZ=Asia/Shanghai
      - APP_DATA_DIR=/data
```

### 更新镜像

```bash
docker pull freeyua/epanel:latest
docker restart epanel
```

---

## 🌐 访问地址

管理面板：

```text
http://localhost:8086
```

用户查询：

```text
http://localhost:8085
```

健康检查：

```text
http://localhost:8086/health
```

---

## ⚙️ 首次配置

1. 打开管理面板：`http://<host>:8086`
2. 首次进入时设置管理密码
3. 进入系统设置中的服务器管理，添加 Emby 服务器
4. 填写 Emby 服务器地址，例如 `http://192.168.1.100:8096`
5. 填写 Emby API Key。API Key 可在 Emby 后台的 API 密钥页面生成
6. 根据需要配置模板用户、到期策略、邮件通知和用户查询 Token

---

## 📦 配置与数据

### 环境变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `APP_DATA_DIR` | `/data` | 运行时数据目录，保存配置、数据库、日志和缓存 |
| `TZ` | `Asia/Shanghai` | 时区，影响定时任务和日期显示 |

### 数据目录

运行后会在数据目录中生成以下内容：

```text
data/
├── config.json
├── log/
└── users/
```

建议备份和迁移时保留整个 `data` 目录。

Docker Compose 部署时，该目录对应项目下的 `./data`。

---

## ⚠️ 重要说明

- 本项目需要可用的 Emby 服务器地址和 API Key
- 用户管理操作会直接作用于对应 Emby 服务器
- 自动禁用或删除过期用户前，请先确认到期策略配置
- 建议定期备份 `data` 目录，尤其是 `config.json` 和用户数据库
- 用户查询页如对公网开放，建议开启 Token 验证并配合反向代理访问控制

---

## ❓ 常见问题

### 默认管理密码是什么？

没有默认密码。首次打开管理面板时需要自行设置管理密码。

### 数据保存在哪里？

默认保存在容器内 `/data`，Docker 示例中已挂载到当前目录的 `./data`。

### 可以管理多台 Emby 吗？

可以。面板支持添加多台 Emby 服务器，并为每台服务器保留独立数据和独立配置覆盖。

### 用户查询页可以关闭或加密吗？

用户查询页支持开启 Token 验证。建议公网部署时开启 Token，并通过反向代理进一步限制访问。

### 如何备份和迁移？

保留整个 `data` 目录即可。迁移到新机器时，将原 `data` 目录挂载到新容器的 `/data`。

---

## 📜 免责声明

1. 本项目仅用于个人媒体服务器管理和学习研究。
2. 请遵守 Emby 相关服务条款和当地法律法规。
3. 使用本项目产生的账号、权限、数据变更风险由使用者自行承担。
4. 请在执行批量禁用、删除、自动到期处理前确认配置无误。

---

## License

MIT License. See [LICENSE](LICENSE).
