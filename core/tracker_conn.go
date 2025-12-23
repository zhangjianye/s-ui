package core

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing/common/network"
)

type ConnectionInfo struct {
	ID         string
	Conn       net.Conn
	PacketConn network.PacketConn
	Inbound    string
	Type       string // "tcp" or "udp"
	// UAP 扩展字段
	User        string // 用户名 (从 metadata.User)
	SourceIP    string // 来源 IP (从 metadata.Source)
	ConnectedAt int64  // 连接时间戳
}

type ConnTracker struct {
	access      sync.Mutex
	connections map[string]*ConnectionInfo
}

func NewConnTracker() *ConnTracker {
	return &ConnTracker{
		connections: make(map[string]*ConnectionInfo),
	}
}

func (c *ConnTracker) generateConnectionID() string {
	return uuid.Must(uuid.NewV4()).String()
}

func (c *ConnTracker) RoutedConnection(ctx context.Context, conn net.Conn, metadata adapter.InboundContext, matchedRule adapter.Rule, matchOutbound adapter.Outbound) net.Conn {
	connID := c.generateConnectionID()
	connInfo := &ConnectionInfo{
		ID:          connID,
		Conn:        conn,
		Inbound:     metadata.Inbound,
		Type:        "tcp",
		User:        metadata.User,
		SourceIP:    metadata.Source.String(),
		ConnectedAt: time.Now().Unix(),
	}

	c.trackConnection(connID, connInfo)

	return c.createWrappedConn(conn, connID)
}

func (c *ConnTracker) RoutedPacketConnection(ctx context.Context, conn network.PacketConn, metadata adapter.InboundContext, matchedRule adapter.Rule, matchOutbound adapter.Outbound) network.PacketConn {
	connID := c.generateConnectionID()
	connInfo := &ConnectionInfo{
		ID:          connID,
		PacketConn:  conn,
		Inbound:     metadata.Inbound,
		Type:        "udp",
		User:        metadata.User,
		SourceIP:    metadata.Source.String(),
		ConnectedAt: time.Now().Unix(),
	}

	c.trackConnection(connID, connInfo)

	return c.createWrappedPacketConn(conn, connID)
}

func (c *ConnTracker) CloseConnByInbound(inbound string) int {
	c.access.Lock()
	defer c.access.Unlock()

	closedCount := 0
	for connID, connInfo := range c.connections {
		if connInfo.Inbound == inbound {
			if connInfo.Conn != nil {
				connInfo.Conn.Close()
			}
			if connInfo.PacketConn != nil {
				connInfo.PacketConn.Close()
			}
			delete(c.connections, connID)
			closedCount++
		}
	}
	return closedCount
}

func (c *ConnTracker) trackConnection(connID string, connInfo *ConnectionInfo) {
	c.access.Lock()
	defer c.access.Unlock()
	c.connections[connID] = connInfo
}

func (c *ConnTracker) untrackConnection(connID string) {
	c.access.Lock()
	defer c.access.Unlock()
	delete(c.connections, connID)
}

func (c *ConnTracker) createWrappedConn(conn net.Conn, connID string) *wrappedConn {
	return &wrappedConn{
		Conn:   conn,
		connID: connID,
	}
}

func (c *ConnTracker) createWrappedPacketConn(conn network.PacketConn, connID string) *wrappedPacketConn {
	return &wrappedPacketConn{
		PacketConn: conn,
		connID:     connID,
	}
}

type wrappedConn struct {
	net.Conn
	connID string
}

func (w *wrappedConn) Close() error {
	connTracker.untrackConnection(w.connID)
	return w.Conn.Close()
}

func (w *wrappedConn) Upstream() any {
	return w.Conn
}

type wrappedPacketConn struct {
	network.PacketConn
	connID string
}

func (w *wrappedPacketConn) Close() error {
	connTracker.untrackConnection(w.connID)
	return w.PacketConn.Close()
}

func (w *wrappedPacketConn) Upstream() any {
	return w.PacketConn
}

// GetUserConnectionCount 获取用户当前连接数
func (c *ConnTracker) GetUserConnectionCount(user string) int {
	c.access.Lock()
	defer c.access.Unlock()

	count := 0
	for _, connInfo := range c.connections {
		if connInfo.User == user {
			count++
		}
	}
	return count
}

// GetUserConnections 获取用户所有连接信息
func (c *ConnTracker) GetUserConnections(user string) []*ConnectionInfo {
	c.access.Lock()
	defer c.access.Unlock()

	var conns []*ConnectionInfo
	for _, connInfo := range c.connections {
		if connInfo.User == user {
			conns = append(conns, connInfo)
		}
	}
	return conns
}

// GetUniqueDeviceCount 获取用户唯一设备数 (基于 SourceIP)
func (c *ConnTracker) GetUniqueDeviceCount(user string) int {
	c.access.Lock()
	defer c.access.Unlock()

	devices := make(map[string]bool)
	for _, connInfo := range c.connections {
		if connInfo.User == user && connInfo.SourceIP != "" {
			devices[connInfo.SourceIP] = true
		}
	}
	return len(devices)
}

// CheckDeviceLimit 检查用户是否超出设备限制
// 返回 true 表示未超限，可以建立新连接
func (c *ConnTracker) CheckDeviceLimit(user string, limit int) bool {
	if limit <= 0 {
		return true // 0 表示无限制
	}
	return c.GetUniqueDeviceCount(user) < limit
}

// GetOnlineUsers 获取当前在线的所有用户列表
func (c *ConnTracker) GetOnlineUsers() []string {
	c.access.Lock()
	defer c.access.Unlock()

	users := make(map[string]bool)
	for _, connInfo := range c.connections {
		if connInfo.User != "" {
			users[connInfo.User] = true
		}
	}

	result := make([]string, 0, len(users))
	for user := range users {
		result = append(result, user)
	}
	return result
}

// GetConnections 获取所有连接信息 (用于从节点上报在线状态)
func (c *ConnTracker) GetConnections() []*ConnectionInfo {
	c.access.Lock()
	defer c.access.Unlock()

	result := make([]*ConnectionInfo, 0, len(c.connections))
	for _, connInfo := range c.connections {
		result = append(result, connInfo)
	}
	return result
}
