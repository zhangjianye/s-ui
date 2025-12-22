package cmd

import (
	"flag"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/alireza0/s-ui/cmd/migration"
	"github.com/alireza0/s-ui/config"
)

// 节点相关命令行参数
var (
	showVersion  bool
	Mode         string
	MasterAddr   string
	NodeToken    string
	NodeId       string
	NodeName     string
	ExternalHost string
	ExternalPort int
)

func init() {
	// 注册节点相关参数
	flag.BoolVar(&showVersion, "v", false, "show version")
	flag.StringVar(&Mode, "mode", "standalone", "node mode: standalone, master, worker")
	flag.StringVar(&MasterAddr, "master", "", "master node address (required for worker mode)")
	flag.StringVar(&NodeToken, "token", "", "node token for authentication (required for worker mode)")
	flag.StringVar(&NodeId, "node-id", "", "unique node identifier (required for worker mode)")
	flag.StringVar(&NodeName, "node-name", "", "node display name (defaults to node-id)")
	flag.StringVar(&ExternalHost, "external-host", "", "external host for client connections")
	flag.IntVar(&ExternalPort, "external-port", 0, "external port for client connections (0 = same as inbound)")
}

// ParseFlags 解析命令行参数并应用到 config
// 返回 true 表示应该启动应用，false 表示已处理完毕（如显示版本）
func ParseFlags() bool {
	flag.Parse()

	if showVersion {
		fmt.Println("S-UI Panel\t", config.GetVersion())
		info, ok := debug.ReadBuildInfo()
		if ok {
			for _, dep := range info.Deps {
				if dep.Path == "github.com/sagernet/sing-box" {
					fmt.Println("Sing-Box\t", dep.Version)
					break
				}
			}
		}
		return false
	}

	// 应用节点配置
	config.SetNodeMode(Mode)
	config.SetMasterAddr(MasterAddr)
	config.SetNodeToken(NodeToken)
	config.SetNodeId(NodeId)
	config.SetNodeName(NodeName)
	config.SetExternalHost(ExternalHost)
	config.SetExternalPort(ExternalPort)

	// 验证 worker 模式必需参数
	if config.IsWorker() {
		if config.GetMasterAddr() == "" {
			fmt.Println("Error: --master is required for worker mode")
			os.Exit(1)
		}
		if config.GetNodeToken() == "" {
			fmt.Println("Error: --token is required for worker mode")
			os.Exit(1)
		}
		if config.GetNodeId() == "" {
			fmt.Println("Error: --node-id is required for worker mode")
			os.Exit(1)
		}
	}

	return true
}

func ParseCmd() {
	// 先解析 flags (已在 init 中注册)
	if !ParseFlags() {
		return
	}

	adminCmd := flag.NewFlagSet("admin", flag.ExitOnError)
	settingCmd := flag.NewFlagSet("setting", flag.ExitOnError)

	var username string
	var password string
	var port int
	var path string
	var subPort int
	var subPath string
	var reset bool
	var show bool
	settingCmd.BoolVar(&reset, "reset", false, "reset all settings")
	settingCmd.BoolVar(&show, "show", false, "show current settings")
	settingCmd.IntVar(&port, "port", 0, "set panel port")
	settingCmd.StringVar(&path, "path", "", "set panel path")
	settingCmd.IntVar(&subPort, "subPort", 0, "set sub port")
	settingCmd.StringVar(&subPath, "subPath", "", "set sub path")

	adminCmd.BoolVar(&show, "show", false, "show first admin credentials")
	adminCmd.BoolVar(&reset, "reset", false, "reset first admin credentials")
	adminCmd.StringVar(&username, "username", "", "set login username")
	adminCmd.StringVar(&password, "password", "", "set login password")

	oldUsage := flag.Usage
	flag.Usage = func() {
		oldUsage()
		fmt.Println()
		fmt.Println("Node Options:")
		fmt.Println("    --mode           node mode: standalone, master, worker (default: standalone)")
		fmt.Println("    --master         master node address (required for worker mode)")
		fmt.Println("    --token          node token for authentication (required for worker mode)")
		fmt.Println("    --node-id        unique node identifier (required for worker mode)")
		fmt.Println("    --node-name      node display name (defaults to node-id)")
		fmt.Println("    --external-host  external host for client connections")
		fmt.Println("    --external-port  external port for client connections")
		fmt.Println()
		fmt.Println("Commands:")
		fmt.Println("    admin          set/reset/show first admin credentials")
		fmt.Println("    uri            Show panel URI")
		fmt.Println("    migrate        migrate form older version")
		fmt.Println("    setting        set/reset/show settings")
		fmt.Println()
		adminCmd.Usage()
		fmt.Println()
		settingCmd.Usage()
	}

	// 检查是否有子命令
	args := flag.Args()
	if len(args) == 0 {
		// 没有子命令，不应该通过 ParseCmd 到达这里
		return
	}

	switch args[0] {
	case "admin":
		err := adminCmd.Parse(args[1:])
		if err != nil {
			fmt.Println(err)
			return
		}
		switch {
		case show:
			showAdmin()
		case reset:
			resetAdmin()
		default:
			updateAdmin(username, password)
			showAdmin()
		}

	case "uri":
		getPanelURI()

	case "migrate":
		migration.MigrateDb()

	case "setting":
		err := settingCmd.Parse(args[1:])
		if err != nil {
			fmt.Println(err)
			return
		}
		switch {
		case show:
			showSetting()
		case reset:
			resetSetting()
		default:
			updateSetting(port, path, subPort, subPath)
			showSetting()
		}
	default:
		fmt.Println("Invalid subcommands")
		flag.Usage()
	}
}
