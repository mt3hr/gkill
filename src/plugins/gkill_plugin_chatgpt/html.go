package main

import (
	"fmt"
	"html"
)

// renderConfigHTML は設定画面HTMLを返す。
func renderConfigHTML(pluginDir string) string {
	convs, err := loadConversations(pluginDir)
	if err != nil {
		return fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="utf-8">
<style>
body{font-family:sans-serif;margin:16px;}
.info{background:#f0f4ff;border-left:4px solid #4466cc;padding:12px;border-radius:4px;margin:12px 0;}
code{background:#eee;padding:2px 6px;border-radius:3px;font-size:0.9em;}
</style></head><body>
<h2>ChatGPT チャット履歴プラグイン</h2>
<div class="info">
<p><strong>セットアップ方法</strong></p>
<ol>
<li>ChatGPT にログイン → 右上のアイコン → <strong>Settings</strong></li>
<li>「Data controls」→「<strong>Export data</strong>」→「Export」</li>
<li>届いたメールのリンクからZIPをダウンロードして解凍</li>
<li><code>conversations-000.json</code>、<code>conversations-001.json</code> 等 (<code>conversations-*.json</code>) または <code>conversations.json</code> をこのプラグインの <code>etc/</code> フォルダに配置する</li>
</ol>
<p>配置先フォルダ: <code>%s</code></p>
</div>
<p style="color:#888">現在: ファイルが見つかりません</p>
</body></html>`, html.EscapeString(pluginDir))
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html><head><meta charset="utf-8">
<style>
body{font-family:sans-serif;margin:16px;}
.ok{background:#f0fff4;border-left:4px solid #44aa66;padding:12px;border-radius:4px;margin:12px 0;}
code{background:#eee;padding:2px 6px;border-radius:3px;font-size:0.9em;}
</style></head><body>
<h2>ChatGPT チャット履歴プラグイン</h2>
<div class="ok">
<p>✓ <strong>%d 件</strong>の会話が読み込まれています</p>
<p>データを更新するには、ChatGPT から再エクスポートして <code>conversations.json</code> を置き換えてください。</p>
</div>
</body></html>`, len(convs))
}
