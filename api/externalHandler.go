package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/database/model"
	"github.com/alireza0/s-ui/logger"
	"github.com/alireza0/s-ui/service"
	"github.com/alireza0/s-ui/util/common"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ExternalHandler 外部 API 处理器 (供 UAP Backend 调用)
type ExternalHandler struct {
	service.ClientService
	service.ConfigService
	service.InboundService
}

// ExternalResponse 外部 API 响应格式
type ExternalResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// UserCreateRequest 创建用户请求
type UserCreateRequest struct {
	UUID                 string `json:"uuid"`                 // 必填，UAP 提供的 UUID
	Name                 string `json:"name"`                 // 必填，用户名
	Enable               *bool  `json:"enable"`               // 是否启用，默认 true
	Volume               int64  `json:"volume"`               // 流量限制 (bytes)，0=无限
	Expiry               int64  `json:"expiry"`               // 过期时间戳，0=永不过期
	TimeLimit            int64  `json:"timeLimit"`            // 时长限制 (秒)，0=无限
	IsPremium            bool   `json:"isPremium"`            // 是否会员
	TrafficResetStrategy string `json:"trafficResetStrategy"` // 流量重置策略
	TimeResetStrategy    string `json:"timeResetStrategy"`    // 时长重置策略
	SpeedLimit           int    `json:"speedLimit"`           // 带宽限制 (Mbps)
	DeviceLimit          int    `json:"deviceLimit"`          // 设备数限制
	Inbounds             []uint `json:"inbounds"`             // 关联的 Inbound IDs
	Desc                 string `json:"desc"`                 // 描述
	Group                string `json:"group"`                // 分组
}

// UserUpdateRequest 更新用户请求
type UserUpdateRequest struct {
	Name                 *string `json:"name,omitempty"`
	Enable               *bool   `json:"enable,omitempty"`
	Volume               *int64  `json:"volume,omitempty"`
	Expiry               *int64  `json:"expiry,omitempty"`
	TimeLimit            *int64  `json:"timeLimit,omitempty"`
	IsPremium            *bool   `json:"isPremium,omitempty"`
	TrafficResetStrategy *string `json:"trafficResetStrategy,omitempty"`
	TimeResetStrategy    *string `json:"timeResetStrategy,omitempty"`
	SpeedLimit           *int    `json:"speedLimit,omitempty"`
	DeviceLimit          *int    `json:"deviceLimit,omitempty"`
	Inbounds             []uint  `json:"inbounds,omitempty"`
	Desc                 *string `json:"desc,omitempty"`
	Group                *string `json:"group,omitempty"`
}

// UserResponse 用户信息响应
type UserResponse struct {
	Id                   uint   `json:"id"`
	UUID                 string `json:"uuid"`
	Name                 string `json:"name"`
	Enable               bool   `json:"enable"`
	Volume               int64  `json:"volume"`
	Up                   int64  `json:"up"`
	Down                 int64  `json:"down"`
	Expiry               int64  `json:"expiry"`
	TimeLimit            int64  `json:"timeLimit"`
	TimeUsed             int64  `json:"timeUsed"`
	IsPremium            bool   `json:"isPremium"`
	TrafficResetStrategy string `json:"trafficResetStrategy"`
	TimeResetStrategy    string `json:"timeResetStrategy"`
	SpeedLimit           int    `json:"speedLimit"`
	DeviceLimit          int    `json:"deviceLimit"`
	Desc                 string `json:"desc"`
	Group                string `json:"group"`
}

// NewExternalHandler 创建外部 API 处理器
func NewExternalHandler(g *gin.RouterGroup) {
	h := &ExternalHandler{}
	h.initRouter(g)
}

func (h *ExternalHandler) initRouter(g *gin.RouterGroup) {
	// API Key 认证中间件
	g.Use(h.apiKeyAuth)

	// 用户管理 API
	users := g.Group("/users")
	{
		users.POST("", h.createUser)         // POST /api/v1/users
		users.GET("/:uuid", h.getUser)       // GET /api/v1/users/{uuid}
		users.PUT("/:uuid", h.updateUser)    // PUT /api/v1/users/{uuid}
		users.DELETE("/:uuid", h.deleteUser) // DELETE /api/v1/users/{uuid}

		// 用户操作
		users.POST("/:uuid/enable", h.enableUser)          // POST /api/v1/users/{uuid}/enable
		users.POST("/:uuid/disable", h.disableUser)        // POST /api/v1/users/{uuid}/disable
		users.POST("/:uuid/reset-traffic", h.resetTraffic) // POST /api/v1/users/{uuid}/reset-traffic
		users.POST("/:uuid/reset-time", h.resetTime)       // POST /api/v1/users/{uuid}/reset-time
	}
}

