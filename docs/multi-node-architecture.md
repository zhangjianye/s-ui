# S-UI 多节点管理架构技术方案

## 1. 概述

### 1.1 背景

S-UI 是一个基于 Sing-Box 的代理服务器管理面板，目前仅支持单节点管理。随着业务规模扩大，需要支持一个主节点管理多个从节点的分布式架构。

### 1.2 目标

- 同一程序通过启动参数区分主节点 (Master) 和从节点 (Worker)
- 从节点 WebUI 提供只读访问
- 从节点主动定时从主节点拉取配置
- 从节点定时向主节点上报统计数据
- 从节点离线时使用本地缓存继续运行

### 1.3 设计原则

**UAP-Aware 设计**: 在实现多节点架构时，将 UAP Backend 需要的基础设施（字段、追踪机制）一并做进去，后续实现 UAP API 时只需"包装暴露"，无需"重构基础"。

这意味着:
- 数据模型一次性扩展到位 (Client 扩展、ClientOnline 扩展)
- 核心追踪机制提前实现 (连接追踪、时长追踪)
- 定时任务框架预先搭建 (时长累计、重置策略)
- Webhook 回调机制预留

### 1.4 非目标

- 主节点高可用（本期不涉及）
- 从节点自动发现（需手动注册）

---

## 2. 架构设计

### 2.1 整体架构

```
                         ┌─────────────────────────────────┐
                         │         主节点 (Master)          │
                         │                                  │
                         │  ┌──────────┐  ┌──────────────┐ │
                         │  │  Web UI  │  │ Node Manager │ │
                         │  │ (完整功能) │  │  (节点管理)   │ │
                         │  └──────────┘  └──────────────┘ │
                         │         │              │        │
                         │  ┌──────┴──────────────┴─────┐  │
                         │  │     SQLite (数据源)        │  │
                         │  └───────────────────────────┘  │
                         │         │                       │
                         │  ┌──────┴──────┐                │
                         │  │  Node API   │ ◄─── 从节点调用 │
                         │  └─────────────┘                │
                         └─────────────┬───────────────────┘
                                       │
              ┌────────────────────────┼────────────────────────┐
              │                        │                        │
              ▼                        ▼                        ▼
┌─────────────────────┐  ┌─────────────────────┐  ┌─────────────────────┐
│   从节点 1 (Worker)  │  │   从节点 2 (Worker)  │  │   从节点 N (Worker)  │
│                     │  │                     │  │                     │
│  ┌───────────────┐  │  │  ┌───────────────┐  │  │  ┌───────────────┐  │
│  │   Web UI      │  │  │  │   Web UI      │  │  │  │   Web UI      │  │
│  │  (只读模式)    │  │  │  │  (只读模式)    │  │  │  │  (只读模式)    │  │
│  └───────────────┘  │  │  └───────────────┘  │  │  └───────────────┘  │
│  ┌───────────────┐  │  │  ┌───────────────┐  │  │  ┌───────────────┐  │
│  │  Sync Service │  │  │  │  Sync Service │  │  │  │  Sync Service │  │
│  │  - 拉取配置   │  │  │  │  - 拉取配置   │  │  │  │  - 拉取配置   │  │
│  │  - 上报统计   │  │  │  │  - 上报统计   │  │  │  │  - 上报统计   │  │
│  └───────────────┘  │  │  └───────────────┘  │  │  └───────────────┘  │
│  ┌───────────────┐  │  │  ┌───────────────┐  │  │  ┌───────────────┐  │
│  │   Sing-Box    │  │  │  │   Sing-Box    │  │  │  │   Sing-Box    │  │
│  └───────────────┘  │  │  └───────────────┘  │  │  └───────────────┘  │
│  ┌───────────────┐  │  │  ┌───────────────┐  │  │  ┌───────────────┐  │
│  │ SQLite (缓存) │  │  │  │ SQLite (缓存) │  │  │  │ SQLite (缓存) │  │
│  └───────────────┘  │  │  └───────────────┘  │  │  └───────────────┘  │
└─────────────────────┘  └─────────────────────┘  └─────────────────────┘
```

### 2.2 数据流向

| 类型 | 数据 | 流向 | 说明 |
|------|------|------|------|
| **分发** | Inbounds | 主→从 | 入站监听配置 |
| **分发** | Outbounds | 主→从 | 出站代理配置 |
| **分发** | Clients | 主→从 | 客户端账号配置 |
| **分发** | TLS | 主→从 | TLS 证书配置 |
| **分发** | Services | 主→从 | 服务配置 |
| **分发** | Endpoints | 主→从 | 端点配置 |
| **分发** | Routing | 主→从 | 路由规则 |
| **聚合** | 流量统计 | 从→主 | 客户端 up/down 流量 |
| **聚合** | 在线状态 | 从→主 | 客户端在线位置 |
| **聚合** | 节点状态 | 从→主 | CPU/内存/连接数 |
| **本地** | 端口设置 | - | 每节点独立 |
| **本地** | 管理员账号 | - | 每节点独立 |

### 2.3 运行模式

| 模式 | 说明 | 启动参数 |
|------|------|---------|
| `standalone` | 单机模式（默认，兼容现有） | 无 |
| `master` | 主节点，管理从节点 | `--mode=master` |
| `worker` | 从节点，受主节点管理 | `--mode=worker --master=<addr> --token=<token>` |

---

## 3. 数据库设计

### 3.1 新增表

#### NodeToken 表（节点邀请码）

```sql
CREATE TABLE node_tokens (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    token      TEXT UNIQUE NOT NULL,     -- 邀请码
    name       TEXT,                     -- 预设节点名称 (可选)
    used       BOOLEAN DEFAULT FALSE,    -- 是否已使用
    used_by    TEXT,                     -- 使用的节点 ID
    expires_at INTEGER,                  -- 过期时间 (可选, 0=永不过期)
    created_at INTEGER
);
```

#### Node 表（节点注册信息）

```sql
CREATE TABLE nodes (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id       TEXT UNIQUE NOT NULL,     -- 节点唯一标识 (从节点注册时提供)
    name          TEXT NOT NULL,            -- 节点名称
    address       TEXT,                     -- 节点内部地址 (从节点注册时上报)
    external_host TEXT,                     -- 节点外部地址 (客户端连接用，用于生成订阅)
    external_port INTEGER DEFAULT 0,        -- 节点外部端口 (0 表示与 Inbound 端口相同)
    token         TEXT NOT NULL,            -- 认证 token (来自 node_tokens)
    enable        BOOLEAN DEFAULT TRUE,     -- 是否启用
    status        TEXT DEFAULT 'offline',   -- online/offline/error
    last_seen     INTEGER,                  -- 最后心跳时间戳
    last_sync     INTEGER,                  -- 最后同步时间戳
    version       TEXT,                     -- 节点版本
    system_info   TEXT,                     -- JSON: CPU/内存等
    created_at    INTEGER,
    updated_at    INTEGER,
    -- UAP API 所需字段
    country       TEXT,                     -- 国家
    city          TEXT,                     -- 城市
    flag          TEXT,                     -- 国家代码 (ISO 3166-1 alpha-2)
    is_premium    BOOLEAN DEFAULT FALSE,    -- 是否仅会员可用
    latency       INTEGER DEFAULT 0         -- 平均延迟 (ms)
);
```

