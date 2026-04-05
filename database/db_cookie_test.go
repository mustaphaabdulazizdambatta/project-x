package database

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCookiePersistence(t *testing.T) {
	tmp := filepath.Join(os.TempDir(), "evil_test.db")
	// ensure clean
	_ = os.Remove(tmp)
	d, err := NewDatabase(tmp)
	if err != nil {
		t.Fatalf("failed to create DB: %v", err)
	}
	defer func() {
		_ = os.Remove(tmp)
	}()

	sid := "test-sid-123"
	if err := d.CreateSession(sid, "phish", "/", "ua", "127.0.0.1"); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	tokens := map[string]map[string]*CookieToken{
		"example.com": {
			"SID": &CookieToken{Name: "SID", Value: "abc123", Path: "/", HttpOnly: true},
		},
	}

	if err := d.SetSessionCookieTokens(sid, tokens); err != nil {
		t.Fatalf("SetSessionCookieTokens failed: %v", err)
	}

	sessions, err := d.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	found := false
	for _, s := range sessions {
		if s.SessionId == sid {
			found = true
			if len(s.CookieTokens) == 0 {
				t.Fatalf("no cookie tokens persisted")
			}
			if m, ok := s.CookieTokens["example.com"]; !ok {
				t.Fatalf("domain not found in tokens: %#v", s.CookieTokens)
			} else {
				if c, ok := m["SID"]; !ok || c.Value != "abc123" {
					t.Fatalf("cookie value mismatch: %#v", s.CookieTokens)
				}
			}
		}
	}
	if !found {
		t.Fatalf("session not found after create")
	}
}
