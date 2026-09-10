package link

import (
    "strings"
    "testing"
)

func TestValidateDestinationURL(t *testing.T) {
    valid := []string{
        "https://example.com",
        "http://example.com/very/long/path?x=1",
        "https://sub.domain.co.id/a/b#frag",
    }
    for _, u := range valid {
        if err := ValidateDestinationURL(u); err != nil {
            t.Errorf("expected %q valid, got error: %v", u, err)
        }
    }

    invalid := []string{
        "javascript:alert(1)", // the classic XSS payload
        "data:text/html;base64,SGVsbG8=",
        "file:///C:/Windows",
        "ftp://example.com",
        "not a url",
        "http://", // no host
        "",
    }
    for _, u := range invalid {
        if err := ValidateDestinationURL(u); err == nil {
            t.Errorf("expected %q to be rejected", u)
        }
    }
}

func TestApplyUTM(t *testing.T) {
    // Appends onto a clean destination.
    got := ApplyUTM("https://example.com/product", UTMInput{Source: "instagram", Medium: "social"})
    if !strings.Contains(got, "utm_source=instagram") || !strings.Contains(got, "utm_medium=social") {
        t.Errorf("UTM params not appended: %s", got)
    }

    // Explicit UTM overwrites an existing param.
    got = ApplyUTM("https://example.com?utm_source=old", UTMInput{Source: "new"})
    if !strings.Contains(got, "utm_source=new") || strings.Contains(got, "utm_source=old") {
        t.Errorf("explicit UTM should overwrite: %s", got)
    }

    // No UTM input → URL untouched.
    got = ApplyUTM("https://example.com?utm_source=old", UTMInput{})
    if got != "https://example.com?utm_source=old" {
        t.Errorf("empty UTM must not modify the URL: %s", got)
    }
}

func TestNormalizeTags(t *testing.T) {
    got := NormalizeTags([]string{" Campaign ", "campaign", "", "Instagram", strings.Repeat("x", 40)})
    want := []string{"campaign", "instagram"}
    if len(got) != len(want) {
        t.Fatalf("got %v, want %v", got, want)
    }
    for i := range want {
        if got[i] != want[i] {
            t.Errorf("got %v, want %v", got, want)
        }
    }
}
