# UAP 协议 sing-box 实现方案

> **文档版本**: v1.0
> **最后更新**: 2025-12-22

## 1. 概述

本文档说明如何在 sing-box 中实现 UAP 协议，以及 sing-box 和 hiddify-sing-box 的区别。

### 1.1 相关仓库

| 仓库 | 说明 | 地址 |
|------|------|------|
| sing-box | 官方 sing-box | https://github.com/SagerNet/sing-box |
| hiddify-sing-box | Hiddify 团队 fork (仅作参考) | https://github.com/hiddify/hiddify-sing-box |
| uap-sing-box | **UAP 官方版本** (基于官方 sing-box) | https://git.uap.io/uap/uap-sing-box |

> **代码来源说明**: UAP 协议是我们自主开发的协议，基于 VLESS 协议规范重新实现。
> 本地目录 `~/work/projects/uap/hiddify-sing-box-main` 包含 UAP 协议源码，
> 命名沿用历史，实际与 hiddify-sing-box 无关。UAP 代码将移植到 uap-sing-box 中。

### 1.2 UAP 协议简介

UAP (Universal Access Protocol) 是我们自主开发的协议，基于 VLESS 协议规范：

| 项目 | VLESS | UAP |
|------|-------|-----|
| Protocol Version | 0 | 1 |
| Addons 格式 | Protobuf | 简化二进制 (长度前缀) |
| 兼容性 | - | 与 VLESS 不兼容 |
| Flow 支持 | xtls-rprx-vision | xtls-rprx-vision |
| Reality 支持 | 完整 | 完整 |

---

## 2. sing-box vs hiddify-sing-box 对比

### 2.1 版本信息

| 项目 | sing-box (官方) | hiddify-sing-box |
|------|-----------------|------------------|
| 当前版本 | v1.13.x | v1.9.4 |
| Go 版本 | go 1.25.5 | go 1.21.4 |
| 更新频率 | 活跃 (每周) | 较慢 |
| 最后更新 | 2024年12月 | 2024年9月 |

### 2.2 代码结构差异

```
sing-box (官方 v1.13.x)          hiddify-sing-box (v1.9.4)
========================          ========================
├── protocol/                     ├── inbound/
│   ├── vless/                    │   ├── vless.go
│   ├── vmess/                    │   ├── vmess.go
│   ├── trojan/                   │   └── ...
│   └── ...                       ├── outbound/
├── adapter/                      │   ├── vless.go
├── service/                      │   ├── vmess.go
└── ...                           │   └── ...
                                  └── ...
```

**关键差异：**
- 官方 v1.12+ 重构了代码结构，将协议实现移到 `protocol/` 目录
- hiddify-sing-box 仍使用旧的 `inbound/` + `outbound/` 结构

### 2.3 hiddify-sing-box 新增功能

#### 2.3.1 Xray 核心集成

```go
// outbound/xray.go
import "github.com/xtls/xray-core/core"

type XrayOutboundOptions struct {
    DialerOptions
    Network          NetworkList        `json:"network,omitempty"`
    XrayOutboundJson *map[string]any    `json:"xray_outbound_raw"`
    Fragment         *conf.Fragment     `json:"xray_fragment"`
    LogLevel         string             `json:"xray_loglevel"`
}
```

- 可直接使用 Xray 配置格式
- 支持 Xray 的高级功能

#### 2.3.2 TLS Fragment (TLS 分片)

```go
// option/fragment.go
type TLSFragmentOptions struct {
    Enabled bool   `json:"enabled,omitempty"`
    Size    string `json:"size,omitempty"`   // 分片大小 (bytes)
    Sleep   string `json:"sleep,omitempty"`  // 分片间隔 (ms)
}
```

**用途：** 绕过 DPI (深度包检测)，将 TLS 握手包分成多个小包发送

**配置示例：**
```json
{
  "outbounds": [
    {
      "type": "vless",
      "tls_fragment": {
        "enabled": true,
        "size": "10-30",
        "sleep": "2-5"
      }
    }
  ]
}
```

#### 2.3.3 TURN Relay 支持

```go
// option/h_turn_udp_proxy.go
type TurnRelayOptions struct {
    ServerOptions
    Username string `json:"username,omitempty"`
    Password string `json:"password,omitempty"`
    Realm    string `json:"realm,omitempty"`
}
```

**用途：** 通过 TURN 服务器中继 UDP 流量

**支持的协议：**
- Hysteria
- Hysteria2
- TUIC
- WireGuard

