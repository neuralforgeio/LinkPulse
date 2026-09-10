package shortid

import (
    "strings"
    "testing"
)

func TestNewLengthAndAlphabet(t *testing.T) {
    for _, n := range []int{1, 6, 7, 8, 32} {
        code, err := New(n)
        if err != nil {
            t.Fatalf("New(%d) error: %v", n, err)
        }
        if len(code) != n {
            t.Errorf("New(%d) returned length %d", n, len(code))
        }
        for _, c := range code {
            if !strings.ContainsRune(alphabet, c) {
                t.Errorf("New(%d) produced %q outside the base62 alphabet", n, c)
            }
        }
    }
}

func TestNewUniqueness(t *testing.T) {
    // 62^7 ≈ 3.5 trillion codes; 10,000 draws should never collide.
    seen := make(map[string]bool)
    for i := 0; i < 10000; i++ {
        code, err := New(7)
        if err != nil {
            t.Fatalf("New error: %v", err)
        }
        if seen[code] {
            t.Fatalf("duplicate code %q after %d draws", code, i)
        }
        seen[code] = true
    }
}
