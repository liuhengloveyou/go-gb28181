/*
观察者：Collector 完成聚合后通知等待方；支持超时注销，用于目录查询等“发请求—等收齐”的同步语义。
*/

package sip

import (
	"fmt"
	"sync"
	"time"
)

// ObserverHandler 返回 true 表示完成任务
type ObserverHandler func(deviceID string, args ...string) bool

// Observer 观察者
type Observer struct {
	data sync.Map // key string -> ObserverHandler
}

// NewObserver 创建观察者
func NewObserver() *Observer {
	return &Observer{}
}

// concRegister 异步注册观察者
func (o *Observer) concRegister(deviceID string, handler ObserverHandler) {
	o.data.Store(deviceID, handler)
}

// Register 同步等待观察者完成任务
func (o *Observer) Register(deviceID string, duration time.Duration, fn ObserverHandler) {
	ch := make(chan struct{}, 1)
	defer close(ch)
	o.concRegister(deviceID, func(did string, args ...string) bool {
		if fn(did, args...) {
			ch <- struct{}{}
			return true
		}
		return false
	})
	// 等待通知或超时
	select {
	// 收到通知
	case <-ch:
	// 超时7秒
	case <-time.After(duration):
		o.data.Delete(deviceID)
	}
}

// DefaultRegister 默认的注册行为
func (o *Observer) DefaultRegister(deviceID string) {
	key := fmt.Sprintf("%s:%d", deviceID, time.Now().UnixMilli())
	o.Register(key, 7*time.Second, func(did string, _ ...string) bool {
		return deviceID == did
	})
}

// RegisterWithTimeout 自定义等待时间
func (o *Observer) RegisterWithTimeout(deviceID string, duration time.Duration) {
	key := fmt.Sprintf("%s:%d", deviceID, time.Now().UnixMilli())
	o.Register(key, duration, func(did string, _ ...string) bool {
		return deviceID == did
	})
}

// Notify 通知观察者
func (o *Observer) Notify(deviceID string, args ...string) {
	o.data.Range(func(key, value any) bool {
		fn, ok := value.(ObserverHandler)
		if !ok {
			return true
		}
		if fn(deviceID, args...) {
			o.data.Delete(key)
		}
		return true
	})
}