**字段说明：**

| 字段 | 用途 | 来源 | 示例 |
|------|------|------|------|
| `node_id` | 节点唯一标识 | 从节点启动参数 | `node-us-west-1` |
| `name` | 节点名称 | 从节点启动参数 | `US West` |
| `address` | 主节点连接从节点的地址 | 从节点注册时上报 | `192.168.1.100:2095` |
| `external_host` | 客户端连接代理的地址 | 从节点启动参数 | `us.example.com` |
| `external_port` | 客户端连接的端口 | 从节点启动参数 | `8443` 或 `0` |
| `token` | 认证 token | 从 node_tokens 表 | `abc123...` |

#### NodeStats 表（节点统计快照）

```sql
CREATE TABLE node_stats (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id     TEXT NOT NULL,
    date_time   INTEGER NOT NULL,
    cpu         REAL,
    memory      REAL,
    connections INTEGER,
    upload      INTEGER,
    download    INTEGER,
    INDEX idx_node_id (node_id),
    INDEX idx_date_time (date_time)
);
```

#### ClientOnline 表（客户端在线状态）

```sql
CREATE TABLE client_onlines (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    client_name  TEXT NOT NULL,
    node_id      TEXT NOT NULL,
    inbound_tag  TEXT,
    source_ip    TEXT,                -- UAP: 设备 IP (用于设备数限制)
    connected_at INTEGER,             -- UAP: 连接时间 (用于时长计算)
    last_seen    INTEGER NOT NULL,
    INDEX idx_client_name (client_name),
    INDEX idx_node_id (node_id)
);
```

#### WebhookConfig 表（UAP 回调配置）

```sql
CREATE TABLE webhook_configs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    callback_url    TEXT NOT NULL,           -- UAP Backend URL
    callback_secret TEXT,                    -- 认证密钥
    enable          BOOLEAN DEFAULT TRUE
);
```

#### ApiKey 表（UAP API 认证）

```sql
CREATE TABLE api_keys (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    key        TEXT UNIQUE NOT NULL,
    name       TEXT,
    enable     BOOLEAN DEFAULT TRUE,
    created_at INTEGER
);
```

### 3.2 修改表

#### Clients 表 (UAP-Aware 扩展)

```sql
ALTER TABLE clients ADD COLUMN uuid TEXT UNIQUE;                    -- UAP 用户标识
ALTER TABLE clients ADD COLUMN is_premium BOOLEAN DEFAULT FALSE;    -- 是否会员
ALTER TABLE clients ADD COLUMN time_limit INTEGER DEFAULT 0;        -- 时长限制 (秒), 0=无限
ALTER TABLE clients ADD COLUMN time_used INTEGER DEFAULT 0;         -- 已用时长 (秒)
ALTER TABLE clients ADD COLUMN time_reset_strategy TEXT DEFAULT 'no_reset';   -- daily/weekly/monthly/no_reset
ALTER TABLE clients ADD COLUMN time_reset_at INTEGER DEFAULT 0;     -- 上次时长重置时间
ALTER TABLE clients ADD COLUMN traffic_reset_strategy TEXT DEFAULT 'no_reset'; -- daily/weekly/monthly/no_reset
ALTER TABLE clients ADD COLUMN traffic_reset_at INTEGER DEFAULT 0;  -- 上次流量重置时间
ALTER TABLE clients ADD COLUMN speed_limit INTEGER DEFAULT 0;       -- 带宽限制 (Mbps), 0=无限
ALTER TABLE clients ADD COLUMN device_limit INTEGER DEFAULT 0;      -- 设备数限制, 0=无限

CREATE INDEX idx_clients_uuid ON clients(uuid);
```

#### Stats 表

新增 `node_id` 字段：

```sql
ALTER TABLE stats ADD COLUMN node_id TEXT DEFAULT 'local';
CREATE INDEX idx_stats_node_id ON stats(node_id);
```

---

## 4. 接口设计

### 4.1 节点 API（主节点提供，从节点调用）

**基础路径**: `/node/`

**认证方式**: Header（注册接口除外）

```
X-Node-Id: <node_id>
X-Node-Token: <token>
```

#### 4.1.1 节点注册（无需认证 Header，使用 Token 参数）

```
POST /node/register

Request:
{
    "token": "invite-token-xxx",       // 邀请码 (从 node_tokens 表)
    "nodeId": "node-us-west-1",        // 节点唯一标识
    "name": "US West",                 // 节点名称
    "address": "192.168.1.100:2095",   // 节点内部地址 (自动获取)
    "externalHost": "us.example.com",  // 外部地址 (客户端连接用)
    "externalPort": 0,                 // 外部端口 (0=与 Inbound 相同)
    "version": "1.3.7"                 // 节点版本
}

Response (成功):
{
    "success": true,
    "msg": "registered",
    "obj": {
        "nodeId": "node-us-west-1",
        "token": "invite-token-xxx"    // 后续请求使用此 token
    }
}

Response (失败 - Token 无效):
{
    "success": false,
    "msg": "invalid or expired token"
}

Response (失败 - Node ID 已存在):
{
    "success": false,
    "msg": "node id already registered"
}
```

**注册流程：**
1. 验证 token 有效性（存在、未使用、未过期）
2. 检查 node_id 是否已被注册
3. 创建 Node 记录
4. 标记 token 为已使用
5. 返回成功，从节点保存 token 用于后续认证

#### 4.1.2 获取配置版本（需认证）

```
GET /node/config/version

Response:
{
    "success": true,
    "version": 1702900000  // Unix 时间戳
}
```

#### 4.1.3 获取完整配置（需认证）

```
GET /node/config

Response:
{
    "success": true,
    "obj": {
        "version": 1702900000,
        "inbounds": [...],
        "outbounds": [...],
        "clients": [...],
        "tls": [...],
        "services": [...],
        "endpoints": [...],
        "config": {...}  // 路由、DNS 等
    }
}
```

#### 4.1.4 上报流量统计（需认证）

```
POST /node/stats

Request:
[
    {
        "dateTime": 1702900000,
        "resource": "user",
        "tag": "client_name",
        "direction": true,    // true=上行, false=下行
        "traffic": 1024000
    },
    ...
]

Response:
{
    "success": true
}
```

#### 4.1.5 上报在线状态（需认证）

```
POST /node/onlines

Request:
{
    "onlines": [
        {
            "user": "user1",
            "inboundTag": "vmess-in",
            "sourceIP": "192.168.1.100",      // UAP: 设备 IP
            "connectedAt": 1702900000          // UAP: 连接时间戳
        },
        {
            "user": "user2",
            "inboundTag": "vless-in",
            "sourceIP": "192.168.1.101",
            "connectedAt": 1702899000
        }
    ]
}

Response:
{
    "success": true
}
```

#### 4.1.6 心跳（需认证）

```
POST /node/heartbeat

Request:
{
    "cpu": 25.5,
    "memory": 60.2,
    "connections": 150,
    "version": "1.3.7"
}

Response:
{
    "success": true,
    "time": 1702900000
}
```

