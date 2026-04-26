package gb28181

import (
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"strings"

	"go-gb28181/sip"
)

type PTZAction string

const (
	PTZActionUp    PTZAction = "up"
	PTZActionDown  PTZAction = "down"
	PTZActionLeft  PTZAction = "left"
	PTZActionRight PTZAction = "right"
	PTZActionStop  PTZAction = "stop"
)

func (a PTZAction) Valid() bool {
	switch a {
	case PTZActionUp, PTZActionDown, PTZActionLeft, PTZActionRight, PTZActionStop:
		return true
	default:
		return false
	}
}

// BuildPTZCmdHex 生成 GB28181 PTZCmd（8 字节十六进制，大写）。
func BuildPTZCmdHex(action PTZAction, speed int) (string, error) {
	if !action.Valid() {
		return "", fmt.Errorf("invalid ptz action: %s", action)
	}
	if speed < 0 {
		speed = 0
	}
	if speed > 255 {
		speed = 255
	}

	cmd := [8]byte{0xA5, 0x0F, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00}
	switch action {
	case PTZActionUp:
		cmd[3] |= 0x08
	case PTZActionDown:
		cmd[3] |= 0x04
	case PTZActionLeft:
		cmd[3] |= 0x02
	case PTZActionRight:
		cmd[3] |= 0x01
	case PTZActionStop:
		// 保持方向位为 0，表示停止。
	}
	cmd[4] = byte(speed) // 水平速度
	cmd[5] = byte(speed) // 垂直速度
	cmd[6] = 0x00        // 缩放速度（当前未暴露 zoom 控制）

	sum := 0
	for i := 0; i < 7; i++ {
		sum += int(cmd[i])
	}
	cmd[7] = byte(sum % 256)
	return strings.ToUpper(hex.EncodeToString(cmd[:])), nil
}

// BuildDeviceControlPTZXML 构造 DeviceControl/PTZCmd 控制 XML（Control）。
func BuildDeviceControlPTZXML(deviceID string, action PTZAction, speed int) ([]byte, error) {
	cmdHex, err := BuildPTZCmdHex(action, speed)
	if err != nil {
		return nil, err
	}
	v := struct {
		XMLName  xml.Name `xml:"Control"`
		CmdType  string   `xml:"CmdType"`
		SN       int      `xml:"SN"`
		DeviceID string   `xml:"DeviceID"`
		PTZCmd   string   `xml:"PTZCmd"`
	}{
		XMLName:  xml.Name{Local: "Control"},
		CmdType:  "DeviceControl",
		SN:       sip.RandInt(100000, 999999),
		DeviceID: strings.TrimSpace(deviceID),
		PTZCmd:   cmdHex,
	}
	return sip.XMLEncode(v)
}
