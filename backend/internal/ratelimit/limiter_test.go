package ratelimit

import (
	"testing"
	"time"
)

func TestAllowBlocksAtLimit(t *testing.T) {
    l := New(3, 20*time.Millisecond)

    for i := 1; i <= 3; i++ {
        res := l.Allow("1.2.3.4")
        if !res.Allowed {
            t.Fatalf("request %d should be allowed", i)
        }
        if res.Remaining != 3-i {
            t.Errorf("request %d: remaining = %d, want %d", i, res.Remaining, 3-i)
        }
    }

    if res := l.Allow("1.2.3.4"); res.Allowed {
        t.Fatal("4th request should be blocked")
    }

    // A different key has its own window.
    if res := l.Allow("5.6.7.8"); !res.Allowed {
        t.Fatal("different key should be allowed")
    }

    // The window resets after it expires.
    time.Sleep(25 * time.Millisecond)
    if res := l.Allow("1.2.3.4"); !res.Allowed {
        t.Fatal("request after window reset should be allowed")
    }
}

func TestDisabledLimiterAllowsEverything(t *testing.T) {
    l := New(0, time.Minute)
    for i := 0; i < 1000; i++ {
        if !l.Allow("k").Allowed {
            t.Fatal("disabled limiter must allow everything")
        }
    }
}
