package gb28181

import (
	"errors"
	"fmt"
	"net"
	"net/http"

	"go-gb28181/sip"
)

// Connection SIP 传输连接（出站 MESSAGE 通常使用 [Server.UDPConn]）。
type Connection = sip.Connection

var (
	// ErrNoSIPResponse 事务在超时时间内未收到对端 SIP 响应。
	ErrNoSIPResponse = errors.New("gb28181/sdk: no SIP response")
	// ErrInvalidEndpoint 设备端点信息不完整。
	ErrInvalidEndpoint = errors.New("gb28181/sdk: invalid device endpoint")
)

// DeviceEndpoint 描述一条国标信令下发目标：设备/通道 SIP 标识、域、当前对端地址及本端用于发送的连接。
// PeerAddr、Conn 一般来自设备 REGISTER/Keepalive 时记录的源地址与平台监听套接字。
type DeviceEndpoint struct {
	DeviceID string // Request-URI 用户部分（20 位国标编码）
	Domain   string // SIP 域，与 URI host 一致
	PeerAddr net.Addr
	Conn     Connection
}

func (ep DeviceEndpoint) validate() error {
	if ep.DeviceID == "" || ep.Domain == "" || ep.PeerAddr == nil || ep.Conn == nil {
		return ErrInvalidEndpoint
	}
	return nil
}

// SIPResponse 返回事务收到的 SIP 响应；未收到则 [ErrNoSIPResponse]。
func SIPResponse(tx *Transaction) (*Response, error) {
	if tx == nil {
		return nil, fmt.Errorf("gb28181/sdk: nil transaction")
	}
	res := tx.GetResponse()
	if res == nil {
		return nil, ErrNoSIPResponse
	}
	return res, nil
}

// SIPResponseOK 要求对端返回 SIP 2xx，否则返回错误（仍返回 *Response 便于读取状态码）。
func SIPResponseOK(tx *Transaction) (*Response, error) {
	res, err := SIPResponse(tx)
	if err != nil {
		return nil, err
	}
	if res.StatusCode() < http.StatusOK || res.StatusCode() >= 300 {
		return res, fmt.Errorf("gb28181/sdk: sip %d %s", res.StatusCode(), res.Reason())
	}
	return res, nil
}
