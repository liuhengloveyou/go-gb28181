# go-gb28181

面向 **GB/T 28181** 的 Go 模块：提供平台侧 SIP 信令、国标 XML（`CmdType`）消息处理、设备注册与目录/心跳等业务能力，并包含与媒体服务等组件对接的扩展点。


## 能力概览

- **API（推荐）**：模块根包 **`go-gb28181`**（`RegisterHandlers`、`NewService` 等）提供信令侧 API；SIP 实现在 **`go-gb28181/sip`**。uNVR 平台逻辑在 **`unvr/gb28181`**（通过 `RegisterHandlers` 接入；媒体/ZLM 遗留配置由 `ApplyServConfig` 注入）。
- **信令**：SIP over UDP/TCP，事务管理，`MESSAGE` / `NOTIFY` 及按报文体 XML 的 `CmdType` 分发。
- **国标报文**：常用 Query 模板（如目录、设备信息）、编解码与路由钩子。
- **平台业务（uNVR）**：`unvr/gb28181` 聚合注册、目录、点播、录像等与业务库/媒体服务的协作；`go.mod` 中 `replace go-gb28181 => ../go-gb28181`。
- **业务接入**：`sdk.NewService(platformFrom)` 创建统一入口（内嵌 `*sdk.Server`），同一对象上 `RegisterHandlers` 与 `QueryCatalog`/`SendMessage` 等完成收与发。实现 `sdk.MessageHandler`（可嵌入 `NopMessageHandler`）即可按 CmdType 回调。
- **示例**：`example` 为长期运行的 **HTTP + SIP 网关**：设备 **REGISTER**（可选 Digest）、**Keepalive**、**Catalog/DeviceInfo** 应答，以及注册后自动或经 HTTP 触发的目录/设备信息 **Query**。

## 快速开始

在模块根目录执行：

```bash
# 网关：SIP UDP :15060 + HTTP :8080（浏览器打开 http://127.0.0.1:8080/ ）
go run ./example -sip-udp :15060 -http :8080 \
  -platform 34020000002000000001 -domain 3402000000

# 可选：设备注册密码（Digest）
go run ./example ... -device-password your_secret

# 可选：不向本机网关发 REGISTER 时，仍可用 send 自测 MESSAGE
go run ./example send -listen :0 -target 127.0.0.1:15060 \
  -platform 34020000002000000001 -device 34020000001320000001 -domain 3402000000 -cmd catalog
```

说明：

- 设备需将平台 SIP 地址指向本机 **`-sip-udp`** 端口，**`-platform` / `-domain`** 与设备配置一致。
- HTTP：`GET /api/devices` 查看在线设备；`POST /api/devices/{id}/catalog` 等主动下发查询。
- `send` 子命令的 `-cmd` 支持 `catalog`、`deviceinfo`。

构建可执行文件时避免将输出命名为 `example`（与目录名冲突），例如：

```bash
go build -o gb28181-example ./example
```

## 作为依赖使用

在业务的 `go.mod` 中引用本模块，例如：

```bash
go get go-gb28181@latest
```

若模块尚未发布到公共代理，可使用 `replace` 指向本地克隆目录。导入路径为 `go-gb28181/...`，与仓库内 `package` 声明一致。

## 日志

通过 **`sdk`** 注入信令日志。SDK **不依赖**第三方日志库；推荐用标准库 **`log`**，也可用 **`log/slog`** 实现结构化输出。

```go
import (
    "log"
    "go-gb28181/sdk"
)

log.SetFlags(log.LstdFlags | log.Lmicroseconds)
sdk.SetLogger(sdk.NewStdLogger(log.Default()))
```

```go
import (
    "log/slog"
    "os"

    "go-gb28181/sdk"
)

l := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
sdk.SetLogger(sdk.NewSLogLogger(l))
```

静默日志：`sdk.SetLogger(nil)`。

自行接入 zap、zerolog 等时，实现 `sdk.Logger`（与 `sip.Logger` 相同）后调用 `sdk.SetLogger` 即可，无需在 go-gb28181 模块内增加依赖。

## 开发与测试

```bash
go test ./...
```

若本地尚未拉齐全部集成依赖，可先保证示例与当前可编译子树通过构建与测试。

## 版本与合规

国标实现需与目标设备、平台版本（如 GB28181-2016 / 2022）及行业管理要求一致；上线前请在真实环境中完成联调与验收。
