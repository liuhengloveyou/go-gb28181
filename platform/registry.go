package platform

import (
	"sync"
)

// DeviceRegistry 国标设备 ID → SIP 传输绑定（进程内会话状态）。
type DeviceRegistry struct {
	peers sync.Map
}

// NewDeviceRegistry 创建空注册表。
func NewDeviceRegistry() *DeviceRegistry {
	return &DeviceRegistry{}
}

// Ensure 返回已有绑定，否则创建并登记空 Endpoint。
func (r *DeviceRegistry) Ensure(deviceGBID string) *Endpoint {
	if v, ok := r.peers.Load(deviceGBID); ok {
		if ep, ok2 := v.(*Endpoint); ok2 {
			return ep
		}
	}
	ep := NewEndpoint()
	if actual, loaded := r.peers.LoadOrStore(deviceGBID, ep); loaded {
		if a, ok := actual.(*Endpoint); ok {
			return a
		}
	}
	return ep
}

// Store 覆盖或写入绑定。
func (r *DeviceRegistry) Store(deviceGBID string, ep *Endpoint) {
	r.peers.Store(deviceGBID, ep)
}

// Load 查询绑定。
func (r *DeviceRegistry) Load(deviceGBID string) (*Endpoint, bool) {
	v, ok := r.peers.Load(deviceGBID)
	if !ok {
		return nil, false
	}
	ep, ok := v.(*Endpoint)
	return ep, ok
}

// Range 遍历绑定（如心跳巡检）。
func (r *DeviceRegistry) Range(fn func(deviceGBID string, ep *Endpoint) bool) {
	r.peers.Range(func(key, value any) bool {
		deviceID, ok := key.(string)
		if !ok {
			return true
		}
		ep, ok := value.(*Endpoint)
		if !ok {
			return true
		}
		return fn(deviceID, ep)
	})
}
