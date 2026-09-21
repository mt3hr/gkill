package mcp

// プラグイン関連の MCP ツール定義とハンドラ、および
// gkill_get_kyous のレスポンスにプラグイン Kyou の本文を埋め込む処理（旧 plugin-tools.mjs）。
//
// read / write / readwrite の3サーバから共有する。サーバごとに API 呼び出しの
// 文脈が違うため、呼び出し口は call(pathname, body) 形式のコールバックで受け取る。
//
// 同一プラグインへ並列に投げない理由（stdio が1本しかない）:
// documents/adr/0602-mcp-inline-plugin-content.md

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

const GetPluginListEndpoint = "/api/get_plugin_list"
const GetPluginContentHTMLEndpoint = "/api/get_plugin_content_html"

var PluginToolNames = []string{"gkill_get_plugin_list"}

// maxPluginContentErrorLength は content_error に載せるメッセージの上限。
// スタックトレース等でレスポンスが膨らむのを防ぐ。
const maxPluginContentErrorLength = 200

// PluginTools はプラグインツール 1 本。
var PluginTools = []*jsonobj.Object{
	tool(
		"gkill_get_plugin_list",
		"List the gkill plugins installed for the current user. Plugins are external programs that feed their own "+
			"data into gkill — for example Claude Code / Claude.ai / ChatGPT conversation logs, Fitbit daily metrics, "+
			"Google location history. "+
			"IMPORTANT — plugins do not all play the same role, and emits_kyou tells you which one you are looking at. "+
			"When emits_kyou is true the plugin supplies kyou: filter gkill_get_kyous with query.reps using an entry of "+
			"rep_names[] (ALWAYS present: the rep names its kyou actually carry — a plugin that wraps several repositories, "+
			"such as Git repositories archived as zip, lists each repository there, and rep_names is [] until its index is "+
			"built) — rep_name itself is the plugin's manifest label and is NOT a query.reps value unless it also appears "+
			"in rep_names — or "+
			"the top-level data_types (its data_type), and pass include_plugin_content:true to get their bodies in the "+
			"same call. query.rep_types does NOT work for plugins — they are not in the canonical rep-type vocabulary "+
			"(a plugin whose provides names a typed kind such as kc or git_commit_log is the exception: its records also "+
			"answer to that rep_types value). "+
			"When emits_kyou is false the plugin supplies no kyou at all, and its data_type / rep_name are NOT query "+
			"values: passing them matches nothing. Read that plugin's data through the route matching provides — today "+
			"provides:[\"gpslog\"] means gkill_get_gps_log. "+
			"provides lists what the plugin supplies beyond kyou metadata (kmemo, kc, urlog, nlog, lantana, timeis, mi, "+
			"git_commit_log, tag, text, notification, gpslog); an absent provides means it supplies plain kyou only. "+
			"Response fields: plugins[] with name, version, description, data_type, rep_name (manifest label), rep_names "+
			"(the query.reps values; always present for kyou-emitting plugins), emits_kyou, provides, "+
			"is_alive (responds to a ping), "+
			"process_running (started; read without side effects), has_last_error, typed_index, and gps_index. "+
			"has_last_error is true when the plugin process wrote something to stderr — that is the signal to look at "+
			"when is_alive is true but no records come back. The text itself is deliberately NOT returned: it is the "+
			"plugin's raw stderr and carries the directory layout of the user's own machine. When it is true and the "+
			"text is needed, ask the person running gkill to read it from the server console, and never copy it into "+
			"documents or commit messages. "+
			"typed_index is the KYOU index and is present only for plugins that declare a non-gpslog provides; it carries "+
			"state (\"ok\" / \"failed\" / \"never_built\"), record_count (unique kyou ids), oldest / newest (how far the "+
			"plugin has actually ingested), truncated, built_at, and — when a build failed — has_last_build_error and "+
			"last_attempt_at (the failure text is withheld for the same reason as last_error). Index build failures "+
			"never reach has_last_error: they happen inside gkill (timeouts, a busy plugin, malformed JSON), so "+
			"has_last_build_error is the one to read for those. last_attempt_at matters because "+
			"rebuilds back off after a failure and then produce no error at all. "+
			"gps_index is separate (GPS points are not kyou) and appears for gpslog plugins once their points have been "+
			"loaded: point_count, oldest / newest, fetched_at. It covers ONLY the points this plugin supplied, counted "+
			"before deduplication — gkill_get_gps_log spans every GPS repository (including native ones) and dedupes, "+
			"so the two numbers are expected to differ, sometimes by an order of magnitude. Do not read a stale newest "+
			"here as proof that GPS recording stopped. Its absence means nothing has requested GPS logs yet, not "+
			"that the plugin is broken — call gkill_get_gps_log with count_only:true to size the real total.",
		schema(jsonobj.Obj(
			"locale_name", jsonobj.Obj("type", "string", "description", "Locale, e.g. ja/en."),
		), nil),
	),
}

