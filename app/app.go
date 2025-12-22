package app

import (
	"fmt"
	"log"

	"github.com/alireza0/s-ui/config"
	"github.com/alireza0/s-ui/core"
	"github.com/alireza0/s-ui/cronjob"
	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/logger"
	"github.com/alireza0/s-ui/service"
	"github.com/alireza0/s-ui/sub"
	"github.com/alireza0/s-ui/web"

	"github.com/op/go-logging"
)

type APP struct {
	service.SettingService
	configService *service.ConfigService
	webServer     *web.Server
	subServer     *sub.Server
	cronJob       *cronjob.CronJob
	logger        *logging.Logger
	core          *core.Core
}

func NewApp() *APP {
	return &APP{}
}

func (a *APP) Init() error {
	log.Printf("%v %v", config.GetName(), config.GetVersion())
	log.Printf("Node Mode: %v", config.GetNodeMode())

	if config.IsWorker() {
		log.Printf("Node ID: %v", config.GetNodeId())
		log.Printf("Master: %v", config.GetMasterAddr())
	}

	a.initLog()

	err := database.InitDB(config.GetDBPath())
	if err != nil {
		return err
	}

	// Init Setting
	a.SettingService.GetAllSetting()

	a.core = core.NewCore()
	a.cronJob = cronjob.NewCronJob()
	a.configService = service.NewConfigService(a.core)

	// Web 和 Sub 服务在所有模式下都启动
	// Worker 模式下 Web 为只读模式（通过 config.IsReadOnly() 判断）
	a.webServer = web.NewServer()
	a.subServer = sub.NewServer()

	// TODO: Worker 模式下初始化同步服务 (Phase 4)
	// if config.IsWorker() {
	//     a.syncService = service.NewSyncService(...)
	// }

	return nil
}

func (a *APP) Start() error {
	loc, err := a.SettingService.GetTimeLocation()
	if err != nil {
		return err
	}

	trafficAge, err := a.SettingService.GetTrafficAge()
	if err != nil {
		return err
	}

	err = a.cronJob.Start(loc, trafficAge)
	if err != nil {
		return err
	}

	err = a.webServer.Start()
	if err != nil {
		return err
	}

	err = a.subServer.Start()
	if err != nil {
		return err
	}

	// Worker 模式下，配置从主节点同步
	if config.IsWorker() {
		// TODO: Phase 4 实现同步服务
		// err = a.syncService.Start()
		// if err != nil {
		//     return err
		// }
		// 暂时不启动 Core，等待同步配置
		logger.Info("Worker mode: waiting for config sync from master")
	} else {
		// Standalone/Master 模式下正常启动 Core
		err = a.configService.StartCore("")
		if err != nil {
			logger.Error(err)
		}
	}

	a.logStartupInfo()

	return nil
}

func (a *APP) logStartupInfo() {
	mode := config.GetNodeMode()
	switch mode {
	case config.ModeMaster:
		logger.Info("Running as MASTER node")
	case config.ModeWorker:
		logger.Info(fmt.Sprintf("Running as WORKER node [%s]", config.GetNodeId()))
	default:
		logger.Info("Running in STANDALONE mode")
	}
}

func (a *APP) Stop() {
	a.cronJob.Stop()
	err := a.subServer.Stop()
	if err != nil {
		logger.Warning("stop Sub Server err:", err)
	}
	err = a.webServer.Stop()
	if err != nil {
		logger.Warning("stop Web Server err:", err)
	}
	err = a.configService.StopCore()
	if err != nil {
		logger.Warning("stop Core err:", err)
	}
}

func (a *APP) initLog() {
	switch config.GetLogLevel() {
	case config.Debug:
		logger.InitLogger(logging.DEBUG)
	case config.Info:
		logger.InitLogger(logging.INFO)
	case config.Warn:
		logger.InitLogger(logging.WARNING)
	case config.Error:
		logger.InitLogger(logging.ERROR)
	default:
		log.Fatal("unknown log level:", config.GetLogLevel())
	}
}

func (a *APP) RestartApp() {
	a.Stop()
	a.Start()
}

func (a *APP) GetCore() *core.Core {
	return a.core
}
