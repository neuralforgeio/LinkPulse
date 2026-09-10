package ratelimit

import (
    "net"
    "net/http"
    "strconv"
    "sync"
    "time"

    "linkpulse/internal/httpx"
)

// Result describes one Allow decision.
type Result struct {
    Allowed   bool
    Limit     int
    Remaining int
    ResetAt   time.Time
}

type window struct {
    count   int
    resetAt time.Time
}

// Limiter tracks one rate limit.
type Limiter struct {
    mu      sync.Mutex
    windows map[string]*window
    limit   int
    window  time.Duration
}

// New builds a Limiter. limit <= 0 disables it (allows everything).
func New(limit int, per time.Duration) *Limiter {
    l := &Limiter{
        windows: map[string]*window{},
        limit:   limit,
        window:  per,
    }
    if limit > 0 {
        go l.janitor()
    }
    return l
}

// Allow consumes one unit for the key, if the limit permits.
func (l *Limiter) Allow(key string) Result {
    if l.limit <= 0 {
        return Result{Allowed: true, ResetAt: time.Now().Add(l.window)}
    }

    l.mu.Lock()
    defer l.mu.Unlock()

    now := time.Now()
    w, ok := l.windows[key]
    if !ok || now.After(w.resetAt) {
        w = &window{resetAt: now.Add(l.window)}
        l.windows[key] = w
    }
    w.count++

    remaining := l.limit - w.count
    if remaining < 0 {
        remaining = 0
    }
    return Result{
        Allowed:   w.count <= l.limit,
        Limit:     l.limit,
        Remaining: remaining,
        ResetAt:   w.resetAt,
    }
}

// janitor evicts expired windows so the map never grows unbounded.
func (l *Limiter) janitor() {
    interval := l.window
    if interval < time.Second {
        interval = time.Second
    }
    ticker := time.NewTicker(interval)
    defer ticker.Stop()
    for range ticker.C {
        l.mu.Lock()
        now := time.Now()
        for key, w := range l.windows {
            if now.After(w.resetAt) {
                delete(l.windows, key)
            }
        }
        l.mu.Unlock()
    }
}

// KeyIP identifies a caller by remote IP (host only, port stripped).
func KeyIP(r *http.Request) string {
    host, _, err := net.SplitHostPort(r.RemoteAddr)
    if err != nil {
        return r.RemoteAddr
    }
    return host
}

// Middleware rate limits requests by keyFn and writes the PRD 9.9
// headers. On limit, responds 429 with Retry-After.
func (l *Limiter) Middleware(keyFn func(r *http.Request) string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            res := l.Allow(keyFn(r))

            if l.limit > 0 {
                w.Header().Set("X-RateLimit-Limit", strconv.Itoa(res.Limit))
                w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
                w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(res.ResetAt.Unix(), 10))
            }

            if !res.Allowed {
                retry := int(time.Until(res.ResetAt).Seconds()) + 1
                if retry < 1 {
                    retry = 1
                }
                w.Header().Set("Retry-After", strconv.Itoa(retry))
                httpx.Error(w, http.StatusTooManyRequests, httpx.CodeRateLimited, "too many requests, slow down")
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}
