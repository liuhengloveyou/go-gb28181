// GB28181 示例：长期运行的 HTTP + SIP 网关，走通「注册 → 心跳 → 平台主动查询目录/设备信息 → 设备应答」闭环。
//
// 启动（SIP UDP 默认 15060，HTTP 默认 8080）：
//
//	go run . -sip-udp :15060 -http :8080 -platform 34020000002000000001 -domain 3402000000
//
// 设备侧将平台 SIP 服务器地址指向本机 UDP 端口，使用相同域与平台编码；可选设备密码：
//
//	go run . ... -device-password 12345678
//
// 可选：一次性向指定地址发 MESSAGE（联调）：
//
//	go run . send -listen :0 -target 127.0.0.1:15060 -platform 34020000002000000001 -device 34020000001320000001 -domain 3402000000 -cmd catalog
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"unicode"

	gb28181 "go-gb28181"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	gb28181.SetLogger(gb28181.NewStdLogger(log.Default()))

	if len(os.Args) >= 2 && os.Args[1] == "send" {
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
		runSend()
		return
	}

	if len(os.Args) >= 2 && (os.Args[1] == "-h" || os.Args[1] == "--help" || os.Args[1] == "help") {
		printUsage()
		return
	}

	runGateway()
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `用法:
  %s [网关参数]          # HTTP + SIP，长期运行（默认）
  %s send [参数]         # 单次发送 MESSAGE（联调）

网关参数:
  -sip-udp 地址         SIP UDP 监听，默认 :15060
  -http 地址            HTTP API，默认 :8080
  -platform 编码        平台 SIP 用户 ID（From），默认 34020000002000000001
  -domain 域            SIP 域（URI host），默认 3402000000
  -device-password 密码 设备注册 Digest，空表示不鉴权
  -no-auto-query        注册成功后不自动发 DeviceInfo/Catalog 查询

HTTP:
  GET  /                 简要说明
  GET  /api/health       健康检查
  GET  /api/devices      已注册设备列表（JSON）
  GET  /api/devices/{id} 单设备摘要
  POST /api/devices/{id}/catalog    主动目录查询
  POST /api/devices/{id}/deviceinfo 主动设备信息查询

`, os.Args[0], os.Args[0])
}

func logExitf(format string, args ...any) {
	log.Printf("[ERROR] "+format, args...)
	os.Exit(1)
}

func validateGBDeviceID(deviceID string) error {
	if len(deviceID) < 18 || len(deviceID) > 20 {
		return fmt.Errorf("device id length must be 18–20")
	}
	for _, ch := range deviceID {
		if !unicode.IsNumber(ch) {
			return fmt.Errorf("device id must be digits")
		}
	}
	return nil
}

// deviceRecord 内存态设备（示例用，非持久化）。
type deviceRecord struct {
	DeviceID       string    `json:"device_id"`
	Domain         string    `json:"domain"`
	RemoteAddr     string    `json:"remote_addr"`
	Online         bool      `json:"online"`
	RegisteredAt   time.Time `json:"registered_at,omitempty"`
	LastKeepalive  time.Time `json:"last_keepalive,omitempty"`
	LastCatalogSN  int       `json:"last_catalog_sn,omitempty"`
	LastCatalogSum int       `json:"last_catalog_sum,omitempty"`
	LastCatalogN   int       `json:"last_catalog_items,omitempty"`
	DeviceName     string    `json:"device_name,omitempty"`
	Manufacturer   string    `json:"manufacturer,omitempty"`
	Model          string    `json:"model,omitempty"`
	Firmware       string    `json:"firmware,omitempty"`
	InfoResult     string    `json:"device_info_result,omitempty"`
}

type gateway struct {
	gb28181.NopMessageHandler

	svc       *gb28181.Service
	domain    string
	password  string
	autoQuery bool

	mu      sync.RWMutex
	devices map[string]*deviceRecord
}

func newGateway(svc *gb28181.Service, domain, password string, autoQuery bool) *gateway {
	return &gateway{
		svc:       svc,
		domain:    domain,
		password:  password,
		autoQuery: autoQuery,
		devices:   make(map[string]*deviceRecord),
	}
}

