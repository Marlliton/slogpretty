package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

const (
	timeFormat = "2006-01-02 15:04:05.000"
)
const (
	reset = "\033[0m"

	black        = 30
	red          = 31
	green        = 32
	yellow       = 33
	blue         = 34
	magenta      = 35
	cyan         = 36
	lightGray    = 37
	darkGray     = 90
	lightRed     = 91
	lightGreen   = 92
	lightYellow  = 93
	lightBlue    = 94
	lightMagenta = 95
	lightCyan    = 96
	white        = 97
)

func colorize(colorCode int, v string) string {
	return fmt.Sprintf("\033[%sm%s%s", strconv.Itoa(colorCode), v, reset)
}

type Options struct {
	Level      slog.Level
	AddSource  bool
	Colorful   bool
	Multiline  bool
	TimeFormat string
}

type ColorTextHandler struct {
	opts Options
	out  io.Writer
	mu   *sync.Mutex
}

func (h *ColorTextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *ColorTextHandler) Handle(ctx context.Context, r slog.Record) error {
	buf := make([]byte, 0, 1024)

	// Timestamp
	if !r.Time.IsZero() {
		timeStr := r.Time.Format(h.opts.TimeFormat)
		if h.opts.Colorful {
			timeStr = colorize(lightGray, timeStr)
		}
		buf = fmt.Appendf(buf, "%s ", timeStr)
	}

	// Level
	levelStr := h.setColorLevel(r.Level)
	buf = fmt.Appendf(buf, "%-7s", levelStr)

	// Source location
	if h.opts.AddSource && r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		source := fmt.Sprintf("%s:%d", f.File, f.Line) // TODO: Retornar somente o nome do arquivo e a linha
		if h.opts.Colorful {
			source = colorize(darkGray, source)
		}
		buf = fmt.Appendf(buf, " %s", source)
	}

	// Message
	msg := r.Message
	msg = colorize(white, msg)
	buf = fmt.Appendf(buf, " %s", msg)

	// Attributes
	r.Attrs(func(a slog.Attr) bool {
		buf = h.appendAttr(buf, a)
		return true
	})

	buf = append(buf, '\n')
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(buf)
	return err
}

func (h *ColorTextHandler) appendAttr(buf []byte, a slog.Attr) []byte {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return buf
	}

	keyColor := lightMagenta
	valColor := lightBlue

	if !h.opts.Colorful {
		keyColor = 0
		valColor = 0
	}

	switch a.Value.Kind() {
	case slog.KindString:
		buf = fmt.Appendf(buf, " %s=%s",
			colorize(keyColor, a.Key),
			colorize(valColor, fmt.Sprintf("%q", a.Value.String())))
	case slog.KindTime:
		buf = fmt.Appendf(buf, " %s=%s",
			colorize(keyColor, a.Key),
			colorize(valColor, fmt.Sprintf("%q", a.Value.Time().Format(h.opts.TimeFormat))))
	case slog.KindInt64, slog.KindUint64, slog.KindFloat64, slog.KindBool:
		buf = fmt.Appendf(buf, " %s=%s",
			colorize(keyColor, a.Key),
			colorize(valColor, a.Value.String()))
	case slog.KindDuration:
		buf = fmt.Appendf(buf, " %s=%s",
			colorize(keyColor, a.Key),
			colorize(valColor, a.Value.String()))
	case slog.KindGroup:
		attrs := a.Value.Group()
		if len(attrs) == 0 {
			return buf
		}

		if a.Key != "" {
			buf = fmt.Appendf(buf, " %s:", colorize(keyColor, a.Key))
		}
		for _, ga := range attrs {
			buf = h.appendAttr(buf, ga)
		}
	default:
		buf = fmt.Appendf(buf, " %s=%s",
			colorize(keyColor, a.Key),
			colorize(valColor, a.Value.String()))
	}

	return buf
}

func (h *ColorTextHandler) setColorLevel(level slog.Level) string {
	switch level {
	case slog.LevelDebug:
		return colorize(lightMagenta, "DEBUG")
	case slog.LevelInfo:
		return colorize(lightCyan, "INFO")
	case slog.LevelWarn:
		return colorize(lightYellow, "WARN")
	case slog.LevelError:
		return colorize(lightRed, "ERROR")
	default:
		return level.String()
	}
}

func New(out io.Writer, opts *Options) *ColorTextHandler {
	if opts == nil {
		opts = &Options{
			Level:    slog.LevelInfo,
			Colorful: true,
		}
	}
	if opts.TimeFormat == "" {
		opts.TimeFormat = timeFormat
	}

	h := &ColorTextHandler{
		out:  out,
		mu:   &sync.Mutex{},
		opts: *opts,
	}
	return h
}

func (h *ColorTextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Implementação simplificada - retorna o mesmo handler
	return h
}

func (h *ColorTextHandler) WithGroup(name string) slog.Handler {
	// Implementação simplificada - retorna o mesmo handler
	return h
}

func main() {
	// Configuração do logger padrão
	h := New(os.Stdout, &Options{
		Level:     slog.LevelDebug,
		AddSource: true,
		Colorful:  true,
		Multiline: true,
	})
	slog.SetDefault(slog.New(h))

	// Exemplos de logs
	slog.Info("Iniciando aplicação", "version", "1.0.0", "env", "development")
	slog.Debug("Configuração carregada", "config", map[string]interface{}{
		"timeout":  "30s",
		"retries":  3,
		"features": []string{"auth", "storage"},
	})
	slog.Warn("Atenção: modo de desenvolvimento ativado")

	err := fmt.Errorf("erro de conexão")
	slog.Error("Falha ao conectar ao banco de dados",
		"error", err,
		"attempt", 3,
		"backoff", time.Second*2)

	slog.Info("Encerrando aplicação", "uptime", time.Minute*5)
}
