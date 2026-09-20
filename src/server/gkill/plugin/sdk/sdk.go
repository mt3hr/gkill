package sdk

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_log"
)

// pluginRequest はgkillからプラグインへのリクエスト。
type pluginRequest struct {
	ID          string             `json:"id"`
	Command     string             `json:"command"`
	Query       *pluginQuery       `json:"query,omitempty"`
	KyouID      string             `json:"kyou_id,omitempty"`
	FormData    map[string]string  `json:"form_data,omitempty"`
	GPSLogQuery *pluginGPSLogQuery `json:"gps_log_query,omitempty"`
}

// pluginGPSLogQuery はget_gps_logsコマンドの取得条件。
type pluginGPSLogQuery struct {
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Offset    int        `json:"offset"`
	Limit     int        `json:"limit"`
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
	ID      string `json:"id"`
	Kyous   []Kyou `json:"kyous,omitempty"`
	Kyou    *Kyou  `json:"kyou,omitempty"`
	RepName string `json:"rep_name,omitempty"`
	// RepNames はポインタで持つ。nil なら欄ごと出さず（gkill は「未対応」と読む）、
	// 空スライスへのポインタなら "rep_names": [] を出す（gkill は「いまは0個」と読む）。
	RepNames       *[]string `json:"rep_names,omitempty"`
	HTML           string    `json:"html,omitempty"`
	Pong           bool      `json:"pong,omitempty"`
	GPSLogs        []GPSLog  `json:"gps_logs,omitempty"`
	HasMoreGPSLogs bool      `json:"has_more_gps_logs,omitempty"`
	Errors         []string  `json:"errors,omitempty"`
}

// pluginGPSLogQueryToQuery はプロトコル上の取得条件を公開型に変換する。
// nilは「期間指定なし・プラグイン既定の件数」を意味する。
func pluginGPSLogQueryToQuery(q *pluginGPSLogQuery) GPSLogQuery {
	if q == nil {
		return GPSLogQuery{}
	}
	return GPSLogQuery{
		StartTime: q.StartTime,
		EndTime:   q.EndTime,
		Offset:    q.Offset,
		Limit:     q.Limit,
	}
}

// 単独モード（--gkill-build-cache）が stdout に書く結果行。
// gkill 側（main/common/generate_plugin_cache.go）はこの2語のどちらかが stdout の全文と
// 一致するときだけ成功とみなす。それ以外（空・別の文字列）は「対応していない古いバイナリか、
// stdout を汚した」として失敗にする。綴りを変えるなら gkill 側も一緒に変えること。
const (
	// BuildCacheResultBuilt は BuildCache が成功したことを表す。
	BuildCacheResultBuilt = "built"
	// BuildCacheResultNoCache は Handler.BuildCache が nil（キャッシュを持たないプラグイン）だったことを表す。
	BuildCacheResultNoCache = "no_cache"
)

// Run はプラグインのメインループを起動する。
// Run はプラグインのメインループを起動する。
// プラグイン作者はHandlerを実装してこの関数を呼び出すだけでよい。
//
// `--gkill-build-cache` 付きで起動されたときは stdio ループに入らず、
// Handler.BuildCache を同期で1回実行して終了する（runBuildCache）。
//
// flag の解析直後に $GKILL_HOME/logs/gkill_plugin_<name>*.log を開く（initLogging）。
// 以後の LogWarn / LogError は stderr とそのファイルの両方へ、LogInfo / LogDebug はファイルへ出る。
// os.Exit の前には必ず closeLogging を通す（defer は os.Exit で走らない）。
func Run(h Handler) {
	pluginDir := flag.String("gkill-plugin-dir", "", "gkillが管理するetcディレクトリのパス")
	userID := flag.String("gkill-user-id", "", "ユーザID")
	protocolVersion := flag.String("gkill-protocol-version", "1", "プロトコルバージョン")
	buildCache := flag.Bool("gkill-build-cache", false, "キャッシュを同期で構築して終了する（stdio ループには入らない）")
	flag.Parse()

	initLogging(*pluginDir, *userID)

	// プロトコルバージョン確認（将来の互換性のため）
	if *protocolVersion != "1" {
		LogError("unsupported protocol version: %s", *protocolVersion)
		closeLogging()
		os.Exit(1)
	}

	// config.jsonが無ければ雛形を作ってから読む。手で編集できるようにするため。
	cfg, err := EnsureConfig(*pluginDir, h.DefaultConfig)
	if err != nil {
		LogWarn("error at load config, continuing with an empty config: %v", err)
		// 設定読み込み失敗は致命的ではないので続行
		cfg = Config{}
	}

	if *buildCache {
		// signal.NotifyContext は張らない。張ると SIGINT の既定動作（即終了）が抑止され、
		// ctx を見ない構築関数が Ctrl+C の後も走り続ける。
		ok := runBuildCache(newCtx(*userID), h, cfg, os.Stdout)
		closeLogging()
		if !ok {
			os.Exit(1)
		}
		return
	}

	closed := runLoop(h, cfg, *pluginDir, *userID, os.Stdin, os.Stdout)
	closeLogging()
	if closed {
		os.Exit(0)
	}
}