func (g *gateway) sipRegister(ctx *gb28181.Context) {
	if err := validateGBDeviceID(ctx.DeviceID); err != nil {
		ctx.String(400, err.Error())
		return
	}

	if g.password != "" {
		hdrs := ctx.Request.GetHeaders("Authorization")
		if len(hdrs) == 0 {
			resp := gb28181.NewResponseFromRequest("", ctx.Request, http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized), nil)
			resp.AppendHeader(&gb28181.GenericHeader{
				HeaderName: "WWW-Authenticate",
				Contents:   fmt.Sprintf(`Digest realm="%s",qop="auth",nonce="%s"`, g.domain, gb28181.RandString(32)),
			})
			_ = ctx.Tx.Respond(resp)
			return
		}
		authenticateHeader := hdrs[0].(*gb28181.GenericHeader)
		auth := gb28181.AuthFromValue(authenticateHeader.Contents)
		auth.SetPassword(g.password)
		auth.SetUsername(ctx.DeviceID)
		auth.SetMethod(ctx.Request.Method())
		auth.SetURI(auth.Get("uri"))
		if auth.CalcResponse() != auth.Get("response") {
			ctx.String(http.StatusUnauthorized, "wrong password")
			return
		}
	}

	respondOK := func() {
		resp := gb28181.NewResponseFromRequest("", ctx.Request, http.StatusOK, "OK", nil)
		resp.AppendHeader(&gb28181.GenericHeader{
			HeaderName: "Date",
			Contents:   time.Now().Format("2006-01-02T15:04:05.000"),
		})
		_ = ctx.Tx.Respond(resp)
	}

	expireStr := ctx.GetHeader("Expires")
	if expireStr == "0" {
		log.Printf("[INFO] 设备注销 device=%s", ctx.DeviceID)
		g.mu.Lock()
		if rec, ok := g.devices[ctx.DeviceID]; ok {
			rec.Online = false
		}
		g.mu.Unlock()
		respondOK()
		return
	}

	g.mu.Lock()
	rec, ok := g.devices[ctx.DeviceID]
	if !ok {
		rec = &deviceRecord{DeviceID: ctx.DeviceID, Domain: g.domain}
		g.devices[ctx.DeviceID] = rec
	}
	rec.RemoteAddr = ctx.Source.String()
	rec.Online = true
	rec.RegisteredAt = time.Now()
	g.mu.Unlock()

	log.Printf("[INFO] 设备注册成功 device=%s from=%s expires=%s", ctx.DeviceID, rec.RemoteAddr, expireStr)
	respondOK()

	if g.autoQuery {
		go g.probeDevice(ctx.DeviceID)
	}
}

func (g *gateway) endpointFor(deviceID string) (gb28181.DeviceEndpoint, bool) {
	g.mu.RLock()
	rec, ok := g.devices[deviceID]
	g.mu.RUnlock()
	if !ok || !rec.Online || g.svc.UDPConn() == nil {
		return gb28181.DeviceEndpoint{}, false
	}
	raddr, err := net.ResolveUDPAddr("udp", rec.RemoteAddr)
	if err != nil {
		return gb28181.DeviceEndpoint{}, false
	}
	return gb28181.DeviceEndpoint{
		DeviceID: deviceID,
		Domain:   g.domain,
		PeerAddr: raddr,
		Conn:     g.svc.UDPConn(),
	}, true
}

func (g *gateway) probeDevice(deviceID string) {
	time.Sleep(300 * time.Millisecond)
	ep, ok := g.endpointFor(deviceID)
	if !ok {
		log.Printf("[WARN] 无法下发查询（设备未在线或无连接）device=%s", deviceID)
		return
	}
	if _, err := g.svc.QueryDeviceInfo(ep, deviceID); err != nil {
		log.Printf("[WARN] QueryDeviceInfo device=%s err=%v", deviceID, err)
	}
	if _, err := g.svc.QueryCatalog(ep, deviceID); err != nil {
		log.Printf("[WARN] QueryCatalog device=%s err=%v", deviceID, err)
	}
}