// apiKeyAuth API Key 认证中间件
func (h *ExternalHandler) apiKeyAuth(c *gin.Context) {
	apiKey := c.GetHeader("X-API-Key")
	if apiKey == "" {
		h.errorResponse(c, http.StatusUnauthorized, "missing X-API-Key header")
		c.Abort()
		return
	}

	// 验证 API Key
	db := database.GetDB()
	var key model.ApiKey
	err := db.Where("key = ? AND enable = ?", apiKey, true).First(&key).Error
	if err != nil {
		h.errorResponse(c, http.StatusUnauthorized, "invalid or disabled API key")
		c.Abort()
		return
	}

	c.Next()
}

// successResponse 成功响应
func (h *ExternalHandler) successResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, ExternalResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// errorResponse 错误响应
func (h *ExternalHandler) errorResponse(c *gin.Context, httpCode int, message string) {
	c.JSON(httpCode, ExternalResponse{
		Code:    httpCode,
		Message: message,
	})
}

// generateClientConfig 生成包含所有协议配置的 Config
func (h *ExternalHandler) generateClientConfig(clientUUID string, name string) json.RawMessage {
	mixedPassword := common.Random(10)
	ssPassword16 := base64.StdEncoding.EncodeToString([]byte(common.Random(12)))[:16]
	ssPassword32 := base64.StdEncoding.EncodeToString([]byte(common.Random(24)))[:32]

	config := map[string]map[string]interface{}{
		"mixed": {
			"username": name,
			"password": mixedPassword,
		},
		"socks": {
			"username": name,
			"password": mixedPassword,
		},
		"http": {
			"username": name,
			"password": mixedPassword,
		},
		"shadowsocks": {
			"name":     name,
			"password": ssPassword32,
		},
		"shadowsocks16": {
			"name":     name,
			"password": ssPassword16,
		},
		"shadowtls": {
			"name":     name,
			"password": ssPassword32,
		},
		"vmess": {
			"name":    name,
			"uuid":    clientUUID,
			"alterId": 0,
		},
		"vless": {
			"name": name,
			"uuid": clientUUID,
			"flow": "xtls-rprx-vision",
		},
		"anytls": {
			"name":     name,
			"password": mixedPassword,
		},
		"trojan": {
			"name":     name,
			"password": mixedPassword,
		},
		"naive": {
			"username": name,
			"password": mixedPassword,
		},
		"hysteria": {
			"name":     name,
			"auth_str": mixedPassword,
		},
		"tuic": {
			"name":     name,
			"uuid":     clientUUID,
			"password": mixedPassword,
		},
		"hysteria2": {
			"name":     name,
			"password": mixedPassword,
		},
		"uap": {
			"name": name,
			"uuid": clientUUID,
		},
	}

	result, _ := json.MarshalIndent(config, "", "  ")
	return result
}