// runBuildCache は Handler.BuildCache を同期実行し、結果行を out に1行だけ書く。
// 成功（built / no_cache）なら true。失敗なら LogError（stderr）して false。
// 出力を引数にしているのはテストのため。
func runBuildCache(ctx context.Context, h Handler, cfg Config, out io.Writer) bool {
	if h.BuildCache == nil {
		fmt.Fprintln(out, BuildCacheResultNoCache)
		logEvent(gkill_log.Info, "build cache", "result", BuildCacheResultNoCache)
		return true
	}
	started := time.Now()
	if err := h.BuildCache(ctx, cfg); err != nil {
		LogError("build cache: %v", err)
		return false
	}
	fmt.Fprintln(out, BuildCacheResultBuilt)
	logEvent(gkill_log.Info, "build cache", "result", BuildCacheResultBuilt, "duration_ms", time.Since(started).Milliseconds())
	return true
}

// runLoop は改行区切りJSONのメッセージループ本体。
// Run から stdin/stdout を渡して使う。入出力を引数にしているのはテストのため。
// close コマンドを受け取って終了した場合に true を返す（Run はこのとき os.Exit(0) する）。
//
// 1コマンドにつき Access で1行（command / id / duration_ms / count / error）を残す。
// 「プロセスが殺され続ける」（期限超過）の調査で、どのコマンドが何ミリ秒かかったかは
// ここにしか出ない。
func runLoop(h Handler, cfg Config, pluginDir, userID string, in io.Reader, out io.Writer) bool {
	encoder := json.NewEncoder(out)
	scanner := bufio.NewScanner(in)
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
			logEvent(gkill_log.Warn, "error at parse plugin request", "error", fmt.Sprintf("%q", err))
			continue
		}

		started := time.Now()
		result := dispatch(h, &cfg, pluginDir, userID, req, encoder)
		logEvent(gkill_log.Access, "plugin command",
			"command", req.Command,
			"id", req.ID,
			"duration_ms", time.Since(started).Milliseconds(),
			"count", result.count,
			"error", result.err)
		if result.closed {
			logEvent(gkill_log.Info, "plugin stop", "reason", "close command")
			return true
		}
	}
	// stdinが閉じられてループを抜けた場合。closeコマンドではないので false。
	if err := scanner.Err(); err != nil {
		logEvent(gkill_log.Warn, "plugin stop", "reason", "stdin read error", "error", fmt.Sprintf("%q", err))
	} else {
		logEvent(gkill_log.Info, "plugin stop", "reason", "stdin eof")
	}
	return false
}

// dispatchResult は1コマンドの結果の要約（Access ログ用）。
type dispatchResult struct {
	// count は返した件数（find_kyous の Kyou 数、get_gps_logs のログ数、rep 名の数）。
	count int
	// err は gkill へ errors として返した文言。成功なら空。
	err string
	// closed は close コマンドを受けたとき true。
	closed bool
}