func (g *gateway) HandleKeepalive(ctx *gb28181.Context, msg *gb28181.KeepaliveNotify) error {
	g.mu.Lock()
	if rec, ok := g.devices[ctx.DeviceID]; ok {
		rec.LastKeepalive = time.Now()
		rec.Online = true
		rec.RemoteAddr = ctx.Source.String()
	}
	g.mu.Unlock()
	log.Printf("[INFO] Keepalive device=%s status=%s sn=%d", ctx.DeviceID, msg.Status, msg.SN)
	return nil
}

func (g *gateway) HandleCatalog(ctx *gb28181.Context, msg *gb28181.CatalogMessage) error {
	g.mu.Lock()
	if rec, ok := g.devices[ctx.DeviceID]; ok {
		rec.LastCatalogSN = msg.SN
		rec.LastCatalogSum = msg.SumNum
		rec.LastCatalogN = len(msg.Items)
	}
	g.mu.Unlock()
	log.Printf("[INFO] Catalog device=%s sum=%d items=%d", ctx.DeviceID, msg.SumNum, len(msg.Items))
	return nil
}

func (g *gateway) HandleDeviceInfo(ctx *gb28181.Context, msg *gb28181.DeviceInfoMessage) error {
	g.mu.Lock()
	if rec, ok := g.devices[ctx.DeviceID]; ok {
		rec.DeviceName = msg.DeviceName
		rec.Manufacturer = msg.Manufacturer
		rec.Model = msg.Model
		rec.Firmware = msg.Firmware
		rec.InfoResult = msg.Result
	}
	g.mu.Unlock()
	log.Printf("[INFO] DeviceInfo device=%s name=%s result=%s", ctx.DeviceID, msg.DeviceName, msg.Result)
	return nil
}

func (g *gateway) jsonDevices(w http.ResponseWriter) {
	g.mu.RLock()
	list := make([]*deviceRecord, 0, len(g.devices))
	for _, rec := range g.devices {
		list = append(list, rec)
	}
	g.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(list)
}

func runGateway() {
	fs := flag.NewFlagSet("gateway", flag.ExitOnError)
	sipUDP := fs.String("sip-udp", ":15060", "SIP UDP 监听地址")
	httpAddr := fs.String("http", ":8080", "HTTP 监听地址")
	platformID := fs.String("platform", "34020000002000000001", "平台 SIP 用户 ID")
	domain := fs.String("domain", "3402000000", "SIP 域")
	devPass := fs.String("device-password", "", "设备注册密码（Digest），空为不鉴权")
	noAuto := fs.Bool("no-auto-query", false, "注册成功后不自动查询 DeviceInfo/Catalog")
	_ = fs.Parse(os.Args[1:])

	uri, err := gb28181.ParseSipURI(fmt.Sprintf("sip:%s@%s", *platformID, *domain))
	if err != nil {
		logExitf("ParseSipURI: %v", err)
	}
	from := &gb28181.Address{
		DisplayName: gb28181.String{Str: "go-gb28181-example"},
		URI:         &uri,
		Params:      gb28181.NewParams(),
	}

	svc := gb28181.NewService(from)
	gw := newGateway(svc, *domain, *devPass, !*noAuto)

	svc.Register(gw.sipRegister)
	svc.RegisterHandlers(gw)

	go svc.ListenUDPServer(*sipUDP)
	for i := 0; i < 150 && svc.UDPConn() == nil; i++ {
		time.Sleep(20 * time.Millisecond)
	}
	if svc.UDPConn() == nil {
		logExitf("SIP UDP 监听未就绪")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		const page = `<!DOCTYPE html><html><head><meta charset="utf-8"><title>go-gb28181 example</title></head><body>
<h1>go-gb28181 示例网关</h1>
<p>SIP UDP 与 HTTP 已启动。设备注册后可在下列接口查看状态或主动查询。</p>
<ul>
<li><a href="/api/health">/api/health</a></li>
<li><a href="/api/devices">/api/devices</a></li>
</ul>
</body></html>`
		_, _ = w.Write([]byte(page))
	})
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("GET /api/devices", func(w http.ResponseWriter, r *http.Request) {
		gw.jsonDevices(w)
	})
	mux.HandleFunc("GET /api/devices/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		gw.mu.RLock()
		rec, ok := gw.devices[id]
		gw.mu.RUnlock()
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(rec)
	})
	mux.HandleFunc("POST /api/devices/{id}/catalog", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		ep, ok := gw.endpointFor(id)
		if !ok {
			http.Error(w, "device offline or unknown", http.StatusBadRequest)
			return
		}
		_, err := gw.svc.QueryCatalog(ep, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"ok":true,"cmd":"catalog","device_id":%q}`+"\n", id)
	})
	mux.HandleFunc("POST /api/devices/{id}/deviceinfo", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		ep, ok := gw.endpointFor(id)
		if !ok {
			http.Error(w, "device offline or unknown", http.StatusBadRequest)
			return
		}
		_, err := gw.svc.QueryDeviceInfo(ep, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"ok":true,"cmd":"deviceinfo","device_id":%q}`+"\n", id)
	})

	httpSrv := &http.Server{Addr: *httpAddr, Handler: mux}
	go func() {
		log.Printf("[INFO] HTTP 监听 %s", *httpAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[ERROR] HTTP: %v", err)
		}
	}()

	log.Printf("[INFO] SIP UDP 监听 %s platform=%s domain=%s digest=%v auto_query=%v",
		*sipUDP, *platformID, *domain, *devPass != "", !*noAuto)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
	svc.Close()
	log.Println("[INFO] 已退出")
}

