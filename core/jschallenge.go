package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/elazarl/goproxy"
)

const (
	challengeCookieName = "_xv"
	challengeTTL        = 3600 // 1 hour
)

// signChallenge returns HMAC-SHA256(ts+":"+path, secret) as hex.
func signChallenge(ts int64, path, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%d:%s", ts, path)))
	return hex.EncodeToString(mac.Sum(nil))
}

// ValidChallengeCookie returns true if the request carries a valid, unexpired
// challenge cookie that was issued for this path.
func ValidChallengeCookie(req *http.Request, path, secret string) bool {
	c, err := req.Cookie(challengeCookieName)
	if err != nil {
		return false
	}
	parts := strings.SplitN(c.Value, ".", 2)
	if len(parts) != 2 {
		return false
	}
	ts, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return false
	}
	if time.Now().Unix()-ts > challengeTTL {
		return false
	}
	expected := signChallenge(ts, path, secret)
	return hmac.Equal([]byte(parts[1]), []byte(expected))
}

// ChallengeResponse serves the JS challenge page. The JS will:
//  1. Detect headless browsers / automation flags.
//  2. Wait 1.5 s (timing gate).
//  3. Set the signed cookie.
//  4. Redirect back to the original URL.
func ChallengeResponse(req *http.Request, path, secret string) (*http.Request, *http.Response) {
	ts := time.Now().Unix()
	sig := signChallenge(ts, path, secret)
	token := fmt.Sprintf("%d.%s", ts, sig)
	cookieDomain := req.Host
	if strings.Contains(cookieDomain, ":") {
		cookieDomain = strings.Split(cookieDomain, ":")[0]
	}

	host := req.Host
	if host == "" {
		host = req.URL.Host
	}
	redirectURL := "https://" + host + req.URL.RequestURI()

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Please wait...</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{background:#f3f2f1;display:flex;align-items:center;justify-content:center;min-height:100vh;font-family:'Segoe UI',sans-serif}
.wrap{text-align:center;padding:32px}
.spinner{width:40px;height:40px;border:3px solid #e1dfdd;border-top-color:#0078d4;border-radius:50%;animation:spin .8s linear infinite;margin:0 auto 16px}
@keyframes spin{to{transform:rotate(360deg)}}
p{color:#605e5c;font-size:15px}
</style>
</head>
<body>
<div class="wrap">
<div class="spinner"></div>
<p>Verifying your browser, please wait...</p>
</div>
<script>
(function(){
  // Block headless / automation
  if(navigator.webdriver||window._phantom||window.__nightmare||
     window.callPhantom||window._selenium||document.__selenium_unwrapped||
     window.domAutomation||window.domAutomationController){
    document.body.innerHTML='';return;
  }
  // Block zero-size screens (headless default)
  if(screen.width<100||screen.height<100){
    document.body.innerHTML='';return;
  }
  var _tok=%q;
  var _url=%q;
  var _dom=%q;
  function _pass(){
    // Set signed cookie then redirect
    var exp=new Date(Date.now()+3600000).toUTCString();
    document.cookie='%s='+_tok+';path=/;domain='+_dom+';expires='+exp;
    window.location.replace(_url);
  }
  // 1.5 s timing gate — bots don't wait
  setTimeout(_pass, 1500);
})();
</script>
</body>
</html>`, token, redirectURL, cookieDomain, challengeCookieName)

	resp := goproxy.NewResponse(req, "text/html; charset=utf-8", http.StatusOK, html)
	resp.Header.Set("Cache-Control", "no-store, no-cache, must-revalidate")
	resp.Header.Set("Pragma", "no-cache")
	return req, resp
}