// createUser 创建用户
func (h *ExternalHandler) createUser(c *gin.Context) {
	var req UserCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	// 验证必填字段
	if req.UUID == "" {
		h.errorResponse(c, http.StatusBadRequest, "uuid is required")
		return
	}
	if req.Name == "" {
		h.errorResponse(c, http.StatusBadRequest, "name is required")
		return
	}

	// 验证 UUID 格式
	if _, err := uuid.Parse(req.UUID); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid uuid format")
		return
	}

	// 检查 UUID 是否已存在
	db := database.GetDB()
	var existingClient model.Client
	if err := db.Where("uuid = ?", req.UUID).First(&existingClient).Error; err == nil {
		h.errorResponse(c, http.StatusConflict, "uuid already exists")
		return
	}

	// 检查 Name 是否已存在
	if err := db.Where("name = ?", req.Name).First(&existingClient).Error; err == nil {
		h.errorResponse(c, http.StatusConflict, "name already exists")
		return
	}

	// 处理 Inbounds (确保不为 null)
	inbounds := req.Inbounds
	if inbounds == nil {
		inbounds = []uint{}
	}
	inboundsJSON, _ := json.Marshal(inbounds)

	// 处理 Enable 默认值
	enable := true
	if req.Enable != nil {
		enable = *req.Enable
	}

	// 生成包含协议配置的 Config
	config := h.generateClientConfig(req.UUID, req.Name)

	// 创建 Client
	client := model.Client{
		UUID:                 req.UUID,
		Name:                 req.Name,
		Enable:               enable,
		Volume:               req.Volume,
		Expiry:               req.Expiry,
		TimeLimit:            req.TimeLimit,
		IsPremium:            req.IsPremium,
		TrafficResetStrategy: req.TrafficResetStrategy,
		TimeResetStrategy:    req.TimeResetStrategy,
		SpeedLimit:           req.SpeedLimit,
		DeviceLimit:          req.DeviceLimit,
		Inbounds:             inboundsJSON,
		Links:                json.RawMessage("[]"),
		Config:               config,
		Desc:                 req.Desc,
		Group:                req.Group,
	}

	// 使用 ConfigService.Save 来保存并触发核心重载
	clientJSON, _ := json.Marshal(client)
	_, err := h.ConfigService.Save("clients", "new", clientJSON, "", "ExternalAPI", getHostname(c))
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "failed to create user: "+err.Error())
		return
	}

	// 重新加载 client 获取生成的 ID 和 Links
	var savedClient model.Client
	db.Where("uuid = ?", req.UUID).First(&savedClient)

	logger.Info("External API: created user ", savedClient.Name, " with UUID ", savedClient.UUID)
	h.successResponse(c, h.toUserResponse(&savedClient))
}

// getUser 获取用户信息
func (h *ExternalHandler) getUser(c *gin.Context) {
	userUUID := c.Param("uuid")

	client, err := h.ClientService.GetByUUID(userUUID)
	if err != nil {
		h.errorResponse(c, http.StatusNotFound, "user not found")
		return
	}

	h.successResponse(c, h.toUserResponse(client))
}

// updateUser 更新用户
func (h *ExternalHandler) updateUser(c *gin.Context) {
	userUUID := c.Param("uuid")

	client, err := h.ClientService.GetByUUID(userUUID)
	if err != nil {
		h.errorResponse(c, http.StatusNotFound, "user not found")
		return
	}

	var req UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	db := database.GetDB()

	// 更新字段
	if req.Name != nil {
		// 检查新名称是否已被使用
		var existingClient model.Client
		if err := db.Where("name = ? AND id != ?", *req.Name, client.Id).First(&existingClient).Error; err == nil {
			h.errorResponse(c, http.StatusConflict, "name already exists")
			return
		}
		client.Name = *req.Name
	}
	if req.Enable != nil {
		client.Enable = *req.Enable
	}
	if req.Volume != nil {
		client.Volume = *req.Volume
	}
	if req.Expiry != nil {
		client.Expiry = *req.Expiry
	}
	if req.TimeLimit != nil {
		client.TimeLimit = *req.TimeLimit
	}
	if req.IsPremium != nil {
		client.IsPremium = *req.IsPremium
	}
	if req.TrafficResetStrategy != nil {
		client.TrafficResetStrategy = *req.TrafficResetStrategy
	}
	if req.TimeResetStrategy != nil {
		client.TimeResetStrategy = *req.TimeResetStrategy
	}
	if req.SpeedLimit != nil {
		client.SpeedLimit = *req.SpeedLimit
	}
	if req.DeviceLimit != nil {
		client.DeviceLimit = *req.DeviceLimit
	}
	if req.Inbounds != nil {
		inboundsJSON, _ := json.Marshal(req.Inbounds)
		client.Inbounds = inboundsJSON
	}
	if req.Desc != nil {
		client.Desc = *req.Desc
	}
	if req.Group != nil {
		client.Group = *req.Group
	}

	// 使用 ConfigService.Save 来保存并触发核心重载
	clientJSON, _ := json.Marshal(client)
	_, err = h.ConfigService.Save("clients", "edit", clientJSON, "", "ExternalAPI", getHostname(c))
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "failed to update user: "+err.Error())
		return
	}

	// 重新加载 client 获取更新后的数据
	var updatedClient model.Client
	db.Where("id = ?", client.Id).First(&updatedClient)

	logger.Info("External API: updated user ", updatedClient.Name)
	h.successResponse(c, h.toUserResponse(&updatedClient))
}

