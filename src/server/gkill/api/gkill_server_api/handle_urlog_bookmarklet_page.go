package gkill_server_api

import (
	"fmt"
	"net/http"
)

func (g *GkillServerAPI) HandleURLogBookmarkletPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, urlogBookmarkletPageHTML)
}

const urlogBookmarkletPageHTML = `<!DOCTYPE html>
<html lang="ja">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>URLog</title>
<style>
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;background:#f5f5f5}
#msg{text-align:center;padding:20px 28px;background:#fff;border-radius:8px;box-shadow:0 2px 8px rgba(0,0,0,.18);font-size:15px;line-height:1.6}
.saving{color:#1976d2}
.ok{color:#388e3c}
.ng{color:#d32f2f}
</style>
</head>
<body>
<div id="msg" class="saving">保存中...</div>
<script>
(function(){
  var sp=new URLSearchParams(location.search);
  fetch('/api/urlog_bookmarklet',{
    method:'POST',
    headers:{'Content-Type':'application/json'},
    body:JSON.stringify({
      url:sp.get('url')||'',
      title:sp.get('title')||'',
      time:sp.get('time')||new Date().toISOString(),
      favicon_url:sp.get('favicon_url')||'',
      description:sp.get('description')||'',
      image_url:sp.get('image_url')||'',
      session_id:sp.get('session_id')||''
    })
  }).then(function(res){
    var m=document.getElementById('msg');
    if(res.ok){
      m.className='ok';
      m.textContent='ブックマーク保存しました';
      setTimeout(function(){window.close();},1500);
    }else{
      m.className='ng';
      m.textContent='保存に失敗しました (HTTP '+res.status+')';
    }
  }).catch(function(e){
    var m=document.getElementById('msg');
    m.className='ng';
    m.textContent='保存に失敗しました: '+e;
  });
})();
</script>
</body>
</html>`
