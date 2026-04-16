/*
Package sip 提供 GB/T 28181 场景下的 SIP 服务端实现。

包含 UDP/TCP 监听、SIP 报文解析与按方法/MESSAGE 内 CmdType 的路由、事务与响应等待、
请求/响应及常用头域构建、Digest 鉴权解析与校验辅助，以及 XML/JSON 编解码、简单 HTTP 工具。

上层 gb28181 包基于本包完成 REGISTER、MESSAGE（Application/MANSCDP+xml）、
INVITE/ACK/BYE（application/sdp）等信令流程。
*/
package sip
