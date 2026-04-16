package gb28181

// 以下为 GB/T 28181 MANSCDP XML 映射，字段尽量与仓库内 message_*.go 及国标附录 A 目录/报警等结构对齐。
// 根元素 Query/Response/Notify/Control 不同，此处不按根元素区分，仅靠子元素解码；未覆盖字段可本地扩展或读 [Context.Request].Body()。

// KeepaliveNotify 心跳（对应 message_keepalive.MessageNotify）。
type KeepaliveNotify struct {
	CmdType  string `xml:"CmdType"`
	SN       int    `xml:"SN"`
	DeviceID string `xml:"DeviceID"`
	Status   string `xml:"Status"`
	Info     string `xml:"Info"`
}

// CatalogDeviceItem 目录 DeviceList/Item（对应 device_model.Channels 的 XML 子集，并补国标常见扩展字段）。
type CatalogDeviceItem struct {
	// DeviceID 在 Item 中表示通道/前端编码（与 gb28181 Channels.ChannelID 一致，XML 标签为 DeviceID）。
	DeviceID            string `xml:"DeviceID"`
	Name                string `xml:"Name"`
	Manufacturer        string `xml:"Manufacturer"`
	Model               string `xml:"Model"`
	Owner               string `xml:"Owner"`
	CivilCode           string `xml:"CivilCode"`
	Address             string `xml:"Address"`
	Parental            int    `xml:"Parental"`
	ParentID            string `xml:"ParentID"`
	SafetyWay           int    `xml:"SafetyWay"`
	RegisterWay         int    `xml:"RegisterWay"`
	Secrecy             int    `xml:"Secrecy"`
	Status              string `xml:"Status"`
	Longitude           string `xml:"Longitude"`
	Latitude            string `xml:"Latitude"`
	IPAddress           string `xml:"IPAddress"`
	Port                int    `xml:"Port"`
	Password            string `xml:"Password"`
	PTZType             int    `xml:"PTZType"`
	PositionType        int    `xml:"PositionType"`
	RoomType            int    `xml:"RoomType"`
	UseType             int    `xml:"UseType"`
	SupplyLightType     int    `xml:"SupplyLightType"`
	DirectionType       int    `xml:"DirectionType"`
	Resolution          string `xml:"Resolution"`
	BusinessGroupID     string `xml:"BusinessGroupID"`
	DownloadSpeed       int    `xml:"DownloadSpeed"`
	SVCSpaceSupportMode int    `xml:"SVCSpaceSupportMode"`
	SVCTimeSupportMode  int    `xml:"SVCTimeSupportMode"`
}

// CatalogMessage 目录 Query/Response（对应 message_catalog.MessageDeviceListResponse 的扁平化）。
type CatalogMessage struct {
	CmdType  string              `xml:"CmdType"`
	SN       int                 `xml:"SN"`
	DeviceID string              `xml:"DeviceID"`
	SumNum   int                 `xml:"SumNum"`
	Items    []CatalogDeviceItem `xml:"DeviceList>Item"`
}

// DeviceInfoMessage 设备信息应答（对应 message_device_info.MessageDeviceInfoResponse）。
type DeviceInfoMessage struct {
	CmdType      string `xml:"CmdType"`
	SN           int    `xml:"SN"`
	DeviceID     string `xml:"DeviceID"`
	DeviceName   string `xml:"DeviceName"`
	Manufacturer string `xml:"Manufacturer"`
	Model        string `xml:"Model"`
	Firmware     string `xml:"Firmware"`
	Result       string `xml:"Result"`
}

// RecordFileItem 录像文件项（对应 message_record_info.RecordItem，仅保留 XML 相关字段）。
type RecordFileItem struct {
	DeviceID   string `xml:"DeviceID"`
	Name       string `xml:"Name"`
	FilePath   string `xml:"FilePath"`
	Address    string `xml:"Address"`
	StartTime  string `xml:"StartTime"`
	EndTime    string `xml:"EndTime"`
	Secrecy    int    `xml:"Secrecy"`
	Type       string `xml:"Type"`
	RecorderID string `xml:"RecorderID"`
}

// RecordInfoMessage 录像目录应答（对应 message_record_info.MessageRecordInfoResponse）。
type RecordInfoMessage struct {
	CmdType  string           `xml:"CmdType"`
	SN       int              `xml:"SN"`
	DeviceID string           `xml:"DeviceID"`
	SumNum   int              `xml:"SumNum"`
	Items    []RecordFileItem `xml:"RecordList>Item"`
}

// BasicParam 基本参数（对应 message_config_download.BasicParam）。
type BasicParam struct {
	Name              string `xml:"Name"`
	Expiration        int    `xml:"Expiration"`
	HeartBeatInterval int    `xml:"HeartBeatInterval"`
	HeartBeatCount    int    `xml:"HeartBeatCount"`
}

