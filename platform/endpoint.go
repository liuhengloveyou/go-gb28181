package platform

import (
	"net"
	"sync"

	"go-gb28181/sip"
)

// Endpoint 单台国标设备在当前进程内的 SIP 传输绑定（不可序列化）。
type Endpoint struct {
	mu     sync.RWMutex
	conn   sip.Connection
	source net.Addr
	to     *sip.Address
}

// NewEndpoint 创建空绑定，后续通过 Bind 更新。
func NewEndpoint() *Endpoint {
	return &Endpoint{}
}

// Bind 更新连接与路由（通常在 REGISTER / Keepalive 时调用）。
func (e *Endpoint) Bind(conn sip.Connection, source net.Addr, to *sip.Address) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.conn = conn
	e.source = source
	e.to = to
}

// Conn 当前用于收发的 SIP Connection（多为共享 UDP）。
func (e *Endpoint) Conn() sip.Connection {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.conn
}

// Source 对端地址（UDP 为设备 contact；用于 SetDestination）。
func (e *Endpoint) Source() net.Addr {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.source
}

// To 设备级 SIP To（发往设备本身的请求）。
func (e *Endpoint) To() *sip.Address {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.to
}

var _ DialogueTarget = (*Endpoint)(nil)
