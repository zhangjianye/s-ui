package core

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/alireza0/s-ui/database/model"

	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing/common/atomic"
	"github.com/sagernet/sing/common/bufio"
	"github.com/sagernet/sing/common/network"
)

type Counter struct {
	read  *atomic.Int64
	write *atomic.Int64
}

type StatsTracker struct {
	access    sync.Mutex
	inbounds  map[string]Counter
	outbounds map[string]Counter
	users     map[string]Counter
}

func NewStatsTracker() *StatsTracker {
	return &StatsTracker{
		inbounds:  make(map[string]Counter),
		outbounds: make(map[string]Counter),
		users:     make(map[string]Counter),
	}
}

func (c *StatsTracker) getReadCounters(inbound string, outbound string, user string) ([]*atomic.Int64, []*atomic.Int64) {
	var readCounter []*atomic.Int64
	var writeCounter []*atomic.Int64
	c.access.Lock()
	defer c.access.Unlock()

	if inbound != "" {
		readCounter = append(readCounter, c.loadOrCreateCounter(&c.inbounds, inbound).read)
		writeCounter = append(writeCounter, c.inbounds[inbound].write)
	}
	if outbound != "" {
		readCounter = append(readCounter, c.loadOrCreateCounter(&c.outbounds, outbound).read)
		writeCounter = append(writeCounter, c.outbounds[outbound].write)
	}
	if user != "" {
		readCounter = append(readCounter, c.loadOrCreateCounter(&c.users, user).read)
		writeCounter = append(writeCounter, c.users[user].write)
	}
	return readCounter, writeCounter
}

func (c *StatsTracker) loadOrCreateCounter(obj *map[string]Counter, name string) Counter {
	counter, loaded := (*obj)[name]
	if loaded {
		return counter
	}
	counter = Counter{read: &atomic.Int64{}, write: &atomic.Int64{}}
	(*obj)[name] = counter
	return counter
}

func (c *StatsTracker) RoutedConnection(ctx context.Context, conn net.Conn, metadata adapter.InboundContext, matchedRule adapter.Rule, matchOutbound adapter.Outbound) net.Conn {
	readCounter, writeCounter := c.getReadCounters(metadata.Inbound, matchOutbound.Tag(), metadata.User)
	return bufio.NewInt64CounterConn(conn, readCounter, writeCounter)
}

func (c *StatsTracker) RoutedPacketConnection(ctx context.Context, conn network.PacketConn, metadata adapter.InboundContext, matchedRule adapter.Rule, matchOutbound adapter.Outbound) network.PacketConn {
	readCounter, writeCounter := c.getReadCounters(metadata.Inbound, matchOutbound.Tag(), metadata.User)
	return bufio.NewInt64CounterPacketConn(conn, readCounter, nil, writeCounter, nil)
}

func (c *StatsTracker) GetStats() *[]model.Stats {
	c.access.Lock()
	defer c.access.Unlock()

	dt := time.Now().Unix()

	s := []model.Stats{}
	for inbound, counter := range c.inbounds {
		down := counter.write.Swap(0)
		up := counter.read.Swap(0)
		if down > 0 || up > 0 {
			s = append(s, model.Stats{
				DateTime:  dt,
				Resource:  "inbound",
				Tag:       inbound,
				Direction: false,
				Traffic:   down,
			}, model.Stats{
				DateTime:  dt,
				Resource:  "inbound",
				Tag:       inbound,
				Direction: true,
				Traffic:   up,
			})
		}
	}

	for outbound, counter := range c.outbounds {
		down := counter.write.Swap(0)
		up := counter.read.Swap(0)
		if down > 0 || up > 0 {
			s = append(s, model.Stats{
				DateTime:  dt,
				Resource:  "outbound",
				Tag:       outbound,
				Direction: false,
				Traffic:   down,
			}, model.Stats{
				DateTime:  dt,
				Resource:  "outbound",
				Tag:       outbound,
				Direction: true,
				Traffic:   up,
			})
		}
	}

	for user, counter := range c.users {
		down := counter.write.Swap(0)
		up := counter.read.Swap(0)
		if down > 0 || up > 0 {
			s = append(s, model.Stats{
				DateTime:  dt,
				Resource:  "user",
				Tag:       user,
				Direction: false,
				Traffic:   down,
			}, model.Stats{
				DateTime:  dt,
				Resource:  "user",
				Tag:       user,
				Direction: true,
				Traffic:   up,
			})
		}
	}
	return &s
}

// UserTimeTracker 用户在线时长追踪器
type UserTimeTracker struct {
	access     sync.Mutex
	onlineTime map[string]int64 // user -> 本周期在线秒数
	lastUpdate map[string]int64 // user -> 上次更新时间戳
}

// NewUserTimeTracker 创建用户时长追踪器
func NewUserTimeTracker() *UserTimeTracker {
	return &UserTimeTracker{
		onlineTime: make(map[string]int64),
		lastUpdate: make(map[string]int64),
	}
}

// UpdateOnlineTime 更新用户在线时长
// 每次调用时，为当前在线的用户累加时间间隔
// interval: 采集间隔秒数 (默认 10 秒)
func (t *UserTimeTracker) UpdateOnlineTime(onlineUsers []string, interval int64) {
	t.access.Lock()
	defer t.access.Unlock()

	now := time.Now().Unix()
	onlineSet := make(map[string]bool)
	for _, user := range onlineUsers {
		onlineSet[user] = true
	}

	// 为在线用户累加时间
	for user := range onlineSet {
		if _, exists := t.onlineTime[user]; !exists {
			t.onlineTime[user] = 0
		}
		t.onlineTime[user] += interval
		t.lastUpdate[user] = now
	}
}

// GetAndResetTime 获取并重置用户在线时长
// 返回 map[user]秒数，同时清空累计数据
func (t *UserTimeTracker) GetAndResetTime() map[string]int64 {
	t.access.Lock()
	defer t.access.Unlock()

	result := make(map[string]int64)
	for user, seconds := range t.onlineTime {
		if seconds > 0 {
			result[user] = seconds
		}
	}

	// 重置
	t.onlineTime = make(map[string]int64)
	t.lastUpdate = make(map[string]int64)

	return result
}

// GetOnlineTime 获取用户当前周期的在线时长 (不重置)
func (t *UserTimeTracker) GetOnlineTime(user string) int64 {
	t.access.Lock()
	defer t.access.Unlock()

	return t.onlineTime[user]
}
