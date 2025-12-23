package service

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"

	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/database/model"
	"github.com/alireza0/s-ui/logger"
	"github.com/alireza0/s-ui/util"
	"github.com/alireza0/s-ui/util/common"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClientService struct{}

func (s *ClientService) Get(id string) (*[]model.Client, error) {
	if id == "" {
		return s.GetAll()
	}
	return s.getById(id)
}

func (s *ClientService) getById(id string) (*[]model.Client, error) {
	db := database.GetDB()
	var client []model.Client
	err := db.Model(model.Client{}).Where("id in ?", strings.Split(id, ",")).Scan(&client).Error
	if err != nil {
		return nil, err
	}

	return &client, nil
}

func (s *ClientService) GetAll() (*[]model.Client, error) {
	db := database.GetDB()
	var clients []model.Client
	err := db.Model(model.Client{}).Select("`id`, `enable`, `name`, `desc`, `group`, `inbounds`, `up`, `down`, `volume`, `expiry`").Scan(&clients).Error
	if err != nil {
		return nil, err
	}
	return &clients, nil
}

func (s *ClientService) Save(tx *gorm.DB, act string, data json.RawMessage, hostname string) ([]uint, error) {
	var err error
	var inboundIds []uint

	switch act {
	case "new", "edit":
		var client model.Client
		err = json.Unmarshal(data, &client)
		if err != nil {
			return nil, err
		}
		// 自动生成 UUID (如果为空)
		if client.UUID == "" {
			client.UUID = uuid.New().String()
		}
		// 同步 UUID 到 Config
		err = s.SyncUUIDToConfig(&client)
		if err != nil {
			return nil, err
		}
		err = s.updateLinksWithFixedInbounds(tx, []*model.Client{&client}, hostname)
		if err != nil {
			return nil, err
		}
		if act == "edit" {
			// Find changed inbounds
			inboundIds, err = s.findInboundsChanges(tx, client)
			if err != nil {
				return nil, err
			}
		} else {
			err = json.Unmarshal(client.Inbounds, &inboundIds)
			if err != nil {
				return nil, err
			}
		}
		err = tx.Save(&client).Error
		if err != nil {
			return nil, err
		}
	case "addbulk":
		var clients []*model.Client
		err = json.Unmarshal(data, &clients)
		if err != nil {
			return nil, err
		}
		// 为每个 client 生成 UUID 并同步到 Config
		for _, client := range clients {
			if client.UUID == "" {
				client.UUID = uuid.New().String()
			}
			err = s.SyncUUIDToConfig(client)
			if err != nil {
				return nil, err
			}
		}
		err = json.Unmarshal(clients[0].Inbounds, &inboundIds)
		if err != nil {
			return nil, err
		}
		err = s.updateLinksWithFixedInbounds(tx, clients, hostname)
		if err != nil {
			return nil, err
		}
		err = tx.Save(clients).Error
		if err != nil {
			return nil, err
		}
	case "del":
		var id uint
		err = json.Unmarshal(data, &id)
		if err != nil {
			return nil, err
		}
		var client model.Client
		err = tx.Where("id = ?", id).First(&client).Error
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(client.Inbounds, &inboundIds)
		if err != nil {
			return nil, err
		}
		err = tx.Where("id = ?", id).Delete(model.Client{}).Error
		if err != nil {
			return nil, err
		}
	default:
		return nil, common.NewErrorf("unknown action: %s", act)
	}

	return inboundIds, nil
}

