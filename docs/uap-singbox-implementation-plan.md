# UAP 协议 sing-box 实现计划

> **关联技术方案**: [uap-singbox-implementation.md](./uap-singbox-implementation.md)
> **仓库地址**: https://git.uap.io/uap/uap-sing-box
> **最后更新**: 2025-12-22

## 实现阶段

### Step 1: 仓库准备

> 参考: [4. 基于官方 sing-box 实现 UAP](./uap-singbox-implementation.md#4-基于官方-sing-box-实现-uap)

- [x] Fork 官方 sing-box 到 uap-sing-box
- [ ] 设置 CI/CD 流程

### Step 2: 常量定义

> 参考: [4.3.1 常量定义](./uap-singbox-implementation.md#431-常量定义)

- [ ] 修改 `constant/proxy.go` 添加 `TypeUAP = "uap"`

### Step 3: 协议实现

> 参考: [4.1 需要新增的文件](./uap-singbox-implementation.md#41-需要新增的文件)

- [ ] 创建 `protocol/uap/` 目录
- [ ] 实现 `protocol/uap/conn.go` (连接处理)
- [ ] 实现 `protocol/uap/inbound.go` (Inbound 实现)
- [ ] 实现 `protocol/uap/outbound.go` (Outbound 实现)
- [ ] 实现 `protocol/uap/packet.go` (数据包处理)

### Step 4: 配置选项

> 参考: [4.3.2 配置选项](./uap-singbox-implementation.md#432-配置选项)

- [ ] 创建 `option/uap.go` (UAPInboundOptions, UAPOutboundOptions)

### Step 5: 注册 UAP

> 参考: [4.2 需要修改的文件](./uap-singbox-implementation.md#42-需要修改的文件)

- [ ] 修改 `option/outbound.go` 添加 UAPOutboundOptions
- [ ] 修改 `include/outbound_default.go` 注册 UAP outbound
- [ ] 创建 `include/uap.go` (UAP 构建标签)

### Step 6: 编译测试

> 参考: [7. 编译与部署](./uap-singbox-implementation.md#7-编译与部署)

- [ ] 编译 sing-box (含 UAP)
- [ ] 验证 `./sing-box version` 输出

### Step 7: 功能验证

> 参考: [6. UAP 配置示例](./uap-singbox-implementation.md#6-uap-配置示例)

- [ ] 创建测试 Inbound 配置
- [ ] 创建测试 Outbound 配置
- [ ] 验证连接建立
- [ ] 验证数据传输
- [ ] 验证 Vision flow
- [ ] 验证 Reality 握手

---

## 进度统计

| 步骤 | 任务数 | 完成数 | 进度 |
|------|--------|--------|------|
| Step 1 | 2 | 1 | 50% |
| Step 2 | 1 | 0 | 0% |
| Step 3 | 5 | 0 | 0% |
| Step 4 | 1 | 0 | 0% |
| Step 5 | 3 | 0 | 0% |
| Step 6 | 2 | 0 | 0% |
| Step 7 | 6 | 0 | 0% |
| **总计** | **20** | **1** | **5%** |