// PluginDiagnosticsWithheldWarning はプラグインの診断文を AI へ返さないときに足す1行。
//
// last_error はプラグインプロセスの生 stderr、typed_index.last_build_error は起動失敗の
// エラー文で、どちらも「利用者の端末のどこに何が置いてあるか」を含む。gkill 側で
// ユーザー名は伏せているが、AI の文脈へ入れば資料やコミットメッセージへ引き写される経路が
// できてしまう。「何か書かれている」ことだけ has_* で伝えれば、
// 「is_alive=true なのに0件」の診断（指摘 D2）は成立する。
// 経緯: documents/adr/0707-redact-environment-specific-strings.md
const PluginDiagnosticsWithheldWarning = "plugin diagnostics are withheld from this response: last_error (raw plugin stderr) and " +
	"typed_index.last_build_error describe the directory layout of the user's own machine, so only " +
	"has_last_error / has_last_build_error are returned. When one is true and the text is needed, ask the " +
	"person running gkill to read it from the server console — do not copy it into documents or commit messages. " +
	"The most common cause of has_last_error with is_alive:true is that the plugin's configured import path " +
	"matches nothing — a real account ran for 20 months at zero records that way — so ask for that path first."

// withoutPluginDiagnostics はプラグイン1件から診断文を落とし、
// 非空だったときだけ has_last_error / has_last_build_error を立てる。
// キーの並びは Node 版（{last_error, typed_index, ...rest} の分割代入）と同じ:
// 残りのキー → has_last_error → typed_index の順。
func withoutPluginDiagnostics(plugin any) any {
	p, ok := plugin.(*jsonobj.Object)
	if !ok || p == nil {
		return plugin
	}
	lastError, hasLastError := p.Get("last_error")
	typedIndex, hasTypedIndex := p.Get("typed_index")
	stripped := jsonobj.New()
	for _, key := range p.Keys() {
		if key == "last_error" || key == "typed_index" {
			continue
		}
		stripped.Set(key, p.Value(key))
	}
	if hasLastError {
		if s, ok := lastError.(string); ok && s != "" {
			stripped.Set("has_last_error", true)
		}
	}
	if hasTypedIndex && !jsonobj.IsUndefined(typedIndex) {
		if ti, ok := typedIndex.(*jsonobj.Object); ok && ti != nil {
			typedRest := jsonobj.New()
			for _, key := range ti.Keys() {
				if key == "last_build_error" {
					continue
				}
				typedRest.Set(key, ti.Value(key))
			}
			if s, ok := ti.String("last_build_error"); ok && s != "" {
				typedRest.Set("has_last_build_error", true)
			}
			stripped.Set("typed_index", typedRest)
		} else {
			stripped.Set("typed_index", typedIndex)
		}
	}
	return stripped
}

// hasWithheldDiagnostics は診断文を実際に落としたかどうかを返す。
// 落としていないのに警告を出すと常時ノイズになる（ADR-0609 と同じ理由）。
func hasWithheldDiagnostics(plugin any) bool {
	p, ok := plugin.(*jsonobj.Object)
	if !ok || p == nil {
		return false
	}
	if p.Value("has_last_error") == true {
		return true
	}
	if ti, ok := p.Object("typed_index"); ok {
		return ti.Value("has_last_build_error") == true
	}
	return false
}

