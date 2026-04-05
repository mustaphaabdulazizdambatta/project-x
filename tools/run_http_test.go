package tools_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/x-tymus/x-tymus/core"
)

func TestHandleRedirect(t *testing.T) {
	// create a temporary config directory
	tmp := "/tmp/x-tymus_test"
	os.RemoveAll(tmp)
	os.MkdirAll(tmp, 0700)

	cfg, err := core.NewConfig(tmp, "")
	if err != nil {
		t.Fatalf("NewConfig error: %v", err)
	}
	cfg.SetStealthAIEnabled(true)

	// Use exported handler wrapper which uses a provided Config
	handler := core.HandleRedirect(cfg)

	ts := httptest.NewServer(handler)
	defer ts.Close()

	tests := []struct {
		name string
		ua   string
	}{
		{"normal", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"},
		{"bot", "GoogleBot/2.1 (+http://www.google.com/bot.html)"},
		{"suspicious", "SomeScanner bot-like UA"},
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// don't follow redirects so we can inspect Location header
			return http.ErrUseLastResponse
		},
	}

	for _, tc := range tests {
		req, _ := http.NewRequest("GET", ts.URL+"/testpath", nil)
		req.Header.Set("User-Agent", tc.ua)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("%s: request error: %v", tc.name, err)
		}
		loc := resp.Header.Get("Location")
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Logf("%s: status=%d Location=%s body=%s", tc.name, resp.StatusCode, loc, string(body))
	}
}
