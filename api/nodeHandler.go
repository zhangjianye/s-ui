package api

import (
	"net/http"

	"github.com/alireza0/s-ui/config"
	"github.com/alireza0/s-ui/database/model"
	"github.com/alireza0/s-ui/service"

	"github.com/gin-gonic/gin"
)

// NodeHandler 节点 API Handler (主节点提供，从节点调用)
type NodeHandler struct {
	nodeService service.NodeService
}

// NewNodeHandler 创建 NodeHandler 并注册路由
func NewNodeHandler(g *gin.RouterGroup) *NodeHandler {
	h := &NodeHandler{}
	h.initRouter(g)
	return h
}

func (h *NodeHandler) initRouter(g *gin.RouterGroup) {
	// 注册接口不需要认证
	g.POST("/register", h.register)

	// 需要认证的接口
	auth := g.Group("")
	auth.Use(h.authMiddleware())
	{
		auth.GET("/config/version", h.getConfigVersion)
		auth.GET("/config", h.getConfig)
		auth.POST("/stats", h.reportStats)
		auth.POST("/onlines", h.reportOnlines)
		auth.POST("/heartbeat", h.heartbeat)
	}
}

// authMiddleware 节点认证中间件
func (h *NodeHandler) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		nodeId := c.GetHeader("X-Node-Id")
		token := c.GetHeader("X-Node-Token")

		if nodeId == "" || token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"msg":     "missing node credentials",
			})
			c.Abort()
			return
		}

		node, err := h.nodeService.AuthenticateNode(nodeId, token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"msg":     err.Error(),
			})
			c.Abort()
			return
		}

		// 将节点信息存入 context
		c.Set("node", node)
		c.Set("nodeId", nodeId)
		c.Next()
	}
}

// RegisterRequest 节点注册请求
type RegisterRequest struct {
	Token        string `json:"token" binding:"required"`
	NodeId       string `json:"nodeId" binding:"required"`
	Name         string `json:"name"`
	Address      string `json:"address"`
	ExternalHost string `json:"externalHost"`
	ExternalPort int    `json:"externalPort"`
	Version      string `json:"version"`
}

// register 处理节点注册
func (h *NodeHandler) register(c *gin.Context) {
	// 只有主节点才能处理注册
	if !config.IsMaster() {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"msg":     "only master node can register workers",
		})
		return
	}

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"msg":     "invalid request: " + err.Error(),
		})
		return
	}

	// 如果没有提供地址，使用请求来源 IP
	if req.Address == "" {
		req.Address = c.ClientIP()
	}

	node, err := h.nodeService.RegisterNode(
		req.Token,
		req.NodeId,
		req.Name,
		req.Address,
		req.ExternalHost,
		req.ExternalPort,
		req.Version,
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"msg":     err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"msg":     "registered",
		"obj": gin.H{
			"nodeId": node.NodeId,
			"token":  node.Token,
		},
	})
}

// getConfigVersion 获取配置版本
func (h *NodeHandler) getConfigVersion(c *gin.Context) {
	version := h.nodeService.GetConfigVersion()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"version": version,
	})
}

// getConfig 获取完整配置
func (h *NodeHandler) getConfig(c *gin.Context) {
	nodeId := c.GetString("nodeId")

	configData, err := h.nodeService.GetFullConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"msg":     err.Error(),
		})
		return
	}

	// 更新节点最后同步时间
	h.nodeService.UpdateLastSync(nodeId)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"obj":     configData,
	})
}

// StatsRequest 统计上报请求
type StatsRequest []struct {
	DateTime  int64  `json:"dateTime"`
	Resource  string `json:"resource"`
	Tag       string `json:"tag"`
	Direction bool   `json:"direction"`
	Traffic   int64  `json:"traffic"`
}

// reportStats 处理统计上报
func (h *NodeHandler) reportStats(c *gin.Context) {
	nodeId := c.GetString("nodeId")

	var req StatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"msg":     "invalid request: " + err.Error(),
		})
		return
	}

	// 转换为 model.Stats
	stats := make([]model.Stats, len(req))
	for i, s := range req {
		stats[i] = model.Stats{
			DateTime:  s.DateTime,
			Resource:  s.Resource,
			Tag:       s.Tag,
			Direction: s.Direction,
			Traffic:   s.Traffic,
			NodeId:    nodeId,
		}
	}

	err := h.nodeService.SaveNodeStats(nodeId, stats)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"msg":     err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// OnlinesRequest 在线状态上报请求
type OnlinesRequest struct {
	Onlines []struct {
		User        string `json:"user"`
		InboundTag  string `json:"inboundTag"`
		SourceIP    string `json:"sourceIP"`
		ConnectedAt int64  `json:"connectedAt"`
	} `json:"onlines"`
}

// reportOnlines 处理在线状态上报
func (h *NodeHandler) reportOnlines(c *gin.Context) {
	nodeId := c.GetString("nodeId")

	var req OnlinesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"msg":     "invalid request: " + err.Error(),
		})
		return
	}

	// 转换为 model.ClientOnline
	onlines := make([]model.ClientOnline, len(req.Onlines))
	for i, o := range req.Onlines {
		onlines[i] = model.ClientOnline{
			ClientName:  o.User,
			InboundTag:  o.InboundTag,
			SourceIP:    o.SourceIP,
			ConnectedAt: o.ConnectedAt,
		}
	}

	err := h.nodeService.SaveOnlineStatus(nodeId, onlines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"msg":     err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// HeartbeatRequest 心跳请求
type HeartbeatRequest struct {
	CPU         float64 `json:"cpu"`
	Memory      float64 `json:"memory"`
	Connections int     `json:"connections"`
	Version     string  `json:"version"`
}

// heartbeat 处理心跳
func (h *NodeHandler) heartbeat(c *gin.Context) {
	nodeId := c.GetString("nodeId")

	var req HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"msg":     "invalid request: " + err.Error(),
		})
		return
	}

	err := h.nodeService.Heartbeat(nodeId, req.CPU, req.Memory, req.Connections, req.Version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"msg":     err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"time":    h.nodeService.GetConfigVersion(),
	})
}
