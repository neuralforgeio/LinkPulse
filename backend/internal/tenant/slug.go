package tenant

import "strings"

// Slugify converts a tenant name into a URL-safe slug: lowercase ASCII
// letters, digits, and dashes (PRD 9.2.1 slug format).
func Slugify(name string) string {
    var b strings.Builder
    lastWasDash := true // prevents a leading dash
    for _, r := range strings.ToLower(name) {
        switch {
        case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
            b.WriteRune(r)
            lastWasDash = false
        default:
            if !lastWasDash {
                b.WriteByte('-')
                lastWasDash = true
            }
        }
    }

    slug := strings.Trim(b.String(), "-")
    if slug == "" {
        return "workspace"
    }
    if len(slug) > 48 {
        slug = slug[:48]
    }
    return slug
}