### 4.2 管理 API（WebUI 调用）

#### 4.2.1 生成节点邀请码

```
POST /api/save

Request:
{
    "object": "nodeTokens",
    "action": "new",
    "data": {
        "name": "US West Node",        // 预设名称 (可选)
        "expiresAt": 1703000000        // 过期时间 (可选, 0=永不过期)
    }
}

Response:
{
    "success": true,
    "obj": {
        "token": "abc123def456...",    // 生成的邀请码
        "expiresAt": 1703000000
    }
}
```

#### 4.2.2 获取邀请码列表

```
GET /api/nodeTokens

Response:
{
    "success": true,
    "obj": [
        {
            "id": 1,
            "token": "abc123...",
            "name": "US West Node",
            "used": false,
            "usedBy": null,
            "expiresAt": 1703000000,
            "createdAt": 1702900000
        },
        {
            "id": 2,
            "token": "def456...",
            "name": null,
            "used": true,
            "usedBy": "node-us-west-1",
            "expiresAt": 0,
            "createdAt": 1702800000
        }
    ]
}
```

#### 4.2.3 删除邀请码

```
POST /api/save

Request:
{
    "object": "nodeTokens",
    "action": "del",
    "data": {
        "id": 1
    }
}
```

#### 4.2.4 获取节点列表

```
GET /api/nodes

Response:
{
    "success": true,
    "obj": [
        {
            "id": 1,
            "nodeId": "node-us-1",
            "name": "US West",
            "address": "192.168.1.100:2095",
            "status": "online",
            "lastSeen": 1702900000,
            "systemInfo": {"cpu": 25.5, "memory": 60.2}
        },
        ...
    ]
}
```

#### 4.2.5 编辑/删除节点

```
POST /api/save

Request (编辑):
{
    "object": "nodes",
    "action": "edit",
    "data": {
        "id": 1,
        "name": "US West Updated",
        "externalHost": "us-new.example.com",
        "externalPort": 8443,
        "enable": true
    }
}

Request (删除):
{
    "object": "nodes",
    "action": "del",
    "data": {
        "id": 1
    }
}
```

**注意**：节点通过从节点自注册创建，管理 API 只能编辑/删除，不能新增。

---

## 5. 核心流程

### 5.1 节点添加流程

```
┌─────────────────────────────────────────────────────────────────┐
│                       节点添加流程                                │
└─────────────────────────────────────────────────────────────────┘

步骤 1: 主节点生成邀请码
┌─────────────────┐
│  管理员在主节点  │
│  WebUI 生成     │
│  节点邀请码     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Token: abc123  │  ◄── 复制给从节点管理员
│  有效期: 24h    │
└─────────────────┘

步骤 2: 从节点启动并注册
┌─────────────────┐
│  ./sui          │
│  --mode=worker  │
│  --master=xxx   │
│  --token=abc123 │  ◄── 使用邀请码
│  --node-id=xxx  │
│  --node-name=xx │
│  --external-host│
└────────┬────────┘
         │
         ▼
┌─────────────────┐     ┌─────────────────┐
│  POST /node/    │────▶│  主节点验证     │
│  register       │     │  Token 有效性   │
└─────────────────┘     └────────┬────────┘
                                 │
                        ┌────────┴────────┐
                        │                 │
                  成功   ▼           失败  ▼
             ┌──────────────┐    ┌──────────────┐
             │ 创建 Node    │    │ 返回错误     │
             │ 标记 Token   │    │ 从节点退出   │
             │ 为已使用     │    └──────────────┘
             └──────┬───────┘
                    │
                    ▼
             ┌──────────────┐
             │ 返回成功     │
             │ 从节点继续   │
             └──────────────┘
```

### 5.2 从节点启动流程

```
┌─────────────────────────────────────────────────────────────────┐
│                       从节点启动流程                              │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │  解析启动参数     │
                    │  --mode=worker   │
                    └────────┬─────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │  初始化数据库     │
                    │  (本地 SQLite)   │
                    └────────┬─────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │  检查是否已注册   │
                    └────────┬─────────┘
                              │
              ┌───────────────┴───────────────┐
              │                               │
        未注册 ▼                         已注册 ▼
    ┌──────────────────┐            ┌──────────────────┐
    │  POST /register  │            │  跳过注册        │
    │  向主节点注册     │            │  使用已保存的    │
    └────────┬─────────┘            │  Token          │
              │                      └────────┬────────┘
              │                               │
              └───────────────┬───────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │  尝试同步配置     │
                    │  GET /node/config│
                    └────────┬─────────┘
                              │
              ┌───────────────┴───────────────┐
              │                               │
        成功   ▼                         失败  ▼
    ┌──────────────────┐            ┌──────────────────┐
    │  应用远程配置     │            │  使用本地缓存     │
    │  保存到本地DB    │            │  (上次同步的配置) │
    └────────┬─────────┘            └────────┬─────────┘
              │                               │
              └───────────────┬───────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │  启动 Sing-Box   │
                    └────────┬─────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │  启动定时任务     │
                    │  - 配置同步      │
                    │  - 统计上报      │
                    │  - 心跳          │
                    └────────┬─────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │  启动 WebUI      │
                    │  (只读模式)      │
                    └──────────────────┘
```

### 5.3 配置同步流程

```
┌─────────────────────────────────────────────────────────────────┐
│                       配置同步流程 (每 60 秒)                     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │  获取远程版本     │
                    │  /config/version │
                    └────────┬─────────┘
                              │
              ┌───────────────┴───────────────┐
              │                               │
    版本相同   ▼                    版本更新   ▼
    ┌──────────────────┐            ┌──────────────────┐
    │  跳过本次同步     │            │  获取完整配置     │
    │  等待下一周期     │            │  GET /node/config│
    └──────────────────┘            └────────┬─────────┘
                                             │
                                             ▼
                                   ┌──────────────────┐
                                   │  更新本地数据库   │
                                   │  - 清除旧数据    │
                                   │  - 插入新数据    │
                                   └────────┬─────────┘
                                             │
                                             ▼
                                   ┌──────────────────┐
                                   │  重启 Sing-Box   │
                                   │  应用新配置      │
                                   └────────┬─────────┘
                                             │
                                             ▼
                                   ┌──────────────────┐
                                   │  更新本地版本号   │
                                   └──────────────────┘
```

### 5.4 统计上报流程

```
┌─────────────────────────────────────────────────────────────────┐
│                       统计上报流程 (每 30 秒)                     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │  收集本地统计     │
                    │  - 用户流量       │
                    │  - 在线状态       │
                    └────────┬─────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │  上报流量统计     │
                    │  POST /node/stats│
                    └────────┬─────────┘
                              │
              ┌───────────────┴───────────────┐
              │                               │
        成功   ▼                         失败  ▼
    ┌──────────────────┐            ┌──────────────────┐
    │  清除已上报数据   │            │  保留待重试      │
    └──────────────────┘            │  下次继续上报    │
                                    └──────────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │  上报在线状态     │
                    │  POST /onlines   │
                    └──────────────────┘
```

---

## 6. 核心层扩展 (UAP-Aware)

### 6.1 连接追踪扩展

修改 `core/tracker_conn.go`，扩展 ConnectionInfo 以支持用户/设备追踪:

```go
type ConnectionInfo struct {
    ID          string
    Conn        net.Conn
    PacketConn  network.PacketConn
    Inbound     string
    Type        string // "tcp" or "udp"

    // UAP 新增
    User        string    // 用户名 (从 metadata.User)
    SourceIP    string    // 来源 IP (从 metadata.Source)
    ConnectedAt int64     // 连接时间戳
}

// 新增方法
func (c *ConnTracker) GetUserConnectionCount(user string) int
func (c *ConnTracker) GetUserConnections(user string) []*ConnectionInfo
func (c *ConnTracker) CheckDeviceLimit(user string, limit int) bool
func (c *ConnTracker) GetUniqueDeviceCount(user string) int
```

### 6.2 时长追踪器

新增 `core/tracker_stats.go` 中的时长追踪:

```go
// 用户在线时长追踪
type UserTimeTracker struct {
    access     sync.Mutex
    onlineTime map[string]int64  // user -> 本周期在线秒数
    lastUpdate map[string]int64  // user -> 上次更新时间戳
}

func NewUserTimeTracker() *UserTimeTracker

// 更新用户在线时长 (每次统计采集时调用)
func (t *UserTimeTracker) UpdateOnlineTime(onlineUsers []string)

// 获取并重置时长 (用于上报)
func (t *UserTimeTracker) GetAndResetTime() map[string]int64
```

### 6.3 CronJob 扩展

修改 `cronjob/cronjob.go`，新增定时任务:

```go
func (c *CronJob) Start(loc *time.Location, trafficAge int) error {
    // ... 现有任务 ...

    // UAP 新增任务
    c.cron.AddJob("@every 10s", NewTimeTrackJob())      // 时长累计
    c.cron.AddJob("@every 1m", NewTimeDepleteJob())     // 时长超限检查
    c.cron.AddJob("@daily", NewResetJob())              // 流量/时长重置
}
```

#### TimeTrackJob

```go
// cronjob/timeTrackJob.go
// 累计用户在线时长
type TimeTrackJob struct {
    service.ClientService
}

func (j *TimeTrackJob) Run() {
    // 1. 从 core.TimeTracker 获取在线用户及时长
    // 2. 累加到 Client.TimeUsed
    // 3. 检查是否超限，超限则禁用并触发 Webhook
}
```

#### ResetJob

```go
// cronjob/resetJob.go
// 按策略重置流量/时长
type ResetJob struct {
    service.ClientService
    service.WebhookService
}

func (j *ResetJob) Run() {
    now := time.Now()

    // 1. 查找需要重置的用户 (根据 ResetStrategy 和 ResetAt)
    // 2. 重置 Up/Down/TimeUsed
    // 3. 更新 ResetAt
    // 4. 重新启用被禁用的用户
    // 5. 触发 Webhook 回调
}
```

### 6.4 Webhook 服务

新增 `service/webhook.go`:

```go
type WebhookService struct{}

// 发送回调通知
func (s *WebhookService) SendCallback(event string, data interface{}) error

// 事件类型
const (
    EventTrafficExceeded = "traffic_exceeded"
    EventTimeExceeded    = "time_exceeded"
    EventTrafficReset    = "traffic_reset"
    EventTimeReset       = "time_reset"
    EventUserExpiring    = "user_expiring"
)

// Webhook 请求体
type WebhookPayload struct {
    Event     string      `json:"event"`
    Timestamp int64       `json:"timestamp"`
    Data      interface{} `json:"data"`
}
```

---

## 7. 订阅服务设计

### 7.1 设计原则

- **从节点不提供订阅服务**：避免配置分散，统一由主节点管理
- **主节点聚合所有节点**：订阅返回所有可用节点的代理配置
- **客户端自主选择**：用户可手动或自动选择使用哪个节点

### 7.2 架构

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           主节点订阅服务                                  │
│                                                                          │
│   ┌─────────────┐      ┌─────────────┐      ┌─────────────────────────┐ │
│   │  Inbound    │      │   Node      │      │      生成结果            │ │
│   │  (模板)     │      │   (地址)    │      │                         │ │
│   ├─────────────┤      ├─────────────┤      ├─────────────────────────┤ │
│   │ type: vmess │      │ US-West     │      │ US-West-vmess           │ │
│   │ port: 443   │  ×   │ us.example  │  =   │ server: us.example.com  │ │
│   │ tls: true   │      │             │      │ port: 443               │ │
│   │             │      ├─────────────┤      ├─────────────────────────┤ │
│   │             │      │ JP-Tokyo    │      │ JP-Tokyo-vmess          │ │
│   │             │      │ jp.example  │      │ server: jp.example.com  │ │
│   └─────────────┘      └─────────────┘      └─────────────────────────┘ │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘

客户端请求: GET /sub/{uuid}?format=clash
返回: 包含所有节点的代理配置
```

### 7.3 生成逻辑

```go
// 主节点订阅服务 (伪代码)
func (s *SubService) GetSubs(uuid string) []ServerConfig {
    client := getClientByUUID(uuid)  // 改用 UUID 查询
    inbounds := getInboundsByClient(client)
    nodes := getEnabledNodes()  // 获取所有启用的节点

    var servers []ServerConfig
    for _, node := range nodes {
        for _, inbound := range inbounds {
            server := buildServerConfig(inbound, node)
            // 使用节点的外部地址
            server.Host = node.ExternalHost
            if node.ExternalPort > 0 {
                server.Port = node.ExternalPort
            } else {
                server.Port = inbound.ListenPort
            }
            // 服务器名称: 节点名-入站标签
            server.Name = fmt.Sprintf("%s-%s", node.Name, inbound.Tag)
            servers = append(servers, server)
        }
    }
    return servers
}
```

### 7.4 输出示例

**Clash 格式：**

```yaml
proxies:
  - name: "US-West-vmess"
    type: vmess
    server: us.example.com
    port: 443
    uuid: client-uuid-xxx
    # ...

  - name: "JP-Tokyo-vmess"
    type: vmess
    server: jp.example.com
    port: 443
    uuid: client-uuid-xxx
    # ...

proxy-groups:
  - name: "Auto"
    type: url-test
    proxies:
      - "US-West-vmess"
      - "JP-Tokyo-vmess"
    url: "http://www.gstatic.com/generate_204"
    interval: 300

  - name: "Select"
    type: select
    proxies:
      - "Auto"
      - "US-West-vmess"
      - "JP-Tokyo-vmess"