// pluginsWithoutIngestCount は「記録を出すのに取り込み件数を名乗れない」プラグインの data_type を返す。
//
// provides を宣言していないプラグインには typed_index が付かない。すると
// gkill_get_plugin_list からは「取り込み0件」と「正常」の区別が付かない ——
// is_alive:true / process_running:true のまま1件も取り込めていない状態が、
// 一覧の上では完全に正常に見える（実測 2026-08-25: 会話ログ系4本のうち1本が
// has_last_error:true で全期間0件だったのに、別途カウントを打つまで分からなかった）。
//
// 件数そのものはここでは出さない。数えるにはプラグイン本体へ問い合わせることになり、
// gkill_get_plugin_list が全プラグインへ直列に往復する形になる（プラグインのハンドラは
// 数十msで返す前提。ADR-0301）。代わりに「数えられない」ことと数え方を名指しする。
func pluginsWithoutIngestCount(plugins []any) []string {
	out := []string{}
	for _, item := range plugins {
		p, ok := item.(*jsonobj.Object)
		if !ok || p == nil {
			continue
		}
		dataType, isString := p.String("data_type")
		if p.Value("emits_kyou") == true && !p.Defined("typed_index") && isString && dataType != "" {
			out = append(out, dataType)
		}
	}
	return out
}

// SummarizePluginToolPayload はプラグインツールの結果の1行サマリを返す。
// 対象外のツール名には false を返すので、呼び出し側は既存の summarize にフォールバックできる。
func SummarizePluginToolPayload(name string, payload *jsonobj.Object) (string, bool) {
	switch name {
	case "gkill_get_plugin_list":
		count := 0
		if plugins, ok := payload.Array("plugins"); ok {
			count = len(plugins)
		}
		summary := "Fetched " + itoa(count) + " plugins."
		// 本文の warnings を読まない経路でも気づけるようにする。
		if warnings, ok := payload.Array("warnings"); ok && len(warnings) != 0 {
			return summary + " (some plugins reported diagnostics; the text is withheld — see warnings)", true
		}
		return summary, true
	default:
		return "", false
	}
}

// IsPluginToolName は name がプラグインツールか。
func IsPluginToolName(name string) bool {
	for _, candidate := range PluginToolNames {
		if candidate == name {
			return true
		}
	}
	return false
}

// PluginCallFunc はサーバ固有の API 呼び出し口（旧 call(pathname, body)）。
type PluginCallFunc func(pathname string, body *jsonobj.Object) (*jsonobj.Object, error)

// HandlePluginToolCall はプラグイン関連ツールを処理する。
func HandlePluginToolCall(call PluginCallFunc, name string, args any) (*jsonobj.Object, error) {
	switch name {
	case "gkill_get_plugin_list":
		normalized, err := NormalizeLocaleOnlyArgs(args)
		if err != nil {
			return nil, err
		}
		response, err := call(GetPluginListEndpoint, normalized)
		if err != nil {
			return nil, err
		}
		plugins := []any{}
		if raw, ok := response.Array("plugins"); ok {
			for _, item := range raw {
				plugins = append(plugins, withoutPluginDiagnostics(item))
			}
		}
		warnings := []any{}
		// 実際に落としたときだけ警告を足す。落としていないのに出すと常時ノイズになる。
		for _, plugin := range plugins {
			if hasWithheldDiagnostics(plugin) {
				warnings = append(warnings, PluginDiagnosticsWithheldWarning)
				break
			}
		}
		uncounted := pluginsWithoutIngestCount(plugins)
		if len(uncounted) != 0 {
			warnings = append(warnings,
				"these plugins feed records but report no ingest count (they declare no provides, so they have no typed_index): "+
					strings.Join(uncounted, ", ")+". is_alive:true does not mean anything was ingested — a plugin can answer pings "+
					"while holding zero records. To check one, call gkill_get_kyous with count_only:true and "+
					`data_types:["<the data_type>"] over the period you expect.`)
		}
		if len(warnings) != 0 {
			return jsonobj.Obj("plugins", plugins, "warnings", warnings), nil
		}
		return jsonobj.Obj("plugins", plugins), nil
	default:
		return nil, NewGkillApiError("Unknown tool: "+name, nil)
	}
}