func runSend() {
	fs := flag.NewFlagSet("send", flag.ExitOnError)
	listen := fs.String("listen", ":0", "本地 SIP UDP 绑定（:0 表示随机端口）")
	target := fs.String("target", "", "对端 UDP 地址，例如 192.168.1.10:5060")
	platformID := fs.String("platform", "34020000002000000001", "本平台编码（From）")
	deviceID := fs.String("device", "", "目标设备/通道编码（To / Request-URI 用户）")
	domain := fs.String("domain", "3402000000", "SIP 域")
	cmd := fs.String("cmd", "catalog", "catalog | deviceinfo")
	_ = fs.Parse(os.Args[1:])

	if *target == "" || *deviceID == "" {
		fs.Usage()
		os.Exit(2)
	}

	raddr, err := net.ResolveUDPAddr("udp", *target)
	if err != nil {
		logExitf("ResolveUDPAddr: %v", err)
	}

	uri, err := gb28181.ParseSipURI(fmt.Sprintf("sip:%s@%s", *platformID, *domain))
	if err != nil {
		logExitf("ParseSipURI(from): %v", err)
	}
	from := &gb28181.Address{
		DisplayName: gb28181.String{Str: "go-gb28181-example"},
		URI:         &uri,
		Params:      gb28181.NewParams(),
	}

	svc := gb28181.NewService(from)
	go svc.ListenUDPServer(*listen)
	for i := 0; i < 100 && svc.UDPConn() == nil; i++ {
		time.Sleep(20 * time.Millisecond)
	}
	if svc.UDPConn() == nil {
		logExitf("UDP 监听未就绪")
	}

	ep := gb28181.DeviceEndpoint{
		DeviceID: *deviceID,
		Domain:   *domain,
		PeerAddr: raddr,
		Conn:     svc.UDPConn(),
	}

	var tx *gb28181.Transaction
	switch *cmd {
	case "catalog":
		tx, err = svc.QueryCatalog(ep, *deviceID)
	case "deviceinfo":
		tx, err = svc.QueryDeviceInfo(ep, *deviceID)
	default:
		logExitf("未知 -cmd: %s", *cmd)
	}
	if err != nil {
		logExitf("发送 MESSAGE 失败: %v", err)
	}
	log.Printf("[INFO] 已发送 MESSAGE to=%s cmd=%s", raddr.String(), *cmd)

	resp, err := gb28181.SIPResponse(tx)
	if err != nil {
		logExitf("未收到 SIP 响应: %v", err)
	}
	log.Printf("[INFO] 收到响应 status=%d reason=%s", resp.StatusCode(), resp.Reason())
	svc.Close()
}
