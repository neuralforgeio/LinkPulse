package email

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "net/http"
    "time"
)

// Sender delivers emails via Resend, or logs them in dev mode.
type Sender struct {
    apiKey string
    from   string
}

// NewSender builds a Sender. An empty apiKey activates dev mode.
func NewSender(apiKey string, log *slog.Logger) *Sender {
    _ = log // used indirectly via Send's caller logging; kept for symmetry
    return &Sender{
        apiKey: apiKey,
        // The only sender available without a verified domain (free tier).
        from: "LinkPulse <onboarding@resend.dev>",
    }
}

// InDevMode reports whether emails are logged instead of sent.
func (s *Sender) InDevMode() bool { return s.apiKey == "" }

// Send delivers an HTML email to one recipient.
func (s *Sender) Send(ctx context.Context, to, subject, html string) error {
    if s.InDevMode() {
        // The caller logs the interesting content (e.g. the OTP code).
        return nil
    }

    payload, err := json.Marshal(map[string]string{
        "from":    s.from,
        "to":      to,
        "subject": subject,
        "html":    html,
    })
    if err != nil {
        return fmt.Errorf("marshal email: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodPost,
        "https://api.resend.com/emails", bytes.NewReader(payload))
    if err != nil {
        return fmt.Errorf("build email request: %w", err)
    }
    req.Header.Set("Authorization", "Bearer "+s.apiKey)
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{Timeout: 10 * time.Second}
    res, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("send email: %w", err)
    }
    defer res.Body.Close()

    if res.StatusCode >= 300 {
        return fmt.Errorf("resend rejected the email: status %d", res.StatusCode)
    }
    return nil
}
