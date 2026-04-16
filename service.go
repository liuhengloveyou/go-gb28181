package gb28181

import (
	"fmt"

	"go-gb28181/sip"
)

// Service 国标信令服务入口：嵌入 [Server]（可直接 Listen、Close、Message 等），并绑定平台 From，
// 在同一对象上提供 [Service.RegisterHandlers] 与主动下发（SendMessage / QueryCatalog 等）。
type Service struct {
	*Server
	platform *Address
}

// NewService 创建信令服务。platform 为平台侧 SIP From（与监听端口、域配置一致），用于收包路由与主动发包。
func NewService(platform *Address) *Service {
	if platform == nil {
		return nil
	}
	return &Service{
		Server:   NewServer(platform),
		platform: platform,
	}
}

// Platform 返回创建服务时使用的平台 From 地址。
func (s *Service) Platform() *Address {
	if s == nil {
		return nil
	}
	return s.platform
}

// RegisterHandlers 将 [MessageHandler] 注册到本服务的 SIP 栈（等价于包函数 [RegisterHandlers](s.Server, …)）。
func (s *Service) RegisterHandlers(h MessageHandler, opts ...RegisterOptions) {
	if s == nil || s.Server == nil {
		return
	}
	RegisterHandlers(s.Server, h, opts...)
}

// SendMessage 向设备发送 MANSCDP XML（SIP MESSAGE）。
func (s *Service) SendMessage(ep DeviceEndpoint, body []byte) (*Transaction, error) {
	if s == nil || s.Server == nil || s.platform == nil {
		return nil, fmt.Errorf("gb28181/sdk: nil service")
	}
	return sendMessage(s.Server, s.platform, ep, body)
}

// QueryCatalog 目录查询（Query CmdType=Catalog）。
func (s *Service) QueryCatalog(ep DeviceEndpoint, catalogDeviceID string) (*Transaction, error) {
	return s.SendMessage(ep, GetCatalogXML(catalogDeviceID))
}

// QueryDeviceInfo 设备信息查询（Query CmdType=DeviceInfo）。
func (s *Service) QueryDeviceInfo(ep DeviceEndpoint, targetDeviceID string) (*Transaction, error) {
	return s.SendMessage(ep, GetDeviceInfoXML(targetDeviceID))
}

// QueryRecordInfo 录像文件目录查询（Query CmdType=RecordInfo）。
func (s *Service) QueryRecordInfo(ep DeviceEndpoint, channelID string, start, end int64) (*Transaction, error) {
	sn := sip.RandInt(100000, 999999)
	return s.SendMessage(ep, sip.GetRecordInfoXML(channelID, sn, start, end))
}

// QueryConfigDownload 配置下载查询（Query CmdType=ConfigDownload）。configType 如 [ConfigTypeBasicParam]。
func (s *Service) QueryConfigDownload(ep DeviceEndpoint, deviceID, configType string, sn int) (*Transaction, error) {
	body, err := BuildConfigDownloadQuery(deviceID, configType, sn)
	if err != nil {
		return nil, err
	}
	return s.SendMessage(ep, body)
}

// SendDeviceConfig 设备配置控制（Control CmdType=DeviceConfig）。
func (s *Service) SendDeviceConfig(ep DeviceEndpoint, body []byte) (*Transaction, error) {
	return s.SendMessage(ep, body)
}

// SendDeviceConfigSnapShot 抓拍等：下发带 SnapShotConfig 的 DeviceConfig Control。
func (s *Service) SendDeviceConfigSnapShot(ep DeviceEndpoint, channelOrDeviceID string, sn int, snap *SnapShot) (*Transaction, error) {
	body, err := BuildDeviceConfigControl(channelOrDeviceID, sn, snap)
	if err != nil {
		return nil, err
	}
	return s.SendMessage(ep, body)
}

func sendMessage(srv *Server, platform *Address, ep DeviceEndpoint, body []byte) (*Transaction, error) {
	if srv == nil || platform == nil {
		return nil, fmt.Errorf("gb28181/sdk: nil server or platform")
	}
	if err := ep.validate(); err != nil {
		return nil, err
	}
	uri, err := ParseSipURI(fmt.Sprintf("sip:%s@%s", ep.DeviceID, ep.Domain))
	if err != nil {
		return nil, err
	}
	toAddr := &Address{
		URI:    &uri,
		Params: NewParams(),
	}
	ct := ContentTypeXML
	hb := NewHeaderBuilder().
		SetTo(toAddr).
		SetFrom(platform).
		SetContact(platform).
		SetContentType(&ct).
		SetMethod(MethodMessage).
		AddVia(&ViaHop{
			Params: NewParams().Add("branch", String{Str: GenerateBranch()}),
		})
	req := NewRequest("", MethodMessage, toAddr.URI, DefaultSipVersion, hb.Build(), body)
	req.SetConnection(ep.Conn)
	req.SetSource(ep.PeerAddr)
	req.SetDestination(ep.PeerAddr)
	return srv.Request(req)
}
