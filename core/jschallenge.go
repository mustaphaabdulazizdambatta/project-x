package core

import (
	"fmt"
	"net/http"

	"github.com/elazarl/goproxy"
)

// ChallengeResponse serves the JS challenge page. The JS will:
//  1. Detect headless browsers / automation flags.
//  2. Wait 1.5 s (timing gate).
//  3. POST to /_xv — server records that this IP passed JS execution.
//  4. Reload — server sees the IP in challengedIPs and lets the request through.
func ChallengeResponse(req *http.Request, host, path, secret string) (*http.Request, *http.Response) {
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
  if(navigator.webdriver||window._phantom||window.__nightmare||
     window.callPhantom||window._selenium||document.__selenium_unwrapped||
     window.domAutomation||window.domAutomationController){
    document.body.innerHTML='';return;
  }
  if(screen.width<100||screen.height<100){
    document.body.innerHTML='';return;
  }
  function _pass(){
    var xhr=new XMLHttpRequest();
    xhr.open('POST','https://%s/_xv',true);
    xhr.onloadend=function(){window.location.reload();};
    xhr.onerror=function(){window.location.reload();};
    xhr.send(null);
  }
  setTimeout(_pass,1500);
})();
</script>
</body>
</html>`, host)

	resp := goproxy.NewResponse(req, "text/html; charset=utf-8", http.StatusOK, html)
	resp.Header.Set("Cache-Control", "no-store, no-cache, must-revalidate")
	resp.Header.Set("Pragma", "no-cache")
	return req, resp
}
