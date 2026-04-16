package gb28181

import "go-gb28181/sip"

// 以下符号供业务实现 REGISTER、Digest 等 SIP 细节时使用，无需直接依赖 [go-gb28181/sip]（可选）。

const MethodRegister = sip.MethodRegister

// GenericHeader 自定义 SIP 头（如 WWW-Authenticate、Date）。
type GenericHeader = sip.GenericHeader

// Authorization SIP Digest 解析与校验（与 [AuthFromValue]、[Authorization.CalcResponse] 配合）。
type Authorization = sip.Authorization

// AuthFromValue 解析 Authorization 头取值。
func AuthFromValue(value string) *Authorization { return sip.AuthFromValue(value) }

// NewResponseFromRequest 由请求构造 SIP 响应（复制 Via/From/To/CSeq 等）。
func NewResponseFromRequest(resID MessageID, req *Request, statusCode int, reason string, body []byte) *Response {
	return sip.NewResponseFromRequest(resID, req, statusCode, reason, body)
}

// RandString 随机字符串（用于 Digest nonce 等）。
func RandString(n int) string { return sip.RandString(n) }