// SnapShot 抓拍配置（对应 message_config_download.SnapShot；出现在 SnapShot / SnapShotConfig 节点下）。
type SnapShot struct {
	SnapNum   int    `xml:"SnapNum"`
	Interval  int    `xml:"Interval"`
	UploadURL string `xml:"UploadURL"`
	SessionID string `xml:"SessionID"`
}

// ConfigDownloadMessage 配置下载 Query 或 Response（合并 message_config_download.ConfigDownloadRequest/Response 常用字段）。
type ConfigDownloadMessage struct {
	CmdType  string `xml:"CmdType"`
	SN       int    `xml:"SN"`
	DeviceID string `xml:"DeviceID"`
	Result   string `xml:"Result"`
	// ConfigType、SnapShotConfig 多见于平台下发的 Query；BasicParam、SnapShot 多见于设备 Response。
	ConfigType     string      `xml:"ConfigType"`
	SnapShotConfig *SnapShot   `xml:"SnapShotConfig"`
	BasicParam     *BasicParam `xml:"BasicParam"`
	SnapShot       *SnapShot   `xml:"SnapShot"`
}

// DeviceConfigMessage 设备配置 Control 或应答（合并 message_device_config.DeviceConfigRequest 与常见 Result 应答）。
type DeviceConfigMessage struct {
	CmdType  string `xml:"CmdType"`
	SN       int    `xml:"SN"`
	DeviceID string `xml:"DeviceID"`
	Result   string `xml:"Result"`
	// SnapShotConfig 对应平台下发的抓拍等配置（Control）。
	SnapShotConfig *SnapShot `xml:"SnapShotConfig"`
}

// AlarmNotify 报警通知（国标报警 Notify 常见字段扩展）。
type AlarmNotify struct {
	CmdType          string `xml:"CmdType"`
	SN               int    `xml:"SN"`
	DeviceID         string `xml:"DeviceID"`
	AlarmPriority    string `xml:"AlarmPriority"`
	AlarmMethod      string `xml:"AlarmMethod"`
	AlarmTime        string `xml:"AlarmTime"`
	AlarmDescription string `xml:"AlarmDescription"`
	Longitude        string `xml:"Longitude"`
	Latitude         string `xml:"Latitude"`
	AlarmType        string `xml:"AlarmType"`
	Info             string `xml:"Info"`
}

// DeviceStatusNotify 设备状态通知。
type DeviceStatusNotify struct {
	CmdType    string `xml:"CmdType"`
	SN         int    `xml:"SN"`
	DeviceID   string `xml:"DeviceID"`
	Result     string `xml:"Result"`
	Online     string `xml:"Online"`
	Status     string `xml:"Status"`
	DutyStatus string `xml:"DutyStatus"`
	DeviceTime string `xml:"DeviceTime"`
}

// MobilePositionNotify 移动位置通知。
type MobilePositionNotify struct {
	CmdType   string `xml:"CmdType"`
	SN        int    `xml:"SN"`
	DeviceID  string `xml:"DeviceID"`
	Longitude string `xml:"Longitude"`
	Latitude  string `xml:"Latitude"`
	Time      string `xml:"Time"`
	Speed     string `xml:"Speed"`
	Direction string `xml:"Direction"`
	Altitude  string `xml:"Altitude"`
}

// MediaStatusNotify 媒体流状态通知。
type MediaStatusNotify struct {
	CmdType    string `xml:"CmdType"`
	SN         int    `xml:"SN"`
	DeviceID   string `xml:"DeviceID"`
	NotifyType string `xml:"NotifyType"`
	Event      string `xml:"Event"`
}

// BroadcastMessage 语音广播相关。
type BroadcastMessage struct {
	CmdType     string `xml:"CmdType"`
	SN          int    `xml:"SN"`
	DeviceID    string `xml:"DeviceID"`
	Result      string `xml:"Result"`
	SourceID    string `xml:"SourceID"`
	TargetID    string `xml:"TargetID"`
	SourceCodec string `xml:"SourceCodec"`
	TargetCodec string `xml:"TargetCodec"`
}

// DeviceControlMessage 设备控制 Control/Response 常见字段（云台、录像、升级等子命令因厂商差异大，仅列常用标签）。
type DeviceControlMessage struct {
	CmdType       string `xml:"CmdType"`
	SN            int    `xml:"SN"`
	DeviceID      string `xml:"DeviceID"`
	Result        string `xml:"Result"`
	PTZCmd        string `xml:"PTZCmd"`
	RecordCmd     string `xml:"RecordCmd"`
	AlarmCmd      string `xml:"AlarmCmd"`
	TeleBoot      string `xml:"TeleBoot"`
	IFrameCmd     string `xml:"IFrameCmd"`
	DragZoomIn    string `xml:"DragZoomIn"`
	DragZoomOut   string `xml:"DragZoomOut"`
	HomePosition  string `xml:"HomePosition"`
	DeviceUpgrade string `xml:"DeviceUpgrade"`
	SDCardFormat  string `xml:"SDCardFormat"`
}
