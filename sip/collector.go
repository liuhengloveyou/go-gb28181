/*
泛型收集器：按设备 ID 聚合分片上报的数据（如目录 Item），定时落盘并通知 Wait 方；支持去重回调。
*/

package sip

import (
	"slices"
	"time"
)

// Collector .
// 1. 收集器
// 2. 分门别类
// 3. 定时同步，超时删除，删除之前再同步一次
// 4. 不会去重
// 如何使用?
// 1. 通过 NewCatalogRecv 创建一个新的收集器
// 2. s.createCh <- deviceID
// 3. s.catalog.msg <- &CollectorMsg[Channel]{Data: &c, Total: msg.SumNum, Key: msg.DeviceID}
type Collector[T any] struct {
	data       map[string]*Content[T]
	msg        chan *CollectorMsg[T]
	createCh   chan string
	noRepeatFn NoRepeatFn[T]
	observer   *Observer
}

func (c *Collector[T]) Run(key string) {
	select {
	case c.createCh <- key:
	default:
	}
}

func (c *Collector[T]) Write(info *CollectorMsg[T]) {
	c.msg <- info
}

type CollectorMsg[T any] struct {
	Key   string
	Data  *T
	Total int
}

type NoRepeatFn[T any] func(*T, *T) bool

// newCollector 创建一个新的收集器
// noRepeatFn 用于提前去重，避免重复数据存储
func NewCollector[T any](noRepeatFn NoRepeatFn[T]) *Collector[T] {
	return &Collector[T]{
		data:       make(map[string]*Content[T]),
		msg:        make(chan *CollectorMsg[T], 512),
		createCh:   make(chan string, 100),
		noRepeatFn: noRepeatFn,
		observer:   NewObserver(),
	}
}

type Content[T any] struct {
	lastUpdateAt time.Time
	data         []*T
	total        int
}

// Wait 在执行 Start 以后，可以调用 Wait 等待
func (c *Collector[T]) Wait(key string) {
	c.observer.DefaultRegister(key)
}

// Start 启动定时任务检查和保存数据
func (c *Collector[T]) Start(save func(string, []*T)) {
	fn := func(k string, data []*T) {
		save(k, data)
		c.observer.Notify(k)
	}

	check := time.NewTicker(time.Second * 3)
	defer check.Stop()
	for {
		select {
		case <-check.C:
			for k, v := range c.data {
				if time.Since(v.lastUpdateAt) > 10*time.Second {
					fn(k, v.data)
					delete(c.data, k)
					continue
				}
				if v.total > 0 && len(v.data) >= v.total {
					fn(k, v.data)
					delete(c.data, k)
					continue
				}
			}
		case v := <-c.createCh:
			c.data[v] = &Content[T]{lastUpdateAt: time.Now(), data: make([]*T, 0, 2), total: -1}
		case msg := <-c.msg:
			data, exist := c.data[msg.Key]
			if !exist {
				sipLog.Info("sip collector: key missing or expired", "key", msg.Key, "data", msg.Data)
				continue
			}
			// 如果数据已存在且无重复，跳过该消息
			if slices.ContainsFunc(data.data, func(v *T) bool {
				return c.noRepeatFn(v, msg.Data)
			}) {
				sipLog.Info("sip collector: duplicate catalog item", "key", msg.Key, "data", msg.Data)
				continue
			}
			// 添加数据到对应的条目并更新最后更新时间和总量
			data.data = append(data.data, msg.Data)
			data.lastUpdateAt = time.Now()
			data.total = msg.Total
		}
	}
}
