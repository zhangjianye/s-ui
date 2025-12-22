# S-UI UAP 协议支持技术方案

## 1. 概述

### 1.1 背景

UAP (Universal Access Protocol) 是基于 VLESS 协议的自定义协议，已在 hiddify-sing-box 中实现。本文档描述如何在 S-UI 中添加对 UAP 协议的支持。

### 1.2 UAP 与 VLESS 的差异

| 项目 | VLESS | UAP |
|------|-------|-----|
| Protocol Version | 0 | 1 |
| Addons 格式 | Protobuf | 简化二进制 (长度前缀) |
| 兼容性 | - | 与 VLESS 不兼容 |
| Flow 支持 | xtls-rprx-vision | xtls-rprx-vision |
| Reality 支持 | 完整 | 完整 |

### 1.3 依赖

S-UI 需要使用支持 UAP 协议的 sing-box 版本：
- **hiddify-sing-box**: 已包含 UAP 协议实现
- 协议类型常量: `TypeUAP = "uap"`

---

## 2. 实现范围

### 2.1 需要修改的功能

| 功能 | 文件 | 修改内容 |
|------|------|----------|
| 链接生成 | `util/genLink.go` | 添加 UAP 链接生成 |
| Inbound 类型 | 前端配置 | 添加 UAP 入站类型 |
| Client 配置 | 前端/后端 | 支持 UAP 用户配置 |
| 订阅输出 | `sub/` | 支持 UAP 格式输出 |

### 2.2 不需要修改的功能

- 核心协议实现 (在 sing-box 中)
- TLS/Reality 配置 (复用现有)
- Transport 配置 (复用现有)

---

## 3. 后端修改

### 3.1 链接生成 (`util/genLink.go`)

#### 3.1.1 添加 UAP 到支持列表

```go
// util/genLink.go:14
var InboundTypeWithLink = []string{
    "socks", "http", "mixed", "shadowsocks", "naive",
    "hysteria", "hysteria2", "anytls", "tuic",
    "vless", "trojan", "vmess",
    "uap",  // 新增
}
```

#### 3.1.2 添加 UAP case

```go
// util/genLink.go - LinkGenerator 函数 switch 语句中添加
case "uap":
    return uapLink(userConfig["uap"], *inbound, Addrs)
```

#### 3.1.3 新增 uapLink 函数

```go
// util/genLink.go - 新增函数

func uapLink(
    userConfig map[string]interface{},
    inbound map[string]interface{},
    addrs []map[string]interface{}) []string {

    uuid, _ := userConfig["uuid"].(string)
    baseParams := getTransportParams(inbound["transport"])
    var links []string

    for _, addr := range addrs {
        params := baseParams
        if tls, ok := addr["tls"].(map[string]interface{}); ok && tls["enabled"].(bool) {
            getTlsParams(&params, tls, "allowInsecure")
            if flow, ok := userConfig["flow"].(string); ok {
                params["flow"] = flow
            }
        }
        port, _ := addr["server_port"].(float64)
        uri := fmt.Sprintf("uap://%s@%s:%.0f", uuid, addr["server"].(string), port)
        uri = addParams(uri, params, addr["remark"].(string))
        links = append(links, uri)
    }

    return links
}
```

### 3.2 UAP 链接格式

```
uap://<uuid>@<host>:<port>?<params>#<remark>

参数说明:
- security: tls | reality
- sni: TLS SNI
- fp: uTLS fingerprint (chrome, firefox, safari, etc.)
- pbk: Reality public key
- sid: Reality short ID
- flow: xtls-rprx-vision
- type: tcp | ws | grpc | httpupgrade
- host: HTTP/WS host
- path: HTTP/WS path
- serviceName: gRPC service name
```

**示例:**

```
# TLS + Vision
uap://uuid@example.com:443?security=tls&sni=example.com&flow=xtls-rprx-vision&fp=chrome#MyServer

# Reality + Vision
uap://uuid@example.com:443?security=reality&pbk=xxx&sid=abc&flow=xtls-rprx-vision&fp=chrome#MyServer

# WebSocket
uap://uuid@example.com:443?security=tls&sni=example.com&type=ws&path=/ws&host=example.com#MyServer
```

---

## 4. 前端修改

### 4.1 Inbound 类型配置

#### 4.1.1 添加 UAP 类型选项

修改 `frontend/src/types/inbound.ts` (或相应类型文件):