// dispatch は1コマンドを処理して応答を書く。cfg は post_config で差し替わるのでポインタで受ける。
func dispatch(h Handler, cfg *Config, pluginDir, userID string, req pluginRequest, encoder *json.Encoder) dispatchResult {
	fail := func(msg string) dispatchResult {
		writeError(encoder, req.ID, msg)
		return dispatchResult{err: msg}
	}

	switch req.Command {
	case "ping":
		resp := pluginResponse{ID: req.ID, Pong: true}
		_ = encoder.Encode(resp)
		return dispatchResult{}

	case "close":
		return dispatchResult{closed: true}

	case "get_rep_name":
		resp := pluginResponse{ID: req.ID, RepName: h.RepName}
		count := 1
		if h.RepNames != nil {
			repNames, err := h.RepNames(newCtx(userID), *cfg)
			if err != nil {
				return fail(err.Error())
			}
			// nil を返されても「いまは0個」として [] を出す。
			// 欄を落とすと gkill が manifest の rep_name にフォールバックしてしまい、
			// 実装しているのに「未対応」と読まれる。
			if repNames == nil {
				repNames = []string{}
			}
			resp.RepNames = &repNames
			count = len(repNames)
		}
		_ = encoder.Encode(resp)
		return dispatchResult{count: count}

	case "find_kyous":
		if h.FindKyous == nil {
			return fail("find_kyous not implemented")
		}
		q := pluginQueryToQuery(req.Query)
		kyous, err := h.FindKyous(newCtx(userID), q, *cfg)
		if err != nil {
			return fail(err.Error())
		}
		resp := pluginResponse{ID: req.ID, Kyous: kyous}
		_ = encoder.Encode(resp)
		return dispatchResult{count: len(kyous)}

	case "get_kyou":
		if h.GetKyou != nil {
			kyou, err := h.GetKyou(newCtx(userID), req.KyouID, *cfg)
			if err != nil {
				return fail(err.Error())
			}
			resp := pluginResponse{ID: req.ID, Kyou: kyou}
			_ = encoder.Encode(resp)
			if kyou == nil {
				return dispatchResult{}
			}
			return dispatchResult{count: 1}
		}
		if h.FindKyous != nil {
			// フォールバック: FindKyousで代替
			kyous, err := h.FindKyous(newCtx(userID), Query{}, *cfg)
			if err != nil {
				return fail(err.Error())
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
			if found == nil {
				return dispatchResult{}
			}
			return dispatchResult{count: 1}
		}
		return fail("get_kyou not implemented")

	case "get_content_html":
		if h.GetContentHTML != nil {
			html, err := h.GetContentHTML(newCtx(userID), req.KyouID, *cfg)
			if err != nil {
				return fail(err.Error())
			}
			resp := pluginResponse{ID: req.ID, HTML: html}
			_ = encoder.Encode(resp)
			return dispatchResult{}
		}
		// デフォルト: シンプルなHTML
		html := fmt.Sprintf(`<html><body><p>%s</p></body></html>`, req.KyouID)
		resp := pluginResponse{ID: req.ID, HTML: html}
		_ = encoder.Encode(resp)
		return dispatchResult{}

	case "get_gps_logs":
		if h.GetGPSLogs == nil {
			return fail("get_gps_logs not implemented")
		}
		page, err := h.GetGPSLogs(newCtx(userID), pluginGPSLogQueryToQuery(req.GPSLogQuery), *cfg)
		if err != nil {
			return fail(err.Error())
		}
		resp := pluginResponse{ID: req.ID, GPSLogs: page.GPSLogs, HasMoreGPSLogs: page.HasMore}
		_ = encoder.Encode(resp)
		return dispatchResult{count: len(page.GPSLogs)}

	case "get_config_html":
		if h.GetConfigHTML != nil {
			html, err := h.GetConfigHTML(newCtx(userID), *cfg)
			if err != nil {
				return fail(err.Error())
			}
			resp := pluginResponse{ID: req.ID, HTML: html}
			_ = encoder.Encode(resp)
			return dispatchResult{}
		}
		// デフォルト: 空フォーム
		resp := pluginResponse{ID: req.ID, HTML: `<html><body><p>設定なし</p></body></html>`}
		_ = encoder.Encode(resp)
		return dispatchResult{}

	case "post_config":
		if h.PostConfig != nil {
			newCfg, err := h.PostConfig(newCtx(userID), req.FormData, *cfg)
			if err != nil {
				return fail(err.Error())
			}
			if err := SaveConfig(pluginDir, newCfg); err != nil {
				return fail(err.Error())
			}
			*cfg = newCfg
		} else {
			// デフォルト: formデータをそのままconfigに保存
			for k, v := range req.FormData {
				(*cfg)[k] = v
			}
			if err := SaveConfig(pluginDir, *cfg); err != nil {
				return fail(err.Error())
			}
		}
		resp := pluginResponse{ID: req.ID}
		_ = encoder.Encode(resp)
		return dispatchResult{}

	default:
		return fail(fmt.Sprintf("unknown command: %s", req.Command))
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
