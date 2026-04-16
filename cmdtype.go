package gb28181

// 国标 GB/T 28181 平台与设备间常用 MANSCDP XML 命令字（<CmdType> 取值）。
// 与设备/平台侧 Query、Response、Notify 根元素配合使用；未列出的命令仍可用 [RouteGroup.Handle] 自行注册。
const (
	CmdKeepalive      = "Keepalive"
	CmdCatalog        = "Catalog"
	CmdDeviceInfo     = "DeviceInfo"
	CmdRecordInfo     = "RecordInfo"
	CmdConfigDownload = "ConfigDownload"
	CmdDeviceConfig   = "DeviceConfig"
	CmdAlarm          = "Alarm"
	CmdDeviceStatus   = "DeviceStatus"
	CmdMobilePosition = "MobilePosition"
	CmdMediaStatus    = "MediaStatus"
	CmdBroadcast      = "Broadcast"
	CmdDeviceControl  = "DeviceControl"
)