```typescript
export type InboundType =
    | 'direct' | 'socks' | 'http' | 'mixed'
    | 'shadowsocks' | 'vmess' | 'vless' | 'trojan'
    | 'naive' | 'hysteria' | 'hysteria2' | 'tuic' | 'anytls'
    | 'tun' | 'redirect' | 'tproxy'
    | 'uap'  // 新增
```

#### 4.1.2 Inbound 编辑弹窗

修改 `frontend/src/layouts/modals/Inbound.vue`:

```vue
<template>
  <!-- 类型选择下拉框添加 UAP -->
  <v-select
    v-model="inbound.type"
    :items="inboundTypes"
    label="Type"
  />

  <!-- UAP 配置区域 (复用 VLESS 配置) -->
  <template v-if="inbound.type === 'uap'">
    <!-- 与 VLESS 相同的配置项 -->
    <v-text-field v-model="user.uuid" label="UUID" />
    <v-select
      v-model="user.flow"
      :items="flowOptions"
      label="Flow"
      clearable
    />
  </template>
</template>

<script setup>
const inboundTypes = [
  // ... 现有类型 ...
  { title: 'UAP', value: 'uap' },
]

const flowOptions = [
  { title: 'None', value: '' },
  { title: 'xtls-rprx-vision', value: 'xtls-rprx-vision' },
]
</script>
```

### 4.2 Client 配置

修改 `frontend/src/layouts/modals/Client.vue`:

Client 的协议配置中添加 UAP:

```typescript
// 默认 Client 配置结构
const defaultConfig = {
  // 现有协议配置...
  vless: { uuid: '', flow: '' },
  vmess: { uuid: '' },
  trojan: { password: '' },
  // 新增 UAP
  uap: { uuid: '', flow: '' },
}
```

**配置界面** (与 VLESS 相同):

```
┌─────────────────────────────────────────────────┐
│  UAP 配置                                        │
├─────────────────────────────────────────────────┤
│  UUID: [________________________________]       │
│        [Generate UUID]                          │
│                                                  │
│  Flow: [xtls-rprx-vision ▼]                     │
│        (留空表示不使用 Vision)                   │
└─────────────────────────────────────────────────┘
```

### 4.3 i18n 国际化

**en.json:**
```json
{
  "inbound": {
    "types": {
      "uap": "UAP"
    }
  },
  "client": {
    "uap": {
      "uuid": "UUID",
      "flow": "Flow"
    }
  }
}
```

**zh-Hans.json:**
```json
{
  "inbound": {
    "types": {
      "uap": "UAP"
    }
  },
  "client": {
    "uap": {
      "uuid": "用户 ID",
      "flow": "流控"
    }
  }
}
```

---

## 5. 订阅输出支持

### 5.1 修改 linkService.go

`sub/linkService.go` 中的 `addClientInfo` 函数需要支持 UAP 协议:

```go
func (s *LinkService) addClientInfo(uri string, clientInfo string) string {
    if len(clientInfo) == 0 {
        return uri
    }
    protocol := strings.Split(uri, "://")
    if len(protocol) < 2 {
        return uri
    }
    switch protocol[0] {
    case "vmess":
        // ... 现有 vmess 处理 ...
    case "vless", "uap":  // UAP 与 VLESS 格式相同
        // URL 格式，直接在 fragment 中追加信息
        return uri + clientInfo
    default:
        return uri + clientInfo
    }
}
```

### 5.2 Clash 订阅格式

UAP 协议在 Clash 订阅中的输出格式 (如果支持):

```yaml
proxies:
  - name: "UAP-Server"
    type: uap          # 需要 Clash 客户端支持
    server: example.com
    port: 443
    uuid: xxxx-xxxx-xxxx
    flow: xtls-rprx-vision
    tls: true
    servername: example.com
    fingerprint: chrome
    reality-opts:
      public-key: xxx
      short-id: abc
```

> **注意**: 标准 Clash 不支持 UAP，需要使用支持 UAP 的客户端 (如自定义 Clash Meta 或专用客户端)。

---

## 6. 代码结构

### 6.1 修改文件清单

| 文件 | 修改内容 |
|------|----------|
| `util/genLink.go` | + InboundTypeWithLink 添加 "uap"<br>+ LinkGenerator 添加 case<br>+ 新增 uapLink 函数 |
| `sub/linkService.go` | + addClientInfo 支持 uap 协议 |
| `frontend/src/types/inbound.ts` | + UAP 类型定义 |
| `frontend/src/layouts/modals/Inbound.vue` | + UAP 入站配置 |
| `frontend/src/layouts/modals/Client.vue` | + UAP 用户配置 |
| `frontend/src/locales/en.json` | + UAP 翻译 |
| `frontend/src/locales/zh-Hans.json` | + UAP 翻译 |

