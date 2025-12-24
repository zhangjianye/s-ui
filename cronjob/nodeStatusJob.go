package cronjob

import (
	"github.com/alireza0/s-ui/config"
	"github.com/alireza0/s-ui/service"
)

type NodeStatusJob struct {
	service.NodeService
}

func NewNodeStatusJob() *NodeStatusJob {
	return &NodeStatusJob{}
}

func (s *NodeStatusJob) Run() {
	// 仅在主节点模式下运行
	if !config.IsMaster() {
		return
	}
	s.NodeService.UpdateNodeStatus()
}
