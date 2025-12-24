package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/database/model"
	"github.com/alireza0/s-ui/logger"
	"github.com/alireza0/s-ui/util/common"

	"gorm.io/gorm"
)

// NodeService 节点管理服务
type NodeService struct{}

// ========== Token 管理 ==========

// GenerateToken 生成节点邀请码
func (s *NodeService) GenerateToken(name string, expiresAt int64) (*model.NodeToken, error) {
	token := generateSecureToken(32)
	nodeToken := &model.NodeToken{
		Token:     token,
		Name:      name,
		ExpiresAt: expiresAt,
	}
	db := database.GetDB()
	err := db.Create(nodeToken).Error
	if err != nil {
		return nil, err
	}
	return nodeToken, nil
}

// GetTokens 获取所有邀请码
func (s *NodeService) GetTokens() ([]model.NodeToken, error) {
	db := database.GetDB()
	var tokens []model.NodeToken
	err := db.Model(model.NodeToken{}).Order("created_at DESC").Find(&tokens).Error
	if err != nil {
		return nil, err
	}
	return tokens, nil
}

// DeleteToken 删除邀请码
func (s *NodeService) DeleteToken(id uint) error {
	db := database.GetDB()
	return db.Where("id = ?", id).Delete(&model.NodeToken{}).Error
}

// ValidateToken 验证邀请码有效性
func (s *NodeService) ValidateToken(token string) (*model.NodeToken, error) {
	db := database.GetDB()
	var nodeToken model.NodeToken
	err := db.Where("token = ?", token).First(&nodeToken).Error
	if err != nil {
		return nil, common.NewError("invalid token")
	}

	// 检查是否已使用
	if nodeToken.Used {
		return nil, common.NewError("token already used")
	}

	// 检查是否过期
	if nodeToken.ExpiresAt > 0 && nodeToken.ExpiresAt < time.Now().Unix() {
		return nil, common.NewError("token expired")
	}

	return &nodeToken, nil
}

// MarkTokenUsed 标记 Token 为已使用
func (s *NodeService) MarkTokenUsed(tx *gorm.DB, tokenId uint, nodeId string) error {
	return tx.Model(&model.NodeToken{}).Where("id = ?", tokenId).Updates(map[string]interface{}{
		"used":    true,
		"used_by": nodeId,
	}).Error
}

// ========== Node 管理 ==========