// isPluginPayload は get_kyous のペイロードがプラグイン由来かを判定する。
func isPluginPayload(value any) bool {
	p, ok := value.(*jsonobj.Object)
	return ok && p != nil && p.Value("kind") == "plugin"
}

// hasPluginContentKey は本文取得に要る rep_name と id を Kyou が持つかを判定する。
// 2026-09-19 までペイロード側にも rep_name / kyou_id が写されていたが、Kyou 側と常に同値で
// 毎件3欄が二重に並ぶだけだったので落とした（ADR-0629）。鍵は Kyou 側から取る。
func hasPluginContentKey(kyou *jsonobj.Object) bool {
	repName, ok1 := kyou.String("rep_name")
	id, ok2 := kyou.String("id")
	return ok1 && repName != "" && ok2 && id != ""
}

// PluginPayloadRef は本文取得の鍵（Kyou 側の rep_name / id）と、本文を書き込む先のペイロード（元オブジェクトの参照）。
type PluginPayloadRef struct {
	RepName string
	KyouID  string
	Payload *jsonobj.Object
}

// CollectPluginPayloads は kyous[] から kind:"plugin" のエントリを取得順に集める。
func CollectPluginPayloads(kyous any) []PluginPayloadRef {
	items, ok := jsonobj.AsArray(kyous)
	if !ok {
		return []PluginPayloadRef{}
	}
	payloads := []PluginPayloadRef{}
	for _, item := range items {
		kyou, ok := item.(*jsonobj.Object)
		if !ok || kyou == nil {
			continue
		}
		payload := kyou.Value("payload")
		if isPluginPayload(payload) && hasPluginContentKey(kyou) {
			repName, _ := kyou.String("rep_name")
			id, _ := kyou.String("id")
			payloads = append(payloads, PluginPayloadRef{RepName: repName, KyouID: id, Payload: payload.(*jsonobj.Object)})
		}
	}
	return payloads
}

// GroupedEntry は RunGroupedWithConcurrency の実行対象。
type GroupedEntry struct {
	Key  string
	Item any
}

// RunGroupedWithConcurrency はキーごとに直列、キー間は並列でタスクを実行する。
//
// gkill のプラグインは1プロセスにつき1ミューテックスで直列化される。しかも Go 側の
// 30秒デッドラインはミューテックス待ちを含むので (plugin_repository_impl.go)、
// 同一プラグインへ同時に投げると待ち時間が期限を食い潰し、期限切れ時の
// Process.Kill() でプラグインプロセスが落ちる。だからキー内は必ず直列にする。
//
// worker が false を返すかエラーを返した場合、そのキーの残りは実行しない。
// エラーは握り潰すので、この関数自体は失敗しない。
func RunGroupedWithConcurrency(entries []GroupedEntry, concurrency int, worker func(item any) (bool, error)) {
	groupIndex := map[string]int{}
	queue := [][]any{}
	for _, entry := range entries {
		if index, ok := groupIndex[entry.Key]; ok {
			queue[index] = append(queue[index], entry.Item)
			continue
		}
		groupIndex[entry.Key] = len(queue)
		queue = append(queue, []any{entry.Item})
	}
	if len(queue) == 0 {
		return
	}
	width := concurrency
	if width > len(queue) {
		width = len(queue)
	}
	if width < 1 {
		width = 1
	}
	var next int64
	var wg sync.WaitGroup
	for range width {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				index := int(atomic.AddInt64(&next, 1) - 1)
				if index >= len(queue) {
					return
				}
				for _, item := range queue[index] {
					keepGoing, err := worker(item)
					if err != nil {
						keepGoing = false
					}
					if !keepGoing {
						break
					}
				}
			}
		}()
	}
	wg.Wait()
}

func shortPluginContentError(err error) string {
	message := err.Error()
	if jsLength(message) > maxPluginContentErrorLength {
		return jsSlice(message, 0, maxPluginContentErrorLength)
	}
	return message
}

