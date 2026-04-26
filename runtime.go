package gb28181

import (
	"fmt"
	"net"
	"sync"
	"time"

	"go-gb28181/platform"
	"go-gb28181/sip"
)

// DeviceSession 协议层设备会话状态（纯内存，不含业务持久化字段）。
type DeviceSession struct {
	PlayMu sync.Mutex

	IsOnline bool
	Address  string
	Password string

	LastKeepaliveAt time.Time
	LastRegisterAt  time.Time
	Expires         int

	KeepaliveInterval uint16
	KeepaliveTimeout  uint16
}

// CatalogItem 为目录聚合保存的项，包含父设备 ID 与通道元数据。
type CatalogItem struct {
	DeviceID string
	Item     CatalogDeviceItem
}

// Runtime 聚合 GB28181 协议态与信令生命周期。
type Runtime struct {
	svc      *Service
	platform *Address

	peers    *platform.DeviceRegistry
	sessions sync.Map
	catalog  *sip.Collector[CatalogItem]
}

func NewRuntime(platformAddr *Address) *Runtime {
	svc := NewService(platformAddr)
	return &Runtime{
		svc:      svc,
		platform: platformAddr,
		peers:    platform.NewDeviceRegistry(),
		catalog: sip.NewCollector(func(c1, c2 *CatalogItem) bool {
			return c1.Item.DeviceID == c2.Item.DeviceID
		}),
	}
}

func (r *Runtime) Service() *Service {
	return r.svc
}

func (r *Runtime) Server() *Server {
	if r == nil || r.svc == nil {
		return nil
	}
	return r.svc.Server
}

func (r *Runtime) Platform() *Address {
	return r.platform
}

func (r *Runtime) SetPlatform(from Address) {
	r.platform = &from
	if r.svc != nil {
		r.svc.platform = &from
		if r.svc.Server != nil {
			r.svc.Server.SetFrom(&from)
		}
	}
}

func (r *Runtime) RegisterHandlers(h MessageHandler, opts ...RegisterOptions) {
	if r == nil || r.svc == nil {
		return
	}
	r.svc.RegisterHandlers(h, opts...)
}

func (r *Runtime) Start(port int) {
	if r == nil || r.svc == nil || r.svc.Server == nil {
		return
	}
	go r.svc.ListenUDPServer(fmt.Sprintf(":%d", port))
	go r.svc.ListenTCPServer(fmt.Sprintf(":%d", port))
}

func (r *Runtime) Close() {
	if r == nil || r.svc == nil || r.svc.Server == nil {
		return
	}
	r.svc.Close()
}

func (r *Runtime) UDPConn(wait time.Duration) sip.Connection {
	if r == nil || r.svc == nil || r.svc.Server == nil {
		return nil
	}
	deadline := time.Now().Add(wait)
	for {
		if conn := r.svc.Server.UDPConn(); conn != nil {
			return conn
		}
		if time.Now().After(deadline) {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (r *Runtime) EnsurePeer(deviceID string) *platform.Endpoint {
	return r.peers.Ensure(deviceID)
}

func (r *Runtime) LoadPeer(deviceID string) (*platform.Endpoint, bool) {
	return r.peers.Load(deviceID)
}

func (r *Runtime) StorePeer(deviceID string, ep *platform.Endpoint) {
	r.peers.Store(deviceID, ep)
}

func (r *Runtime) RangePeers(fn func(deviceGBID string, ep *platform.Endpoint) bool) {
	r.peers.Range(fn)
}

func (r *Runtime) EnsureSession(deviceID string) *DeviceSession {
	if v, ok := r.sessions.Load(deviceID); ok {
		if st, ok2 := v.(*DeviceSession); ok2 {
			return st
		}
	}
	st := &DeviceSession{}
	if actual, loaded := r.sessions.LoadOrStore(deviceID, st); loaded {
		if a, ok := actual.(*DeviceSession); ok {
			return a
		}
	}
	return st
}

func (r *Runtime) LoadSession(deviceID string) (*DeviceSession, bool) {
	v, ok := r.sessions.Load(deviceID)
	if !ok {
		return nil, false
	}
	st, ok := v.(*DeviceSession)
	return st, ok
}

func (r *Runtime) StoreSession(deviceID string, st *DeviceSession) {
	r.sessions.Store(deviceID, st)
}

func (r *Runtime) RangeSessions(fn func(deviceID string, st *DeviceSession) bool) {
	r.sessions.Range(func(key, value any) bool {
		deviceID, ok := key.(string)
		if !ok {
			return true
		}
		st, ok := value.(*DeviceSession)
		if !ok {
			return true
		}
		return fn(deviceID, st)
	})
}

func (r *Runtime) CatalogRun(deviceID string) {
	r.catalog.Run(deviceID)
}

func (r *Runtime) CatalogWrite(deviceID string, total int, item CatalogDeviceItem) {
	v := CatalogItem{DeviceID: deviceID, Item: item}
	r.catalog.Write(&sip.CollectorMsg[CatalogItem]{
		Key:   deviceID,
		Data:  &v,
		Total: total,
	})
}

func (r *Runtime) CatalogWait(deviceID string) {
	r.catalog.Wait(deviceID)
}

func (r *Runtime) StartCatalogLoop(save func(deviceID string, items []*CatalogItem)) {
	go r.catalog.Start(save)
}

func (r *Runtime) WrapRequest(t platform.DialogueTarget, method string, contentType *sip.ContentType, body []byte, opts ...platform.RequestOption) (*sip.Transaction, error) {
	if r == nil || r.svc == nil || r.svc.Server == nil || r.platform == nil {
		return nil, fmt.Errorf("runtime not ready")
	}
	return platform.OutboundRequest(r.svc.Server, r.platform, t, method, contentType, body, opts...)
}

// PTZControl 使用 Runtime 已登记的设备 Endpoint 下发云台方向控制。
func (r *Runtime) PTZControl(deviceID, channelOrDeviceID string, action PTZAction, speed int) (*Transaction, error) {
	if r == nil || r.svc == nil || r.svc.Server == nil {
		return nil, fmt.Errorf("runtime not ready")
	}
	st, ok := r.LoadSession(deviceID)
	if !ok || st == nil || !st.IsOnline {
		return nil, fmt.Errorf("device offline")
	}
	ep, ok := r.LoadPeer(deviceID)
	if !ok || ep == nil {
		return nil, fmt.Errorf("device endpoint not found")
	}
	body, err := BuildDeviceControlPTZXML(channelOrDeviceID, action, speed)
	if err != nil {
		return nil, err
	}
	ct := ContentTypeXML
	return r.WrapRequest(ep, MethodMessage, &ct, body)
}

func (r *Runtime) NewChannelTarget(deviceID, domain, channelID string) (*platform.ChannelTarget, bool) {
	ep, ok := r.peers.Load(deviceID)
	if !ok || ep == nil {
		return nil, false
	}
	ch, err := platform.NewChannelTarget(ep, domain, channelID)
	if err != nil {
		return nil, false
	}
	return ch, true
}

func (r *Runtime) RebindUDPDevice(conn sip.Connection, deviceID, contactHost string, timeout time.Duration) error {
	ep, err := platform.NewUDPEndpoint(conn, deviceID, contactHost)
	if err != nil {
		return err
	}
	if err = platform.CheckUDPAddrReachable(ep.Source(), timeout); err != nil {
		return err
	}
	r.peers.Store(deviceID, ep)
	return nil
}

func (r *Runtime) PeerSource(deviceID string) net.Addr {
	ep, ok := r.peers.Load(deviceID)
	if !ok || ep == nil {
		return nil
	}
	return ep.Source()
}
