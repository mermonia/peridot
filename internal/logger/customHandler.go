package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"sync"
	"time"
)

// Example CustomHandler output:
// TIME [DEBUG] Failed to find file. [ file_name="foo.txt" error_code="2" ]
type CustomHandler struct {
	opts CustomOptions
	goas []groupOrAttributes
	mu   *sync.Mutex
	out  io.Writer
}

type CustomOptions struct {
	// Minimum level for slog.Records to be actually handled. If not specified, it will
	// default to slog.LevelDebug
	Level       slog.Leveler
	ReplaceAttr func(groups []string, a slog.Attr)
	HidePC      bool
}

type groupOrAttributes struct {
	group string
	attrs []slog.Attr
}

func NewCustomHandler(out io.Writer, opts *CustomOptions) *CustomHandler {
	handler := &CustomHandler{out: out, mu: &sync.Mutex{}}

	if opts != nil {
		handler.opts = *opts
	}

	if handler.opts.Level == nil {
		handler.opts.Level = slog.LevelDebug
	}

	return handler
}

func (h *CustomHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *CustomHandler) Handle(ctx context.Context, r slog.Record) error {
	buf := make([]byte, 0, 1024)

	// [TIME]
	if !r.Time.IsZero() {
		buf = h.appendTime(buf, r.Time)
	}

	// [LEVEL]
	buf = h.appendLevel(buf, r.Level)

	// /main.go:17
	if r.PC != 0 && !h.opts.HidePC {
		buf = h.appendPC(buf, r.PC)
	}

	// Message, as plain text
	buf = h.appendMessage(buf, r.Message)

	// Handle state from WithGroup / WithAttrs
	goas := h.goas
	// WithGroup appends the group whose name it's called with as a group to the
	// handler. Since subsequent attributes are added to that group (including attributes
	// from the record "r"), if there are no attributes in r the last group will be empty.
	if r.NumAttrs() == 0 {
		for len(goas) > 0 && goas[len(goas)-1].group != "" {
			goas = goas[:len(goas)-1]
		}
	}

	// All attributes and groups go inside brackets []
	hasAttributes := (r.NumAttrs() + len(goas)) > 0
	if hasAttributes {
		buf = fmt.Append(buf, "[ ")
	}

	for _, goa := range goas {
		if goa.group != "" {
			buf = fmt.Appendf(buf, "%s:{ ", goa.group)
		} else {
			for _, a := range goa.attrs {
				buf = h.appendAttr(buf, a)
			}
		}
	}

	// Rest of the attributes
	r.Attrs(func(a slog.Attr) bool {
		buf = h.appendAttr(buf, a)
		return true
	})

	for _, goa := range goas {
		if goa.group != "" {
			buf = fmt.Appendf(buf, "}")
		}
	}

	if hasAttributes {
		buf = fmt.Append(buf, "]")
	}

	buf = fmt.Append(buf, "\n")

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(buf)
	return err
}

func (h *CustomHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttributes{group: name})
}

func (h *CustomHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) <= 0 {
		return h
	}
	return h.withGroupOrAttrs(groupOrAttributes{attrs: attrs})
}

func (h *CustomHandler) withGroupOrAttrs(goa groupOrAttributes) *CustomHandler {
	h2 := *h
	h2.goas = make([]groupOrAttributes, len(h.goas)+1)
	copy(h2.goas, h.goas)
	h2.goas[len(h2.goas)-1] = goa
	return &h2
}

func (h *CustomHandler) appendMessage(buf []byte, msg string) []byte {
	buf = fmt.Appendf(buf, "%s", msg)
	buf = h.appendSeparator(buf)
	return buf
}

func (h *CustomHandler) appendTime(buf []byte, t time.Time) []byte {
	buf = fmt.Appendf(buf, "[%s]", t.Format(time.RFC3339))
	buf = h.appendSeparator(buf)
	return buf
}

func (h *CustomHandler) appendLevel(buf []byte, l slog.Level) []byte {
	buf = fmt.Appendf(buf, "[%s]", l.String())
	buf = h.appendSeparator(buf)
	return buf
}

func (h *CustomHandler) appendPC(buf []byte, pc uintptr) []byte {
	fs := runtime.CallersFrames([]uintptr{pc})
	f, _ := fs.Next()
	buf = fmt.Appendf(buf, "%s:%d", f.File, f.Line)
	buf = h.appendSeparator(buf)
	return buf
}

func (h *CustomHandler) appendAttr(buf []byte, a slog.Attr) []byte {
	a.Value = a.Value.Resolve()

	if a.Equal(slog.Attr{}) {
		return buf
	}

	switch a.Value.Kind() {
	case slog.KindString:
		// Quote string values: k="value"
		buf = fmt.Appendf(buf, "%s=%q", a.Key, a.Value.String())
	case slog.KindGroup:
		// Circle groups with curly braces: g:{ k="v" k2="v2" }
		attrs := a.Value.Group()
		if len(attrs) <= 0 {
			return buf
		}

		if a.Key != "" {
			buf = fmt.Appendf(buf, "%s:", a.Key)
		}

		buf = fmt.Appendf(buf, "{ ")

		for _, atr := range attrs {
			buf = h.appendAttr(buf, atr)
		}

		buf = fmt.Appendf(buf, "}")
	default:
		buf = fmt.Appendf(buf, "%s=%v", a.Key, a.Value)
	}

	buf = h.appendSeparator(buf)
	return buf
}

func (h *CustomHandler) appendSeparator(buf []byte) []byte {
	separator := " "
	buf = fmt.Appendf(buf, separator)
	return buf
}
