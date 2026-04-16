package gb28181

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go-gb28181/sip"
)

// RegisterHooks 定义 SIP REGISTER 流程中的业务钩子（可选实现）。
type RegisterHooks interface {
	// ValidateRegister 在 REGISTER 进入鉴权前执行（可做设备 ID 校验、预加载上下文等）。
	ValidateRegister(ctx *Context) error
	// OnUnregister 在 Expires=0 注销时执行（业务状态落库）。
	OnUnregister(ctx *Context) error
	// OnRegister 在成功注册时执行（业务状态落库）。
	OnRegister(ctx *Context, expires int) error
	// AfterRegister 在成功应答后执行（可触发目录/设备信息拉取）。
	AfterRegister(ctx *Context) error
}

// AuthProvider 定义 SIP REGISTER 鉴权信息提供器（可选实现）。
type AuthProvider interface {
	// RegisterAuth 返回本次 REGISTER 的鉴权参数；password 为空表示跳过鉴权。
	RegisterAuth(ctx *Context) (username, password, realm string, err error)
}

// MessageHandler 国标 MESSAGE/NOTIFY 业务入口：SDK 已按 CmdType 解析 XML，将具体结构指针传入各 Handle。
// 不需要实现的命令可嵌入 [NopMessageHandler]，再只重写关心的方法。
type MessageHandler interface {
	// HandleKeepalive 处理设备保活通知（Keepalive）。
	HandleKeepalive(ctx *Context, msg *KeepaliveNotify) error
	// HandleCatalog 处理设备目录响应/通知（Catalog）。
	HandleCatalog(ctx *Context, msg *CatalogMessage) error
	// HandleDeviceInfo 处理设备信息响应（DeviceInfo）。
	HandleDeviceInfo(ctx *Context, msg *DeviceInfoMessage) error
	// HandleRecordInfo 处理录像信息响应（RecordInfo）。
	HandleRecordInfo(ctx *Context, msg *RecordInfoMessage) error
	// HandleConfigDownload 处理配置下载响应（ConfigDownload）。
	HandleConfigDownload(ctx *Context, msg *ConfigDownloadMessage) error
	// HandleDeviceConfig 处理设备配置响应（DeviceConfig）。
	HandleDeviceConfig(ctx *Context, msg *DeviceConfigMessage) error
	// HandleAlarm 处理报警通知（Alarm）。
	HandleAlarm(ctx *Context, msg *AlarmNotify) error
	// HandleDeviceStatus 处理设备状态通知（DeviceStatus）。
	HandleDeviceStatus(ctx *Context, msg *DeviceStatusNotify) error
	// HandleMobilePosition 处理移动位置通知（MobilePosition）。
	HandleMobilePosition(ctx *Context, msg *MobilePositionNotify) error
	// HandleMediaStatus 处理媒体状态通知（MediaStatus）。
	HandleMediaStatus(ctx *Context, msg *MediaStatusNotify) error
	// HandleBroadcast 处理语音广播消息（Broadcast）。
	HandleBroadcast(ctx *Context, msg *BroadcastMessage) error
	// HandleDeviceControl 处理设备控制消息（DeviceControl）。
	HandleDeviceControl(ctx *Context, msg *DeviceControlMessage) error
}

// NopMessageHandler 空实现，供嵌入后按需覆盖个别 Handle。
// 各方法会打一行标准日志，便于确认请求落到了默认分支。
type NopMessageHandler struct{}

func nopHandlerLog(cmd string, ctx *Context) {
	sip.LogDebug("go-gb28181 NopMessageHandler", "cmd", cmd, "device_id", ctx.DeviceID)
}

