package redirect

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/mileusna/useragent"

	"linkpulse/internal/clickbuffer"
)

// shortCodeRegex matches our code format — anything else (favicon.ico,
// random junk) is rejected before touching the database.
var shortCodeRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,32}$`)

// HandlerConfig carries the redirect handler's deployment settings.
type HandlerConfig struct {
	IPSalt             string
	FrontendOrigin     string
	JWTSecret          string
	CookieSecure       bool
	CookieSameSiteNone bool
}

// Handler resolves GET /{code} and verifies link passwords.
type Handler struct {
	svc                *Service
	buf                *clickbuffer.Buffer
	log                *slog.Logger
	ipSalt             string
	frontend           string
	jwtSecret          string
	cookieSecure       bool
	cookieSameSiteNone bool
}

// NewHandler builds the redirect handler.
func NewHandler(svc *Service, buf *clickbuffer.Buffer, log *slog.Logger, cfg HandlerConfig) *Handler {
	return &Handler{
		svc:                svc,
		buf:                buf,
		log:                log,
		ipSalt:             cfg.IPSalt,
		frontend:           cfg.FrontendOrigin,
		jwtSecret:          cfg.JWTSecret,
		cookieSecure:       cfg.CookieSecure,
		cookieSameSiteNone: cfg.CookieSameSiteNone,
	}
}

// Resolve handles GET /{code}.
//
// Note on max_clicks: the counter is bumped asynchronously, so a couple
// of extra clicks can slip past the limit in the same second — an
// accepted trade-off for never blocking a redirect (PRD 9.5.4).
func (h *Handler) Resolve(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")

	if !shortCodeRegex.MatchString(code) {
		http.Error(w, "link not found", http.StatusNotFound)
		return
	}

	link, err := h.svc.Find(r.Context(), code)
	switch {
	case errors.Is(err, ErrNotFound):
		http.Error(w, "link not found", http.StatusNotFound)
		return
	case err != nil:
		h.log.Error("redirect lookup failed",
			"error", err, "code", code, "request_id", middleware.GetReqID(r.Context()))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Status checks (PRD 9.5.2, 9.13).
	switch {
	case link.Status == "disabled":
		http.Error(w, "this link has been disabled", http.StatusGone)
		return
	case link.ExpiresAt != nil && link.ExpiresAt.Before(time.Now()):
		http.Error(w, "this link has expired", http.StatusGone)
		return
	case link.MaxClicks != nil && link.ClickCount >= *link.MaxClicks:
		http.Error(w, "this link reached its click limit", http.StatusGone)
		return
	}

	// Password-protected links: a verified password grants a 30-minute
	// link-access cookie bound to this code (PRD 9.5.3).
	if link.PasswordHash != nil {
		granted := false
		if cookie, err := r.Cookie(linkAccessCookieName(code)); err == nil {
			granted = parseLinkAccessToken(h.jwtSecret, cookie.Value, code)
		}
		if !granted {
			http.Redirect(w, r, h.frontend+"/protected/"+code, http.StatusFound)
			return
		}
	}

	// Record the click — never blocking, never failing the redirect.
	// The IP is stored only as a salted hash (PRD 9.5.4, 16.11).
	device, browser, osName := parseUserAgent(r.UserAgent())
	h.buf.Enqueue(clickbuffer.ClickEvent{
		ID:           uuid.New(),
		LinkID:       link.ID,
		TenantID:     link.TenantID,
		ShortCode:    link.ShortCode,
		RequestID:    middleware.GetReqID(r.Context()),
		IPHash:       hashIP(r.RemoteAddr, h.ipSalt),
		UserAgentRaw: r.UserAgent(),
		DeviceType:   device,
		Browser:      browser,
		OS:           osName,
		Referrer:     r.Referer(),
		Source:       link.UTMSource,
		Medium:       link.UTMMedium,
		Campaign:     link.UTMCampaign,
	})

	// PRD 9.5.2: 302 by default, never cached.
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, link.DestinationURL, http.StatusFound)
}

// hashIP converts "host:port" into a salted SHA-256 hash. The raw IP
// is never stored anywhere.
func hashIP(remoteAddr, salt string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	sum := sha256.Sum256([]byte(salt + host))
	return hex.EncodeToString(sum[:])
}

// parseUserAgent extracts device type, browser, and OS.
func parseUserAgent(raw string) (device, browser, osName string) {
	ua := useragent.Parse(raw)
	switch {
	case ua.Mobile:
		device = "mobile"
	case ua.Tablet:
		device = "tablet"
	default:
		device = "desktop"
	}
	return device, ua.Name, ua.OS
}
