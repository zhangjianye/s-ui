package cronjob

import (
	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/logger"
	"github.com/alireza0/s-ui/service"
	"github.com/alireza0/s-ui/util/common"
)

// ResetJob 流量/时长重置任务
// 每天执行一次，按策略重置用户的流量和时长
type ResetJob struct {
	service.ClientService
	service.InboundService
	service.WebhookService
}

func NewResetJob() *ResetJob {
	return new(ResetJob)
}

func (j *ResetJob) Run() {
	// 重置流量
	trafficResetClients, trafficInboundIds, err := j.ClientService.ResetTrafficByStrategy()
	if err != nil {
		logger.Warning("Reset traffic by strategy failed: ", err)
	} else {
		for _, client := range trafficResetClients {
			j.WebhookService.SendClientEvent(
				service.EventTrafficReset,
				client.Name,
				client.UUID,
				client.TrafficResetStrategy,
			)
		}
	}

	// 重置时长
	timeResetClients, timeInboundIds, err := j.ClientService.ResetTimeByStrategy()
	if err != nil {
		logger.Warning("Reset time by strategy failed: ", err)
	} else {
		for _, client := range timeResetClients {
			j.WebhookService.SendClientEvent(
				service.EventTimeReset,
				client.Name,
				client.UUID,
				client.TimeResetStrategy,
			)
		}
	}

	// 合并需要重启的 inbound
	allInboundIds := common.UnionUintArray(trafficInboundIds, timeInboundIds)

	// 重启受影响的 inbound
	if len(allInboundIds) > 0 {
		err := j.InboundService.RestartInbounds(database.GetDB(), allInboundIds)
		if err != nil {
			logger.Error("Unable to restart inbounds after reset: ", err)
		}
	}
}