#### 2.3.4 InvalidConfig 容错处理

```go
// outbound/InvalidConfig.go
type InvalidConfig struct {
    myOutboundAdapter
    err error
}
```

**用途：** 单个 outbound 配置错误时不中断整个服务，而是标记为无效

#### 2.3.5 其他改动

| 功能 | 文件 | 说明 |
|------|------|------|
| 依赖排序优化 | `adapter/router.go` | `SortedOutboundsByDependenciesHiddify()` |
| ProxyProtocol | `inbound/default_tcp.go` | 支持代理协议头 |
| 命令行增强 | `experimental/libbox/` | 更多控制命令 |

### 2.4 功能对比表

| 功能 | sing-box (官方) | hiddify-sing-box |
|------|-----------------|------------------|
| VLESS | ✓ | ✓ |
| VMess | ✓ | ✓ |
| Trojan | ✓ | ✓ |
| Shadowsocks | ✓ | ✓ |
| Hysteria/2 | ✓ | ✓ |
| TUIC | ✓ | ✓ |
| WireGuard | ✓ | ✓ |
| AnyTLS | ✓ (新) | ✗ |
| Tailscale | ✓ (新) | ✗ |
| NaiveProxy outbound | ✓ (新) | ✗ |
| **Xray 集成** | ✗ | ✓ |
| **TLS Fragment** | ✗ | ✓ |
| **TURN Relay** | ✗ | ✓ |
| **UAP** | ✗ | ✗ (需添加) |

---

## 3. UAP 实现方案选择

### 3.1 方案对比

| 方案 | 基于 | 优点 | 缺点 |
|------|------|------|------|
| A | 官方 sing-box | 最新功能、安全修复、活跃维护 | 需适配新代码结构 |
| B | hiddify-sing-box | TLS Fragment 等功能、结构简单 | 版本较旧、缺少新功能 |

### 3.2 推荐方案

**推荐方案 A：基于官方 sing-box 最新版添加 UAP**

理由：
1. 官方版本更新频繁，安全性更好
2. hiddify 的 TLS Fragment 等功能对 UAP 不是必需的
3. UAP 代码量小 (~10 个文件)，移植成本可控
4. 长期维护更容易

---

## 4. 基于官方 sing-box 实现 UAP

### 4.1 需要新增的文件

```
sing-box/
├── protocol/
│   └── uap/                      # 新增目录
│       ├── conn.go               # 连接处理
│       ├── inbound.go            # Inbound 实现
│       ├── outbound.go           # Outbound 实现
│       └── packet.go             # 数据包处理
├── option/
│   └── uap.go                    # 新增: UAP 配置选项
└── include/
    └── uap.go                    # 新增: UAP 构建标签
```

### 4.2 需要修改的文件

| 文件 | 修改内容 |
|------|----------|
| `constant/proxy.go` | 添加 `TypeUAP = "uap"` |
| `protocol/uap/` | 添加 UAP 协议实现 (自研代码) |
| `option/outbound.go` | 添加 `UAPOutboundOptions` |
| `include/outbound_default.go` | 注册 UAP outbound |

### 4.3 UAP 协议核心代码

#### 4.3.1 常量定义

```go
// constant/proxy.go
const (
    // ... 现有常量 ...
    TypeUAP = "uap"  // 新增
)
```

#### 4.3.2 配置选项

```go
// option/uap.go
package option

type UAPInboundOptions struct {
    ListenOptions
    Users     []UAPUser                  `json:"users,omitempty"`
    TLS       *InboundTLSOptions         `json:"tls,omitempty"`
    Multiplex *InboundMultiplexOptions   `json:"multiplex,omitempty"`
    Transport *V2RayTransportOptions     `json:"transport,omitempty"`
}

type UAPUser struct {
    Name string `json:"name"`
    UUID string `json:"uuid"`
    Flow string `json:"flow,omitempty"`
}

type UAPOutboundOptions struct {
    DialerOptions
    ServerOptions
    UUID      string                     `json:"uuid"`
    Flow      string                     `json:"flow,omitempty"`
    Network   NetworkList                `json:"network,omitempty"`
    TLS       *OutboundTLSOptions        `json:"tls,omitempty"`
    Multiplex *OutboundMultiplexOptions  `json:"multiplex,omitempty"`
    Transport *V2RayTransportOptions     `json:"transport,omitempty"`
}
```

#### 4.3.3 协议实现 (从 VLESS 复制修改)