func (s *ClientService) updateLinksWithFixedInbounds(tx *gorm.DB, clients []*model.Client, hostname string) error {
	var err error
	var inbounds []model.Inbound
	var inboundIds []uint

	err = json.Unmarshal(clients[0].Inbounds, &inboundIds)
	if err != nil {
		return err
	}

	// Zero inbounds means removing local links only
	if len(inboundIds) > 0 {
		err = tx.Model(model.Inbound{}).Preload("Tls").Where("id in ? and type in ?", inboundIds, util.InboundTypeWithLink).Find(&inbounds).Error
		if err != nil {
			return err
		}
	}
	for index, client := range clients {
		var clientLinks []map[string]string
		err = json.Unmarshal(client.Links, &clientLinks)
		if err != nil {
			return err
		}

		newClientLinks := []map[string]string{}
		for _, inbound := range inbounds {
			newLinks := util.LinkGenerator(client.Config, &inbound, hostname)
			for _, newLink := range newLinks {
				newClientLinks = append(newClientLinks, map[string]string{
					"remark": inbound.Tag,
					"type":   "local",
					"uri":    newLink,
				})
			}
		}

		// Add non local links
		for _, clientLink := range clientLinks {
			if clientLink["type"] != "local" {
				newClientLinks = append(newClientLinks, clientLink)
			}
		}

		clients[index].Links, err = json.MarshalIndent(newClientLinks, "", "  ")
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *ClientService) UpdateClientsOnInboundAdd(tx *gorm.DB, initIds string, inboundId uint, hostname string) error {
	clientIds := strings.Split(initIds, ",")
	var clients []model.Client
	err := tx.Model(model.Client{}).Where("id in ?", clientIds).Find(&clients).Error
	if err != nil {
		return err
	}
	var inbound model.Inbound
	err = tx.Model(model.Inbound{}).Preload("Tls").Where("id = ?", inboundId).Find(&inbound).Error
	if err != nil {
		return err
	}
	for _, client := range clients {
		// Add inbounds
		var clientInbounds []uint
		json.Unmarshal(client.Inbounds, &clientInbounds)
		clientInbounds = append(clientInbounds, inboundId)
		client.Inbounds, err = json.MarshalIndent(clientInbounds, "", "  ")
		if err != nil {
			return err
		}
		// Add links
		var clientLinks, newClientLinks []map[string]string
		json.Unmarshal(client.Links, &clientLinks)
		newLinks := util.LinkGenerator(client.Config, &inbound, hostname)
		for _, newLink := range newLinks {
			newClientLinks = append(newClientLinks, map[string]string{
				"remark": inbound.Tag,
				"type":   "local",
				"uri":    newLink,
			})
		}
		for _, clientLink := range clientLinks {
			if clientLink["remark"] != inbound.Tag {
				newClientLinks = append(newClientLinks, clientLink)
			}
		}

		client.Links, err = json.MarshalIndent(newClientLinks, "", "  ")
		if err != nil {
			return err
		}
		err = tx.Save(&client).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *ClientService) UpdateClientsOnInboundDelete(tx *gorm.DB, id uint, tag string) error {
	var clients []model.Client
	err := tx.Table("clients").
		Where("EXISTS (SELECT 1 FROM json_each(clients.inbounds) WHERE json_each.value = ?)", id).
		Find(&clients).Error
	if err != nil {
		return err
	}
	for _, client := range clients {
		// Delete inbounds
		var clientInbounds, newClientInbounds []uint
		json.Unmarshal(client.Inbounds, &clientInbounds)
		for _, clientInbound := range clientInbounds {
			if clientInbound != id {
				newClientInbounds = append(newClientInbounds, clientInbound)
			}
		}
		client.Inbounds, err = json.MarshalIndent(newClientInbounds, "", "  ")
		if err != nil {
			return err
		}
		// Delete links
		var clientLinks, newClientLinks []map[string]string
		json.Unmarshal(client.Links, &clientLinks)
		for _, clientLink := range clientLinks {
			if clientLink["remark"] != tag {
				newClientLinks = append(newClientLinks, clientLink)
			}
		}
		client.Links, err = json.MarshalIndent(newClientLinks, "", "  ")
		if err != nil {
			return err
		}
		err = tx.Save(&client).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *ClientService) UpdateLinksByInboundChange(tx *gorm.DB, inbounds *[]model.Inbound, hostname string, oldTag string) error {
	var err error
	for _, inbound := range *inbounds {
		var clients []model.Client
		err = tx.Table("clients").
			Where("EXISTS (SELECT 1 FROM json_each(clients.inbounds) WHERE json_each.value = ?)", inbound.Id).
			Find(&clients).Error
		if err != nil {
			return err
		}
		for _, client := range clients {
			var clientLinks, newClientLinks []map[string]string
			json.Unmarshal(client.Links, &clientLinks)
			newLinks := util.LinkGenerator(client.Config, &inbound, hostname)
			for _, newLink := range newLinks {
				newClientLinks = append(newClientLinks, map[string]string{
					"remark": inbound.Tag,
					"type":   "local",
					"uri":    newLink,
				})
			}
			for _, clientLink := range clientLinks {
				if clientLink["type"] != "local" || (clientLink["remark"] != inbound.Tag && clientLink["remark"] != oldTag) {
					newClientLinks = append(newClientLinks, clientLink)
				}
			}

			client.Links, err = json.MarshalIndent(newClientLinks, "", "  ")
			if err != nil {
				return err
			}
			err = tx.Save(&client).Error
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *ClientService) DepleteClients() ([]uint, error) {
	var err error
	var clients []model.Client
	var changes []model.Changes
	var users []string
	var inboundIds []uint

	now := time.Now().Unix()
	db := database.GetDB()

	tx := db.Begin()
	defer func() {
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	err = tx.Model(model.Client{}).Where("enable = true AND ((volume >0 AND up+down > volume) OR (expiry > 0 AND expiry < ?))", now).Scan(&clients).Error
	if err != nil {
		return nil, err
	}

	dt := time.Now().Unix()
	for _, client := range clients {
		logger.Debug("Client ", client.Name, " is going to be disabled")
		users = append(users, client.Name)
		var userInbounds []uint
		json.Unmarshal(client.Inbounds, &userInbounds)
		// Find changed inbounds
		inboundIds = common.UnionUintArray(inboundIds, userInbounds)
		changes = append(changes, model.Changes{
			DateTime: dt,
			Actor:    "DepleteJob",
			Key:      "clients",
			Action:   "disable",
			Obj:      json.RawMessage("\"" + client.Name + "\""),
		})
	}

	// Save changes
	if len(changes) > 0 {
		err = tx.Model(model.Client{}).Where("enable = true AND ((volume >0 AND up+down > volume) OR (expiry > 0 AND expiry < ?))", now).Update("enable", false).Error
		if err != nil {
			return nil, err
		}
		err = tx.Model(model.Changes{}).Create(&changes).Error
		if err != nil {
			return nil, err
		}
		LastUpdate = dt
	}

	return inboundIds, nil
}

func (s *ClientService) findInboundsChanges(tx *gorm.DB, client model.Client) ([]uint, error) {
	var err error
	var oldClient model.Client
	var oldInboundIds, newInboundIds []uint
	err = tx.Model(model.Client{}).Where("id = ?", client.Id).First(&oldClient).Error
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(oldClient.Inbounds, &oldInboundIds)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(client.Inbounds, &newInboundIds)
	if err != nil {
		return nil, err
	}

	// Check client.Config changes
	if !bytes.Equal(oldClient.Config, client.Config) ||
		oldClient.Name != client.Name ||
		oldClient.Enable != client.Enable {
		return common.UnionUintArray(oldInboundIds, newInboundIds), nil
	}

	// Check client.Inbounds changes
	diffInbounds := common.DiffUintArray(oldInboundIds, newInboundIds)

	return diffInbounds, nil
}

// SyncUUIDToConfig 将 Client.UUID 同步到 Config 中各协议的 uuid 字段
// 支持的协议: vless, vmess, tuic, uap
func (s *ClientService) SyncUUIDToConfig(client *model.Client) error {
	if client.UUID == "" {
		return nil
	}

	var config map[string]map[string]interface{}
	err := json.Unmarshal(client.Config, &config)
	if err != nil {
		// 如果解析失败，可能是空 config，跳过同步
		return nil
	}

	// 同步 UUID 到支持 UUID 的协议
	for _, proto := range []string{"vless", "vmess", "tuic", "uap"} {
		if cfg, ok := config[proto]; ok {
			cfg["uuid"] = client.UUID
		}
	}

	client.Config, err = json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return nil
}

// GetByUUID 通过 UUID 获取 Client
func (s *ClientService) GetByUUID(uuid string) (*model.Client, error) {
	db := database.GetDB()
	var client model.Client
	err := db.Model(model.Client{}).Where("uuid = ?", uuid).First(&client).Error
	if err != nil {
		return nil, err
	}
	return &client, nil
}

// ========== UAP 时长追踪方法 ==========

// UpdateOnlineTime 从 core TimeTracker 获取时长并累加到数据库
func (s *ClientService) UpdateOnlineTime() error {
	if !corePtr.IsRunning() {
		return nil
	}

	// 获取并重置时长追踪数据
	timeData := corePtr.GetInstance().TimeTracker().GetAndResetTime()
	if len(timeData) == 0 {
		return nil
	}

	db := database.GetDB()
	tx := db.Begin()
	var err error
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	for userName, seconds := range timeData {
		err = tx.Model(&model.Client{}).
			Where("name = ?", userName).
			UpdateColumn("time_used", gorm.Expr("time_used + ?", seconds)).Error
		if err != nil {
			return err
		}
	}

	return nil
}

// DepleteTimeExceededClients 禁用时长超限的用户
func (s *ClientService) DepleteTimeExceededClients() ([]uint, []model.Client, error) {
	var err error
	var clients []model.Client
	var changes []model.Changes
	var inboundIds []uint

	db := database.GetDB()
	tx := db.Begin()
	defer func() {
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	// 查找启用且时长超限的用户 (time_limit > 0 且 time_used >= time_limit)
	err = tx.Model(model.Client{}).
		Where("enable = ? AND time_limit > 0 AND time_used >= time_limit", true).
		Scan(&clients).Error
	if err != nil {
		return nil, nil, err
	}

	if len(clients) == 0 {
		return nil, nil, nil
	}

	dt := time.Now().Unix()
	for _, client := range clients {
		logger.Debug("Client ", client.Name, " time limit exceeded, disabling")
		var userInbounds []uint
		json.Unmarshal(client.Inbounds, &userInbounds)
		inboundIds = common.UnionUintArray(inboundIds, userInbounds)
		changes = append(changes, model.Changes{
			DateTime: dt,
			Actor:    "TimeDepleteJob",
			Key:      "clients",
			Action:   "disable",
			Obj:      json.RawMessage("\"" + client.Name + "\""),
		})
	}

	// 禁用超限用户
	err = tx.Model(model.Client{}).
		Where("enable = ? AND time_limit > 0 AND time_used >= time_limit", true).
		Update("enable", false).Error
	if err != nil {
		return nil, nil, err
	}

	// 记录变更
	if len(changes) > 0 {
		err = tx.Model(model.Changes{}).Create(&changes).Error
		if err != nil {
			return nil, nil, err
		}
		LastUpdate = dt
	}

	return inboundIds, clients, nil
}

// ResetTrafficByStrategy 按策略重置流量
func (s *ClientService) ResetTrafficByStrategy() ([]model.Client, []uint, error) {
	var err error
	var clients []model.Client
	var inboundIds []uint

	now := time.Now()
	db := database.GetDB()
	tx := db.Begin()
	defer func() {
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	// 查找需要重置流量的用户
	// traffic_reset_strategy != 'no_reset' AND traffic_reset_at <= now
	err = tx.Model(model.Client{}).
		Where("traffic_reset_strategy != ? AND traffic_reset_strategy != '' AND traffic_reset_at > 0 AND traffic_reset_at <= ?",
			"no_reset", now.Unix()).
		Scan(&clients).Error
	if err != nil {
		return nil, nil, err
	}

	if len(clients) == 0 {
		return nil, nil, nil
	}

	dt := now.Unix()
	var changes []model.Changes
	for _, client := range clients {
		logger.Debug("Resetting traffic for client: ", client.Name)
		var userInbounds []uint
		json.Unmarshal(client.Inbounds, &userInbounds)
		inboundIds = common.UnionUintArray(inboundIds, userInbounds)

		// 计算下次重置时间
		nextResetAt := calculateNextResetTime(client.TrafficResetStrategy, now)

		// 重置流量并更新下次重置时间，同时重新启用用户
		err = tx.Model(&model.Client{}).Where("id = ?", client.Id).Updates(map[string]interface{}{
			"up":               0,
			"down":             0,
			"enable":           true,
			"traffic_reset_at": nextResetAt,
		}).Error
		if err != nil {
			return nil, nil, err
		}

		changes = append(changes, model.Changes{
			DateTime: dt,
			Actor:    "ResetJob",
			Key:      "clients",
			Action:   "reset_traffic",
			Obj:      json.RawMessage("\"" + client.Name + "\""),
		})
	}

	if len(changes) > 0 {
		err = tx.Create(&changes).Error
		if err != nil {
			return nil, nil, err
		}
		LastUpdate = dt
	}

	return clients, inboundIds, nil
}

// ResetTimeByStrategy 按策略重置时长
func (s *ClientService) ResetTimeByStrategy() ([]model.Client, []uint, error) {
	var err error
	var clients []model.Client
	var inboundIds []uint

	now := time.Now()
	db := database.GetDB()
	tx := db.Begin()
	defer func() {
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	// 查找需要重置时长的用户
	err = tx.Model(model.Client{}).
		Where("time_reset_strategy != ? AND time_reset_strategy != '' AND time_reset_at > 0 AND time_reset_at <= ?",
			"no_reset", now.Unix()).
		Scan(&clients).Error
	if err != nil {
		return nil, nil, err
	}

	if len(clients) == 0 {
		return nil, nil, nil
	}

	dt := now.Unix()
	var changes []model.Changes
	for _, client := range clients {
		logger.Debug("Resetting time for client: ", client.Name)
		var userInbounds []uint
		json.Unmarshal(client.Inbounds, &userInbounds)
		inboundIds = common.UnionUintArray(inboundIds, userInbounds)

		// 计算下次重置时间
		nextResetAt := calculateNextResetTime(client.TimeResetStrategy, now)

		// 重置时长并更新下次重置时间，同时重新启用用户
		err = tx.Model(&model.Client{}).Where("id = ?", client.Id).Updates(map[string]interface{}{
			"time_used":     0,
			"enable":        true,
			"time_reset_at": nextResetAt,
		}).Error
		if err != nil {
			return nil, nil, err
		}

		changes = append(changes, model.Changes{
			DateTime: dt,
			Actor:    "ResetJob",
			Key:      "clients",
			Action:   "reset_time",
			Obj:      json.RawMessage("\"" + client.Name + "\""),
		})
	}

	if len(changes) > 0 {
		err = tx.Create(&changes).Error
		if err != nil {
			return nil, nil, err
		}
		LastUpdate = dt
	}

	return clients, inboundIds, nil
}

// calculateNextResetTime 计算下次重置时间
func calculateNextResetTime(strategy string, now time.Time) int64 {
	switch strategy {
	case "daily":
		// 明天同一时间
		return now.AddDate(0, 0, 1).Unix()
	case "weekly":
		// 下周同一时间
		return now.AddDate(0, 0, 7).Unix()
	case "monthly":
		// 下月同一时间
		return now.AddDate(0, 1, 0).Unix()
	case "yearly":
		// 明年同一时间
		return now.AddDate(1, 0, 0).Unix()
	default:
		return 0
	}
}