// GetNodes 获取所有节点
func (s *NodeService) GetNodes() ([]model.Node, error) {
	db := database.GetDB()
	var nodes []model.Node
	err := db.Model(model.Node{}).Order("created_at DESC").Find(&nodes).Error
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

// GetNode 根据 ID 获取节点
func (s *NodeService) GetNode(id uint) (*model.Node, error) {
	db := database.GetDB()
	var node model.Node
	err := db.Where("id = ?", id).First(&node).Error
	if err != nil {
		return nil, err
	}
	return &node, nil
}

// GetNodeByNodeId 根据 NodeId 获取节点
func (s *NodeService) GetNodeByNodeId(nodeId string) (*model.Node, error) {
	db := database.GetDB()
	var node model.Node
	err := db.Where("node_id = ?", nodeId).First(&node).Error
	if err != nil {
		return nil, err
	}
	return &node, nil
}

// GetEnabledOnlineNodes 获取启用且在线的节点
func (s *NodeService) GetEnabledOnlineNodes() ([]model.Node, error) {
	db := database.GetDB()
	var nodes []model.Node
	err := db.Where("enable = ? AND status = ?", true, "online").Find(&nodes).Error
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

// RegisterNode 注册节点 (从节点调用)
func (s *NodeService) RegisterNode(token, nodeId, name, address, externalHost string, externalPort int, version string) (*model.Node, error) {
	// 验证 Token
	nodeToken, err := s.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	// 检查 NodeId 是否已存在
	db := database.GetDB()
	var existingNode model.Node
	err = db.Where("node_id = ?", nodeId).First(&existingNode).Error
	if err == nil {
		return nil, common.NewError("node id already registered")
	}

	// 如果 Token 有预设名称且未提供名称，使用预设名称
	if name == "" && nodeToken.Name != "" {
		name = nodeToken.Name
	}
	if name == "" {
		name = nodeId
	}

	// 开始事务
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 创建节点
	node := &model.Node{
		NodeId:       nodeId,
		Name:         name,
		Address:      address,
		ExternalHost: externalHost,
		ExternalPort: externalPort,
		Token:        token,
		Status:       "online",
		LastSeen:     time.Now().Unix(),
		Version:      version,
	}

	err = tx.Create(node).Error
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// 标记 Token 为已使用
	err = s.MarkTokenUsed(tx, nodeToken.Id, nodeId)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	tx.Commit()
	logger.Info("Node registered: ", nodeId, " (", name, ")")
	return node, nil
}

// UpdateNode 更新节点信息 (管理 API)
func (s *NodeService) UpdateNode(id uint, data map[string]interface{}) error {
	db := database.GetDB()
	return db.Model(&model.Node{}).Where("id = ?", id).Updates(data).Error
}

// DeleteNode 删除节点
func (s *NodeService) DeleteNode(id uint) error {
	db := database.GetDB()
	return db.Where("id = ?", id).Delete(&model.Node{}).Error
}

// Save 保存节点 (管理 API 调用)
func (s *NodeService) Save(tx *gorm.DB, act string, data json.RawMessage) error {
	var err error

	switch act {
	case "edit":
		var node model.Node
		err = json.Unmarshal(data, &node)
		if err != nil {
			return err
		}
		// 只允许更新部分字段
		err = tx.Model(&model.Node{}).Where("id = ?", node.Id).Updates(map[string]interface{}{
			"name":          node.Name,
			"external_host": node.ExternalHost,
			"external_port": node.ExternalPort,
			"enable":        node.Enable,
			"country":       node.Country,
			"city":          node.City,
			"flag":          node.Flag,
			"is_premium":    node.IsPremium,
		}).Error
	case "del":
		var id uint
		err = json.Unmarshal(data, &id)
		if err != nil {
			return err
		}
		// 先获取节点信息
		var node model.Node
		if err = tx.Where("id = ?", id).First(&node).Error; err != nil {
			return err
		}
		// 删除相关数据
		tx.Where("node_id = ?", node.NodeId).Delete(&model.ClientOnline{})
		tx.Where("node_id = ?", node.NodeId).Delete(&model.NodeStats{})
		// 可选：重置 token 允许复用
		tx.Model(&model.NodeToken{}).Where("used_by = ?", node.NodeId).Updates(map[string]interface{}{
			"used":    false,
			"used_by": "",
		})
		// 删除节点
		err = tx.Where("id = ?", id).Delete(&model.Node{}).Error
	default:
		return common.NewErrorf("unknown action: %s", act)
	}

	return err
}

// ========== 认证 ==========

// AuthenticateNode 验证节点请求
func (s *NodeService) AuthenticateNode(nodeId, token string) (*model.Node, error) {
	db := database.GetDB()
	var node model.Node
	err := db.Where("node_id = ? AND token = ?", nodeId, token).First(&node).Error
	if err != nil {
		return nil, common.NewError("invalid node credentials")
	}
	if !node.Enable {
		return nil, common.NewError("node is disabled")
	}
	return &node, nil
}

// ========== 心跳和状态 ==========

// Heartbeat 处理节点心跳
func (s *NodeService) Heartbeat(nodeId string, cpu, memory float64, connections int, version, externalHost string, externalPort int) error {
	db := database.GetDB()
	systemInfo, _ := json.Marshal(map[string]interface{}{
		"cpu":         cpu,
		"memory":      memory,
		"connections": connections,
	})
	updates := map[string]interface{}{
		"status":        "online",
		"last_seen":     time.Now().Unix(),
		"version":       version,
		"system_info":   systemInfo,
		"external_port": externalPort,
	}
	// 更新外部地址（如果提供）
	if externalHost != "" {
		updates["external_host"] = externalHost
	}
	// 使用 Select 强制更新 external_port（即使为 0）
	return db.Model(&model.Node{}).Where("node_id = ?", nodeId).
		Select("status", "last_seen", "version", "system_info", "external_port", "external_host").
		Updates(updates).Error
}

// UpdateNodeStatus 更新节点状态 (定时任务调用)
func (s *NodeService) UpdateNodeStatus() error {
	db := database.GetDB()
	now := time.Now().Unix()

	// 超过 60 秒未心跳，标记为 offline
	err := db.Model(&model.Node{}).
		Where("status = ? AND last_seen < ?", "online", now-60).
		Update("status", "offline").Error
	if err != nil {
		return err
	}

	// 超过 5 分钟未心跳，标记为 error
	err = db.Model(&model.Node{}).
		Where("status = ? AND last_seen < ?", "offline", now-300).
		Update("status", "error").Error
	return err
}

// ========== 配置同步 ==========

// GetConfigVersion 获取配置版本 (使用 LastUpdate 时间戳)
func (s *NodeService) GetConfigVersion() int64 {
	return LastUpdate
}

// GetFullConfig 获取完整配置 (从节点同步用)
func (s *NodeService) GetFullConfig() (map[string]interface{}, error) {
	db := database.GetDB()
	result := make(map[string]interface{})

	// 获取 Inbounds
	var inbounds []model.Inbound
	if err := db.Preload("Tls").Find(&inbounds).Error; err != nil {
		return nil, err
	}
	result["inbounds"] = inbounds

	// 获取 Outbounds
	var outbounds []model.Outbound
	if err := db.Find(&outbounds).Error; err != nil {
		return nil, err
	}
	result["outbounds"] = outbounds

	// 获取 Clients
	var clients []model.Client
	if err := db.Find(&clients).Error; err != nil {
		return nil, err
	}
	result["clients"] = clients

	// 获取 TLS
	var tls []model.Tls
	if err := db.Find(&tls).Error; err != nil {
		return nil, err
	}
	result["tls"] = tls

	// 获取 Services
	var services []model.Service
	if err := db.Find(&services).Error; err != nil {
		return nil, err
	}
	result["services"] = services

	// 获取 Endpoints
	var endpoints []model.Endpoint
	if err := db.Find(&endpoints).Error; err != nil {
		return nil, err
	}
	result["endpoints"] = endpoints

	// 获取系统配置
	var settingService SettingService
	configStr, err := settingService.GetConfig()
	if err != nil {
		return nil, err
	}
	result["config"] = json.RawMessage(configStr)

	// 版本号
	result["version"] = LastUpdate

	return result, nil
}

// UpdateLastSync 更新节点最后同步时间
func (s *NodeService) UpdateLastSync(nodeId string) error {
	db := database.GetDB()
	return db.Model(&model.Node{}).Where("node_id = ?", nodeId).Update("last_sync", time.Now().Unix()).Error
}

// ========== 统计上报 ==========

// SaveNodeStats 保存从节点上报的统计数据
func (s *NodeService) SaveNodeStats(nodeId string, stats []model.Stats) error {
	if len(stats) == 0 {
		return nil
	}

	db := database.GetDB()
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 使用索引访问以修改原始切片
	for i := range stats {
		stats[i].NodeId = nodeId
		// 更新 Client 流量
		if stats[i].Resource == "user" {
			if stats[i].Direction {
				err := tx.Model(&model.Client{}).Where("name = ?", stats[i].Tag).
					UpdateColumn("up", gorm.Expr("up + ?", stats[i].Traffic)).Error
				if err != nil {
					tx.Rollback()
					return err
				}
			} else {
				err := tx.Model(&model.Client{}).Where("name = ?", stats[i].Tag).
					UpdateColumn("down", gorm.Expr("down + ?", stats[i].Traffic)).Error
				if err != nil {
					tx.Rollback()
					return err
				}
			}
		}
	}

	// 保存统计记录
	if err := tx.Create(&stats).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

// SaveOnlineStatus 保存从节点上报的在线状态
func (s *NodeService) SaveOnlineStatus(nodeId string, onlines []model.ClientOnline) error {
	db := database.GetDB()
	now := time.Now().Unix()

	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除该节点的旧在线记录
	if err := tx.Where("node_id = ?", nodeId).Delete(&model.ClientOnline{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 插入新的在线记录
	for i := range onlines {
		onlines[i].NodeId = nodeId
		onlines[i].LastSeen = now
	}

	if len(onlines) > 0 {
		if err := tx.Create(&onlines).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	tx.Commit()
	return nil
}

// ========== 辅助函数 ==========

// generateSecureToken 生成安全随机 Token
func generateSecureToken(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// 如果加密随机数生成失败，使用时间戳作为后备方案
		// 这种情况极少发生，通常表示系统熵池耗尽
		logger.Warning("crypto/rand failed, using fallback: ", err)
		return hex.EncodeToString([]byte(time.Now().String()))[:length*2]
	}
	return hex.EncodeToString(bytes)
}
