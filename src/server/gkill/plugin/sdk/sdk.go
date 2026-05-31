package sdk

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"
)

// pluginRequest はgkillからプラグインへのリクエスト。
type pluginRequest struct {
	ID       string            `json:"id"`
	Command  string            `json:"command"`
	Query    *pluginQuery      `json:"query,omitempty"`
	KyouID   string            `json:"kyou_id,omitempty"`
	FormData map[string]string `json:"form_data,omitempty"`
}

// pluginQuery はfind_kyousコマンドの検索条件。
type pluginQuery struct {
	Words             []string   `json:"words"`
	NotWords          []string   `json:"not_words"`
	WordsAnd          bool       `json:"words_and"`
	Tags              []string   `json:"tags"`
	NotTags           []string   `json:"not_tags"`
	TagsAnd           bool       `json:"tags_and"`
	CalendarStartDate *time.Time `json:"calendar_start_date,omitempty"`
	CalendarEndDate   *time.Time `json:"calendar_end_date,omitempty"`
	IsDeleted         bool       `json:"is_deleted"`
	OnlyLatestData    bool       `json:"only_latest_data"`
	Limit             int        `json:"limit"`
}

// pluginResponse はプラグインからgkillへのレスポンス。
type pluginResponse struct {
	ID      string   `json:"id"`
	Kyous   []Kyou   `json:"kyous,omitempty"`
	Kyou    *Kyou    `json:"kyou,omitempty"`
	RepName string   `json:"rep_name,omitempty"`
	HTML    string   `json:"html,omitempty"`
	Pong    bool     `json:"pong,omitempty"`
	Errors  []string `json:"errors,omitempty"`
}

// Run はプラグインのメインループを起動する。
// プラグイン作者はHandlerを実装してこの関数を呼び出すだけでよい。
func Run(h Handler) {
	pluginDir := flag.String("gkill-plugin-dir", "", "gkillが管理するetcディレクトリのパス")
	userID := flag.String("gkill-user-id", "", "ユーザID")
	protocolVersion := flag.String("gkill-protocol-version", "1", "プロトコルバージョン")
	flag.Parse()

	// プロトコルバージョン確認（将来の互換性のため）
	if *protocolVersion != "1" {
		fmt.Fprintf(os.Stderr, "unsupported protocol version: %s\n", *protocolVersion)
		os.Exit(1)
	}

	cfg, err := LoadConfig(*pluginDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error at load config: %v\n", err)
		// 設定読み込み失敗は致命的ではないので続行
		cfg = Config{}
	}

	encoder := json.NewEncoder(os.Stdout)
	scanner := bufio.NewScanner(os.Stdin)
	// 大きなレスポンスに備えてバッファを拡張
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, len(buf))

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req pluginRequest
		if err := json.Unmarshal(line, &req); err != nil {
			writeError(encoder, "", fmt.Sprintf("error at parse request: %v", err))
			continue
		}

		switch req.Command {
		case "ping":
			resp := pluginResponse{ID: req.ID, Pong: true}
			_ = encoder.Encode(resp)

		case "close":
			os.Exit(0)

		case "get_rep_name":
			resp := pluginResponse{ID: req.ID, RepName: h.RepName}
			_ = encoder.Encode(resp)

		case "find_kyous":
			if h.FindKyous == nil {
				writeError(encoder, req.ID, "find_kyous not implemented")
				continue
			}
			q := pluginQueryToQuery(req.Query)
			kyous, err := h.FindKyous(newCtx(*userID), q, cfg)
			if err != nil {
				writeError(encoder, req.ID, err.Error())
				continue
			}
			resp := pluginResponse{ID: req.ID, Kyous: kyous}
			_ = encoder.Encode(resp)

		case "get_kyou":
			if h.GetKyou != nil {
				kyou, err := h.GetKyou(newCtx(*userID), req.KyouID, cfg)
				if err != nil {
					writeError(encoder, req.ID, err.Error())
					continue
				}
				resp := pluginResponse{ID: req.ID, Kyou: kyou}
				_ = encoder.Encode(resp)
			} else if h.FindKyous != nil {
				// フォールバック: FindKyousで代替
				kyous, err := h.FindKyous(newCtx(*userID), Query{}, cfg)
				if err != nil {
					writeError(encoder, req.ID, err.Error())
					continue
				}
				var found *Kyou
				for i := range kyous {
					if kyous[i].ID == req.KyouID {
						found = &kyous[i]
						break
					}
				}
				resp := pluginResponse{ID: req.ID, Kyou: found}
				_ = encoder.Encode(resp)
			} else {
				writeError(encoder, req.ID, "get_kyou not implemented")
			}

		case "get_content_html":
			if h.GetContentHTML != nil {
				html, err := h.GetContentHTML(newCtx(*userID), req.KyouID, cfg)
				if err != nil {
					writeError(encoder, req.ID, err.Error())
					continue
				}
				resp := pluginResponse{ID: req.ID, HTML: html}
				_ = encoder.Encode(resp)
			} else {
				// デフォルト: シンプルなHTML
				html := fmt.Sprintf(`<html><body><p>%s</p></body></html>`, req.KyouID)
				resp := pluginResponse{ID: req.ID, HTML: html}
				_ = encoder.Encode(resp)
			}

		case "get_config_html":
			if h.GetConfigHTML != nil {
				html, err := h.GetConfigHTML(newCtx(*userID), cfg)
				if err != nil {
					writeError(encoder, req.ID, err.Error())
					continue
				}
				resp := pluginResponse{ID: req.ID, HTML: html}
				_ = encoder.Encode(resp)
			} else {
				// デフォルト: 空フォーム
				resp := pluginResponse{ID: req.ID, HTML: `<html><body><p>設定なし</p></body></html>`}
				_ = encoder.Encode(resp)
			}

		case "post_config":
			if h.PostConfig != nil {
				newCfg, err := h.PostConfig(newCtx(*userID), req.FormData, cfg)
				if err != nil {
					writeError(encoder, req.ID, err.Error())
					continue
				}
				if err := SaveConfig(*pluginDir, newCfg); err != nil {
					writeError(encoder, req.ID, err.Error())
					continue
				}
				cfg = newCfg
			} else {
				// デフォルト: formデータをそのままconfigに保存
				for k, v := range req.FormData {
					cfg[k] = v
				}
				if err := SaveConfig(*pluginDir, cfg); err != nil {
					writeError(encoder, req.ID, err.Error())
					continue
				}
			}
			resp := pluginResponse{ID: req.ID}
			_ = encoder.Encode(resp)

		default:
			writeError(encoder, req.ID, fmt.Sprintf("unknown command: %s", req.Command))
		}
	}
}

func newCtx(userID string) context.Context {
	return context.WithValue(context.Background(), ctxKeyUserID{}, userID)
}

type ctxKeyUserID struct{}

func writeError(enc *json.Encoder, id, msg string) {
	_ = enc.Encode(pluginResponse{ID: id, Errors: []string{msg}})
}

func pluginQueryToQuery(pq *pluginQuery) Query {
	if pq == nil {
		return Query{}
	}
	return Query{
		Words:             pq.Words,
		NotWords:          pq.NotWords,
		WordsAnd:          pq.WordsAnd,
		Tags:              pq.Tags,
		NotTags:           pq.NotTags,
		TagsAnd:           pq.TagsAnd,
		CalendarStartDate: pq.CalendarStartDate,
		CalendarEndDate:   pq.CalendarEndDate,
		IsDeleted:         pq.IsDeleted,
		OnlyLatestData:    pq.OnlyLatestData,
		Limit:             pq.Limit,
	}
}
