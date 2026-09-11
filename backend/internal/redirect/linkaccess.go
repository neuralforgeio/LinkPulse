package redirect

import (
    "encoding/json"
    "errors"
    "net/http"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/golang-jwt/jwt/v5"

    "linkpulse/internal/auth"
    "linkpulse/internal/httpx"
)

// linkAccessTTL is how long a verified password unlocks one link.
const linkAccessTTL = 30 * time.Minute

// linkAccessCookieName builds the cookie name for one short code.
func linkAccessCookieName(code string) string { return "link_access_" + code }

// newLinkAccessToken signs a JWT bound to one short code.
func newLinkAccessToken(secret, code string) (string, time.Time, error) {
    now := time.Now()
    expiresAt := now.Add(linkAccessTTL)

    claims := jwt.MapClaims{
        "sub": code,
        "typ": "link_access", // distinguishes these from user JWTs
        "iat": now.Unix(),
        "exp": expiresAt.Unix(),
    }

    signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
    if err != nil {
        return "", time.Time{}, err
    }
    return signed, expiresAt, nil
}

// parseLinkAccessToken reports whether the token is a valid link-access
// token for THIS code. The "typ" and "sub" checks mean a user access
// token or a token for another code is always rejected.
func parseLinkAccessToken(secret, token, code string) bool {
    parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
        return []byte(secret), nil
    }, jwt.WithValidMethods([]string{"HS256"}))
    if err != nil || !parsed.Valid {
        return false
    }

    claims, ok := parsed.Claims.(jwt.MapClaims)
    if !ok {
        return false
    }
    return claims["typ"] == "link_access" && claims["sub"] == code
}

// VerifyPassword handles POST /api/v1/public/links/{shortCode}/verify-password
// (PRD 9.5.3). On success it sets an HttpOnly cookie that unlocks the
// redirect for 30 minutes. Rate limited per IP like the redirect itself.
func (h *Handler) VerifyPassword(w http.ResponseWriter, r *http.Request) {
    code := chi.URLParam(r, "shortCode")
    if !shortCodeRegex.MatchString(code) {
        httpx.Error(w, http.StatusNotFound, httpx.CodeNotFound, "link not found")
        return
    }

    link, err := h.svc.Find(r.Context(), code)
    if errors.Is(err, ErrNotFound) {
        httpx.Error(w, http.StatusNotFound, httpx.CodeNotFound, "link not found")
        return
    }
    if err != nil {
        h.log.Error("verify-password lookup failed", "error", err, "code", code)
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
        return
    }

    // Non-protected links have nothing to verify.
    if link.PasswordHash == nil {
        httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "this link is not password protected")
        return
    }

    r.Body = http.MaxBytesReader(w, r.Body, 2048)
    var in struct {
        Password string `json:"password"`
    }
    if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
        httpx.Error(w, http.StatusBadRequest, httpx.CodeBadRequest, "invalid JSON body")
        return
    }

    ok, err := auth.VerifyPassword(in.Password, *link.PasswordHash)
    if err != nil {
        h.log.Error("verify-password hash check failed", "error", err, "code", code)
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
        return
    }
    if !ok {
        httpx.Error(w, http.StatusUnauthorized, httpx.CodeUnauthorized, "incorrect password")
        return
    }

    token, expiresAt, err := newLinkAccessToken(h.jwtSecret, code)
    if err != nil {
        h.log.Error("verify-password token sign failed", "error", err, "code", code)
        httpx.Error(w, http.StatusInternalServerError, httpx.CodeInternalError, "internal error")
        return
    }

    sameSite := http.SameSiteLaxMode
    if h.cookieSameSiteNone {
        sameSite = http.SameSiteNoneMode
    }
    http.SetCookie(w, &http.Cookie{
        Name:     linkAccessCookieName(code),
        Value:    token,
        Path:     "/", // must reach GET /{code}
        HttpOnly: true,
        Secure:   h.cookieSecure,
        SameSite: sameSite,
        MaxAge:   int(time.Until(expiresAt).Seconds()),
    })

    httpx.Success(w, http.StatusOK, map[string]any{
        "verified":   true,
        "expires_at": expiresAt,
    })
}
