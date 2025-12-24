package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/alireza0/s-ui/config"
	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/database/model"
	"github.com/alireza0/s-ui/logger"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

// SyncService 从节点同步服务
type SyncService struct {
	configService *ConfigService

	masterAddr   string
	nodeId       string
	nodeToken    string
	localVersion int64
	running      bool
	stopChan     chan struct{}
	wg           sync.WaitGroup
	client       *http.Client
	mutex        sync.Mutex
	stopOnce     sync.Once

	// 待上报的统计数据 (上报失败时保留)
	pendingStats []model.Stats
}

// NewSyncService 创建同步服务
func NewSyncService(configService *ConfigService) *SyncService {
	return &SyncService{
		configService: configService,
		masterAddr:    config.GetMasterAddr(),
		nodeId:        config.GetNodeId(),
		nodeToken:     config.GetNodeToken(),
		stopChan:      make(chan struct{}),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Start 启动同步服务
func (s *SyncService) Start() error {
	if !config.IsWorker() {
		return nil
	}

	s.mutex.Lock()
	if s.running {
		s.mutex.Unlock()
		return nil
	}
	s.running = true
	s.mutex.Unlock()

	logger.Info("Starting sync service...")

	// 尝试注册或验证已注册
	if err := s.ensureRegistered(); err != nil {
		logger.Warning("Failed to register with master: ", err)
		// 继续运行，使用本地缓存
	}

	// 首次同步配置
	if err := s.syncConfig(); err != nil {
		logger.Warning("Failed to sync config: ", err)
		// 使用本地缓存继续
	}

	// 启动定时任务
	s.wg.Add(3)
	go s.configSyncLoop()
	go s.statsReportLoop()
	go s.heartbeatLoop()

	return nil
}

// Stop 停止同步服务
func (s *SyncService) Stop() {
	s.stopOnce.Do(func() {
		s.mutex.Lock()
		if !s.running {
			s.mutex.Unlock()
			return
		}
		s.running = false
		s.mutex.Unlock()

		close(s.stopChan)
		s.wg.Wait()
		logger.Info("Sync service stopped")
	})
}

// ========== 注册 ==========

// ensureRegistered 确保节点已注册
func (s *SyncService) ensureRegistered() error {
	// 检查本地是否已有注册信息
	if s.checkLocalRegistration() {
		logger.Info("Node already registered, using saved token")
		return nil
	}

	// 向主节点注册
	return s.register()
}

// checkLocalRegistration 检查本地注册状态
func (s *SyncService) checkLocalRegistration() bool {
	// 检查是否有保存的 token (通过命令行参数传入的 token 在首次注册后会被保存)
	// 这里简单判断：如果有 nodeToken 且能成功验证，则认为已注册
	if s.nodeToken == "" {
		return false
	}

	// 尝试获取配置版本来验证 token 是否有效
	_, err := s.getConfigVersion()
	return err == nil
}

// register 向主节点注册
func (s *SyncService) register() error {
	reqBody := map[string]interface{}{
		"token":        s.nodeToken,
		"nodeId":       s.nodeId,
		"name":         config.GetNodeName(),
		"externalHost": config.GetExternalHost(),
		"externalPort": config.GetExternalPort(),
		"version":      config.GetVersion(),
	}

	resp, err := s.doRequest("POST", "/node/register", reqBody, false)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("registration failed: %s", resp.Msg)
	}

	logger.Info("Node registered successfully")
	return nil
}

// ========== 配置同步 ==========

// configSyncLoop 配置同步循环
func (s *SyncService) configSyncLoop() {
	defer s.wg.Done()

	interval := time.Duration(config.GetSyncConfigInterval()) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			if err := s.syncConfigIfNeeded(); err != nil {
				logger.Warning("Config sync failed: ", err)
			}
		}
	}
}

// syncConfigIfNeeded 检查并同步配置
func (s *SyncService) syncConfigIfNeeded() error {
	// 获取远程版本
	remoteVersion, err := s.getConfigVersion()
	if err != nil {
		return err
	}

	// 版本相同，跳过
	if remoteVersion == s.localVersion {
		return nil
	}

	logger.Info("Config version changed, syncing...")
	return s.syncConfig()
}

// getConfigVersion 获取远程配置版本
func (s *SyncService) getConfigVersion() (int64, error) {
	resp, err := s.doRequest("GET", "/node/config/version", nil, true)
	if err != nil {
		return 0, err
	}

	if !resp.Success {
		return 0, fmt.Errorf("get version failed: %s", resp.Msg)
	}

	version, ok := resp.Raw["version"].(float64)
	if !ok {
		return 0, fmt.Errorf("invalid version format")
	}

	return int64(version), nil
}

