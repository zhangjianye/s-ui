package cronjob

import (
	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/logger"
	"github.com/alireza0/s-ui/service"
)

// TimeTrackJob 用户在线时长追踪任务
// 每 10 秒执行一次，累加用户在线时长
type TimeTrackJob struct {
	service.ClientService
}

func NewTimeTrackJob() *TimeTrackJob {
	return new(TimeTrackJob)
}

func (j *TimeTrackJob) Run() {
	// 从 core 获取时长追踪数据并累加到数据库
	err := j.ClientService.UpdateOnlineTime()
	if err != nil {
		logger.Warning("Update online time failed: ", err)
	}
}

// TimeDepleteJob 时长超限检查任务
// 每 1 分钟执行一次，检查并禁用时长超限用户
type TimeDepleteJob struct {
	service.ClientService
	service.InboundService
	service.WebhookService
}

func NewTimeDepleteJob() *TimeDepleteJob {
	return new(TimeDepleteJob)
}

func (j *TimeDepleteJob) Run() {
	inboundIds, disabledClients, err := j.ClientService.DepleteTimeExceededClients()
	if err != nil {
		logger.Warning("Deplete time exceeded clients failed: ", err)
		return
	}

	// 发送 Webhook 通知
	for _, client := range disabledClients {
		j.WebhookService.SendClientEvent(
			service.EventTimeExceeded,
			client.Name,
			client.UUID,
			"time_limit_exceeded",
		)
	}

	// 重启受影响的 inbound
	if len(inboundIds) > 0 {
		err := j.InboundService.RestartInbounds(database.GetDB(), inboundIds)
		if err != nil {
			logger.Error("Unable to restart inbounds after time deplete: ", err)
		}
	}
}
