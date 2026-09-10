package shortid

import (
    "crypto/rand"
    "fmt"
)

// base62: digits, uppercase, lowercase — URL-safe, no look-alikes issue.
const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// New returns a random base62 code of length n (e.g. n=7 → "aB3xK9p").
// Rejection sampling keeps the distribution uniform: bytes that would
// cause modulo bias are discarded, so every character is equally likely.
func New(n int) (string, error) {
    const maxByte = byte(256 - (256 % len(alphabet)))

    out := make([]byte, 0, n)
    buf := make([]byte, n*2)
    for len(out) < n {
        if _, err := rand.Read(buf); err != nil {
            return "", fmt.Errorf("read random bytes: %w", err)
        }
        for _, b := range buf {
            if b >= maxByte {
                continue // rejected — avoids modulo bias
            }
            out = append(out, alphabet[int(b)%len(alphabet)])
            if len(out) == n {
                break
            }
        }
    }
    return string(out), nil
}