// deleteUser 删除用户
func (h *ExternalHandler) deleteUser(c *gin.Context) {
	userUUID := c.Param("uuid")

	client, err := h.ClientService.GetByUUID(userUUID)
	if err != nil {
		h.errorResponse(c, http.StatusNotFound, "user not found")
		return
	}

	// 使用 ConfigService.Save 来删除并触发核心重载
	idJSON, _ := json.Marshal(client.Id)
	_, err = h.ConfigService.Save("clients", "del", idJSON, "", "ExternalAPI", getHostname(c))
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "failed to delete user: "+err.Error())
		return
	}

	logger.Info("External API: deleted user ", client.Name)
	h.successResponse(c, nil)
}

// enableUser 启用用户
func (h *ExternalHandler) enableUser(c *gin.Context) {
	userUUID := c.Param("uuid")

	client, err := h.ClientService.GetByUUID(userUUID)
	if err != nil {
		h.errorResponse(c, http.StatusNotFound, "user not found")
		return
	}

	client.Enable = true

	// 使用 ConfigService.Save 来保存并触发核心重载
	clientJSON, _ := json.Marshal(client)
	_, err = h.ConfigService.Save("clients", "edit", clientJSON, "", "ExternalAPI", getHostname(c))
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "failed to enable user: "+err.Error())
		return
	}

	logger.Info("External API: enabled user ", client.Name)
	h.successResponse(c, nil)
}

// disableUser 禁用用户
func (h *ExternalHandler) disableUser(c *gin.Context) {
	userUUID := c.Param("uuid")

	client, err := h.ClientService.GetByUUID(userUUID)
	if err != nil {
		h.errorResponse(c, http.StatusNotFound, "user not found")
		return
	}

	client.Enable = false

	// 使用 ConfigService.Save 来保存并触发核心重载
	clientJSON, _ := json.Marshal(client)
	_, err = h.ConfigService.Save("clients", "edit", clientJSON, "", "ExternalAPI", getHostname(c))
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "failed to disable user: "+err.Error())
		return
	}

	logger.Info("External API: disabled user ", client.Name)
	h.successResponse(c, nil)
}

// resetTraffic 重置用户流量
func (h *ExternalHandler) resetTraffic(c *gin.Context) {
	userUUID := c.Param("uuid")

	client, err := h.ClientService.GetByUUID(userUUID)
	if err != nil {
		h.errorResponse(c, http.StatusNotFound, "user not found")
		return
	}

	client.Up = 0
	client.Down = 0
	client.Enable = true

	// 使用 ConfigService.Save 来保存并触发核心重载
	clientJSON, _ := json.Marshal(client)
	_, err = h.ConfigService.Save("clients", "edit", clientJSON, "", "ExternalAPI", getHostname(c))
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "failed to reset traffic: "+err.Error())
		return
	}

	logger.Info("External API: reset traffic for user ", client.Name)
	h.successResponse(c, nil)
}

// resetTime 重置用户时长
func (h *ExternalHandler) resetTime(c *gin.Context) {
	userUUID := c.Param("uuid")

	client, err := h.ClientService.GetByUUID(userUUID)
	if err != nil {
		h.errorResponse(c, http.StatusNotFound, "user not found")
		return
	}

	client.TimeUsed = 0
	client.Enable = true

	// 使用 ConfigService.Save 来保存并触发核心重载
	clientJSON, _ := json.Marshal(client)
	_, err = h.ConfigService.Save("clients", "edit", clientJSON, "", "ExternalAPI", getHostname(c))
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "failed to reset time: "+err.Error())
		return
	}

	logger.Info("External API: reset time for user ", client.Name)
	h.successResponse(c, nil)
}

// toUserResponse 转换为用户响应
func (h *ExternalHandler) toUserResponse(client *model.Client) *UserResponse {
	return &UserResponse{
		Id:                   client.Id,
		UUID:                 client.UUID,
		Name:                 client.Name,
		Enable:               client.Enable,
		Volume:               client.Volume,
		Up:                   client.Up,
		Down:                 client.Down,
		Expiry:               client.Expiry,
		TimeLimit:            client.TimeLimit,
		TimeUsed:             client.TimeUsed,
		IsPremium:            client.IsPremium,
		TrafficResetStrategy: client.TrafficResetStrategy,
		TimeResetStrategy:    client.TimeResetStrategy,
		SpeedLimit:           client.SpeedLimit,
		DeviceLimit:          client.DeviceLimit,
		Desc:                 client.Desc,
		Group:                client.Group,
	}
}