// InlineOptions は InlinePluginContents の設定。0 / 空は既定値。
type InlineOptions struct {
	MaxTextLength   int
	Format          string
	MaxKyous        int
	TotalTextLength int
	Concurrency     int
	Deadline        time.Duration
	MaxHTMLLength   int
	LocaleName      string
	Now             func() time.Time
}

// InlineStats は InlinePluginContents の集計。
type InlineStats struct {
	Requested       int
	Inlined         int
	Truncated       int
	Skipped         int
	Errors          int
	TotalTextLength int
}

type inlineEntry struct {
	repName  string
	kyouID   string
	payloads []*jsonobj.Object
}

type inlineResult struct {
	html        string
	htmlClipped bool
	err         string
	failed      bool
}

// InlinePluginContents は kyous[] のプラグインペイロードに本文を埋め込む。
//
// ペイロードを破壊的に更新し、個別の失敗は content_status に落として
// gkill_get_kyous 全体は落とさない (この関数は失敗しない)。
//
// 実行中のリクエストは絶対に abort しない。gkill 側は HTTP リクエストの
// コンテキストをそのままプラグイン呼び出しに渡しており、abort すると
// プラグインプロセスが kill されるため。デッドラインは「新しいリクエストを
// 始めない」ことだけで実現する。
func InlinePluginContents(call PluginCallFunc, kyous []any, options InlineOptions) InlineStats {
	maxTextLength := options.MaxTextLength
	if maxTextLength == 0 {
		maxTextLength = DefaultInlinePluginContentMaxTextLength
	}
	format := options.Format
	if format == "" {
		format = DefaultPluginContentFormat
	}
	maxKyous := options.MaxKyous
	if maxKyous == 0 {
		maxKyous = MaxInlinePluginContentKyous
	}
	totalTextLength := options.TotalTextLength
	if totalTextLength == 0 {
		totalTextLength = InlinePluginContentTotalTextLength
	}
	concurrency := options.Concurrency
	if concurrency == 0 {
		concurrency = InlinePluginContentRepConcurrency
	}
	deadline := options.Deadline
	if deadline == 0 {
		deadline = InlinePluginContentDeadlineMS * time.Millisecond
	}
	maxHTMLLength := options.MaxHTMLLength
	if maxHTMLLength == 0 {
		maxHTMLLength = MaxInlinePluginContentHTMLLength
	}
	now := options.Now
	if now == nil {
		now = timeNow
	}

	payloads := CollectPluginPayloads(kyous)
	stats := InlineStats{Requested: len(payloads)}
	if len(payloads) == 0 {
		return stats
	}

	markSkipped := func(targets []*jsonobj.Object, reason string) {
		for _, payload := range targets {
			payload.Set("content_status", "skipped")
			payload.Set("content_skipped_reason", reason)
		}
		stats.Skipped += len(targets)
	}

	// 同じ (rep_name, kyou_id) が複数件返ることがあるので、取得は1回にまとめる。
	// 件数上限は「取得しにいく対象の数」に対して掛ける。
	entries := []*inlineEntry{}
	entryByKey := map[string]*inlineEntry{}
	for _, ref := range payloads {
		key := ref.RepName + ref.KyouID
		if hit, ok := entryByKey[key]; ok {
			hit.payloads = append(hit.payloads, ref.Payload)
			continue
		}
		if len(entryByKey) >= maxKyous {
			markSkipped([]*jsonobj.Object{ref.Payload}, "max_kyous")
			continue
		}
		entry := &inlineEntry{repName: ref.RepName, kyouID: ref.KyouID, payloads: []*jsonobj.Object{ref.Payload}}
		entryByKey[key] = entry
		entries = append(entries, entry)
	}

	var mu sync.Mutex
	results := map[*inlineEntry]inlineResult{}
	// ある rep で打ち切ったとき、その rep の未処理エントリに付ける理由。
	repStopReason := map[string]string{}
	startedAt := now()

	grouped := make([]GroupedEntry, 0, len(entries))
	for _, entry := range entries {
		grouped = append(grouped, GroupedEntry{Key: entry.repName, Item: entry})
	}
	RunGroupedWithConcurrency(grouped, concurrency, func(item any) (bool, error) {
		entry := item.(*inlineEntry)
		if now().Sub(startedAt) >= deadline {
			mu.Lock()
			repStopReason[entry.repName] = "deadline"
			mu.Unlock()
			return false, nil
		}
		body := jsonobj.Obj("rep_name", entry.repName, "kyou_id", entry.kyouID)
		if options.LocaleName != "" {
			body.Set("locale_name", options.LocaleName)
		}
		response, err := call(GetPluginContentHTMLEndpoint, body)
		if err != nil {
			mu.Lock()
			results[entry] = inlineResult{err: shortPluginContentError(err), failed: true}
			// タイムアウトはプラグインプロセスを殺しているので、同じ rep に投げ続けても
			// コールドスタートで待たされるだけ。その rep の残りは諦める。
			repStopReason[entry.repName] = "rep_error"
			mu.Unlock()
			return false, nil
		}
		rawHTML := ""
		if response != nil {
			if s, ok := response.String("html"); ok {
				rawHTML = s
			}
		}
		html := rawHTML
		if jsLength(rawHTML) > maxHTMLLength {
			html = jsSlice(rawHTML, 0, maxHTMLLength)
		}
		mu.Lock()
		results[entry] = inlineResult{html: html, htmlClipped: jsLength(html) != jsLength(rawHTML)}
		mu.Unlock()
		return true, nil
	})

	wantText := format == "text" || format == "both"
	wantHTML := format == "html" || format == "both"

	// 予算は取得完了順ではなく Kyou の並び順で適用する。
	// ネットワークのタイミングによらず同じ入力から同じ出力になる。
	used := 0
	for _, entry := range entries {
		result, ok := results[entry]
		if !ok {
			reason, hasReason := repStopReason[entry.repName]
			if !hasReason {
				reason = "deadline"
			}
			markSkipped(entry.payloads, reason)
			continue
		}
		if result.failed {
			for _, payload := range entry.payloads {
				payload.Set("content_status", "error")
				payload.Set("content_error", result.err)
			}
			stats.Errors += len(entry.payloads)
			continue
		}
		converted := HtmlToText(result.html, maxTextLength)
		cost := 0
		if wantText {
			cost += jsLength(converted.Text)
		}
		if wantHTML {
			cost += jsLength(result.html)
		}
		// 1件目は必ず載せる。そうしないと「1件だけ全文が欲しい」ケースで
		// 上限に関係なく常に空振りしてしまう。
		if used > 0 && used+cost > totalTextLength {
			markSkipped(entry.payloads, "budget")
			continue
		}
		used += cost
		truncated := converted.Truncated || result.htmlClipped
		for _, payload := range entry.payloads {
			if wantText {
				payload.Set("content_text", converted.Text)
			}
			if wantHTML {
				payload.Set("content_html", result.html)
			}
			if truncated {
				payload.Set("content_status", "truncated")
			} else {
				payload.Set("content_status", "ok")
			}
		}
		stats.Inlined += len(entry.payloads)
		if truncated {
			stats.Truncated += len(entry.payloads)
		}
	}
	stats.TotalTextLength = used
	return stats
}

// SummarizeInlinePluginContent は get_kyous のサマリ行に足す一文を返す。
// インライン化していないときは空文字を返す。
func SummarizeInlinePluginContent(stats *InlineStats) string {
	if stats == nil || stats.Requested == 0 {
		return ""
	}
	notes := []string{}
	if stats.Truncated > 0 {
		notes = append(notes, itoa(stats.Truncated)+" truncated")
	}
	if stats.Skipped > 0 {
		notes = append(notes, itoa(stats.Skipped)+" not fetched")
	}
	if stats.Errors > 0 {
		notes = append(notes, itoa(stats.Errors)+" failed")
	}
	suffix := ""
	if len(notes) > 0 {
		suffix = " (" + strings.Join(notes, ", ") + ")"
	}
	return " Embedded plugin content for " + itoa(stats.Inlined) + " of " + itoa(stats.Requested) + " plugin kyous" + suffix + "."
}
