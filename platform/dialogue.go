package platform

import (
	"net"

	"go-gb28181/sip"
)

// DialogueTarget 描述平台主动发 SIP 时的对端：连接、源地址、To 头（设备或通道）。
type DialogueTarget interface {
	To() *sip.Address
	Conn() sip.Connection
	Source() net.Addr
}