```

### 7.5 从节点订阅服务处理

从节点的订阅服务有两种处理方式：

| 方式 | 实现 | 说明 |
|------|------|------|
| **禁用** | `subPort = 0` | 从节点不启动订阅服务 |
| **代理** | 转发到主节点 | 从节点收到请求后转发给主节点处理 |

**推荐方式**：禁用从节点订阅服务，客户端统一访问主节点获取订阅。

---

## 8. 代码结构

### 8.1 新增文件

```
s-ui/
├── database/
│   └── model/
│       ├── node.go              # Node, NodeToken, NodeStats, ClientOnline 模型
│       └── webhook.go           # WebhookConfig, ApiKey 模型 (UAP)
├── service/
│   ├── node.go                  # NodeService - 节点管理
│   ├── sync.go                  # SyncService - 从节点同步
│   └── webhook.go               # WebhookService - 回调服务 (UAP)
├── cronjob/
│   ├── timeTrackJob.go          # 时长追踪任务 (UAP)
│   └── resetJob.go              # 重置策略任务 (UAP)
├── api/
│   ├── nodeHandler.go           # 节点 API Handler
│   └── externalHandler.go       # 外部 API Handler (UAP)
└── frontend/
    └── src/
        ├── views/
        │   ├── Nodes.vue        # 节点管理页面
        │   └── ApiKeys.vue      # API Key 管理页面 (UAP)
        ├── layouts/
        │   └── modals/
        │       ├── Node.vue     # 节点编辑弹窗
        │       └── ApiKey.vue   # API Key 编辑弹窗 (UAP)
        ├── components/
        │   └── NodeSelector.vue # 节点选择器
        └── types/
            ├── Node.ts          # 节点类型定义
            └── Client.ts        # Client 类型扩展 (UAP)
```

### 8.2 修改文件

```
s-ui/
├── config/
│   └── config.go                # + 节点配置结构
├── cmd/
│   └── cmd.go                   # + 节点命令行参数
├── app/
│   └── app.go                   # + 节点服务初始化
├── database/
│   ├── db.go                    # + 新模型迁移
│   └── model/
│       └── model.go             # + Client UAP 字段, Stats.NodeId
├── core/
│   ├── tracker_conn.go          # + ConnectionInfo 扩展 (UAP)
│   └── tracker_stats.go         # + UserTimeTracker (UAP)
├── cronjob/
│   └── cronjob.go               # + 注册 UAP 新任务
├── service/
│   ├── stats.go                 # + 多节点统计方法
│   └── client.go                # + UUID 同步到 Config (UAP)
├── sub/
│   └── subService.go            # + 订阅查询改用 UUID (UAP)
├── web/
│   └── web.go                   # + 节点 API 路由 + 外部 API 路由
├── api/
│   ├── apiHandler.go            # + 只读检查 + nodes API
│   └── apiService.go            # + 节点管理方法
└── frontend/
    └── src/
        ├── store/
        │   └── modules/
        │       └── data.ts      # + 节点状态, API Keys
        ├── router/
        │   └── index.ts         # + 节点路由, API Keys 路由
        ├── layouts/
        │   ├── default/
        │   │   ├── Drawer.vue   # + 节点菜单, API Keys 菜单
        │   │   └── AppBar.vue   # + 节点选择器, 只读标签
        │   └── modals/
        │       └── Client.vue   # + UAP 字段 (UUID, 时长限制, 设备限制等)
        └── views/
            ├── Clients.vue      # + 显示 UAP 扩展字段
            └── Settings.vue     # + Webhook 配置 (UAP)
```

### 8.3 前端 UI 改造说明

#### 菜单结构

```
原有菜单                    修改后菜单
─────────────              ─────────────
Home                       Home
Inbounds                   Inbounds
Clients                    Clients
Outbounds                  Outbounds
Endpoints                  Endpoints
Services                   Services
TLS                        TLS
Basics                     Basics
Rules                      Rules
DNS                        DNS
                           ─────────────
                           Nodes        ← 新增 (仅 Master 模式)
                           ApiKeys      ← 新增 (仅 Master 模式)
                           ─────────────
