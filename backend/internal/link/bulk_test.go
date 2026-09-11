package link

import (
	"strings"
	"testing"
)

func TestParseImportCSV(t *testing.T) {
	t.Run("requires destination_url header", func(t *testing.T) {
		_, uerr := ParseImportCSV([]byte("title\nMy Link"))
		if uerr == nil {
			t.Fatal("expected error for missing destination_url column")
		}
		if !strings.Contains(uerr.Message, "destination_url") {
			t.Fatalf("unexpected message: %s", uerr.Message)
		}
	})

	t.Run("parses named columns in any order", func(t *testing.T) {
		csv := "title,destination_url,tags\n" +
			"\"Docs, main\",https://example.com/docs,a; b\n" +
			"Blog,https://example.com/blog,\n"
		rows, uerr := ParseImportCSV([]byte(csv))
		if uerr != nil {
			t.Fatalf("unexpected error: %v", uerr)
		}
		if len(rows) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(rows))
		}
		if rows[0].DestinationURL != "https://example.com/docs" {
			t.Errorf("row 0 destination: %q", rows[0].DestinationURL)
		}
		if rows[0].Title != "Docs, main" {
			t.Errorf("row 0 title (quoted comma): %q", rows[0].Title)
		}
		if len(rows[0].Tags) != 2 || rows[0].Tags[0] != "a" || rows[0].Tags[1] != "b" {
			t.Errorf("row 0 tags: %v", rows[0].Tags)
		}
		if len(rows[1].Tags) != 0 {
			t.Errorf("row 1 tags should be empty: %v", rows[1].Tags)
		}
	})

	t.Run("skips blank lines", func(t *testing.T) {
		csv := "destination_url\n\nhttps://example.com\n"
		rows, uerr := ParseImportCSV([]byte(csv))
		if uerr != nil {
			t.Fatalf("unexpected error: %v", uerr)
		}
		if len(rows) != 1 {
			t.Fatalf("expected 1 row, got %d", len(rows))
		}
	})

	t.Run("rejects empty csv", func(t *testing.T) {
		if _, uerr := ParseImportCSV([]byte("")); uerr == nil {
			t.Fatal("expected error for empty CSV")
		}
	})

	t.Run("supports custom_code column", func(t *testing.T) {
		csv := "destination_url,custom_code\nhttps://example.com,mycode\n"
		rows, uerr := ParseImportCSV([]byte(csv))
		if uerr != nil {
			t.Fatalf("unexpected error: %v", uerr)
		}
		if rows[0].CustomCode != "mycode" {
			t.Errorf("custom code: %q", rows[0].CustomCode)
		}
	})

	t.Run("header case-insensitive", func(t *testing.T) {
		csv := "Destination_URL,Title\nhttps://example.com,Hi\n"
		rows, uerr := ParseImportCSV([]byte(csv))
		if uerr != nil {
			t.Fatalf("unexpected error: %v", uerr)
		}
		if rows[0].DestinationURL != "https://example.com" {
			t.Errorf("destination: %q", rows[0].DestinationURL)
		}
	})
}

func TestParseTagsField(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"a; b;c", []string{"a", "b", "c"}},
		{"a, b,c", []string{"a", "b", "c"}},
		{"  Mixed ; Case ", []string{"mixed", "case"}},
	}
	for _, tc := range cases {
		got := ParseTagsField(tc.in)
		if len(got) != len(tc.want) {
			t.Errorf("ParseTagsField(%q) = %v, want %v", tc.in, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("ParseTagsField(%q)[%d] = %q, want %q", tc.in, i, got[i], tc.want[i])
			}
		}
	}
}

func TestExportCSV(t *testing.T) {
	s := &Service{baseURL: "http://localhost:8080"}
	links := []LinkOut{
		{
			ShortCode:      "promo26",
			ShortURL:       "http://localhost:8080/promo26",
			DestinationURL: "https://example.com/promo?utm_source=x",
			Title:          "Promo, with comma",
			Status:         "active",
			ClickCount:     1280,
			Tags:           []string{"campaign", "social"},
		},
	}
	raw, err := s.ExportCSV(links)
	if err != nil {
		t.Fatalf("export failed: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, "short_code,short_url,title,destination_url,status,click_count,created_at,expires_at,tags") {
		t.Errorf("header missing: %s", text)
	}
	if !strings.Contains(text, "\"Promo, with comma\"") {
		t.Errorf("quoted comma title not escaped: %s", text)
	}
	if !strings.Contains(text, "campaign;social") {
		t.Errorf("tags not joined: %s", text)
	}
}
