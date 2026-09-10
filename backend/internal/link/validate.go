package link

import (
    "net/url"
    "regexp"
    "strings"

    "linkpulse/internal/httpx"
)

// Rules from PRD 9.4.1.
const (
    maxDestinationLen = 2048
    maxTitleLen       = 200
    maxTags           = 10
    maxTagLen         = 32
    minLinkPassword   = 4
)

// customCodeRegex is the alias format (PRD 9.4.1).
var customCodeRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,32}$`)

// UTMInput carries the UTM builder fields (PRD 9.12).
type UTMInput struct {
    Source   string `json:"source"`
    Medium   string `json:"medium"`
    Campaign string `json:"campaign"`
    Term     string `json:"term"`
    Content  string `json:"content"`
}

func (u UTMInput) provided() bool {
    return u.Source != "" || u.Medium != "" || u.Campaign != "" ||
        u.Term != "" || u.Content != ""
}

// ValidateDestinationURL accepts absolute http/https URLs only. The
// scheme allow-list blocks javascript:, file:, data: and friends at the
// door (PRD 9.4.1, 16.14–16.15).
func ValidateDestinationURL(raw string) *httpx.UserError {
    if len(raw) > maxDestinationLen {
        return &httpx.UserError{
            Status:  422,
            Code:    httpx.CodeValidationError,
            Message: "destination_url is too long (max 2048 characters)",
        }
    }
    u, err := url.Parse(raw)
    if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
        return &httpx.UserError{
            Status:  422,
            Code:    httpx.CodeValidationError,
            Message: "destination_url must be a valid http:// or https:// URL",
        }
    }
    return nil
}

// ValidateCustomCode checks the alias format; empty means "generate one".
func ValidateCustomCode(code string) *httpx.UserError {
    if code == "" {
        return nil
    }
    if !customCodeRegex.MatchString(code) {
        return &httpx.UserError{
            Status:  422,
            Code:    httpx.CodeValidationError,
            Message: "custom_code must be 3-32 characters: letters, digits, dash, underscore",
        }
    }
    return nil
}

// ApplyUTM merges explicitly provided UTM fields into the destination
// URL's query string. Explicit input overwrites existing values; fields
// left empty keep whatever the destination already carries
// (PRD 9.12: overwrite when filled explicitly).
func ApplyUTM(destination string, utm UTMInput) string {
    if !utm.provided() {
        return destination
    }
    u, err := url.Parse(destination)
    if err != nil {
        return destination
    }
    q := u.Query()
    set := func(key, val string) {
        if val != "" {
            q.Set(key, val)
        }
    }
    set("utm_source", utm.Source)
    set("utm_medium", utm.Medium)
    set("utm_campaign", utm.Campaign)
    set("utm_term", utm.Term)
    set("utm_content", utm.Content)
    u.RawQuery = q.Encode()
    return u.String()
}

// NormalizeTags trims, lowercases, dedupes, and caps the tag list.
func NormalizeTags(tags []string) []string {
    if len(tags) == 0 {
        return []string{}
    }
    seen := make(map[string]bool)
    out := []string{}
    for _, t := range tags {
        t = strings.ToLower(strings.TrimSpace(t))
        if t == "" || len(t) > maxTagLen || seen[t] {
            continue
        }
        seen[t] = true
        out = append(out, t)
        if len(out) == maxTags {
            break
        }
    }
    return out
}