```go
// protocol/uap/protocol.go
package uap

const (
    Version    = 1  // UAP 版本号 (VLESS 是 0)
    FlowVision = "xtls-rprx-vision"
)

type Request struct {
    UUID        [16]byte
    Command     byte
    Destination M.Socksaddr
    Flow        string
}

// ReadRequest 读取请求
func ReadRequest(reader io.Reader) (*Request, error) {
    var request Request

    // 读取版本号
    var version uint8
    err := binary.Read(reader, binary.BigEndian, &version)
    if err != nil {
        return nil, err
    }
    if version != Version {
        return nil, E.New("unknown version: ", version)
    }

    // 读取 UUID
    _, err = io.ReadFull(reader, request.UUID[:])
    if err != nil {
        return nil, err
    }

    // 读取 Addons (简化二进制格式，非 Protobuf)
    var addonsLen uint8
    err = binary.Read(reader, binary.BigEndian, &addonsLen)
    if err != nil {
        return nil, err
    }
    if addonsLen > 0 {
        addonsBytes := make([]byte, addonsLen)
        _, err = io.ReadFull(reader, addonsBytes)
        if err != nil {
            return nil, err
        }
        addons, err := readAddons(bytes.NewReader(addonsBytes))
        if err != nil {
            return nil, err
        }
        request.Flow = addons.Flow
    }

    // 读取命令和目标地址
    err = binary.Read(reader, binary.BigEndian, &request.Command)
    if err != nil {
        return nil, err
    }
    if request.Command != vmess.CommandMux {
        request.Destination, err = vmess.AddressSerializer.ReadAddrPort(reader)
        if err != nil {
            return nil, err
        }
    }

    return &request, nil
}
```

### 4.4 实现计划

> 详细实现计划请参考: [uap-singbox-implementation-plan.md](./uap-singbox-implementation-plan.md)

---

## 5. 基于 hiddify-sing-box 实现 UAP (备选方案)

如果选择基于 hiddify-sing-box：

### 5.1 需要新增的文件

```
hiddify-sing-box/
├── inbound/
│   └── uap.go                    # UAP Inbound
├── outbound/
│   └── uap.go                    # UAP Outbound
├── option/
│   └── uap.go                    # UAP 配置选项
└── protocol/
    └── uap/                      # UAP 协议实现 (新建目录)
        ├── client.go
        ├── constant.go
        ├── protocol.go
        ├── service.go
        ├── vision.go
        ├── vision_reality.go
        └── vision_utls.go
```

### 5.2 复制 UAP 协议文件

已有的 UAP 实现可以直接使用（源目录为历史命名）：

```bash
# 源目录 (历史命名，实际是 UAP 自研代码)
UAP_SRC=~/work/projects/uap/hiddify-sing-box-main

# 目标目录 (新的 uap-sing-box 仓库)
UAP_DEST=~/work/projects/uap/uap-sing-box

# 复制协议实现
cp -r $UAP_SRC/protocol/uap $UAP_DEST/protocol/

# 复制 inbound/outbound
cp $UAP_SRC/inbound/uap.go $UAP_DEST/inbound/
cp $UAP_SRC/outbound/uap.go $UAP_DEST/outbound/

# 复制配置选项
cp $UAP_SRC/option/uap.go $UAP_DEST/option/
```

### 5.3 需要修改的文件

| 文件 | 修改内容 |
|------|----------|
| `constant/proxy.go` | 添加 `TypeUAP = "uap"` |
| `inbound/builder.go` | 注册 UAP inbound |
| `outbound/builder.go` | 注册 UAP outbound |
| `option/outbound.go` | 添加 UAP 到 Outbound 联合类型 |
| `option/inbound.go` | 添加 UAP 到 Inbound 联合类型 |

---

## 6. UAP 配置示例

### 6.1 Inbound 配置

```json
{
  "inbounds": [
    {
      "type": "uap",
      "tag": "uap-in",
      "listen": "::",
      "listen_port": 443,
      "users": [
        {
          "name": "user1",
          "uuid": "550e8400-e29b-41d4-a716-446655440000",
          "flow": "xtls-rprx-vision"
        }
      ],
      "tls": {
        "enabled": true,
        "server_name": "example.com",
        "reality": {
          "enabled": true,
          "handshake": {
            "server": "www.microsoft.com",
            "server_port": 443
          },
          "private_key": "your-private-key",
          "short_id": ["abcd1234"]
        }
      }
    }
  ]
}
```

### 6.2 Outbound 配置

