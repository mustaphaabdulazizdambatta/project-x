package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"

	"github.com/x-tymus/x-tymus/core"
)

func main() {
	// create a temporary config directory
	tmp := "/tmp/x-tymus_test"
	os.RemoveAll(tmp)
	os.MkdirAll(tmp, 0700)

	cfg, err := core.NewConfig(tmp, "")
	if err != nil {
		fmt.Printf("NewConfig error: %v\n", err)
		return
	}
	cfg.SetStealthAIEnabled(true)

	hs, err := core.NewHttpServer()
	if err != nil {
		fmt.Printf("NewHttpServer error: %v\n", err)
		return
	}
	hs.Cfg = cfg

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
			fmt.Printf("%s: request error: %v\n", tc.name, err)
			continue
		}
		loc := resp.Header.Get("Location")
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("%s: status=%d Location=%s body=%s\n", tc.name, resp.StatusCode, loc, string(body))
	}
}
