/*
package gb28181 提供基于 SIP 的 GB/T 28181 信令侧能力，是 go-gb28181 对外推荐集成的稳定 API。

业务侧实现 [MessageHandler]（可嵌入 [NopMessageHandler]），[RegisterHandlers] 会按 CmdType 解析 XML
为结构体再回调各 Handle。收与发统一使用 [NewService]（嵌入 [Server]）与 [DeviceEndpoint]。

业务侧优先使用本包 API；底层 SIP 实现在子包 [go-gb28181/sip]，本包为类型别名与薄封装。
*/
package gb28181

import (
	"io"
	"log/slog"

	"go-gb28181/sip"
)

// Logger 信令栈日志接口，语义与标准库 log/slog 的键值参数一致。
type Logger = sip.Logger

// SetLogger 注入日志实现；nil 表示静默。
func SetLogger(l Logger) { sip.SetLogger(l) }

// NewSLogLogger 使用指定 *slog.Logger 作为后端。
func NewSLogLogger(l *slog.Logger) Logger { return sip.NewSLogLogger(l) }

// NewTextSLogLogger 文本格式输出到 w（nil 为 stderr）。
func NewTextSLogLogger(w io.Writer) Logger { return sip.NewTextSLogLogger(w) }

// 以下为 SIP / 国标信令常用类型的别名，便于在业务代码中以 sdk 为前缀使用。
type (
	Address        = sip.Address
	URI            = sip.URI
	Server         = sip.Server
	Context        = sip.Context
	String         = sip.String
	ViaHop         = sip.ViaHop
	Params         = sip.Params
	ContentType    = sip.ContentType
	RouteGroup     = sip.RouteGroup
	HeadersBuilder = sip.HeadersBuilder
	Header         = sip.Header
	MessageID      = sip.MessageID
	Request        = sip.Request
	Response       = sip.Response
	Transaction    = sip.Transaction
)

// 常用 SIP / MANSCDP 常量。
var (
	ContentTypeXML    = sip.ContentTypeXML
	MethodMessage     = sip.MethodMessage
	DefaultSipVersion = sip.DefaultSipVersion
)

// ParseSipURI 解析 sip: URI。
func ParseSipURI(uriStr string) (URI, error) { return sip.ParseSipURI(uriStr) }

// NewParams 构造 SIP 参数表。
func NewParams() Params { return sip.NewParams() }

// NewServer 创建 SIP 服务端（UDP/TCP 由 Listen* 启动）。
func NewServer(from *Address) *Server { return sip.NewServer(from) }

// XMLDecode 解码国标 XML（含 GB2312 等容错）。
func XMLDecode(data []byte, v any) error { return sip.XMLDecode(data, v) }

// GetCatalogXML 生成目录查询 Query 报文体。
func GetCatalogXML(id string) []byte { return sip.GetCatalogXML(id) }

// GetDeviceInfoXML 生成设备信息查询 Query 报文体。
func GetDeviceInfoXML(id string) []byte { return sip.GetDeviceInfoXML(id) }

// GetRecordInfoXML 生成录像文件目录查询 Query 报文体；sn 建议唯一，可用内部随机数。
func GetRecordInfoXML(channelID string, sn int, start, end int64) []byte {
	return sip.GetRecordInfoXML(channelID, sn, start, end)
}

// NewHeaderBuilder 构造 SIP 头域。
func NewHeaderBuilder() *HeadersBuilder { return sip.NewHeaderBuilder() }

// GenerateBranch 生成 Via branch。
func GenerateBranch() string { return sip.GenerateBranch() }

// NewRequest 构造 SIP 请求。
func NewRequest(messID MessageID, method string, recipient *URI, sipVersion string, hdrs []Header, body []byte) *Request {
	return sip.NewRequest(messID, method, recipient, sipVersion, hdrs, body)
}
