package agents

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yurifrl/cly/pkg/style"
)

type logField struct {
	Key   string
	Value any
}

func lf(key string, value any) logField {
	return logField{Key: key, Value: value}
}

func logInfo(msg string, fields ...logField) {
	logWith(os.Stdout, "INFO", msg, fields...)
}

func logSuccess(msg string, fields ...logField) {
	logWith(os.Stdout, "OK", msg, fields...)
}

func logWarn(msg string, fields ...logField) {
	logWith(os.Stdout, "WARN", msg, fields...)
}

func logError(msg string, fields ...logField) {
	logWith(os.Stderr, "ERROR", msg, fields...)
}

func logWith(out *os.File, level string, msg string, fields ...logField) {
	timestamp := style.SubtleStyle.Render(time.Now().Format("15:04:05"))

	levelText := level
	switch level {
	case "OK":
		levelText = style.GreenStyle.Bold(true).Render(level)
	case "WARN":
		levelText = style.YellowStyle.Render(level)
	case "ERROR":
		levelText = style.RedStyle.Bold(true).Render(level)
	default:
		levelText = style.BlueStyle.Bold(true).Render(level)
	}

	var b strings.Builder
	b.WriteString(timestamp)
	b.WriteString(" ")
	b.WriteString(levelText)
	b.WriteString(" ")
	b.WriteString(msg)

	for _, field := range fields {
		if field.Key == "" {
			continue
		}
		b.WriteString(" ")
		b.WriteString(style.SubtleStyle.Render(field.Key))
		b.WriteString("=")
		b.WriteString(formatFieldValue(field.Value))
	}

	fmt.Fprintln(out, b.String())
}

func formatFieldValue(v any) string {
	s := fmt.Sprint(v)
	s = strings.ReplaceAll(s, "\n", "\\n")
	if strings.ContainsAny(s, " \t") {
		return fmt.Sprintf("%q", s)
	}
	return s
}
