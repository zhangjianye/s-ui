# S-UI 多节点管理架构实现计划

> **关联技术方案**: [multi-node-architecture.md](./multi-node-architecture.md)
> **最后更新**: 2025-12-23

## 依赖关系

```
Phase 1 ──► Phase 1.5 ──► Phase 2 ──► Phase 3 ──► Phase 4 ──► Phase 4.5 ──► Phase 5 ──► Phase 6
   │            │                         │           │            │
   │            │                         │           │            └─ 依赖 Phase 1.5 (UserTimeTracker)
   │            │                         │           └─ 依赖 Phase 3 (NodeService)
   │            │                         └─ 依赖 Phase 1 (Node 模型)
   │            └─ 依赖 Phase 1 (Client 模型扩展)
   └─ 无前置依赖
```

## 实现阶段

### Phase 1: 数据库与模型

> 参考: [3. 数据库设计](./multi-node-architecture.md#3-数据库设计)
> **依赖**: 无

- [x] 新增 `database/model/node.go` (Node, NodeToken, NodeStats, ClientOnline)
- [x] 修改 `database/model/model.go` (Client 扩展, Stats.NodeId)
- [x] 新增 `database/model/webhook.go` (WebhookConfig, ApiKey)
- [x] 修改 `database/db.go` (AutoMigrate)

### Phase 1.5: 核心层改造 (UAP-Aware)

> 参考: [6. 核心层扩展](./multi-node-architecture.md#6-核心层扩展-uap-aware)
> **依赖**: Phase 1 (Client 模型扩展)

- [x] 修改 `core/tracker_conn.go` (扩展 ConnectionInfo: User, SourceIP, ConnectedAt)
- [x] 修改 `core/tracker_stats.go` (新增 UserTimeTracker)

### Phase 2: 配置与启动

> 参考: [9. 配置说明](./multi-node-architecture.md#9-配置说明)
> **依赖**: Phase 1.5

- [x] 修改 `config/config.go` (节点配置)
- [x] 修改 `cmd/cmd.go` (命令行参数)
- [x] 修改 `app/app.go` (初始化流程)

### Phase 3: 主节点服务

> 参考: [4. 接口设计](./multi-node-architecture.md#4-接口设计)
> **依赖**: Phase 2, Phase 1 (Node 模型)

- [x] 新增 `service/node.go`
- [x] 新增 `api/nodeHandler.go`
- [x] 修改 `service/stats.go`
- [x] 修改 `web/web.go`
- [x] 修改 `api/apiHandler.go`
- [x] 修改 `api/apiService.go`
- [x] 修改 `service/config.go` (nodes Save 支持)

### Phase 4: 从节点同步

> 参考: [5. 核心流程](./multi-node-architecture.md#5-核心流程)
> **依赖**: Phase 3 (NodeService 提供 Token 验证、配置获取)

- [ ] 新增 `service/sync.go`
- [x] 修改 `api/apiHandler.go` (只读检查) - 已在 Phase 3 完成

### Phase 4.5: CronJob 扩展 (UAP-Aware)

> 参考: [6.3 CronJob 扩展](./multi-node-architecture.md#63-cronjob-扩展)
> **依赖**: Phase 1.5 (UserTimeTracker), Phase 4

- [ ] 新增 `cronjob/timeTrackJob.go` (时长追踪任务)
- [ ] 新增 `cronjob/resetJob.go` (重置策略任务)
- [ ] 新增 `service/webhook.go` (Webhook 回调服务)
- [ ] 修改 `cronjob/cronjob.go` (注册新任务)

### Phase 5: 前端改造

> 参考: [8.3 前端 UI 改造说明](./multi-node-architecture.md#83-前端-ui-改造说明)
> **依赖**: Phase 4.5 (后端 API 就绪)

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
- [ ] 修改 `frontend/src/layouts/modals/Client.vue` (添加 UAP 字段)
- [ ] 修改 `frontend/src/views/Clients.vue` (显示 UAP 扩展字段列)
- [ ] 修改 `frontend/src/views/Settings.vue` (添加 Webhook 配置区域)

**国际化 (i18n):**
- [ ] 修改 `frontend/src/locales/en.json` (添加 nodes、apikeys 等翻译)
- [ ] 修改 `frontend/src/locales/zh-Hans.json` (添加节点管理、API 密钥等翻译)

### Phase 6: UAP Backend 对接

> 参考: [13. UAP Backend 对接](./multi-node-architecture.md#13-uap-backend-对接)
> **依赖**: Phase 5

- [ ] 修改 `service/client.go` (UUID 同步到 Config)
- [ ] 修改 `sub/subService.go` (订阅查询改用 UUID)
- [ ] 新增 `api/externalHandler.go` (外部 API)
- [ ] 修改 `web/web.go` (注册外部 API 路由)

---

## 进度统计

| 阶段 | 任务数 | 完成数 | 进度 |
|------|--------|--------|------|
| Phase 1 | 4 | 4 | 100% |
| Phase 1.5 | 2 | 2 | 100% |
| Phase 2 | 3 | 3 | 100% |
| Phase 3 | 7 | 7 | 100% |
| Phase 4 | 2 | 1 | 50% |
| Phase 4.5 | 4 | 0 | 0% |
| Phase 5 | 16 | 0 | 0% |
| Phase 6 | 4 | 0 | 0% |
| **总计** | **42** | **17** | **40%** |
