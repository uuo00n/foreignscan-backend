package middleware

import "testing"

func TestParseAllowedOrigins_Default(t *testing.T) {
	got := parseAllowedOrigins("")
	if len(got) != 2 {
		t.Fatalf("expected 2 default origins, got %d", len(got))
	}
	if got[0] != "http://localhost:8080" || got[1] != "http://127.0.0.1:8080" {
		t.Fatalf("unexpected defaults: %#v", got)
	}
}

func TestParseAllowedOrigins_CommaSeparated(t *testing.T) {
	got := parseAllowedOrigins(" http://a.com , http://b.com ")
	if len(got) != 2 {
		t.Fatalf("expected 2 parsed origins, got %d", len(got))
	}
	if got[0] != "http://a.com" || got[1] != "http://b.com" {
		t.Fatalf("unexpected parsed origins: %#v", got)
	}
}

func TestHasWildcard(t *testing.T) {
	if !hasWildcard([]string{"http://localhost:8080", "*"}) {
		t.Fatalf("expected wildcard to be detected")
	}
	if hasWildcard([]string{"http://localhost:8080"}) {
		t.Fatalf("did not expect wildcard")
	}
}
