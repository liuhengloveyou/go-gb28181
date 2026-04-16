package gb28181

import (
	"encoding/xml"

	"go-gb28181/sip"
)

// 配置查询类型（ConfigDownload Query 的 ConfigType），与 message_config_download 一致。
const (
	ConfigTypeBasicParam = "BasicParam"
)

// BuildConfigDownloadQuery 构造 ConfigDownload 查询 XML（Query）。sn 为 0 时由内部随机生成。
func BuildConfigDownloadQuery(deviceID, configType string, sn int) ([]byte, error) {
	if sn == 0 {
		sn = sip.RandInt(100000, 999999)
	}
	v := struct {
		XMLName    xml.Name `xml:"Query"`
		CmdType    string   `xml:"CmdType"`
		SN         int      `xml:"SN"`
		DeviceID   string   `xml:"DeviceID"`
		ConfigType string   `xml:"ConfigType"`
	}{
		XMLName:    xml.Name{Local: "Query"},
		CmdType:    CmdConfigDownload,
		SN:         sn,
		DeviceID:   deviceID,
		ConfigType: configType,
	}
	return sip.XMLEncode(v)
}

// BuildDeviceConfigControl 构造 DeviceConfig 控制 XML（Control），可选携带抓拍等 SnapShotConfig。
func BuildDeviceConfigControl(deviceID string, sn int, snap *SnapShot) ([]byte, error) {
	if sn == 0 {
		sn = sip.RandInt(100000, 999999)
	}
	v := struct {
		XMLName        xml.Name  `xml:"Control"`
		CmdType        string    `xml:"CmdType"`
		SN             int       `xml:"SN"`
		DeviceID       string    `xml:"DeviceID"`
		SnapShotConfig *SnapShot `xml:"SnapShotConfig,omitempty"`
	}{
		XMLName:        xml.Name{Local: "Control"},
		CmdType:        CmdDeviceConfig,
		SN:             sn,
		DeviceID:       deviceID,
		SnapShotConfig: snap,
	}
	return sip.XMLEncode(v)
}
