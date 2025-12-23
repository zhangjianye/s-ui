package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/database/model"
	"github.com/alireza0/s-ui/logger"
)

// Webhook 事件类型
const (
	EventTrafficExceeded = "traffic_exceeded"
	EventTimeExceeded    = "time_exceeded"
	EventTrafficReset    = "traffic_reset"
	EventTimeReset       = "time_reset"
	EventUserExpired     = "user_expired"
	EventUserDisabled    = "user_disabled"
)

// WebhookPayload Webhook 请求体
type WebhookPayload struct {
	Event     string      `json:"event"`
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// ClientEventData 客户端事件数据
type ClientEventData struct {
	ClientName string `json:"clientName"`
	UUID       string `json:"uuid,omitempty"`
	Reason     string `json:"reason,omitempty"`
}

// WebhookService Webhook 回调服务
type WebhookService struct{}

// GetConfig 获取 Webhook 配置
func (s *WebhookService) GetConfig() (*model.WebhookConfig, error) {
	db := database.GetDB()
	var config model.WebhookConfig
	err := db.First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// SaveConfig 保存 Webhook 配置
func (s *WebhookService) SaveConfig(config *model.WebhookConfig) error {
	db := database.GetDB()
	if config.Id == 0 {
		return db.Create(config).Error
	}
	return db.Save(config).Error
}

// SendCallback 发送 Webhook 回调 (异步)
func (s *WebhookService) SendCallback(event string, data interface{}) error {
	config, err := s.GetConfig()
	if err != nil {
		// 区分"记录不存在"和其他错误
		logger.Debug("No webhook config found, skipping callback")
		return nil
	}

	if !config.Enable || config.CallbackURL == "" {
		return nil
	}

	payload := WebhookPayload{
		Event:     event,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}

	// 异步发送，避免阻塞 cronjob
	go s.send(config, payload)
	return nil
}

// SendClientEvent 发送客户端相关事件
func (s *WebhookService) SendClientEvent(event string, clientName string, uuid string, reason string) error {
	data := ClientEventData{
		ClientName: clientName,
		UUID:       uuid,
		Reason:     reason,
	}
	return s.SendCallback(event, data)
}

// send 发送 HTTP 请求 (内部方法，由 goroutine 调用)
func (s *WebhookService) send(config *model.WebhookConfig, payload WebhookPayload) {
	jsonBody, err := json.Marshal(payload)
	if err != nil {
		logger.Warning("Webhook marshal failed: ", err)
		return
	}

	req, err := http.NewRequest("POST", config.CallbackURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		logger.Warning("Webhook request create failed: ", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// 如果配置了签名密钥，添加签名
	if config.CallbackSecret != "" {
		signature := s.sign(jsonBody, config.CallbackSecret)
		req.Header.Set("X-Webhook-Signature", signature)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Warning("Webhook callback failed: ", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		logger.Warning("Webhook callback returned status: ", resp.StatusCode)
	}
}

// sign 使用 HMAC-SHA256 签名
func (s *WebhookService) sign(payload []byte, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}
