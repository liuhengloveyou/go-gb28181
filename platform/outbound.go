package platform

import (
	"go-gb28181/sip"
)

// RequestOption 可选修改即将发出的 sip.Request。
type RequestOption func(*sip.Request)

// OutboundRequest 平台主动发请求：统一 From/Contact/Via，并交给 sip.Server 建事务发送。
func OutboundRequest(svr *sip.Server, from *sip.Address, t DialogueTarget, method string, contentType *sip.ContentType, body []byte, opts ...RequestOption) (*sip.Transaction, error) {
	to := t.To()
	conn := t.Conn()
	source := t.Source()

	hb := sip.NewHeaderBuilder().
		SetTo(to).
		SetFrom(from).
		SetContentType(contentType).
		SetMethod(method).
		SetContact(from).
		AddVia(&sip.ViaHop{
			Params: sip.NewParams().Add("branch", sip.String{Str: sip.GenerateBranch()}),
		})

	req := sip.NewRequest("", method, to.URI, sip.DefaultSipVersion, hb.Build(), body)
	req.SetConnection(conn)
	req.SetSource(source)
	req.SetDestination(source)

	for _, opt := range opts {
		opt(req)
	}

	return svr.Request(req)
}