// syncConfig 同步完整配置
func (s *SyncService) syncConfig() error {
	resp, err := s.doRequest("GET", "/node/config", nil, true)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("get config failed: %s", resp.Msg)
	}

	// 解析配置
	obj, ok := resp.Raw["obj"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid config format")
	}

	// 应用配置到本地数据库
	if err := s.applyConfig(obj); err != nil {
		return err
	}

	// 更新本地版本
	if version, ok := obj["version"].(float64); ok {
		s.localVersion = int64(version)
	}

	// 重启 Core 应用新配置
	if err := s.configService.RestartCore(); err != nil {
		logger.Warning("Failed to restart core: ", err)
	}

	logger.Info("Config synced successfully, version: ", s.localVersion)
	return nil
}

// applyConfig 应用配置到本地数据库
func (s *SyncService) applyConfig(configData map[string]interface{}) error {
	db := database.GetDB()
	tx := db.Begin()

	var err error
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// 清除并重新插入各类配置
	// Inbounds
	if inboundsData, ok := configData["inbounds"]; ok {
		if err = tx.Where("1 = 1").Delete(&model.Inbound{}).Error; err != nil {
			return err
		}
		inboundsJSON, _ := json.Marshal(inboundsData)
		var inbounds []model.Inbound
		if jsonErr := json.Unmarshal(inboundsJSON, &inbounds); jsonErr == nil && len(inbounds) > 0 {
			if err = tx.Create(&inbounds).Error; err != nil {
				return err
			}
		}
	}

	// Outbounds
	if outboundsData, ok := configData["outbounds"]; ok {
		if err = tx.Where("1 = 1").Delete(&model.Outbound{}).Error; err != nil {
			return err
		}
		outboundsJSON, _ := json.Marshal(outboundsData)
		var outbounds []model.Outbound
		if jsonErr := json.Unmarshal(outboundsJSON, &outbounds); jsonErr == nil && len(outbounds) > 0 {
			if err = tx.Create(&outbounds).Error; err != nil {
				return err
			}
		}
	}

	// Clients
	if clientsData, ok := configData["clients"]; ok {
		if err = tx.Where("1 = 1").Delete(&model.Client{}).Error; err != nil {
			return err
		}
		clientsJSON, _ := json.Marshal(clientsData)
		var clients []model.Client
		if jsonErr := json.Unmarshal(clientsJSON, &clients); jsonErr == nil && len(clients) > 0 {
			if err = tx.Create(&clients).Error; err != nil {
				return err
			}
		}
	}

	// TLS
	if tlsData, ok := configData["tls"]; ok {
		if err = tx.Where("1 = 1").Delete(&model.Tls{}).Error; err != nil {
			return err
		}
		tlsJSON, _ := json.Marshal(tlsData)
		var tlsConfigs []model.Tls
		if jsonErr := json.Unmarshal(tlsJSON, &tlsConfigs); jsonErr == nil && len(tlsConfigs) > 0 {
			if err = tx.Create(&tlsConfigs).Error; err != nil {
				return err
			}
		}
	}

	// Services
	if servicesData, ok := configData["services"]; ok {
		if err = tx.Where("1 = 1").Delete(&model.Service{}).Error; err != nil {
			return err
		}
		servicesJSON, _ := json.Marshal(servicesData)
		var services []model.Service
		if jsonErr := json.Unmarshal(servicesJSON, &services); jsonErr == nil && len(services) > 0 {
			if err = tx.Create(&services).Error; err != nil {
				return err
			}
		}
	}

	// Endpoints
	if endpointsData, ok := configData["endpoints"]; ok {
		if err = tx.Where("1 = 1").Delete(&model.Endpoint{}).Error; err != nil {
			return err
		}
		endpointsJSON, _ := json.Marshal(endpointsData)
		var endpoints []model.Endpoint
		if jsonErr := json.Unmarshal(endpointsJSON, &endpoints); jsonErr == nil && len(endpoints) > 0 {
			if err = tx.Create(&endpoints).Error; err != nil {
				return err
			}
		}
	}

	// 系统配置 (config)
	if configJSON, ok := configData["config"]; ok {
		configBytes, _ := json.Marshal(configJSON)
		if err = tx.Model(&model.Setting{}).Where("key = ?", "config").Update("value", string(configBytes)).Error; err != nil {
			return err
		}
	}

	return tx.Commit().Error
}

// ========== 统计上报 ==========

// statsReportLoop 统计上报循环
func (s *SyncService) statsReportLoop() {
	defer s.wg.Done()

	interval := time.Duration(config.GetSyncStatsInterval()) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.reportStats()
			s.reportOnlines()
		}
	}
}