func (NopMessageHandler) HandleKeepalive(ctx *Context, _ *KeepaliveNotify) error {
	nopHandlerLog(CmdKeepalive, ctx)
	return nil
}
func (NopMessageHandler) HandleCatalog(ctx *Context, _ *CatalogMessage) error {
	nopHandlerLog(CmdCatalog, ctx)
	return nil
}
func (NopMessageHandler) HandleDeviceInfo(ctx *Context, _ *DeviceInfoMessage) error {
	nopHandlerLog(CmdDeviceInfo, ctx)
	return nil
}
func (NopMessageHandler) HandleRecordInfo(ctx *Context, _ *RecordInfoMessage) error {
	nopHandlerLog(CmdRecordInfo, ctx)
	return nil
}
func (NopMessageHandler) HandleConfigDownload(ctx *Context, _ *ConfigDownloadMessage) error {
	nopHandlerLog(CmdConfigDownload, ctx)
	return nil
}
func (NopMessageHandler) HandleDeviceConfig(ctx *Context, _ *DeviceConfigMessage) error {
	nopHandlerLog(CmdDeviceConfig, ctx)
	return nil
}
func (NopMessageHandler) HandleAlarm(ctx *Context, _ *AlarmNotify) error {
	nopHandlerLog(CmdAlarm, ctx)
	return nil
}
func (NopMessageHandler) HandleDeviceStatus(ctx *Context, _ *DeviceStatusNotify) error {
	nopHandlerLog(CmdDeviceStatus, ctx)
	return nil
}
func (NopMessageHandler) HandleMobilePosition(ctx *Context, _ *MobilePositionNotify) error {
	nopHandlerLog(CmdMobilePosition, ctx)
	return nil
}
func (NopMessageHandler) HandleMediaStatus(ctx *Context, _ *MediaStatusNotify) error {
	nopHandlerLog(CmdMediaStatus, ctx)
	return nil
}
func (NopMessageHandler) HandleBroadcast(ctx *Context, _ *BroadcastMessage) error {
	nopHandlerLog(CmdBroadcast, ctx)
	return nil
}
func (NopMessageHandler) HandleDeviceControl(ctx *Context, _ *DeviceControlMessage) error {
	nopHandlerLog(CmdDeviceControl, ctx)
	return nil
}

// RegisterOptions 控制 [RegisterHandlers] 的挂载方式与默认 SIP 响应。
type RegisterOptions struct {
	// SkipNotify 为 true 时仅注册 SIP MESSAGE，不向 NOTIFY 挂载（默认 false，即 MESSAGE 与 NOTIFY 各挂一份）。
	SkipNotify bool
	// SuccessText 处理成功时 SIP 200 应答正文，默认 "OK"。
	SuccessText string
	// OnError 非 nil 时替代默认错误响应（XML 解析失败默认 400，业务返回 error 默认 500）。
	OnError func(ctx *Context, err error)
	// RegisterHooks 为 REGISTER 流程提供业务钩子（可选）。
	RegisterHooks RegisterHooks
	// AuthProvider 为 REGISTER 流程提供鉴权信息（可选）。
	AuthProvider AuthProvider
}

// RegisterHandlers 将 MessageHandler 按 CmdType 挂到 MESSAGE，默认同时挂到 NOTIFY；进入业务前完成 XML 解析。
func RegisterHandlers(s *Server, h MessageHandler, opts ...RegisterOptions) {
	if s == nil || h == nil {
		return
	}
	opt := RegisterOptions{SuccessText: "OK"}
	if len(opts) > 0 {
		opt = opts[0]
		if opt.SuccessText == "" {
			opt.SuccessText = "OK"
		}
	}

	registerHandler(s, h, opt)

	msg := s.Message()
	registerOne := func(cmd string, fn sip.HandlerFunc) {
		msg.Handle(cmd, fn)
		if !opt.SkipNotify {
			s.Notify().Handle(cmd, fn)
		}
	}

	registerMsg(registerOne, CmdKeepalive, h, func(x MessageHandler, ctx *Context, m *KeepaliveNotify) error {
		return x.HandleKeepalive(ctx, m)
	}, opt)
	registerMsg(registerOne, CmdCatalog, h, func(x MessageHandler, ctx *Context, m *CatalogMessage) error {
		return x.HandleCatalog(ctx, m)
	}, opt)
	registerMsg(registerOne, CmdDeviceInfo, h, func(x MessageHandler, ctx *Context, m *DeviceInfoMessage) error {
		return x.HandleDeviceInfo(ctx, m)
	}, opt)
	registerMsg(registerOne, CmdRecordInfo, h, func(x MessageHandler, ctx *Context, m *RecordInfoMessage) error {
		return x.HandleRecordInfo(ctx, m)
	}, opt)
	registerMsg(registerOne, CmdConfigDownload, h, func(x MessageHandler, ctx *Context, m *ConfigDownloadMessage) error {
		return x.HandleConfigDownload(ctx, m)
	}, opt)
	registerMsg(registerOne, CmdDeviceConfig, h, func(x MessageHandler, ctx *Context, m *DeviceConfigMessage) error {
		return x.HandleDeviceConfig(ctx, m)
	}, opt)
	registerMsg(registerOne, CmdAlarm, h, func(x MessageHandler, ctx *Context, m *AlarmNotify) error {
		return x.HandleAlarm(ctx, m)
	}, opt)
	registerMsg(registerOne, CmdDeviceStatus, h, func(x MessageHandler, ctx *Context, m *DeviceStatusNotify) error {
		return x.HandleDeviceStatus(ctx, m)
	}, opt)
	registerMsg(registerOne, CmdMobilePosition, h, func(x MessageHandler, ctx *Context, m *MobilePositionNotify) error {
		return x.HandleMobilePosition(ctx, m)
	}, opt)
	registerMsg(registerOne, CmdMediaStatus, h, func(x MessageHandler, ctx *Context, m *MediaStatusNotify) error {
		return x.HandleMediaStatus(ctx, m)
	}, opt)
	registerMsg(registerOne, CmdBroadcast, h, func(x MessageHandler, ctx *Context, m *BroadcastMessage) error {
		return x.HandleBroadcast(ctx, m)
	}, opt)
	registerMsg(registerOne, CmdDeviceControl, h, func(x MessageHandler, ctx *Context, m *DeviceControlMessage) error {
		return x.HandleDeviceControl(ctx, m)
	}, opt)
}

