package platform

import (
	"fmt"
	"net"

	"go-gb28181/sip"
)

// ChannelTarget 通道级 SIP 目标（复用设备 Endpoint 的 Conn/Source，独立 To）。
type ChannelTarget struct {
	ChannelID string
	peer      *Endpoint
	to        *sip.Address
}

// NewChannelTarget 基于设备绑定与平台域构造通道对话目标。
func NewChannelTarget(peer *Endpoint, sipDomain, channelID string) (*ChannelTarget, error) {
	if peer == nil {
		return nil, fmt.Errorf("peer is nil")
	}
	uri, err := sip.ParseURI(fmt.Sprintf("sip:%s@%s", channelID, sipDomain))
	if err != nil {
		return nil, err
	}
	to := &sip.Address{URI: uri, Params: sip.NewParams()}
	return &ChannelTarget{ChannelID: channelID, peer: peer, to: to}, nil
}

// Conn implements DialogueTarget.
func (c *ChannelTarget) Conn() sip.Connection { return c.peer.Conn() }

// Source implements DialogueTarget.
func (c *ChannelTarget) Source() net.Addr { return c.peer.Source() }

// To implements DialogueTarget.
func (c *ChannelTarget) To() *sip.Address { return c.to }

var _ DialogueTarget = (*ChannelTarget)(nil)
