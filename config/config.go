package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

//go:embed version
var version string

//go:embed name
var name string

type LogLevel string

const (
	Debug LogLevel = "debug"
	Info  LogLevel = "info"
	Warn  LogLevel = "warn"
	Error LogLevel = "error"
)

// NodeMode 节点运行模式
type NodeMode string

const (
	ModeStandalone NodeMode = "standalone" // 单机模式 (默认)
	ModeMaster     NodeMode = "master"     // 主节点模式
	ModeWorker     NodeMode = "worker"     // 从节点模式
)

// 节点配置 (通过命令行参数或环境变量设置)
var (
	nodeMode         NodeMode = ModeStandalone
	nodeId           string
	nodeName         string
	masterAddr       string
	nodeToken        string
	externalHost     string
	externalPort     int
	syncConfigInterval int = 60 // 秒
	syncStatsInterval  int = 30 // 秒
)

func GetVersion() string {
	return strings.TrimSpace(version)
}

func GetName() string {
	return strings.TrimSpace(name)
}

func GetLogLevel() LogLevel {
	if IsDebug() {
		return Debug
	}
	logLevel := os.Getenv("SUI_LOG_LEVEL")
	if logLevel == "" {
		return Info
	}
	return LogLevel(logLevel)
}

func IsDebug() bool {
	return os.Getenv("SUI_DEBUG") == "true"
}

func GetDBFolderPath() string {
	dbFolderPath := os.Getenv("SUI_DB_FOLDER")
	if dbFolderPath == "" {
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
		if err != nil {
			// Cross-platform fallback path
			if runtime.GOOS == "windows" {
				return "C:\\Program Files\\s-ui\\db"
			}
			return "/usr/local/s-ui/db"
		}
		dbFolderPath = filepath.Join(dir, "db")
	}
	return dbFolderPath
}

func GetDBPath() string {
	return fmt.Sprintf("%s/%s.db", GetDBFolderPath(), GetName())
}

// ========== 节点配置 Setters ==========

// SetNodeMode 设置节点模式
func SetNodeMode(mode string) {
	switch NodeMode(mode) {
	case ModeMaster, ModeWorker:
		nodeMode = NodeMode(mode)
	default:
		nodeMode = ModeStandalone
	}
}

// SetNodeId 设置节点 ID
func SetNodeId(id string) {
	nodeId = id
}

// SetNodeName 设置节点名称
func SetNodeName(n string) {
	nodeName = n
}

// SetMasterAddr 设置主节点地址
func SetMasterAddr(addr string) {
	masterAddr = addr
}

// SetNodeToken 设置节点 Token
func SetNodeToken(token string) {
	nodeToken = token
}

// SetExternalHost 设置外部主机地址
func SetExternalHost(host string) {
	externalHost = host
}

// SetExternalPort 设置外部端口
func SetExternalPort(port int) {
	externalPort = port
}

// SetSyncConfigInterval 设置配置同步间隔
func SetSyncConfigInterval(interval int) {
	if interval > 0 {
		syncConfigInterval = interval
	}
}

// SetSyncStatsInterval 设置统计上报间隔
func SetSyncStatsInterval(interval int) {
	if interval > 0 {
		syncStatsInterval = interval
	}
}

// ========== 节点配置 Getters ==========

// GetNodeMode 获取节点模式
func GetNodeMode() NodeMode {
	// 环境变量优先
	if envMode := os.Getenv("SUI_NODE_MODE"); envMode != "" {
		switch NodeMode(envMode) {
		case ModeMaster, ModeWorker:
			return NodeMode(envMode)
		}
	}
	return nodeMode
}

// GetNodeId 获取节点 ID
func GetNodeId() string {
	if envId := os.Getenv("SUI_NODE_ID"); envId != "" {
		return envId
	}
	return nodeId
}

// GetNodeName 获取节点名称
func GetNodeName() string {
	if envName := os.Getenv("SUI_NODE_NAME"); envName != "" {
		return envName
	}
	if nodeName != "" {
		return nodeName
	}
	// 默认使用 nodeId
	return GetNodeId()
}

// GetMasterAddr 获取主节点地址
func GetMasterAddr() string {
	if envAddr := os.Getenv("SUI_MASTER_ADDR"); envAddr != "" {
		return envAddr
	}
	return masterAddr
}

// GetNodeToken 获取节点 Token
func GetNodeToken() string {
	if envToken := os.Getenv("SUI_NODE_TOKEN"); envToken != "" {
		return envToken
	}
	return nodeToken
}

// GetExternalHost 获取外部主机地址
func GetExternalHost() string {
	if envHost := os.Getenv("SUI_EXTERNAL_HOST"); envHost != "" {
		return envHost
	}
	return externalHost
}

// GetExternalPort 获取外部端口
func GetExternalPort() int {
	if envPort := os.Getenv("SUI_EXTERNAL_PORT"); envPort != "" {
		if port, err := strconv.Atoi(envPort); err == nil {
			return port
		}
	}
	return externalPort
}

// GetSyncConfigInterval 获取配置同步间隔 (秒)
func GetSyncConfigInterval() int {
	if envInterval := os.Getenv("SUI_SYNC_CONFIG_INTERVAL"); envInterval != "" {
		if interval, err := strconv.Atoi(envInterval); err == nil && interval > 0 {
			return interval
		}
	}
	return syncConfigInterval
}

// GetSyncStatsInterval 获取统计上报间隔 (秒)
func GetSyncStatsInterval() int {
	if envInterval := os.Getenv("SUI_SYNC_STATS_INTERVAL"); envInterval != "" {
		if interval, err := strconv.Atoi(envInterval); err == nil && interval > 0 {
			return interval
		}
	}
	return syncStatsInterval
}

// ========== 便捷判断函数 ==========

// IsStandalone 是否单机模式
func IsStandalone() bool {
	return GetNodeMode() == ModeStandalone
}

// IsMaster 是否主节点模式
func IsMaster() bool {
	return GetNodeMode() == ModeMaster
}

// IsWorker 是否从节点模式
func IsWorker() bool {
	return GetNodeMode() == ModeWorker
}

// IsReadOnly 是否只读模式 (从节点为只读)
func IsReadOnly() bool {
	return IsWorker()
}
