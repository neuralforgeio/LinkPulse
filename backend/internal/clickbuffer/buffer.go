package clickbuffer

import (
    "context"
    "log/slog"
    "sync/atomic"
    "time"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

// ClickEvent is one recorded redirect, ready to be inserted.
type ClickEvent struct {
    ID           uuid.UUID
    LinkID       uuid.UUID
    TenantID     uuid.UUID
    ShortCode    string
    RequestID    string
    IPHash       string
    UserAgentRaw string
    DeviceType   string
    Browser      string
    OS           string
    Referrer     string
    Source       string
    Medium       string
    Campaign     string
}

// Options configures the buffer (PRD 17.1 defaults).
type Options struct {
    Size          int           // channel capacity
    FlushInterval time.Duration // flush at least this often
    BatchSize     int           // flush when this many events queue up
}

// Buffer is the in-memory click pipeline.
type Buffer struct {
    ch            chan ClickEvent
    db            *pgxpool.Pool
    log           *slog.Logger
    flushInterval time.Duration
    batchSize     int
    done          chan struct{}
    dropped       atomic.Int64
}

// New builds a Buffer. Call Start before enqueueing, Stop on shutdown.
func New(db *pgxpool.Pool, log *slog.Logger, opts Options) *Buffer {
    if opts.Size <= 0 {
        opts.Size = 5000
    }
    if opts.FlushInterval <= 0 {
        opts.FlushInterval = time.Second
    }
    if opts.BatchSize <= 0 {
        opts.BatchSize = 500
    }
    return &Buffer{
        ch:            make(chan ClickEvent, opts.Size),
        db:            db,
        log:           log,
        flushInterval: opts.FlushInterval,
        batchSize:     opts.BatchSize,
        done:          make(chan struct{}),
    }
}

// Start launches the flush worker.
func (b *Buffer) Start() {
    go b.run()
}

// Enqueue adds an event without ever blocking the redirect path. When
// the buffer is full the event is dropped and counted — a redirect must
// never fail because analytics is busy (PRD 9.5.5).
func (b *Buffer) Enqueue(ev ClickEvent) {
    select {
    case b.ch <- ev:
    default:
        n := b.dropped.Add(1)
        b.log.Warn("click event dropped: buffer full",
            "dropped_total", n, "request_id", ev.RequestID)
    }
}

// Stop drains the buffer and flushes everything left. Call this during
// graceful shutdown BEFORE closing the pool.
func (b *Buffer) Stop() {
    close(b.ch)
    <-b.done
}

func (b *Buffer) run() {
    defer close(b.done)
    ticker := time.NewTicker(b.flushInterval)
    defer ticker.Stop()

    events := make([]ClickEvent, 0, b.batchSize)
    for {
        select {
        case ev, ok := <-b.ch:
            if !ok {
                // Stop() called: drain what's left and exit.
                b.flush(events)
                return
            }
            events = append(events, ev)
            if len(events) >= b.batchSize {
                events = b.flush(events)
            }
        case <-ticker.C:
            events = b.flush(events)
        }
    }
}

// flush inserts the batch in one transaction, bumps click_count for
// each affected link, and returns a reused empty slice. On database
// failure the batch is dropped and logged — the PRD allows dropping
// under pressure rather than blocking or losing redirects.
func (b *Buffer) flush(events []ClickEvent) []ClickEvent {
    if len(events) == 0 {
        return events[:0]
    }

    started := time.Now()
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    tx, err := b.db.Begin(ctx)
    if err != nil {
        b.log.Error("click flush: begin failed", "error", err, "events", len(events))
        return events[:0]
    }
    defer tx.Rollback(ctx)

    batch := &pgx.Batch{}
    for _, ev := range events {
        batch.Queue(`
            INSERT INTO click_events (id, link_id, tenant_id, short_code,
                request_id, ip_hash, user_agent_raw, device_type, browser, os,
                referrer, source, medium, campaign)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
            ev.ID, ev.LinkID, ev.TenantID, ev.ShortCode,
            ev.RequestID, ev.IPHash, ev.UserAgentRaw, ev.DeviceType, ev.Browser, ev.OS,
            ev.Referrer, ev.Source, ev.Medium, ev.Campaign)
    }
    if err := tx.SendBatch(ctx, batch).Close(); err != nil {
        b.log.Error("click flush: insert failed", "error", err, "events", len(events))
        return events[:0]
    }

    // One counter bump per distinct link in this batch.
    counts := make(map[uuid.UUID]int64)
    for _, ev := range events {
        counts[ev.LinkID]++
    }
    for linkID, n := range counts {
        if _, err := tx.Exec(ctx,
            `UPDATE links SET click_count = click_count + $1 WHERE id = $2`,
            n, linkID); err != nil {
            b.log.Error("click flush: count update failed", "error", err, "link_id", linkID)
            return events[:0]
        }
    }

    if err := tx.Commit(ctx); err != nil {
        b.log.Error("click flush: commit failed", "error", err, "events", len(events))
        return events[:0]
    }

    b.log.Info("click buffer flushed",
        "events", len(events),
        "links", len(counts),
        "duration_ms", time.Since(started).Milliseconds())
    return events[:0]
}