### 6.2 与现有代码的关系

```
                    ┌─────────────────┐
                    │   sing-box      │
                    │  (hiddify版)    │
                    │                 │
                    │  ┌───────────┐  │
                    │  │ UAP 协议  │  │
                    │  │ 实现      │  │
                    │  └───────────┘  │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │      S-UI       │
                    │                 │
   ┌────────────────┼─────────────────┼────────────────┐
   │                │                 │                │
   ▼                ▼                 ▼                ▼
┌──────┐      ┌──────────┐      ┌──────────┐     ┌──────────┐
│ 前端 │      │ Inbound  │      │  Client  │     │   订阅   │
│ 配置 │      │   配置   │      │   配置   │     │   输出   │
└──────┘      └──────────┘      └──────────┘     └──────────┘
   │                │                 │                │
   │                │                 │                │
   ▼                ▼                 ▼                ▼
┌──────────────────────────────────────────────────────────┐
│                      genLink.go                          │
│                                                          │
│  uapLink() ─────► uap://uuid@host:port?params#remark    │
└──────────────────────────────────────────────────────────┘
```

---

## 7. 实现计划

### Phase 1: 后端链接生成

- [ ] 修改 `util/genLink.go` 添加 UAP 支持
- [ ] 修改 `sub/linkService.go` 支持 UAP 协议
- [ ] 单元测试

### Phase 2: 前端 Inbound 配置

- [ ] 添加 UAP 类型到 Inbound 下拉选项
- [ ] 复用 VLESS 的配置界面组件
- [ ] 添加 i18n 翻译

### Phase 3: 前端 Client 配置

- [ ] Client 配置中添加 UAP 协议
- [ ] UUID 生成和 Flow 选择
- [ ] 添加 i18n 翻译

### Phase 4: 测试验证

- [ ] 创建 UAP Inbound
- [ ] 创建 Client 并关联 UAP Inbound
- [ ] 验证链接生成格式
- [ ] 验证订阅输出
- [ ] 使用支持 UAP 的客户端测试连接

---

## 8. 测试用例

### 8.1 链接生成测试

**输入:**
```json
{
  "inbound": {
    "type": "uap",
    "tag": "uap-in",
    "listen_port": 443
  },
  "tls": {
    "enabled": true,
    "server_name": "example.com",
    "reality": {
      "enabled": true,
      "public_key": "abc123",
      "short_id": ["def"]
    }
  },
  "client": {
    "uap": {
      "uuid": "550e8400-e29b-41d4-a716-446655440000",
      "flow": "xtls-rprx-vision"
    }
  }
}
```

**期望输出:**
```
uap://550e8400-e29b-41d4-a716-446655440000@example.com:443?security=reality&pbk=abc123&sid=def&flow=xtls-rprx-vision&fp=chrome#uap-in
```

### 8.2 配置生成测试

确保 S-UI 生成的 sing-box 配置包含正确的 UAP inbound:

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
          "private_key": "xxx",
          "short_id": ["def"]
        }
      }
    }
  ]
}
```

---

## 附录

### A. UAP 协议源码位置 (hiddify-sing-box)

```
hiddify-sing-box/
├── constant/
│   └── proxy.go              # TypeUAP = "uap"
├── inbound/
│   └── uap.go                # UAP Inbound 实现
├── outbound/
│   └── uap.go                # UAP Outbound 实现
├── option/
│   └── uap.go                # UAP 配置选项
├── protocol/
│   └── uap/
│       ├── client.go         # 客户端协议
│       ├── constant.go       # TLS 常量
│       ├── protocol.go       # 协议编解码
│       ├── service.go        # 服务端协议
│       ├── vision.go         # XTLS Vision
│       ├── vision_reality.go # Reality 支持
│       └── vision_utls.go    # uTLS 支持
└── docs/
    └── configuration/
        ├── inbound/uap.md    # Inbound 配置文档
        └── outbound/uap.md   # Outbound 配置文档
```

### B. 参考链接

- [VLESS 协议规范](https://xtls.github.io/development/protocols/vless.html)
- [Reality 协议](https://github.com/XTLS/REALITY)
- [sing-box 文档](https://sing-box.sagernet.org/)