// reportStats 上报流量统计
func (s *SyncService) reportStats() {
	db := database.GetDB()

	// 获取待上报的统计数据
	var stats []model.Stats
	// 获取最近一个周期的数据
	threshold := time.Now().Unix() - int64(config.GetSyncStatsInterval()*2)
	err := db.Where("date_time > ? AND node_id = ?", threshold, config.GetNodeId()).Find(&stats).Error
	if err != nil {
		logger.Warning("Failed to get stats: ", err)
		return
	}

	// 加上之前失败的数据
	s.mutex.Lock()
	stats = append(s.pendingStats, stats...)
	s.pendingStats = nil
	s.mutex.Unlock()

	if len(stats) == 0 {
		return
	}

	// 转换为上报格式
	reqBody := make([]map[string]interface{}, len(stats))
	for i, stat := range stats {
		reqBody[i] = map[string]interface{}{
			"dateTime":  stat.DateTime,
			"resource":  stat.Resource,
			"tag":       stat.Tag,
			"direction": stat.Direction,
			"traffic":   stat.Traffic,
		}
	}

	resp, err := s.doRequest("POST", "/node/stats", reqBody, true)
	if err != nil || !resp.Success {
		// 上报失败，保留数据待重试
		s.mutex.Lock()
		s.pendingStats = append(s.pendingStats, stats...)
		// 限制待上报数据量，防止无限增长
		if len(s.pendingStats) > 10000 {
			s.pendingStats = s.pendingStats[len(s.pendingStats)-10000:]
		}
		s.mutex.Unlock()
		if err != nil {
			logger.Warning("Failed to report stats: ", err)
		}
		return
	}

	// 上报成功，清除本地已上报数据
	db.Where("date_time > ? AND node_id = ?", threshold, config.GetNodeId()).Delete(&model.Stats{})
}

// reportOnlines 上报在线状态
func (s *SyncService) reportOnlines() {
	// 从 core 获取在线用户
	if !corePtr.IsRunning() {
		return
	}

	connections := corePtr.GetInstance().ConnTracker().GetConnections()
	if len(connections) == 0 {
		return
	}

	onlines := make([]map[string]interface{}, 0)
	seen := make(map[string]bool)

	for _, conn := range connections {
		if conn.User == "" {
			continue
		}
		// 去重 (同一用户+IP 只报一次)
		key := conn.User + "|" + conn.SourceIP
		if seen[key] {
			continue
		}
		seen[key] = true

		onlines = append(onlines, map[string]interface{}{
			"user":        conn.User,
			"inboundTag":  conn.Inbound,
			"sourceIP":    conn.SourceIP,
			"connectedAt": conn.ConnectedAt,
		})
	}

	if len(onlines) == 0 {
		return
	}

	reqBody := map[string]interface{}{
		"onlines": onlines,
	}

	resp, err := s.doRequest("POST", "/node/onlines", reqBody, true)
	if err != nil {
		logger.Warning("Failed to report onlines: ", err)
		return
	}
	if !resp.Success {
		logger.Warning("Report onlines failed: ", resp.Msg)
	}
}

// ========== 心跳 ==========

// heartbeatLoop 心跳循环
func (s *SyncService) heartbeatLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.sendHeartbeat()
		}
	}
}

// sendHeartbeat 发送心跳
func (s *SyncService) sendHeartbeat() {
	cpuPercent := 0.0
	memPercent := 0.0
	connections := 0

	// 获取 CPU 使用率
	if cpuStats, err := cpu.Percent(0, false); err == nil && len(cpuStats) > 0 {
		cpuPercent = cpuStats[0]
	}

	// 获取内存使用率
	if memStats, err := mem.VirtualMemory(); err == nil {
		memPercent = memStats.UsedPercent
	}

	// 获取连接数
	if corePtr.IsRunning() {
		connections = len(corePtr.GetInstance().ConnTracker().GetConnections())
	}

	reqBody := map[string]interface{}{
		"cpu":          cpuPercent,
		"memory":       memPercent,
		"connections":  connections,
		"version":      config.GetVersion(),
		"externalHost": config.GetExternalHost(),
		"externalPort": config.GetExternalPort(),
	}

	resp, err := s.doRequest("POST", "/node/heartbeat", reqBody, true)
	if err != nil {
		logger.Debug("Heartbeat failed: ", err)
		return
	}

	if !resp.Success {
		logger.Debug("Heartbeat rejected: ", resp.Msg)
	}
}

// ========== HTTP 请求 ==========

// APIResponse API 响应
type APIResponse struct {
	Success bool                   `json:"success"`
	Msg     string                 `json:"msg"`
	Obj     interface{}            `json:"obj"`
	Raw     map[string]interface{} `json:"-"`
}

// doRequest 发送请求到主节点
func (s *SyncService) doRequest(method, path string, body interface{}, auth bool) (*APIResponse, error) {
	// 正确拼接 URL: masterAddr + masterPath + path
	masterAddr := strings.TrimSuffix(s.masterAddr, "/")
	masterPath := strings.Trim(config.GetMasterPath(), "/")
	path = strings.TrimPrefix(path, "/")
	url := masterAddr + "/" + masterPath + "/" + path

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if auth {
		req.Header.Set("X-Node-Id", s.nodeId)
		req.Header.Set("X-Node-Token", s.nodeToken)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result APIResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	// 保存原始响应用于解析复杂字段
	json.Unmarshal(respBody, &result.Raw)

	return &result, nil
}
