# S-UI 技术债务 & 待办事项

> 记录已知但暂不修复的技术问题，按优先级排序

---

## 中等优先级

### 1. SyncService.reportStats 统计数据重复上报/误删风险

**位置**: `service/sync.go:reportStats()`

**问题描述**:
```go
// 获取待上报的统计数据
threshold := time.Now().Unix() - int64(config.GetSyncStatsInterval()*2)
err := db.Where("date_time > ? AND node_id = ?", threshold, config.GetNodeId()).Find(&stats).Error
// ... 上报到主节点 ...
// 上报成功后删除
db.Where("date_time > ? AND node_id = ?", threshold, config.GetNodeId()).Delete(&model.Stats{})
```

在 `Find` 和 `Delete` 之间如果有新数据写入，这些数据会被误删但未上报。

**影响范围**:
- 从节点统计上报 (Worker 模式)
- 仅在上报周期内恰好有新数据写入时发生
- 可能丢失少量流量统计

**建议修复方案**:

1. **方案 A**: 添加 `reported` 标记字段
   ```go
   // model/model.go
   type Stats struct {
       // ...
       Reported bool `json:"reported" gorm:"default:false"`
   }
   ```

2. **方案 B**: 使用 ID 范围删除
   ```go
   // 记录最大 ID
   var maxId uint
   db.Model(&model.Stats{}).Select("MAX(id)").Scan(&maxId)
   // 上报后按 ID 删除
   db.Where("id <= ?", maxId).Delete(&model.Stats{})
   ```

3. **方案 C**: 使用事务 + SELECT FOR UPDATE

**建议修复时间**: Phase 4.5 (CronJob 扩展) 或 Phase 6

**关联文档**:
- [多节点架构技术方案](./multi-node-architecture.md) - 5.3 统计同步流程
- [实现计划](./multi-node-architecture-plan.md) - Phase 4

**记录日期**: 2025-12-23

---

## 低优先级

### 2. ClientService 中 json.Unmarshal 错误被忽略

**位置**: `service/client.go`

**问题描述**:
```go
// line 462, 529, 601
json.Unmarshal(client.Inbounds, &userInbounds)  // 错误被忽略
```

多处 `json.Unmarshal` 调用忽略了返回的错误，如果 `Inbounds` JSON 格式异常会静默失败。

**影响范围**:
- `DepleteTimeExceededClients()` - 时长超限禁用
- `ResetTrafficByStrategy()` - 流量重置
- `ResetTimeByStrategy()` - 时长重置

**实际风险**: 极低
- `Inbounds` 字段由系统内部控制，格式固定为 `[]uint`
- 只有数据库被外部篡改时才可能出错

**建议修复方案**:
```go
if err := json.Unmarshal(client.Inbounds, &userInbounds); err != nil {
    logger.Warning("Failed to parse inbounds for client ", client.Name, ": ", err)
    continue
}
```

**建议修复时间**: 无紧迫性，可在相关代码重构时顺带修复

**关联文档**:
- [实现计划](./multi-node-architecture-plan.md) - Phase 4.5

**记录日期**: 2025-12-23

---

### 3. NodeSelector.vue 节点选择器组件

**位置**: `frontend/src/components/NodeSelector.vue` (待创建)

**功能描述**:
在需要选择节点的地方提供统一的节点选择下拉组件，支持：
- 单选/多选节点
- 显示节点状态 (在线/离线)
- 显示节点标识 (国旗 emoji)

**依赖**: Node store (`frontend/src/store/modules/data.ts`)

**使用场景**:
- 客户端编辑弹窗中选择可用节点
- 批量操作时选择目标节点

**记录日期**: 2025-12-23

---

### 4. Clients.vue 显示 UAP 扩展字段

**位置**: `frontend/src/views/Clients.vue`

**功能描述**:
在客户端列表表格中添加 UAP 相关字段列：
- `timeLimit` - 时长限制
- `timeUsed` - 已用时长
- `trafficResetStrategy` - 流量重置策略
- `timeResetStrategy` - 时长重置策略

**依赖**: Client 模型已包含 UAP 字段 (`frontend/src/types/clients.ts`)

**备注**: 字段已在 Client 编辑弹窗中实现，仅需在列表中展示

**记录日期**: 2025-12-23

---

## 已解决

*(暂无)*
