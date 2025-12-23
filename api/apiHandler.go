package api

import (
	"strings"

	"github.com/alireza0/s-ui/config"
	"github.com/alireza0/s-ui/util/common"

	"github.com/gin-gonic/gin"
)

type APIHandler struct {
	ApiService
	apiv2 *APIv2Handler
}

func NewAPIHandler(g *gin.RouterGroup, a2 *APIv2Handler) {
	a := &APIHandler{
		apiv2: a2,
	}
	a.initRouter(g)
}

func (a *APIHandler) initRouter(g *gin.RouterGroup) {
	g.Use(func(c *gin.Context) {
		path := c.Request.URL.Path
		if !strings.HasSuffix(path, "login") && !strings.HasSuffix(path, "logout") {
			checkLogin(c)
		}
	})
	g.POST("/:postAction", a.postHandler)
	g.GET("/:getAction", a.getHandler)
}

func (a *APIHandler) postHandler(c *gin.Context) {
	loginUser := GetLoginUser(c)
	action := c.Param("postAction")

	// Worker 模式下只读操作限制
	if config.IsReadOnly() {
		// 允许的只读操作
		allowedActions := map[string]bool{
			"login":       true,
			"linkConvert": true,
		}
		if !allowedActions[action] {
			jsonMsg(c, "failed", common.NewError("readonly mode: write operations are not allowed"))
			return
		}
	}

	switch action {
	case "login":
		a.ApiService.Login(c)
	case "changePass":
		a.ApiService.ChangePass(c)
	case "save":
		a.ApiService.Save(c, loginUser)
	case "restartApp":
		a.ApiService.RestartApp(c)
	case "restartSb":
		a.ApiService.RestartSb(c)
	case "linkConvert":
		a.ApiService.LinkConvert(c)
	case "importdb":
		a.ApiService.ImportDb(c)
	case "addToken":
		a.ApiService.AddToken(c)
		a.apiv2.ReloadTokens()
	case "deleteToken":
		a.ApiService.DeleteToken(c)
		a.apiv2.ReloadTokens()
	// 节点管理 (仅 Master 模式)
	case "generateNodeToken":
		a.ApiService.GenerateNodeToken(c)
	case "deleteNodeToken":
		a.ApiService.DeleteNodeToken(c)
	// API Key 管理
	case "createApiKey":
		a.ApiService.CreateApiKey(c)
	case "updateApiKey":
		a.ApiService.UpdateApiKey(c)
	case "deleteApiKey":
		a.ApiService.DeleteApiKey(c)
	// Webhook 配置
	case "saveWebhookConfig":
		a.ApiService.SaveWebhookConfig(c)
	default:
		jsonMsg(c, "failed", common.NewError("unknown action: ", action))
	}
}

func (a *APIHandler) getHandler(c *gin.Context) {
	action := c.Param("getAction")

	switch action {
	case "logout":
		a.ApiService.Logout(c)
	case "load":
		a.ApiService.LoadData(c)
	case "inbounds", "outbounds", "endpoints", "services", "tls", "clients", "config":
		err := a.ApiService.LoadPartialData(c, []string{action})
		if err != nil {
			jsonMsg(c, action, err)
		}
		return
	case "users":
		a.ApiService.GetUsers(c)
	case "settings":
		a.ApiService.GetSettings(c)
	case "stats":
		a.ApiService.GetStats(c)
	case "status":
		a.ApiService.GetStatus(c)
	case "onlines":
		a.ApiService.GetOnlines(c)
	case "logs":
		a.ApiService.GetLogs(c)
	case "changes":
		a.ApiService.CheckChanges(c)
	case "keypairs":
		a.ApiService.GetKeypairs(c)
	case "getdb":
		a.ApiService.GetDb(c)
	case "tokens":
		a.ApiService.GetTokens(c)
	// 节点管理 (仅 Master 模式)
	case "nodes":
		a.ApiService.GetNodes(c)
	case "nodeTokens":
		a.ApiService.GetNodeTokens(c)
	// 节点模式信息
	case "nodeMode":
		a.ApiService.GetNodeMode(c)
	// API Key 管理
	case "apiKeys":
		a.ApiService.GetApiKeys(c)
	// Webhook 配置
	case "webhookConfig":
		a.ApiService.GetWebhookConfig(c)
	default:
		jsonMsg(c, "failed", common.NewError("unknown action: ", action))
	}
}