func registerMsg[T any](
	registerOne func(string, sip.HandlerFunc),
	cmd string,
	h MessageHandler,
	call func(MessageHandler, *Context, *T) error,
	opt RegisterOptions,
) {
	registerOne(cmd, func(c *sip.Context) {
		var v T
		if err := XMLDecode(c.Request.Body(), &v); err != nil {
			if opt.OnError != nil {
				opt.OnError(c, err)
			} else {
				c.String(400, err.Error())
			}
			return
		}
		if err := call(h, c, &v); err != nil {
			if opt.OnError != nil {
				opt.OnError(c, err)
			} else {
				c.String(500, err.Error())
			}
			return
		}
		c.String(200, opt.SuccessText)
	})
}

func registerHandler(s *Server, h MessageHandler, opt RegisterOptions) {
	hooks := opt.RegisterHooks
	authp := opt.AuthProvider
	s.Register(func(ctx *sip.Context) {
		if hooks != nil {
			if err := hooks.ValidateRegister(ctx); err != nil {
				ctx.String(http.StatusBadRequest, err.Error())
				return
			}
		}

		if authp != nil {
			username, password, realm, err := authp.RegisterAuth(ctx)
			if err != nil {
				if opt.OnError != nil {
					opt.OnError(ctx, err)
				} else {
					ctx.String(http.StatusInternalServerError, err.Error())
				}
				return
			}
			if password != "" {
				if username == "" {
					username = ctx.DeviceID
				}
				if realm == "" {
					realm = "gb28181"
				}
				if err := checkDigestAuth(ctx, username, password, realm); err != nil {
					return
				}
			}
		}

		expire := ctx.GetHeader("Expires")
		if expire == "0" {
			if hooks != nil {
				if err := hooks.OnUnregister(ctx); err != nil {
					if opt.OnError != nil {
						opt.OnError(ctx, err)
					} else {
						ctx.String(http.StatusInternalServerError, err.Error())
					}
					return
				}
			}
			respondRegisterOK(ctx)
			return
		}

		expires, _ := strconv.Atoi(expire)
		if hooks != nil {
			if err := hooks.OnRegister(ctx, expires); err != nil {
				if opt.OnError != nil {
					opt.OnError(ctx, err)
				} else {
					ctx.String(http.StatusInternalServerError, err.Error())
				}
				return
			}
		}
		respondRegisterOK(ctx)
		if hooks != nil {
			_ = hooks.AfterRegister(ctx)
		}
	})
}

func respondRegisterOK(ctx *sip.Context) {
	resp := sip.NewResponseFromRequest("", ctx.Request, http.StatusOK, "OK", nil)
	resp.AppendHeader(&sip.GenericHeader{
		HeaderName: "Date",
		Contents:   time.Now().Format("2006-01-02T15:04:05.000"),
	})
	_ = ctx.Tx.Respond(resp)
}

func checkDigestAuth(ctx *sip.Context, username, password, realm string) error {
	hdrs := ctx.Request.GetHeaders("Authorization")
	if len(hdrs) == 0 {
		resp := sip.NewResponseFromRequest("", ctx.Request, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized), nil)
		resp.AppendHeader(&sip.GenericHeader{
			HeaderName: "WWW-Authenticate",
			Contents:   fmt.Sprintf(`Digest realm="%s",qop="auth",nonce="%s"`, realm, sip.RandString(32)),
		})
		_ = ctx.Tx.Respond(resp)
		return fmt.Errorf("missing authorization")
	}
	authenticateHeader := hdrs[0].(*sip.GenericHeader)
	auth := sip.AuthFromValue(authenticateHeader.Contents)
	auth.SetPassword(password)
	auth.SetUsername(username)
	auth.SetMethod(ctx.Request.Method())
	auth.SetURI(auth.Get("uri"))
	if auth.CalcResponse() != auth.Get("response") {
		ctx.String(http.StatusUnauthorized, "wrong password")
		return fmt.Errorf("wrong password")
	}
	return nil
}
