package service

import (
	"time"

	"github.com/alireza0/s-ui/config"
	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/database/model"

	"gorm.io/gorm"
)

type onlines struct {
	Inbound  []string `json:"inbound,omitempty"`
	User     []string `json:"user,omitempty"`
	Outbound []string `json:"outbound,omitempty"`
}

var onlineResources = &onlines{}

type StatsService struct {
}

func (s *StatsService) SaveStats(enableTraffic bool) error {
	if !corePtr.IsRunning() {
		return nil
	}
	stats := corePtr.GetInstance().StatsTracker().GetStats()

	// Reset onlines
	onlineResources.Inbound = nil
	onlineResources.Outbound = nil
	onlineResources.User = nil

	if len(*stats) == 0 {
		return nil
	}

	// 获取当前节点 ID
	nodeId := getLocalNodeId()

	var err error
	db := database.GetDB()
	tx := db.Begin()
	defer func() {
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	for i := range *stats {
		stat := &(*stats)[i]
		// 设置节点 ID
		stat.NodeId = nodeId

		if stat.Resource == "user" {
			if stat.Direction {
				err = tx.Model(model.Client{}).Where("name = ?", stat.Tag).
					UpdateColumn("up", gorm.Expr("up + ?", stat.Traffic)).Error
			} else {
				err = tx.Model(model.Client{}).Where("name = ?", stat.Tag).
					UpdateColumn("down", gorm.Expr("down + ?", stat.Traffic)).Error
			}
			if err != nil {
				return err
			}
		}
		if stat.Direction {
			switch stat.Resource {
			case "inbound":
				onlineResources.Inbound = append(onlineResources.Inbound, stat.Tag)
			case "outbound":
				onlineResources.Outbound = append(onlineResources.Outbound, stat.Tag)
			case "user":
				onlineResources.User = append(onlineResources.User, stat.Tag)
			}
		}
	}

	if !enableTraffic {
		return nil
	}
	return tx.Create(&stats).Error
}

// getLocalNodeId 获取本地节点 ID
func getLocalNodeId() string {
	if config.IsWorker() {
		return config.GetNodeId()
	}
	return "local"
}

func (s *StatsService) GetStats(resource string, tag string, limit int) ([]model.Stats, error) {
	var err error
	var result []model.Stats

	currentTime := time.Now().Unix()
	timeDiff := currentTime - (int64(limit) * 3600)

	db := database.GetDB()
	resources := []string{resource}
	if resource == "endpoint" {
		resources = []string{"inbound", "outbound"}
	}
	err = db.Model(model.Stats{}).Where("resource in ? AND tag = ? AND date_time > ?", resources, tag, timeDiff).Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *StatsService) GetOnlines() (onlines, error) {
	return *onlineResources, nil
}

// GetAllOnlines 获取所有节点的在线状态 (主节点模式)
func (s *StatsService) GetAllOnlines() (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// 本地在线状态
	result["local"] = onlineResources

	// 如果是主节点，还需要获取从节点的在线状态
	if config.IsMaster() {
		db := database.GetDB()
		var clientOnlines []model.ClientOnline
		// 获取最近 60 秒内的在线记录
		threshold := time.Now().Unix() - 60
		err := db.Where("last_seen > ?", threshold).Find(&clientOnlines).Error
		if err != nil {
			return nil, err
		}

		// 按节点分组
		nodeOnlines := make(map[string]*onlines)
		for _, co := range clientOnlines {
			if _, ok := nodeOnlines[co.NodeId]; !ok {
				nodeOnlines[co.NodeId] = &onlines{}
			}
			// 用户在线
			nodeOnlines[co.NodeId].User = appendUnique(nodeOnlines[co.NodeId].User, co.ClientName)
			// 入站在线
			if co.InboundTag != "" {
				nodeOnlines[co.NodeId].Inbound = appendUnique(nodeOnlines[co.NodeId].Inbound, co.InboundTag)
			}
		}

		for nodeId, online := range nodeOnlines {
			result[nodeId] = online
		}
	}

	return result, nil
}

// GetStatsByNode 按节点获取统计
func (s *StatsService) GetStatsByNode(resource string, tag string, nodeId string, limit int) ([]model.Stats, error) {
	var result []model.Stats

	currentTime := time.Now().Unix()
	timeDiff := currentTime - (int64(limit) * 3600)

	db := database.GetDB()
	resources := []string{resource}
	if resource == "endpoint" {
		resources = []string{"inbound", "outbound"}
	}

	query := db.Model(model.Stats{}).Where("resource in ? AND tag = ? AND date_time > ?", resources, tag, timeDiff)
	if nodeId != "" {
		query = query.Where("node_id = ?", nodeId)
	}

	err := query.Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetClientOnlineDevices 获取客户端在线设备 (UAP 设备限制用)
func (s *StatsService) GetClientOnlineDevices(clientName string) ([]model.ClientOnline, error) {
	db := database.GetDB()
	var onlines []model.ClientOnline
	threshold := time.Now().Unix() - 60
	err := db.Where("client_name = ? AND last_seen > ?", clientName, threshold).Find(&onlines).Error
	if err != nil {
		return nil, err
	}
	return onlines, nil
}

// GetUniqueDeviceCount 获取客户端唯一设备数
func (s *StatsService) GetUniqueDeviceCount(clientName string) (int, error) {
	db := database.GetDB()
	var count int64
	threshold := time.Now().Unix() - 60
	err := db.Model(&model.ClientOnline{}).
		Where("client_name = ? AND last_seen > ?", clientName, threshold).
		Distinct("source_ip").
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func (s *StatsService) DelOldStats(days int) error {
	oldTime := time.Now().AddDate(0, 0, -(days)).Unix()
	db := database.GetDB()
	return db.Where("date_time < ?", oldTime).Delete(model.Stats{}).Error
}

// DelOldClientOnlines 清理过期的在线记录
func (s *StatsService) DelOldClientOnlines() error {
	// 清理 5 分钟前的记录
	oldTime := time.Now().Unix() - 300
	db := database.GetDB()
	return db.Where("last_seen < ?", oldTime).Delete(&model.ClientOnline{}).Error
}

// appendUnique 追加去重
func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}
