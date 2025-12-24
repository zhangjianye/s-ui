# S-UI UAP 版本部署指南

> **文档版本**: v1.0
> **最后更新**: 2025-12-23

## 目录

1. [概述](#1-概述)
2. [系统要求](#2-系统要求)
3. [从源码构建](#3-从源码构建)
4. [单节点部署](#4-单节点部署)
5. [多节点部署](#5-多节点部署)
6. [Docker 部署](#6-docker-部署)
7. [UAP 协议配置](#7-uap-协议配置)
8. [配置参考](#8-配置参考)
9. [常见问题](#9-常见问题)

---

## 1. 概述

S-UI UAP 版本是基于 S-UI 的增强版本，包含以下特性：

- **UAP 协议支持**: 基于 VLESS 的自定义协议，支持 `uap://` 链接格式
- **多节点架构**: 支持 Master/Worker 分布式部署
- **增强的用户管理**: 时长限制、流量/时长重置策略
- **API 密钥管理**: 支持外部系统集成
- **Webhook 回调**: 事件通知机制

---

## 2. 系统要求

### 2.1 硬件要求

| 组件 | 最低配置 | 推荐配置 |
|------|----------|----------|
| CPU | 1 核 | 2+ 核 |
| 内存 | 512 MB | 1+ GB |
| 磁盘 | 1 GB | 10+ GB |

### 2.2 软件要求

| 软件 | 版本要求 |
|------|----------|
| 操作系统 | Linux (amd64/arm64), Windows, macOS |
| Go | 1.22+ (仅构建时需要) |
| Node.js | 18+ (仅构建时需要) |
| Docker | 20.10+ (可选) |

### 2.3 网络要求

| 端口 | 用途 | 说明 |
|------|------|------|
| 2095 | 管理面板 | 可配置 |
| 2096 | 订阅服务 | 可配置 |
| 443/80 | 代理服务 | 根据配置 |

---

## 3. 从源码构建

### 3.1 克隆仓库

```bash
git clone https://github.com/alireza0/s-ui.git
cd s-ui
```

### 3.2 构建前端

```bash
cd frontend
npm install
npm run build
cd ..
```

### 3.3 复制前端文件

```bash
mkdir -p web/html
rm -rf web/html/*
cp -r frontend/dist/* web/html/
```

### 3.4 构建后端

**重要**: 必须使用以下 build tags：

```bash
go build -tags "with_quic,with_utls,with_gvisor,with_wireguard" -o s-ui ./main.go
```

**Build Tags 说明**：

| Tag | 说明 |
|-----|------|
| `with_quic` | QUIC 协议支持 (Hysteria, TUIC) |
| `with_utls` | uTLS 指纹伪装 |
| `with_gvisor` | gVisor 网络栈 (Tun 模式) |
| `with_wireguard` | WireGuard 支持 |

### 3.5 一键构建脚本

```bash
#!/bin/bash
# build.sh

set -e

echo "Building frontend..."
cd frontend
npm install
npm run build
cd ..

echo "Copying frontend files..."
mkdir -p web/html
rm -rf web/html/*
cp -r frontend/dist/* web/html/

echo "Building backend..."
go build -tags "with_quic,with_utls,with_gvisor,with_wireguard" -o s-ui ./main.go

echo "Build complete: ./s-ui"
```

---

## 4. 单节点部署

### 4.1 使用 Systemd 服务

**步骤 1**: 创建目录并复制文件

```bash
sudo mkdir -p /usr/local/s-ui
sudo cp s-ui /usr/local/s-ui/
sudo chmod +x /usr/local/s-ui/s-ui
```

**步骤 2**: 创建 systemd 服务文件

```bash
sudo cat > /etc/systemd/system/s-ui.service << 'EOF'
[Unit]
Description=S-UI Service
After=network.target
Wants=network.target

[Service]
Type=simple
WorkingDirectory=/usr/local/s-ui/
ExecStart=/usr/local/s-ui/s-ui
Restart=on-failure
RestartSec=10s

[Install]
WantedBy=multi-user.target
EOF
```

**步骤 3**: 启动服务

```bash
sudo systemctl daemon-reload
sudo systemctl enable s-ui --now
```

**步骤 4**: 检查状态

```bash
sudo systemctl status s-ui
```

### 4.2 默认访问信息

| 项目 | 值 |
|------|-----|
| 面板地址 | http://IP:2095/app/ |
| 订阅地址 | http://IP:2096/sub/ |
| 用户名 | admin |
| 密码 | admin |

**首次登录后请立即修改密码！**

---

## 5. 多节点部署

### 5.1 架构概述

```
                    ┌─────────────────┐
                    │  Master Node    │
                    │  (主节点)        │
                    │  - 完整 WebUI   │
                    │  - 配置管理     │
                    │  - 数据汇总     │
                    └────────┬────────┘
                             │
            ┌────────────────┼────────────────┐
            │                │                │
            ▼                ▼                ▼
    ┌───────────────┐ ┌───────────────┐ ┌───────────────┐
    │  Worker Node  │ │  Worker Node  │ │  Worker Node  │
    │  (从节点 1)    │ │  (从节点 2)    │ │  (从节点 N)    │
    │  - 只读 WebUI │ │  - 只读 WebUI │ │  - 只读 WebUI │
    │  - 代理服务   │ │  - 代理服务   │ │  - 代理服务   │
    └───────────────┘ └───────────────┘ └───────────────┘
```

### 5.2 部署主节点 (Master)

主节点使用默认配置启动，无需额外参数：

```bash
# 以 Master 模式运行 (默认)
/usr/local/s-ui/s-ui
```

或者显式指定：

```bash
/usr/local/s-ui/s-ui --mode master
```

**主节点 systemd 服务**：

```ini
[Unit]
Description=S-UI Master Node
After=network.target

[Service]
Type=simple
WorkingDirectory=/usr/local/s-ui/
ExecStart=/usr/local/s-ui/s-ui --mode master
Restart=on-failure
RestartSec=10s

[Install]
WantedBy=multi-user.target
```

### 5.3 生成邀请码

1. 登录主节点 WebUI
2. 进入 **设置 → 节点管理 → 邀请码**
3. 点击 **生成邀请码**
4. 设置名称和有效期
5. 复制生成的 Token

### 5.4 部署从节点 (Worker)

**步骤 1**: 在从节点服务器上安装 S-UI

```bash
sudo mkdir -p /usr/local/s-ui
sudo cp s-ui /usr/local/s-ui/
sudo chmod +x /usr/local/s-ui/s-ui
```

**步骤 2**: 创建从节点 systemd 服务

有两种配置方式：

**方式 A: 使用环境变量（推荐）**

创建环境配置文件：

```bash
sudo cat > /usr/local/s-ui/worker.env << 'EOF'
SUI_NODE_MODE=worker
SUI_MASTER_ADDR=https://master.example.com:2095
SUI_NODE_TOKEN=YOUR_INVITE_TOKEN
SUI_NODE_ID=worker-hk-01
SUI_NODE_NAME=香港节点01
SUI_EXTERNAL_HOST=hk.example.com
SUI_EXTERNAL_PORT=443
EOF

# 保护配置文件（包含敏感 token）
sudo chmod 600 /usr/local/s-ui/worker.env
```

创建 systemd 服务：

```bash
sudo cat > /etc/systemd/system/s-ui.service << 'EOF'
[Unit]
Description=S-UI Worker Node
After=network.target
Wants=network.target

[Service]
Type=simple
WorkingDirectory=/usr/local/s-ui/
EnvironmentFile=/usr/local/s-ui/worker.env
ExecStart=/usr/local/s-ui/s-ui
Restart=on-failure
RestartSec=10s
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF
```

**方式 B: 使用命令行参数**

```bash
sudo cat > /etc/systemd/system/s-ui.service << 'EOF'
[Unit]
Description=S-UI Worker Node
After=network.target
Wants=network.target

[Service]
Type=simple
WorkingDirectory=/usr/local/s-ui/
ExecStart=/usr/local/s-ui/s-ui \
  --mode worker \
  --master "https://master.example.com:2095" \
  --token "YOUR_INVITE_TOKEN" \
  --node-id "worker-hk-01" \
  --node-name "香港节点01" \
  --external-host "hk.example.com" \
  --external-port 443
Restart=on-failure
RestartSec=10s
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF
```

**步骤 3**: 启动服务

```bash
sudo systemctl daemon-reload
sudo systemctl enable s-ui --now

# 检查状态
sudo systemctl status s-ui

# 查看日志
journalctl -u s-ui -f
```

### 5.5 从节点配置参数

| 参数 | 说明 | 示例 |
|------|------|------|
| `--mode` | 运行模式 | `worker` |
| `--master` | 主节点地址 | `https://master.example.com:2095` |
| `--master-path` | API 路径前缀（默认 /app） | `/app` |
| `--token` | 邀请码 | `abc123...` |
| `--node-id` | 节点唯一标识 | `node-hk-01` |
| `--node-name` | 节点显示名称 | `香港节点 01` |
| `--external-host` | 外部连接地址 | `hk.example.com` |
| `--external-port` | 外部连接端口 | `443` |

### 5.6 节点信息配置

从节点可配置以下元数据（用于在主节点显示）：

| 字段 | 说明 |
|------|------|
| 名称 | 节点显示名称 |
| 国家/城市 | 地理位置信息 |
| 外部地址 | 用户连接地址 |
| 标志 | 是否为高级节点 |

---

## 6. Docker 部署

### 6.1 构建镜像

```bash
# 克隆仓库
git clone https://github.com/alireza0/s-ui.git
cd s-ui

# 构建镜像
docker build -t s-ui-uap:latest .
```

### 6.2 单节点 Docker Compose

```yaml
# docker-compose.yml
version: '3.8'

services:
  s-ui:
    image: s-ui-uap:latest
    container_name: s-ui
    restart: unless-stopped
    ports:
      - "2095:2095"  # 管理面板
      - "2096:2096"  # 订阅服务
      - "443:443"    # HTTPS 代理
      - "80:80"      # HTTP
    volumes:
      - ./db:/app/db
      - ./cert:/app/cert
    environment:
      - TZ=Asia/Shanghai
```

启动：

```bash
docker compose up -d
```

### 6.3 多节点 Docker Compose

**主节点 (master-compose.yml)**:

```yaml
version: '3.8'

services:
  s-ui-master:
    image: s-ui-uap:latest
    container_name: s-ui-master
    restart: unless-stopped
    command: ["./s-ui", "--mode", "master"]
    ports:
      - "2095:2095"
      - "2096:2096"
    volumes:
      - ./db:/app/db
      - ./cert:/app/cert
    environment:
      - TZ=Asia/Shanghai
```

**从节点 (worker-compose.yml)**:

```yaml
version: '3.8'

services:
  s-ui-worker:
    image: s-ui-uap:latest
    container_name: s-ui-worker
    restart: unless-stopped
    command: >
      ./s-ui --mode worker
      --master https://master.example.com:2095
      --token YOUR_INVITE_TOKEN
      --node-id worker-01
    ports:
      - "2095:2095"
      - "443:443"
    volumes:
      - ./db:/app/db
      - ./cert:/app/cert
    environment:
      - TZ=Asia/Shanghai
```

---

## 7. UAP 协议配置

### 7.1 创建 UAP 入站

1. 登录管理面板
2. 进入 **入站管理**
3. 点击 **添加**
4. 选择类型: **uap**
5. 配置参数：

| 参数 | 说明 | 示例 |
|------|------|------|
| Tag | 入站标签 | `uap-443` |
| 监听地址 | 绑定 IP | `::` (所有地址) |
| 监听端口 | 绑定端口 | `443` |
| TLS | 必须启用 | 选择 TLS 配置 |
| 传输层 | 可选 | ws, grpc, http 等 |

### 7.2 UAP 链接格式

```
uap://<uuid>@<host>:<port>?security=tls&sni=<domain>&fp=<fingerprint>&flow=<flow>#<remark>
```

**参数说明**：

| 参数 | 说明 |
|------|------|
| `uuid` | 用户 UUID |
| `host` | 服务器地址 |
| `port` | 服务器端口 |
| `security` | 安全类型 (tls/reality) |
| `sni` | TLS SNI |
| `fp` | TLS 指纹 |
| `flow` | 流控 (xtls-rprx-vision) |

### 7.3 用户 UAP 配置

在客户端配置中，UAP 协议需要以下字段：

| 字段 | 说明 |
|------|------|
| UUID | 用户唯一标识 |
| Flow | 流控设置 (可选) |

这些配置可在 **用户管理 → 配置** 标签页中编辑。

### 7.4 UAP 扩展字段

UAP 版本支持以下用户扩展字段：

| 字段 | 说明 | 默认值 |
|------|------|--------|
| 时长限制 | 最大在线时长（小时） | 0 (无限) |
| 流量重置 | 流量重置策略 | 不重置 |
| 时长重置 | 时长重置策略 | 不重置 |

**重置策略选项**：
- 不重置
- 每日
- 每周
- 每月
- 每年

---

## 8. 配置参考

### 8.1 环境变量

**通用配置：**

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `SUI_LOG_LEVEL` | 日志级别 | `info` |
| `SUI_DEBUG` | 调试模式 | `false` |
| `SUI_BIN_FOLDER` | Sing-Box 目录 | `bin` |
| `SUI_DB_FOLDER` | 数据库目录 | `db` |
| `SINGBOX_API` | Sing-Box API 地址 | - |

**节点配置（Worker 模式）：**

> **注意**: 环境变量优先级高于命令行参数。如果同时设置，环境变量的值会覆盖命令行参数。

| 变量 | 说明 | 示例 |
|------|------|------|
| `SUI_NODE_MODE` | 节点模式 | `worker` |
| `SUI_MASTER_ADDR` | 主节点地址 | `https://master.example.com:2095` |
| `SUI_MASTER_PATH` | API 路径前缀 | `/app` (默认) |
| `SUI_NODE_TOKEN` | 认证令牌 | `abc123...` |
| `SUI_NODE_ID` | 节点唯一标识 | `worker-hk-01` |
| `SUI_NODE_NAME` | 节点显示名称 | `香港节点01` |
| `SUI_EXTERNAL_HOST` | 外部连接地址 | `hk.example.com` |
| `SUI_EXTERNAL_PORT` | 外部连接端口 | `443` |
| `SUI_SYNC_CONFIG_INTERVAL` | 配置同步间隔（秒） | `60` (默认) |
| `SUI_SYNC_STATS_INTERVAL` | 统计上报间隔（秒） | `30` (默认) |

### 8.2 命令行参数

```bash
s-ui [flags]

Flags:
  -v                    显示版本
  --mode string         运行模式 (standalone/master/worker，默认 standalone)
  --master string       主节点地址 (worker 模式必填)
  --master-path string  主节点 API 路径前缀 (默认 /app)
  --token string        认证令牌 (worker 模式必填)
  --node-id string      节点唯一标识 (worker 模式必填)
  --node-name string    节点显示名称 (默认使用 node-id)
  --external-host string  外部连接地址
  --external-port int   外部连接端口 (0 = 使用入站端口)
```

### 8.3 目录结构

```
/usr/local/s-ui/
├── s-ui              # 主程序
├── bin/
│   └── sing-box      # Sing-Box 核心
├── db/
│   └── s-ui.db       # SQLite 数据库
├── cert/             # 证书目录
└── web/
    └── html/         # 前端文件
```

---

## 9. 常见问题

### 9.1 构建失败

**问题**: `undefined: tun.DefaultNIC`

**解决**: 确保使用正确的 build tags：

```bash
go build -tags "with_quic,with_utls,with_gvisor,with_wireguard" -o s-ui ./main.go
```

### 9.2 从节点无法连接主节点

**检查项**：
1. 主节点防火墙是否放行 2095 端口
2. 邀请码是否过期
3. 网络连通性测试：`curl https://master:2095/api/node/ping`

### 9.3 UAP 链接无法导入

**检查项**：
1. 确认客户端支持 UAP 协议
2. 检查链接格式是否正确
3. 确认 TLS 配置已正确设置

### 9.4 订阅更新失败

**检查项**：
1. 订阅端口 (2096) 是否放行
2. 客户端订阅地址是否正确
3. 检查日志：`journalctl -u s-ui -f`

### 9.5 日志查看

```bash
# Systemd 日志
journalctl -u s-ui -f

# Docker 日志
docker logs -f s-ui

# 调试模式
SUI_DEBUG=true SUI_LOG_LEVEL=debug /usr/local/s-ui/s-ui
```

---

## 更新日志

| 版本 | 日期 | 说明 |
|------|------|------|
| v1.0 | 2025-12-23 | 初始版本 |
