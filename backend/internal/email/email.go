package email

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/smtp"
    "os"
    "time"
)

// Sender delivers emails via SMTP, Resend, or dev-log mode.
type Sender struct {
    mode     string
    smtpHost string
    smtpUser string
    smtpPass string
    apiKey   string
    from     string
}

// NewSender builds a Sender from environment configuration.
// Priority: SMTP > Resend > dev.
func NewSender() *Sender {
    smtpHost := os.Getenv("SMTP_HOST")
    smtpUser := os.Getenv("SMTP_USER")
    smtpPass := os.Getenv("SMTP_PASS")
    if smtpHost != "" && smtpUser != "" && smtpPass != "" {
        return &Sender{
            mode:     "smtp",
            smtpHost: smtpHost,
            smtpUser: smtpUser,
            smtpPass: smtpPass,
            from:     smtpUser,
        }
    }

    apiKey := os.Getenv("RESEND_API_KEY")
    if apiKey != "" {
        return &Sender{
            mode:   "resend",
            apiKey: apiKey,
            from:   "LinkPulse <onboarding@resend.dev>",
        }
    }

    return &Sender{mode: "dev"}
}

// Mode returns the delivery mode: "smtp", "resend", or "dev".
func (s *Sender) Mode() string { return s.mode }

// InDevMode reports whether emails are logged instead of sent.
func (s *Sender) InDevMode() bool { return s.mode == "dev" }

// Send delivers an HTML email to one recipient.
func (s *Sender) Send(ctx context.Context, to, subject, html string) error {
    switch s.mode {
    case "smtp":
        return s.sendSMTP(to, subject, html)
    case "resend":
        return s.sendResend(ctx, to, subject, html)
    default:
        return nil
    }
}

// sendSMTP sends via SMTP — delivers to ANY address (PRD: OTP must work
// for all users, not just the account owner).
func (s *Sender) sendSMTP(to, subject, html string) error {
    from := fmt.Sprintf("LinkPulse <%s>", s.smtpUser)

    msg := fmt.Sprintf(
        "From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
        from, to, subject, html,
    )

    addr := s.smtpHost + ":587"
    auth := smtp.PlainAuth("", s.smtpUser, s.smtpPass, s.smtpHost)
    return smtp.SendMail(addr, auth, s.smtpUser, []string{to}, []byte(msg))
}

// sendResend sends via the Resend REST API.
func (s *Sender) sendResend(ctx context.Context, to, subject, html string) error {
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
        body, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
        return fmt.Errorf("resend rejected (status %d): %s", res.StatusCode, string(body))
    }
    return nil
}
