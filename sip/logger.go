/*
包级日志：标准 Logger 接口，默认 log/slog。应用侧请通过 go-gb28181/sdk 配置日志（如 sdk.NewStdLogger、sdk.NewSLogLogger）。
*/

package sip

import (
	"io"
	"log/slog"
	"os"
)

// Logger 为 SIP 栈使用的最小日志抽象。args 与 [log/slog.Logger.Info] 一致：
// 可为 slog.Attr，或 key、value 交替出现（偶数个）。
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// sipLog 包内默认实现，由 SetLogger 替换。
var sipLog Logger = NewSLogLogger(slog.Default())

// SetLogger 注入日志实现；nil 表示静默（等价于丢弃日志）。
func SetLogger(l Logger) {
	if l == nil {
		sipLog = noopLogger{}
		return
	}
	sipLog = l
}

// NewSLogLogger 使用指定 slog.Logger；nil 时返回静默实现。
func NewSLogLogger(l *slog.Logger) Logger {
	if l == nil {
		return noopLogger{}
	}
	return slogLogger{l: l}
}

// NewTextSLogLogger 便于示例或调试：文本输出到 w（nil 时使用 stderr）。
func NewTextSLogLogger(w io.Writer) Logger {
	if w == nil {
		w = os.Stderr
	}
	h := slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug})
	return slogLogger{l: slog.New(h)}
}

type slogLogger struct {
	l *slog.Logger
}

func (s slogLogger) Debug(msg string, args ...any) { s.l.Debug(msg, args...) }
func (s slogLogger) Info(msg string, args ...any)  { s.l.Info(msg, args...) }
func (s slogLogger) Error(msg string, args ...any) { s.l.Error(msg, args...) }

type noopLogger struct{}

func (noopLogger) Debug(string, ...any) {}
func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}

// LogDebug 写入当前注入的 Logger，供本模块内非 sip 子包使用，避免标准库 log。
func LogDebug(msg string, args ...any) { sipLog.Debug(msg, args...) }

// LogInfo 写入当前注入的 Logger。
func LogInfo(msg string, args ...any) { sipLog.Info(msg, args...) }

// LogError 写入当前注入的 Logger。
func LogError(msg string, args ...any) { sipLog.Error(msg, args...) }