```json
{
  "outbounds": [
    {
      "type": "uap",
      "tag": "uap-out",
      "server": "example.com",
      "server_port": 443,
      "uuid": "550e8400-e29b-41d4-a716-446655440000",
      "flow": "xtls-rprx-vision",
      "tls": {
        "enabled": true,
        "server_name": "example.com",
        "reality": {
          "enabled": true,
          "public_key": "your-public-key",
          "short_id": "abcd1234"
        },
        "utls": {
          "enabled": true,
          "fingerprint": "chrome"
        }
      }
    }
  ]
}
```

---

## 7. 编译与部署

### 7.1 编译环境

**前置要求:**
- Go 1.22+ (推荐 1.24.x)
- Git
- Make (可选)

```bash
# 检查 Go 版本
go version

# 克隆仓库
git clone https://git.uap.io/uap/uap-sing-box.git
cd uap-sing-box
```

### 7.2 编译命令

```bash
# 基本编译 (含 UAP 及常用功能)
go build -tags "with_quic,with_wireguard,with_utls,with_reality_server,with_gvisor" \
    -o sing-box ./cmd/sing-box

# 精简编译 (仅核心功能)
go build -tags "with_utls,with_reality_server" \
    -o sing-box ./cmd/sing-box

# 验证 UAP 支持
./sing-box version
```

**编译标签说明:**
| 标签 | 说明 |
|------|------|
| `with_quic` | QUIC 传输支持 (Hysteria2, TUIC) |
| `with_wireguard` | WireGuard 出站支持 |
| `with_utls` | uTLS 指纹伪装 |
| `with_reality_server` | Reality 服务端支持 |
| `with_gvisor` | Tun 模式 gVisor 支持 |

### 7.3 配置测试

```bash
# 检查配置语法
./sing-box check -c config.json

# 格式化配置
./sing-box format -c config.json

# 前台运行 (调试)
./sing-box run -c config.json
```

### 7.4 部署方式

#### 方式一: systemd 服务

```bash
# 复制二进制文件
sudo cp sing-box /usr/local/bin/
sudo chmod +x /usr/local/bin/sing-box

# 创建配置目录
sudo mkdir -p /etc/sing-box
sudo cp config.json /etc/sing-box/

# 创建 systemd 服务
sudo tee /etc/systemd/system/sing-box.service > /dev/null <<EOF
[Unit]
Description=sing-box service
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/sing-box run -c /etc/sing-box/config.json
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
EOF

# 启动服务
sudo systemctl daemon-reload
sudo systemctl enable sing-box
sudo systemctl start sing-box
sudo systemctl status sing-box
```

#### 方式二: Docker 部署

```dockerfile
# Dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -tags "with_quic,with_utls,with_reality_server" \
    -o sing-box ./cmd/sing-box

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/sing-box /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/sing-box"]
CMD ["run", "-c", "/etc/sing-box/config.json"]
```

```bash
# 构建镜像
docker build -t uap-sing-box:latest .

# 运行容器
docker run -d --name sing-box \
    -v /path/to/config.json:/etc/sing-box/config.json \
    -p 443:443 -p 80:80 \
    --restart unless-stopped \
    uap-sing-box:latest
```

### 7.5 连接测试

使用支持 UAP 的客户端连接并验证：
- 连接建立
- 数据传输
- Vision flow 工作正常
- Reality 握手成功

---

## 8. 总结

### 8.1 推荐路线

```
                    ┌─────────────────────────────────────┐
                    │         官方 sing-box v1.13.x        │
                    │                                     │
                    │  + 最新功能 (AnyTLS, Tailscale...)  │
                    │  + 安全更新                          │
                    │  + 活跃维护                          │
                    └──────────────┬──────────────────────┘
                                   │
                                   │ fork + 添加 UAP
                                   ▼
                    ┌─────────────────────────────────────┐
                    │           uap-sing-box              │
                    │                                     │
                    │  = 官方 sing-box                     │
                    │  + UAP 协议 (~10 个文件)             │
                    └─────────────────────────────────────┘
```

### 8.2 后续维护

- 定期从官方 sing-box 合并更新
- UAP 协议改动时同步更新
- 保持与 S-UI 的兼容性

---

## 相关文档

- [S-UI 多节点管理架构技术方案](./multi-node-architecture.md) - 主从节点架构、数据同步、UAP-Aware 设计
- [S-UI UAP 协议支持技术方案](./uap-protocol-support.md) - S-UI 中 UAP 链接生成、前端配置、订阅输出