Admins                     Admins
Settings                   Settings
```

**Drawer.vue 菜单逻辑:**
```typescript
const menu = computed(() => {
  const baseMenu = [
    { title: 'pages.home', icon: 'mdi-home', path: '/' },
    { title: 'pages.inbounds', icon: 'mdi-cloud-download', path: '/inbounds' },
    { title: 'pages.clients', icon: 'mdi-account-multiple', path: '/clients' },
    { title: 'pages.outbounds', icon: 'mdi-cloud-upload', path: '/outbounds' },
    { title: 'pages.endpoints', icon: 'mdi-cloud-tags', path: '/endpoints' },
    { title: 'pages.services', icon: 'mdi-server', path: '/services' },
    { title: 'pages.tls', icon: 'mdi-certificate', path: '/tls' },
    { title: 'pages.basics', icon: 'mdi-application-cog', path: '/basics' },
    { title: 'pages.rules', icon: 'mdi-routes', path: '/rules' },
    { title: 'pages.dns', icon: 'mdi-dns', path: '/dns' },
  ]

  // Master 模式才显示
  if (dataStore.isMaster()) {
    baseMenu.push(
      { title: 'pages.nodes', icon: 'mdi-server-network', path: '/nodes' },
      { title: 'pages.apikeys', icon: 'mdi-key', path: '/apikeys' },
    )
  }

  baseMenu.push(
    { title: 'pages.admins', icon: 'mdi-account-tie', path: '/admins' },
    { title: 'pages.settings', icon: 'mdi-cog', path: '/settings' },
  )

  return baseMenu
})
```

#### 多节点管理 UI

**Nodes.vue - 节点管理页面:**
- 节点列表表格 (ID, 名称, 地址, 状态, 最后心跳)
- 邀请码管理 (生成/删除邀请码)
- 节点编辑/删除操作

**NodeSelector.vue - 节点选择器 (AppBar):**
- 下拉选择: All Nodes / Local / Node1 / Node2 ...
- 切换后刷新数据视图，筛选统计数据

**只读模式 (Worker):**
- AppBar 显示 "Read-Only" 标签
- 所有编辑按钮 `disabled`
- 通过 `dataStore.isReadOnly()` 检查

#### UAP 扩展 UI

**Client.vue 弹窗新增字段:**
```
┌─────────────────────────────────────────────────┐
│  Client 编辑                                     │
├─────────────────────────────────────────────────┤
│  基本信息                                        │
│  ├─ 名称: [________]                            │
│  ├─ UUID: [________] (自动生成/UAP提供)         │
│  └─ 会员: [✓] 是会员                            │
│                                                  │
│  流量限制                                        │
│  ├─ 流量配额: [____] GB                         │
│  ├─ 重置策略: [每日 ▼]                          │
│  └─ 带宽限制: [____] Mbps (0=无限)              │
│                                                  │
│  时长限制                                        │
│  ├─ 时长配额: [____] 小时                       │
│  ├─ 已用时长: 2.5 小时                          │
│  └─ 重置策略: [每月 ▼]                          │
│                                                  │
│  设备限制                                        │
│  └─ 最大设备数: [____] (0=无限)                 │
│                                                  │
│  过期时间                                        │
│  └─ 过期日期: [2025-12-31]                      │
└─────────────────────────────────────────────────┘
```

**Clients.vue 列表扩展列:**
| 名称 | 流量 | 时长 | 设备 | 状态 | 操作 |
|------|------|------|------|------|------|
| user1 | 5.2/10 GB | 2.5/10 h | 2/3 | ✓ | ... |

**Settings.vue Webhook 配置区域:**
```
┌─────────────────────────────────────────────────┐
│  Webhook 配置                                    │
├─────────────────────────────────────────────────┤
│  回调 URL: [https://uap.example.com/callback]   │
│  密钥: [********]                               │
│  启用: [✓]                                      │
│                                                  │
│  事件通知:                                       │
│  [✓] 流量超限  [✓] 时长超限                     │
│  [✓] 流量重置  [✓] 时长重置                     │
│  [✓] 用户即将过期                               │
└─────────────────────────────────────────────────┘
```

**ApiKeys.vue - API Key 管理页面:**
- API Key 列表 (名称, Key前缀, 状态, 创建时间)
- 创建/删除 API Key
- Key 只在创建时显示完整内容

#### i18n 国际化配置

新增菜单项需要添加翻译，修改 `frontend/src/locales/` 下的语言文件：

**en.json:**
```json
{
  "pages": {
    "nodes": "Nodes",
    "apikeys": "API Keys"
  },
  "nodes": {
    "title": "Node Management",
    "addToken": "Generate Invite Token",
    "tokenList": "Invite Tokens",
    "nodeList": "Registered Nodes",
    "status": {
      "online": "Online",
      "offline": "Offline",
      "error": "Error"
    }
  },
  "apikeys": {
    "title": "API Key Management",
    "add": "Create API Key",
    "copyWarning": "Key will only be shown once"
  }
}
```

**zh-Hans.json:**
```json
{
  "pages": {
    "nodes": "节点管理",
    "apikeys": "API 密钥"
  },
  "nodes": {
    "title": "节点管理",
    "addToken": "生成邀请码",
    "tokenList": "邀请码列表",
    "nodeList": "已注册节点",
    "status": {
      "online": "在线",
      "offline": "离线",
      "error": "错误"
    }
  },
  "apikeys": {
    "title": "API 密钥管理",
    "add": "创建 API 密钥",
    "copyWarning": "密钥仅显示一次"
  }
}
```

---

## 9. 配置说明

### 9.1 环境变量

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `SUI_NODE_MODE` | 节点模式 (standalone/master/worker) | standalone |
| `SUI_NODE_ID` | 节点唯一标识 | - |
| `SUI_NODE_NAME` | 节点名称 | - |
| `SUI_MASTER_ADDR` | 主节点地址 (worker 模式必需) | - |
| `SUI_NODE_TOKEN` | 节点认证 token (worker 模式必需) | - |
| `SUI_SYNC_CONFIG_INTERVAL` | 配置同步间隔 (秒) | 60 |
| `SUI_SYNC_STATS_INTERVAL` | 统计上报间隔 (秒) | 30 |

### 9.2 命令行参数

```bash
# 单机模式 (默认)
./sui

# 主节点
./sui --mode=master

# 从节点 (完整参数)
./sui --mode=worker \
      --master=https://master.example.com:2095 \
      --token=your-secure-token \
      --node-id=node-us-west-1 \
      --node-name="US West 1" \
      --external-host=us.example.com \
      --external-port=443

# 查看帮助
./sui --help
```

**参数说明:**

| 参数 | 说明 | 必需 |
|------|------|------|
| `--mode` | 运行模式 (standalone/master/worker) | 否，默认 standalone |
| `--master` | 主节点地址 | worker 模式必需 |
| `--token` | 节点邀请码 | worker 模式必需 |
| `--node-id` | 节点唯一标识 | worker 模式必需 |
| `--node-name` | 节点显示名称 | 否，默认使用 node-id |
| `--external-host` | 客户端连接地址 (用于订阅) | 否 |
| `--external-port` | 客户端连接端口 (0=与 Inbound 相同) | 否，默认 0 |

### 9.3 配置文件 (可选)

```json
// config/node.json
{
    "mode": "worker",
    "nodeId": "node-us-west-1",
    "nodeName": "US West 1",
    "master": {
        "address": "https://master.example.com:2095",
        "token": "your-secure-token"
    },
    "sync": {
        "configInterval": 60,
        "statsInterval": 30
    }
}
```

---

## 10. 安全考虑

### 10.1 通信安全

- 主从通信建议使用 HTTPS
- Token 使用 32 字符随机字符串
- Token 在传输和存储时应加密

### 10.2 认证机制

- 每个从节点有唯一的 Node ID 和 Token
- Token 由主节点生成，在注册节点时分配
- 主节点验证每个请求的 Node ID + Token

### 10.3 权限控制

- Worker 模式 API 层拦截所有写操作
- WebUI 检测模式，禁用编辑按钮

---

## 11. 容错机制

### 11.1 从节点离线处理

| 场景 | 行为 |
|------|------|
| 启动时连不上主节点 | 使用本地数据库缓存启动 |
| 运行中主节点断开 | 继续使用当前配置运行 |
| 主节点恢复 | 自动重连，同步最新配置 |

### 11.2 统计数据可靠性

- 上报失败时保留数据，下次重试
- 本地统计数据定期清理（保留 7 天）
- 主节点汇总后可清理原始数据

### 11.3 配置版本控制

- 使用时间戳作为配置版本号
- 从节点只在版本更新时拉取完整配置
- 配置变更记录在 Changes 表供审计

---

## 12. 实现计划

### Phase 1: 数据库与模型

- [ ] 新增 `database/model/node.go` (Node, NodeToken, NodeStats, ClientOnline)
- [ ] 修改 `database/model/model.go` (Client 扩展, Stats.NodeId)
- [ ] 新增 `database/model/webhook.go` (WebhookConfig, ApiKey)
- [ ] 修改 `database/db.go` (AutoMigrate)

### Phase 1.5: 核心层改造 (UAP-Aware)

- [ ] 修改 `core/tracker_conn.go` (扩展 ConnectionInfo: User, SourceIP, ConnectedAt)
- [ ] 修改 `core/tracker_stats.go` (新增 UserTimeTracker)

### Phase 2: 配置与启动

- [ ] 修改 `config/config.go` (节点配置)
- [ ] 修改 `cmd/cmd.go` (命令行参数)
- [ ] 修改 `app/app.go` (初始化流程)

### Phase 3: 主节点服务

- [ ] 新增 `service/node.go`
- [ ] 新增 `api/nodeHandler.go`
- [ ] 修改 `service/stats.go`
- [ ] 修改 `web/web.go`
- [ ] 修改 `api/apiHandler.go`
- [ ] 修改 `api/apiService.go`

### Phase 4: 从节点同步

- [ ] 新增 `service/sync.go`
- [ ] 修改 `api/apiHandler.go` (只读检查)

### Phase 4.5: CronJob 扩展 (UAP-Aware)

- [ ] 新增 `cronjob/timeTrackJob.go` (时长追踪任务)
- [ ] 新增 `cronjob/resetJob.go` (重置策略任务)
- [ ] 新增 `service/webhook.go` (Webhook 回调服务)
- [ ] 修改 `cronjob/cronjob.go` (注册新任务)

### Phase 5: 前端改造

**多节点管理 UI:**
- [ ] 新增 `frontend/src/views/Nodes.vue` (节点管理页面)
- [ ] 新增 `frontend/src/layouts/modals/Node.vue` (节点编辑弹窗)
- [ ] 新增 `frontend/src/components/NodeSelector.vue` (节点选择器)
- [ ] 新增 `frontend/src/types/Node.ts` (节点类型定义)
- [ ] 修改 `frontend/src/store/modules/data.ts` (节点状态)
- [ ] 修改 `frontend/src/router/index.ts` (节点路由)
- [ ] 修改 `frontend/src/layouts/default/Drawer.vue` (节点菜单)
- [ ] 修改 `frontend/src/layouts/default/AppBar.vue` (节点选择器 + 只读标签)

**UAP 扩展 UI:**
- [ ] 新增 `frontend/src/views/ApiKeys.vue` (API Key 管理页面)
- [ ] 新增 `frontend/src/layouts/modals/ApiKey.vue` (API Key 编辑弹窗)
- [ ] 新增 `frontend/src/types/Client.ts` (Client UAP 类型扩展)
- [ ] 修改 `frontend/src/layouts/modals/Client.vue` (添加 UAP 字段: UUID, 时长限制, 设备限制等)
- [ ] 修改 `frontend/src/views/Clients.vue` (显示 UAP 扩展字段列)
- [ ] 修改 `frontend/src/views/Settings.vue` (添加 Webhook 配置区域)

### Phase 6: UAP Backend 对接

- [ ] 修改 `service/client.go` (UUID 同步到 Config)
- [ ] 修改 `sub/subService.go` (订阅查询改用 UUID)
- [ ] 新增 `api/externalHandler.go` (外部 API)
- [ ] 修改 `web/web.go` (注册外部 API 路由)

---

## 13. UAP Backend 对接

### 13.1 设计决策

| 决策项 | 选择 | 说明 |
|--------|------|------|
| 用户标识 | Client 新增 UUID 字段 | UAP Backend 创建用户时提供 UUID |
| 订阅 URL | `/sub/{uuid}` | 用 UUID 替代 Name 作为订阅标识 |
| 协议 UUID | 自动同步 | Config.*.uuid 自动使用 Client.UUID |
| API 认证 | X-API-Key Header | 新增 API Key 认证中间件 |

### 13.2 Client 模型扩展

```go
type Client struct {
    // 现有字段
    Id       uint            `json:"id" gorm:"primaryKey;autoIncrement"`
    Enable   bool            `json:"enable"`
    Name     string          `json:"name"`
    Config   json.RawMessage `json:"config,omitempty"`
    Inbounds json.RawMessage `json:"inbounds"`
    Links    json.RawMessage `json:"links,omitempty"`
    Volume   int64           `json:"volume"`           // 流量限制 (bytes)
    Expiry   int64           `json:"expiry"`           // 过期时间戳
    Down     int64           `json:"down"`             // 下行流量
    Up       int64           `json:"up"`               // 上行流量
    Desc     string          `json:"desc"`
    Group    string          `json:"group"`

    // UAP 新增字段
    UUID               string `json:"uuid" gorm:"unique"`            // UAP 用户标识
    IsPremium          bool   `json:"isPremium"`                     // 是否会员
    TimeLimit          int64  `json:"timeLimit"`                     // 时长限制 (秒), 0=无限
    TimeUsed           int64  `json:"timeUsed"`                      // 已用时长 (秒)
    TimeResetStrategy  string `json:"timeResetStrategy"`             // daily/weekly/monthly/no_reset
    TimeResetAt        int64  `json:"timeResetAt"`                   // 上次时长重置时间
    TrafficResetStrategy string `json:"trafficResetStrategy"`        // daily/weekly/monthly/no_reset
    TrafficResetAt     int64  `json:"trafficResetAt"`                // 上次流量重置时间
    SpeedLimit         int    `json:"speedLimit"`                    // 带宽限制 (Mbps), 0=无限
    DeviceLimit        int    `json:"deviceLimit"`                   // 设备数限制, 0=无限
}
```

### 13.3 UUID 自动同步

保存 Client 时自动将 UUID 同步到各协议配置:

```go
func (s *ClientService) syncUUIDToConfig(client *model.Client) error {
    var config map[string]map[string]interface{}
    json.Unmarshal(client.Config, &config)

    // 同步 UUID 到支持 UUID 的协议
    for _, proto := range []string{"vless", "vmess", "tuic"} {
        if cfg, ok := config[proto]; ok {
            cfg["uuid"] = client.UUID
        }
    }

    client.Config, _ = json.Marshal(config)
    return nil
}
```

### 13.4 外部 API (UAP Backend 调用)

**基础路径**: `/api/v1/`

**认证方式**: `X-API-Key` Header

**响应格式**:
```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

#### 用户管理 API (8 个)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/users` | 创建用户 (传入 UUID) |
| PUT | `/api/v1/users/{uuid}` | 更新用户 |
| DELETE | `/api/v1/users/{uuid}` | 删除用户 |
| GET | `/api/v1/users/{uuid}` | 获取用户信息 |
| POST | `/api/v1/users/{uuid}/enable` | 启用用户 |
| POST | `/api/v1/users/{uuid}/disable` | 禁用用户 |
| POST | `/api/v1/users/{uuid}/reset-traffic` | 重置流量 |
| POST | `/api/v1/users/{uuid}/reset-time` | 重置时长 |

#### 订阅管理 API (2 个)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/users/{uuid}/subscription` | 获取订阅链接 |
| POST | `/api/v1/users/{uuid}/subscription/regenerate` | 重新生成订阅链接 |

#### 流量统计 API (4 个)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/users/{uuid}/traffic` | 获取用户流量 |
| POST | `/api/v1/users/traffic/batch` | 批量获取流量 |
| GET | `/api/v1/users/{uuid}/traffic/by-node` | 分节点流量统计 |
| GET | `/api/v1/users/{uuid}/online` | 获取在线状态 |

#### 节点信息 API (3 个)

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/nodes` | 获取节点列表 |
| GET | `/api/v1/nodes/{node_id}` | 获取节点状态 |
| GET | `/api/v1/stats` | 获取系统统计 |

### 13.5 API Key 管理

```go
type ApiKey struct {
    Id        uint   `json:"id" gorm:"primaryKey;autoIncrement"`
    Key       string `json:"key" gorm:"unique;not null"`
    Name      string `json:"name"`
    Enable    bool   `json:"enable" gorm:"default:true"`
    CreatedAt int64  `json:"createdAt" gorm:"autoCreateTime"`
}
```

### 13.6 UAP 功能实现状态

| 功能 | UAP 需求 | 实现阶段 | 说明 |
|------|---------|---------|------|
| time_limit | 时长限制 (秒) | Phase 1 (模型) + Phase 4.5 (CronJob) | Client 新增字段 + TimeTrackJob |
| time_used | 已用时长 | Phase 1.5 (核心层) + Phase 4.5 (CronJob) | UserTimeTracker + TimeTrackJob |
| speed_limit | 带宽限制 | Phase 1 (模型) | 字段预留，依赖 sing-box rate limiter |
| device_limit | 设备数限制 | Phase 1.5 (核心层) | ConnTracker 扩展 + CheckDeviceLimit |
| Webhook 回调 | 超限/重置通知 | Phase 4.5 (CronJob) | WebhookService + 事件触发 |
| 流量重置策略 | 按周期重置流量 | Phase 1 (模型) + Phase 4.5 (CronJob) | ResetJob |
| 时长重置策略 | 按周期重置时长 | Phase 1 (模型) + Phase 4.5 (CronJob) | ResetJob |

---

## 附录

### A. 数据模型定义 (Go)

```go
// database/model/node.go

package model

import "encoding/json"

// NodeToken 节点邀请码
type NodeToken struct {
    Id        uint   `json:"id" gorm:"primaryKey;autoIncrement"`
    Token     string `json:"token" gorm:"unique;not null"`
    Name      string `json:"name"`                       // 预设节点名称 (可选)
    Used      bool   `json:"used" gorm:"default:false"`
    UsedBy    string `json:"usedBy"`                     // 使用的节点 ID
    ExpiresAt int64  `json:"expiresAt"`                  // 过期时间 (0=永不过期)
    CreatedAt int64  `json:"createdAt" gorm:"autoCreateTime"`
}

// Node 节点注册信息
type Node struct {
    Id           uint            `json:"id" gorm:"primaryKey;autoIncrement"`
    NodeId       string          `json:"nodeId" gorm:"unique;not null"`
    Name         string          `json:"name" gorm:"not null"`
    Address      string          `json:"address"`                       // 内部地址 (从节点注册时上报)
    ExternalHost string          `json:"externalHost"`                  // 外部地址 (客户端连接，用于生成订阅)
    ExternalPort int             `json:"externalPort" gorm:"default:0"` // 外部端口 (0 表示与 Inbound 相同)
    Token        string          `json:"token" gorm:"not null"`
    Enable       bool            `json:"enable" gorm:"default:true"`
    Status       string          `json:"status" gorm:"default:'offline'"`
    LastSeen     int64           `json:"lastSeen"`
    LastSync     int64           `json:"lastSync"`
    Version      string          `json:"version"`
    SystemInfo   json.RawMessage `json:"systemInfo" gorm:"type:text"`
    CreatedAt    int64           `json:"createdAt" gorm:"autoCreateTime"`
    UpdatedAt    int64           `json:"updatedAt" gorm:"autoUpdateTime"`

    // UAP API 所需字段
    Country   string `json:"country"`              // 国家
    City      string `json:"city"`                 // 城市
    Flag      string `json:"flag"`                 // 国家代码 (ISO 3166-1 alpha-2)
    IsPremium bool   `json:"isPremium"`            // 是否仅会员可用
    Latency   int    `json:"latency"`              // 平均延迟 (ms)
}

// NodeStats 节点统计快照
type NodeStats struct {
    Id          uint64  `json:"id" gorm:"primaryKey;autoIncrement"`
    NodeId      string  `json:"nodeId" gorm:"index;not null"`
    DateTime    int64   `json:"dateTime" gorm:"index;not null"`
    CPU         float64 `json:"cpu"`
    Memory      float64 `json:"memory"`
    Connections int     `json:"connections"`
    Upload      int64   `json:"upload"`
    Download    int64   `json:"download"`
}

// ClientOnline 客户端在线状态
type ClientOnline struct {
    Id          uint64 `json:"id" gorm:"primaryKey;autoIncrement"`
    ClientName  string `json:"clientName" gorm:"index;not null"`
    NodeId      string `json:"nodeId" gorm:"index;not null"`
    InboundTag  string `json:"inboundTag"`
    SourceIP    string `json:"sourceIP"`              // UAP: 设备 IP (用于设备数限制)
    ConnectedAt int64  `json:"connectedAt"`           // UAP: 连接时间 (用于时长计算)
    LastSeen    int64  `json:"lastSeen" gorm:"not null"`
}

// WebhookConfig Webhook 配置 (UAP 回调)
type WebhookConfig struct {
    Id             uint   `json:"id" gorm:"primaryKey;autoIncrement"`
    CallbackURL    string `json:"callbackUrl"`
    CallbackSecret string `json:"callbackSecret"`
    Enable         bool   `json:"enable" gorm:"default:true"`
}

// ApiKey API Key (UAP 认证)
type ApiKey struct {
    Id        uint   `json:"id" gorm:"primaryKey;autoIncrement"`
    Key       string `json:"key" gorm:"unique;not null"`
    Name      string `json:"name"`
    Enable    bool   `json:"enable" gorm:"default:true"`
    CreatedAt int64  `json:"createdAt" gorm:"autoCreateTime"`
}
```

### B. 前端类型定义 (TypeScript)

```typescript
// frontend/src/types/Node.ts

export interface NodeToken {
    id?: number
    token: string
    name?: string
    used: boolean
    usedBy?: string
    expiresAt: number
    createdAt: number
}

export interface Node {
    id?: number
    nodeId: string
    name: string
    address: string          // 内部地址 (主节点管理通信)
    externalHost: string     // 外部地址 (客户端连接，用于生成订阅)
    externalPort: number     // 外部端口 (0 表示与 Inbound 相同)
    token?: string
    enable: boolean
    status: 'online' | 'offline' | 'error'
    lastSeen: number
    lastSync: number
    version: string
    systemInfo?: SystemInfo

    // UAP API 所需字段
    country?: string         // 国家
    city?: string            // 城市
    flag?: string            // 国家代码 (ISO 3166-1 alpha-2)
    isPremium?: boolean      // 是否仅会员可用
    latency?: number         // 平均延迟 (ms)
}

export interface SystemInfo {
    cpu: number
    memory: number
    connections: number
}

export interface NodeStats {
    nodeId: string
    dateTime: number
    cpu: number
    memory: number
    upload: number
    download: number
}

export interface ClientOnline {
    id?: number
    clientName: string
    nodeId: string
    inboundTag?: string
    sourceIP?: string          // UAP: 设备 IP
    connectedAt?: number       // UAP: 连接时间
    lastSeen: number
}

// 注: Client UAP 扩展字段定义在 frontend/src/types/Client.ts 中

export interface WebhookConfig {
    id?: number
    callbackUrl: string
    callbackSecret?: string
    enable: boolean
}

export interface ApiKey {
    id?: number
    key: string
    name?: string
    enable: boolean
    createdAt?: number
}
```

```typescript
// frontend/src/types/Client.ts (UAP 扩展)

export interface Client {
    // 现有字段
    id?: number
    enable: boolean
    name: string
    config?: Record<string, any>
    inbounds: number[]
    links?: ClientLink[]
    volume: number              // 流量限制 (bytes)
    expiry: number              // 过期时间戳
    down: number                // 下行流量
    up: number                  // 上行流量
    desc?: string
    group?: string

    // UAP 新增字段
    uuid?: string                    // UAP 用户标识
    isPremium?: boolean              // 是否会员
    timeLimit?: number               // 时长限制 (秒), 0=无限
    timeUsed?: number                // 已用时长 (秒)
    timeResetStrategy?: ResetStrategy
    timeResetAt?: number             // 上次时长重置时间
    trafficResetStrategy?: ResetStrategy
    trafficResetAt?: number          // 上次流量重置时间
    speedLimit?: number              // 带宽限制 (Mbps), 0=无限
    deviceLimit?: number             // 设备数限制, 0=无限
}

export type ResetStrategy = 'daily' | 'weekly' | 'monthly' | 'no_reset'

export interface ClientLink {
    type: 'local' | 'external' | 'sub'
    remark: string
    uri: string
}
```
