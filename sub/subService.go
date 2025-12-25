package sub

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/alireza0/s-ui/config"
	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/database/model"
	"github.com/alireza0/s-ui/service"
	"github.com/alireza0/s-ui/util"
)

type SubService struct {
	service.SettingService
	service.NodeService
	LinkService
}

func (s *SubService) GetSubs(subId string) (*string, []string, error) {
	var err error

	db := database.GetDB()
	client := &model.Client{}
	// 优先尝试 UUID 查询，如果失败则尝试 Name 查询 (向后兼容)
	err = db.Model(model.Client{}).Where("enable = true and uuid = ?", subId).First(client).Error
	if err != nil {
		// UUID 查询失败，尝试 Name 查询
		err = db.Model(model.Client{}).Where("enable = true and name = ?", subId).First(client).Error
		if err != nil {
			return nil, nil, err
		}
	}

	clientInfo := ""
	subShowInfo, _ := s.SettingService.GetSubShowInfo()
	if subShowInfo {
		clientInfo = s.getClientInfo(client)
	}

	linksArray := s.LinkService.GetLinks(&client.Links, "all", clientInfo)

	// 主节点模式：为每个从节点复制链接
	if config.IsMaster() {
		linksArray, err = s.expandLinksForNodes(linksArray)
		if err != nil {
			return nil, nil, err
		}
	}

	result := strings.Join(linksArray, "\n")

	updateInterval, _ := s.SettingService.GetSubUpdates()
	headers := util.GetHeaders(client, updateInterval)

	subEncode, _ := s.SettingService.GetSubEncode()
	if subEncode {
		result = base64.StdEncoding.EncodeToString([]byte(result))
	}

	return &result, headers, nil
}

func (s *SubService) getClientInfo(c *model.Client) string {
	now := time.Now().Unix()

	var result []string
	if vol := c.Volume - (c.Up + c.Down); vol > 0 {
		result = append(result, fmt.Sprintf("%s%s", s.formatTraffic(vol), "📊"))
	}
	if c.Expiry > 0 {
		result = append(result, fmt.Sprintf("%d%s⏳", (c.Expiry-now)/86400, "Days"))
	}
	if len(result) > 0 {
		return " " + strings.Join(result, " ")
	} else {
		return " ♾"
	}
}

func (s *SubService) formatTraffic(trafficBytes int64) string {
	if trafficBytes < 1024 {
		return fmt.Sprintf("%.2fB", float64(trafficBytes)/float64(1))
	} else if trafficBytes < (1024 * 1024) {
		return fmt.Sprintf("%.2fKB", float64(trafficBytes)/float64(1024))
	} else if trafficBytes < (1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fMB", float64(trafficBytes)/float64(1024*1024))
	} else if trafficBytes < (1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fGB", float64(trafficBytes)/float64(1024*1024*1024))
	} else if trafficBytes < (1024 * 1024 * 1024 * 1024 * 1024) {
		return fmt.Sprintf("%.2fTB", float64(trafficBytes)/float64(1024*1024*1024*1024))
	} else {
		return fmt.Sprintf("%.2fEB", float64(trafficBytes)/float64(1024*1024*1024*1024*1024))
	}
}

// expandLinksForNodes 在主节点模式下，为每个在线从节点复制链接
func (s *SubService) expandLinksForNodes(links []string) ([]string, error) {
	nodes, err := s.NodeService.GetEnabledOnlineNodes()
	if err != nil || len(nodes) == 0 {
		// 没有从节点，返回空
		return []string{}, nil
	}

	var result []string
	for _, node := range nodes {
		if node.ExternalHost == "" {
			continue
		}
		for _, link := range links {
			newLink := s.replaceHostInLink(link, node.ExternalHost, node.ExternalPort, node.Name)
			if newLink != "" {
				result = append(result, newLink)
			}
		}
	}
	return result, nil
}

// replaceHostInLink 替换链接中的服务器地址
func (s *SubService) replaceHostInLink(link, newHost string, newPort int, nodeName string) string {
	u, err := url.Parse(link)
	if err != nil {
		return link
	}

	switch u.Scheme {
	case "vmess":
		return s.replaceVmessHost(link, newHost, newPort, nodeName)
	case "vless", "trojan", "hy", "hysteria", "hy2", "hysteria2", "tuic", "anytls", "uap":
		return s.replaceStandardLink(u, newHost, newPort, nodeName)
	case "ss", "shadowsocks":
		return s.replaceSsHost(u, newHost, newPort, nodeName)
	default:
		return link
	}
}

// replaceVmessHost 替换 vmess 链接中的服务器地址
func (s *SubService) replaceVmessHost(link, newHost string, newPort int, nodeName string) string {
	// vmess://base64{...}
	parts := strings.SplitN(link, "://", 2)
	if len(parts) != 2 {
		return link
	}

	dataByte, err := util.B64StrToByte(parts[1])
	if err != nil {
		return link
	}

	var vmessJson map[string]interface{}
	if err := json.Unmarshal(dataByte, &vmessJson); err != nil {
		return link
	}

	// 替换地址
	vmessJson["add"] = newHost
	if newPort > 0 {
		vmessJson["port"] = newPort
	}

	// 更新备注，添加节点名称
	if ps, ok := vmessJson["ps"].(string); ok {
		vmessJson["ps"] = nodeName + "-" + ps
	}

	result, err := json.Marshal(vmessJson)
	if err != nil {
		return link
	}
	return "vmess://" + util.ByteToB64Str(result)
}

// replaceStandardLink 替换标准格式链接中的服务器地址 (vless, trojan, etc.)
func (s *SubService) replaceStandardLink(u *url.URL, newHost string, newPort int, nodeName string) string {
	// 格式: scheme://userinfo@host:port?params#remark
	port := u.Port()
	if newPort > 0 {
		port = fmt.Sprintf("%d", newPort)
	}

	// 构建新的 host:port
	newHostPort := newHost
	if port != "" {
		newHostPort = fmt.Sprintf("%s:%s", newHost, port)
	}

	// 更新备注
	fragment := u.Fragment
	if fragment != "" {
		fragment = nodeName + "-" + fragment
	} else {
		fragment = nodeName
	}

	// 重建链接
	newURL := &url.URL{
		Scheme:   u.Scheme,
		User:     u.User,
		Host:     newHostPort,
		RawQuery: u.RawQuery,
		Fragment: fragment,
	}
	return newURL.String()
}

// replaceSsHost 替换 shadowsocks 链接中的服务器地址
func (s *SubService) replaceSsHost(u *url.URL, newHost string, newPort int, nodeName string) string {
	// ss://base64(method:password)@host:port#remark
	// 或 ss://base64(method:password@host:port)#remark
	port := u.Port()
	if newPort > 0 {
		port = fmt.Sprintf("%d", newPort)
	}

	newHostPort := newHost
	if port != "" {
		newHostPort = fmt.Sprintf("%s:%s", newHost, port)
	}

	fragment := u.Fragment
	if fragment != "" {
		fragment = nodeName + "-" + fragment
	} else {
		fragment = nodeName
	}

	newURL := &url.URL{
		Scheme:   u.Scheme,
		User:     u.User,
		Host:     newHostPort,
		RawQuery: u.RawQuery,
		Fragment: fragment,
	}
	return newURL.String()
}
