package platform

import (
	"fmt"
	"net"
	"time"

	"go-gb28181/sip"
)

// NewUDPEndpoint 用共享 UDP Connection 与库中 contact 地址构建设备级 Endpoint（启动恢复等）。
func NewUDPEndpoint(conn sip.Connection, deviceGBID, contactHost string) (*Endpoint, error) {
	uri, err := sip.ParseURI(fmt.Sprintf("sip:%s@%s", deviceGBID, contactHost))
	if err != nil {
		return nil, err
	}
	addr, err := net.ResolveUDPAddr("udp", contactHost)
	if err != nil {
		return nil, err
	}
	ep := NewEndpoint()
	ep.Bind(conn, addr, &sip.Address{URI: uri, Params: sip.NewParams()})
	return ep, nil
}

// CheckUDPAddrReachable 对 UDP 地址做一次短时拨测（启动时剔除不可达设备）。
func CheckUDPAddrReachable(addr net.Addr, timeout time.Duration) error {
	if addr == nil {
		return fmt.Errorf("nil addr")
	}
	if addr.Network() == "tcp" {
		return nil
	}
	c, err := net.DialTimeout("udp", addr.String(), timeout)
	if err != nil {
		return err
	}
	return c.Close()
}
