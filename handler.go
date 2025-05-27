package slogpretty

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// ==================== Types ====================
type groupOrAttrs struct {
	group string
	attrs []slog.Attr
}

type SlogPretty struct {
	opts Options
	goas []groupOrAttrs
	out  io.Writer
	mu   *sync.Mutex
}

// ==================== Initialization ====================
func New(out io.Writer, opts *Options) *SlogPretty {
	if opts == nil {
		opts = DefaultOptions()
	}
	if opts.TimeFormat == "" {
		opts.TimeFormat = DefaultTimeFormat
	}

	h := &SlogPretty{
		out:  out,
		mu:   &sync.Mutex{},
		opts: *opts,
	}
	return h
}

// ==================== Handler Interface Methods ====================
func (h *SlogPretty) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *SlogPretty) Handle(ctx context.Context, r slog.Record) error {
	buf := make([]byte, 0, 1024)

	buf = h.appendHeader(buf, r)
	h.removeEmptyGroup(r)
	if h.opts.Multiline {
		buf = h.appendMultilineGroupOrAttrs(buf, 1)
		buf = h.appendMultilineAttrs(buf, r)
	} else {
		buf = h.appendInLineAttrs(buf, r)
	}

	buf = append(buf, '\n')
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(buf)
	return err
}

func (h *SlogPretty) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	return h.withGroupOrAttrs(groupOrAttrs{attrs: attrs})
}

func (h *SlogPretty) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttrs{group: name})
}

// ==================== Helper Methods ====================
func (h *SlogPretty) appendHeader(buf []byte, r slog.Record) []byte {
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

	// Message
	msg := r.Message
	msg = colorize(white, msg)
	buf = fmt.Appendf(buf, " %s", msg)

	// Source location
	if h.opts.AddSource && r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		file := filepath.Base(f.File)
		source := fmt.Sprintf("source: %s:%d", file, f.Line)
		if h.opts.Colorful {
			source = colorize(darkGray, source)
		}
		buf = fmt.Appendf(buf, " %s", source)
	}
	return buf
}

func (h *SlogPretty) withGroupOrAttrs(goa groupOrAttrs) *SlogPretty {
	h2 := *h
	h2.goas = make([]groupOrAttrs, len(h.goas)+1)
	copy(h2.goas, h.goas)
	h2.goas[len(h2.goas)-1] = goa

	return &h2
}

func (h *SlogPretty) removeEmptyGroup(r slog.Record) {
	if r.NumAttrs() == 0 {
		for len(h.goas) > 0 && h.goas[len(h.goas)-1].group != "" {
			h.goas = h.goas[:len(h.goas)-1]
		}
	}
}

func (h *SlogPretty) appendMultilineGroupOrAttrs(buf []byte, level int) []byte {
	for _, goa := range h.goas {
		if goa.group != "" {
			buf = fmt.Appendf(buf, "%s%s:", strings.Repeat("  ", level), colorize(magenta, goa.group))
			level++
		} else {
			if len(goa.attrs) > 0 {
				buf = append(buf, '\n')
			}
			for _, a := range goa.attrs {
				buf = h.appendAttr(buf, a, true, level)
			}
		}
	}

	return buf
}

func (h *SlogPretty) appendMultilineAttrs(buf []byte, r slog.Record) []byte {
	attrCount := 0
	r.Attrs(func(a slog.Attr) bool {
		attrCount++
		return true
	})

	if attrCount == 0 {
		return buf
	}

	buf = append(buf, '\n')

	r.Attrs(func(a slog.Attr) bool {
		buf = h.appendAttr(buf, a, true, 1)
		return true
	})

	return buf
}

func (h *SlogPretty) appendInLineAttrs(buf []byte, r slog.Record) []byte {
	for _, goa := range h.goas {
		if goa.group != "" {
			if h.opts.Colorful {
				buf = fmt.Appendf(buf, " %s:", colorize(cyan, goa.group))
			} else {
				buf = fmt.Appendf(buf, " %s:", goa.group)
			}
		}
		for _, attr := range goa.attrs {
			buf = h.appendAttr(buf, attr, false, 0)
		}
	}
	r.Attrs(func(a slog.Attr) bool {
		buf = h.appendAttr(buf, a, false, 0)
		return true
	})

	return buf
}

func (h *SlogPretty) appendAttr(buf []byte, a slog.Attr, multiline bool, level int) []byte {
	// Identation
	indent := strings.Repeat(" ", 2*level)

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
		val := a.Value.String()
		if multiline {
			buf = fmt.Appendf(buf, "%s%s: %s\n",
				indent,
				colorize(keyColor, a.Key),
				colorize(valColor, val))
		} else {
			buf = fmt.Appendf(buf, " %s=%s",
				colorize(keyColor, a.Key),
				colorize(valColor, fmt.Sprintf("%q", val)))
		}
	case slog.KindTime:
		val := a.Value.Time().Format(h.opts.TimeFormat)
		if multiline {
			buf = fmt.Appendf(buf, "%s%s: %s\n",
				indent,
				colorize(keyColor, a.Key),
				colorize(valColor, val))
		} else {
			buf = fmt.Appendf(buf, " %s=%s",
				colorize(keyColor, a.Key),
				colorize(valColor, fmt.Sprintf("%q", val)))
		}
	case slog.KindInt64, slog.KindUint64, slog.KindFloat64, slog.KindBool:
		val := a.Value.String()
		if multiline {
			buf = fmt.Appendf(buf, "%s%s: %s\n",
				indent,
				colorize(keyColor, a.Key),
				colorize(valColor, val))
		} else {
			buf = fmt.Appendf(buf, " %s=%s",
				colorize(keyColor, a.Key),
				colorize(valColor, val))
		}
	case slog.KindDuration:
		val := a.Value.String()
		if multiline {
			buf = fmt.Appendf(buf, "%s%s: %s\n",
				indent,
				colorize(keyColor, a.Key),
				colorize(valColor, val))
		} else {
			buf = fmt.Appendf(buf, " %s=%s",
				colorize(keyColor, a.Key),
				colorize(valColor, val))
		}
	case slog.KindGroup:
		attrs := a.Value.Group()
		if len(attrs) == 0 {
			return buf
		}

		if multiline {
			buf = fmt.Appendf(buf, "%s%s:\n", indent, colorize(keyColor, a.Key))
			for _, ga := range attrs {
				buf = h.appendAttr(buf, ga, multiline, level+1)
			}
		} else {
			buf = fmt.Appendf(buf, " %s:", colorize(keyColor, a.Key))
			for _, ga := range attrs {
				buf = h.appendAttr(buf, ga, multiline, 2)
			}
		}
	default:
		if multiline {
			buf = fmt.Appendf(buf, "%s%s: %s\n",
				indent,
				colorize(keyColor, a.Key),
				colorize(valColor, a.Value.String()))
		} else {
			buf = fmt.Appendf(buf, " %s=%s",
				colorize(keyColor, a.Key),
				colorize(valColor, a.Value.String()))
		}
	}

	return buf
}

func (h *SlogPretty) setColorLevel(level slog.Level) string {
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
