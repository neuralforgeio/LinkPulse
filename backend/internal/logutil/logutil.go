package logutil

import (
    "context"
    "errors"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
    "sync"

    "log/slog"

    "gopkg.in/natefinch/lumberjack.v2"
)

// ANSI escape codes. Set NO_COLOR=1 to disable (CI, piping to files).
const (
    ansiReset  = "\033[0m"
    ansiDim    = "\033[2m"
    ansiRed    = "\033[31;1m"
    ansiGreen  = "\033[32;1m"
    ansiYellow = "\033[33;1m"
    ansiCyan   = "\033[36m"
)

var noColor = os.Getenv("NO_COLOR") != ""

func colorize(code, s string) string {
    if noColor {
        return s
    }
    return code + s + reset()
}

func reset() string {
    if noColor {
        return ""
    }
    return ansiReset
}

func levelColor(l slog.Level) string {
    switch {
    case l >= slog.LevelError:
        return ansiRed
    case l >= slog.LevelWarn:
        return ansiYellow
    case l >= slog.LevelInfo:
        return ansiGreen
    default:
        return ansiCyan
    }
}

// PrettyHandler renders one short, colored line per record:
//
//	19:12:10 INFO  POST /api/v1/auth/login → 200 (51ms) request_id=…
//
// The date is omitted on purpose — it lives in the JSON file log.
type PrettyHandler struct {
    mu    sync.Mutex
    w     io.Writer
    level slog.Level
    attrs []slog.Attr
}

// NewPretty builds a PrettyHandler writing to w.
func NewPretty(w io.Writer, level slog.Level) *PrettyHandler {
    return &PrettyHandler{w: w, level: level}
}

func (h *PrettyHandler) Enabled(_ context.Context, l slog.Level) bool {
    return l >= h.level
}

func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
    var b strings.Builder

    b.WriteString(colorize(ansiDim, r.Time.Format("15:04:05")))
    b.WriteByte(' ')

    b.WriteString(colorize(levelColor(r.Level), fmt.Sprintf("%-5s", r.Level.String())))
    b.WriteByte(' ')

    b.WriteString(r.Message)

    for _, a := range h.attrs {
        appendAttr(&b, a)
    }
    r.Attrs(func(a slog.Attr) bool {
        appendAttr(&b, a)
        return true
    })
    b.WriteByte('\n')

    h.mu.Lock()
    defer h.mu.Unlock()
    _, err := io.WriteString(h.w, b.String())
    return err
}

func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    return &PrettyHandler{
        w:     h.w,
        level: h.level,
        attrs: append(append([]slog.Attr{}, h.attrs...), attrs...),
    }
}

// WithGroup is not used by this app; pass through unchanged.
func (h *PrettyHandler) WithGroup(string) slog.Handler { return h }

func appendAttr(b *strings.Builder, a slog.Attr) {
    if a.Equal(slog.Attr{}) {
        return
    }
    b.WriteByte(' ')
    if a.Key == "error" {
        // Errors deserve attention: always red.
        b.WriteString(colorize(ansiRed, fmt.Sprintf("%s=%v", a.Key, a.Value)))
        return
    }
    b.WriteString(colorize(ansiDim, fmt.Sprintf("%s=%v", a.Key, a.Value)))
}

// MultiHandler fans every record out to all wrapped handlers
// (console + file), so one log call lands in both places.
type MultiHandler struct {
    handlers []slog.Handler
}

// NewMulti wraps handlers into a single slog.Handler.
func NewMulti(handlers ...slog.Handler) *MultiHandler {
    return &MultiHandler{handlers: handlers}
}

func (m *MultiHandler) Enabled(ctx context.Context, l slog.Level) bool {
    for _, h := range m.handlers {
        if h.Enabled(ctx, l) {
            return true
        }
    }
    return false
}

func (m *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
    var errs []error
    for _, h := range m.handlers {
        if h.Enabled(ctx, r.Level) {
            if err := h.Handle(ctx, r.Clone()); err != nil {
                errs = append(errs, err)
            }
        }
    }
    return errors.Join(errs...)
}

func (m *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
    handlers := make([]slog.Handler, len(m.handlers))
    for i, h := range m.handlers {
        handlers[i] = h.WithAttrs(attrs)
    }
    return &MultiHandler{handlers: handlers}
}

func (m *MultiHandler) WithGroup(name string) slog.Handler {
    handlers := make([]slog.Handler, len(m.handlers))
    for i, h := range m.handlers {
        handlers[i] = h.WithGroup(name)
    }
    return &MultiHandler{handlers: handlers}
}

// Options configures the root logger.
type Options struct {
    // Level: debug | info | warn | error (default info).
    Level string
    // Console format: pretty | json (default pretty). Production runs
    // json on the console; the file is always JSON.
    Format string
    // File path for the rotating JSON log. Empty disables file logging.
    File string
}

// New builds the root logger: a colored console handler for humans and,
// when File is set, a rotating JSON file handler for history.
func New(opts Options) *slog.Logger {
    level := parseLevel(opts.Level)

    var console slog.Handler = NewPretty(os.Stdout, level)
    if strings.EqualFold(opts.Format, "json") {
        console = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
    }

    handlers := []slog.Handler{console}

    if opts.File != "" {
        if err := os.MkdirAll(filepath.Dir(opts.File), 0o755); err != nil {
            // Bootstrap-time warning: the logger itself is not ready yet.
            fmt.Fprintf(os.Stderr, "logutil: cannot create log directory: %v\n", err)
        } else {
            handlers = append(handlers, slog.NewJSONHandler(newRotatingWriter(opts.File), &slog.HandlerOptions{Level: level}))
        }
    }

    return slog.New(NewMulti(handlers...))
}

// newRotatingWriter returns a size-rotating writer: 10 MB per file,
// 5 backups kept, older files gzip-compressed.
func newRotatingWriter(path string) io.Writer {
    return &lumberjack.Logger{
        Filename:   path,
        MaxSize:    10,
        MaxBackups: 5,
        Compress:   true,
    }
}

func parseLevel(s string) slog.Level {
    switch strings.ToLower(s) {
    case "debug":
        return slog.LevelDebug
    case "warn":
        return slog.LevelWarn
    case "error":
        return slog.LevelError
    default:
        return slog.LevelInfo
    }
}
