package main

import (
	"context"
	"os"
	"strings"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

const repName = "ChatGPT"
const dataType = "chatgpt_conversation"

func extractpluginDir(args []string) string {
	for i, a := range args {
		if a == "--gkill-plugin-dir" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func main() {
	pluginDir := extractpluginDir(os.Args)

	sdk.Run(sdk.Handler{
		RepName: repName,

		FindKyous: func(ctx context.Context, q sdk.Query, cfg sdk.Config) ([]sdk.Kyou, error) {
			msgs, err := globalCache.GetMessages(pluginDir)
			if err != nil {
				return []sdk.Kyou{}, nil
			}

			var kyous []sdk.Kyou
			for _, msg := range msgs {
				relatedTime := unixToTimeFromCache(msg.RelatedTimeUnix)

				// カレンダーフィルタ
				if q.CalendarStartDate != nil && relatedTime.Before(*q.CalendarStartDate) {
					continue
				}
				if q.CalendarEndDate != nil && relatedTime.After(*q.CalendarEndDate) {
					continue
				}

				// ワードフィルタ（メッセージ単位）
				if !matchWordsText(msg.Text, q) {
					continue
				}

				createTime := unixToTimeFromCache(msg.CreateTimeUnix)

				k := sdk.Kyou{
					ID:          msg.MsgID,
					RepName:     repName,
					DataType:    dataType,
					RelatedTime: relatedTime,
					CreateTime:  createTime,
					UpdateTime:  createTime,
					CreateApp:   "gkill_plugin_chatgpt",
					UpdateApp:   "gkill_plugin_chatgpt",
				}
				kyous = append(kyous, k)
			}

			if q.Limit > 0 && len(kyous) > q.Limit {
				kyous = kyous[:q.Limit]
			}

			return kyous, nil
		},

		GetContentHTML: func(ctx context.Context, kyouID string, cfg sdk.Config) (string, error) {
			convTitle, msg, err := globalCache.GetMsgByID(pluginDir, kyouID)
			if err != nil {
				return "<html><body><p>メッセージが見つかりません</p></body></html>", nil
			}
			return renderSingleMsgHTML(convTitle, msg), nil
		},

		GetConfigHTML: func(ctx context.Context, cfg sdk.Config) (string, error) {
			return renderConfigHTML(pluginDir), nil
		},
	})
}

// matchWordsText はワード検索条件にメッセージテキストが合致するかチェックする。
func matchWordsText(text string, q sdk.Query) bool {
	if len(q.Words) == 0 && len(q.NotWords) == 0 {
		return true
	}

	target := strings.ToLower(text)

	if len(q.Words) > 0 {
		if q.WordsAnd {
			for _, w := range q.Words {
				if !strings.Contains(target, strings.ToLower(w)) {
					return false
				}
			}
		} else {
			matched := false
			for _, w := range q.Words {
				if strings.Contains(target, strings.ToLower(w)) {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}
	}

	for _, w := range q.NotWords {
		if strings.Contains(target, strings.ToLower(w)) {
			return false
		}
	}
	return true
}

// renderSingleMsgHTML は1メッセージのみのHTMLを生成する。
// テーマはpostMessage経由で親ページから受け取り動的に切り替える。
func renderSingleMsgHTML(convTitle string, msg cachedMessage) string {
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html><head><meta charset="utf-8">
<style>
:root {
  --bg: #ffffff;
  --text: #333333;
  --msg-user-bg: #dbeafe;
  --msg-assistant-bg: #f3f4f6;
  --sender-color: #6b7280;
  --ts-color: #9ca3af;
  --title-color: #9ca3af;
  --scrollbar-thumb: #2672ed;
  --scrollbar-track: #e5e7eb;
}
[data-theme="dark"] {
  --bg: #212121;
  --text: #e0e0e0;
  --msg-user-bg: #1a3557;
  --msg-assistant-bg: #2d2d2d;
  --sender-color: #aaaaaa;
  --ts-color: #888888;
  --title-color: #888888;
  --scrollbar-thumb: #2672ed;
  --scrollbar-track: #424242;
}
html, body { height: auto; margin: 0; overflow: visible; }
body { font-family: sans-serif; padding: 12px; font-size: 14px;
  background: var(--bg); color: var(--text); }
.conv-title { font-size: 0.85em; color: var(--title-color); margin-bottom: 8px; }
.msg { padding: 8px 12px; border-radius: 8px; white-space: pre-wrap;
  word-break: break-word; line-height: 1.5; }
.user { background: var(--msg-user-bg); }
.assistant { background: var(--msg-assistant-bg); }
.sender { font-size: 0.75em; color: var(--sender-color); margin-bottom: 4px; }
.ts { font-size: 0.7em; color: var(--ts-color); margin-top: 4px; }
::-webkit-scrollbar { width: 6px; height: 6px; }
::-webkit-scrollbar-track { background: var(--scrollbar-track); }
::-webkit-scrollbar-thumb { background: var(--scrollbar-thumb); border-radius: 3px; }
</style>
<script>
(function() {
  function notifySize() {
    window.parent.postMessage({
      gkill_iframe_size: {
        width: document.documentElement.scrollWidth,
        height: document.documentElement.scrollHeight
      }
    }, '*');
  }
  window.addEventListener('message', function(e) {
    if (e.data && e.data.gkill_theme) {
      document.documentElement.setAttribute('data-theme', e.data.gkill_theme);
      setTimeout(notifySize, 10);
    }
  });
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', notifySize);
  } else {
    notifySize();
  }
  if (window.ResizeObserver) {
    new ResizeObserver(notifySize).observe(document.documentElement);
  }
})();
</script>
</head><body>`)

	if convTitle != "" {
		sb.WriteString(`<div class="conv-title">`)
		sb.WriteString(htmlEscape(convTitle))
		sb.WriteString(`</div>`)
	}

	text := strings.TrimSpace(msg.Text)
	class := "assistant"
	senderLabel := "ChatGPT"
	if msg.Sender == "user" {
		class = "user"
		senderLabel = "あなた"
	}
	ts := ""
	if msg.RelatedTimeUnix != 0 {
		ts = unixToTimeFromCache(msg.RelatedTimeUnix).Format("2006-01-02 15:04")
	}
	sb.WriteString(`<div class="msg `)
	sb.WriteString(class)
	sb.WriteString(`"><div class="sender">`)
	sb.WriteString(htmlEscape(senderLabel))
	sb.WriteString(`</div>`)
	sb.WriteString(htmlEscape(text))
	sb.WriteString(`<div class="ts">`)
	sb.WriteString(ts)
	sb.WriteString(`</div></div>`)

	sb.WriteString(`</body></html>`)
	return sb.String()
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&#34;")
	return s
}
