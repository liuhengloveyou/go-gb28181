package gb28181

import (
	"fmt"
	"log"
	"strings"
)

// stdLog 将标准库 log 适配为 Logger；args 按「键、值」交替拼接为一行（与 slog 风格兼容）。
type stdLog struct {
	l *log.Logger
}

// NewStdLogger 使用给定 *log.Logger 输出信令日志；nil 时使用 [log.Default]。
func NewStdLogger(l *log.Logger) Logger {
	if l == nil {
		l = log.Default()
	}
	return stdLog{l: l}
}

func formatLogArgs(args []any) string {
	if len(args) == 0 {
		return ""
	}
	var b strings.Builder
	for i := 0; i < len(args); {
		if i > 0 {
			b.WriteByte(' ')
		}
		if i+1 < len(args) {
			fmt.Fprintf(&b, "%v=%v", args[i], args[i+1])
			i += 2
			continue
		}
		fmt.Fprintf(&b, "%v", args[i])
		i++
	}
	return b.String()
}

func (s stdLog) Debug(msg string, args ...any) {
	s.l.Printf("[DEBUG] %s %s", msg, formatLogArgs(args))
}

func (s stdLog) Info(msg string, args ...any) {
	s.l.Printf("[INFO] %s %s", msg, formatLogArgs(args))
}

func (s stdLog) Error(msg string, args ...any) {
	s.l.Printf("[ERROR] %s %s", msg, formatLogArgs(args))
}
