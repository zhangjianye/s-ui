# S-UI UAP 协议支持实现计划

> **关联技术方案**: [uap-protocol-support.md](./uap-protocol-support.md)
> **最后更新**: 2025-12-22

## 前置依赖

- [ ] uap-sing-box 完成 UAP 协议实现（参考 [uap-singbox-implementation-plan.md](./uap-singbox-implementation-plan.md)）

## 实现阶段

### Phase 1: 后端链接生成

> 参考: [3. 后端修改](./uap-protocol-support.md#3-后端修改)

- [ ] 修改 `util/genLink.go` 添加 UAP 支持
  - [ ] 添加 "uap" 到 InboundTypeWithLink 列表
  - [ ] 添加 uapLink case
  - [ ] 实现 uapLink 函数
- [ ] 修改 `sub/linkService.go` 支持 UAP 协议
- [ ] 编写单元测试

### Phase 2: 前端 Inbound 配置

> 参考: [4. 前端修改](./uap-protocol-support.md#4-前端修改)

- [ ] 添加 UAP 类型到 Inbound 下拉选项
- [ ] 复用 VLESS 的配置界面组件
- [ ] 添加 i18n 翻译 (en.json, zh-Hans.json)

### Phase 3: 前端 Client 配置

> 参考: [4.2 Client 配置](./uap-protocol-support.md#42-client-配置)

- [ ] Client 配置中添加 UAP 协议
- [ ] UUID 生成功能
- [ ] Flow 选择功能
- [ ] 添加 i18n 翻译

### Phase 4: 测试验证

> 参考: [8. 测试用例](./uap-protocol-support.md#8-测试用例)

- [ ] 创建 UAP Inbound
- [ ] 创建 Client 并关联 UAP Inbound
- [ ] 验证链接生成格式
- [ ] 验证订阅输出
- [ ] 使用支持 UAP 的客户端测试连接

---

## 进度统计

| 阶段 | 任务数 | 完成数 | 进度 |
|------|--------|--------|------|
| Phase 1 | 5 | 0 | 0% |
| Phase 2 | 3 | 0 | 0% |
| Phase 3 | 4 | 0 | 0% |
| Phase 4 | 5 | 0 | 0% |
| **总计** | **17** | **0** | **0%** |
