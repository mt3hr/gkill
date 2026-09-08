package gkill_server_api

// HandleGetKyousMCP (v2) のリクエストレベルの補助群。
// 複合カーソル・data_types/num/idf_kinds フィルタ・group_by バケット化・未知値警告。
// 契約の全体と却下案: documents/adr/0604-mcp-composite-cursor-strict-limits.md

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/gkill_plugin"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// mcpCursor は get_kyous_mcp のページングカーソルの解釈結果。
// hasID が真なら複合形式（時刻+ID）で、同一時刻のかたまりの途中からでも再開できる。
type mcpCursor struct {
	time  time.Time
	id    string
	hasID bool
}

// mcpCursorSeparator は複合カーソルの区切り。
// RFC3339Nano にはコロンが2つ続く並びが現れない（コロンは常に数字に挟まれる）ので、
// 左端の "::" で割れば ID 側に "::" が含まれていても（プラグインIDは任意文字列）安全。
const mcpCursorSeparator = "::"

// encodeMCPCursor は複合カーソル文字列を作る。
// 時刻は RFC3339Nano（秒へ切り捨てると同じ秒の内側が漏れるため）。
//
// タイムゾーンはローカルへ揃える。保存層は書いた側のオフセットをそのまま往復させるので
// (KFTL はローカル、MCP と Web は UTC で書く。localTime のコメント参照)、揃えないと
// 同じ検索の next_cursor が行によって "+09:00::id" と "Z::id" に割れる。
// parseMCPCursor はどちらも同じ瞬間として解釈するので往復は前から成立していたが、
// 応答の他の時刻はすべて localTime を通しており、カーソルだけが素通しだった
// (2026-08-25 の実利用レビュー)。
func encodeMCPCursor(t time.Time, id string) string {
	return t.In(time.Local).Format(time.RFC3339Nano) + mcpCursorSeparator + id
}

// parseMCPCursor はカーソル文字列を解釈する。
// 複合形式 {RFC3339Nano}::{ID} を優先し、旧形式（RFC3339 単独・日付のみ）も受理する
// （移行中の in-flight カーソルや、API を直接叩くクライアントを壊さないため）。
// どの形式でも解釈できないときはエラー（黙って1ページ目に戻すとページングが終わらない）。
func parseMCPCursor(cursor string) (mcpCursor, error) {
	if idx := strings.Index(cursor, mcpCursorSeparator); idx >= 0 {
		t, err := time.Parse(time.RFC3339Nano, cursor[:idx])
		if err != nil {
			return mcpCursor{}, fmt.Errorf("error at parse composite cursor time %q: %w", cursor, err)
		}
		return mcpCursor{time: t, id: cursor[idx+len(mcpCursorSeparator):], hasID: true}, nil
	}
	// 旧形式。Goの time.Parse は layout に小数秒が無くても入力側の小数秒を受理する
	if t, err := time.Parse(time.RFC3339, cursor); err == nil {
		return mcpCursor{time: t}, nil
	}
	// 日付のみ(YYYY-MM-DD)も受ける。MCPサーバは正規化してから送るが、
	// APIを直接叩くクライアントはそのまま送ってくる
	if t, err := time.ParseInLocation(time.DateOnly, cursor, time.Local); err == nil {
		return mcpCursor{time: t}, nil
	}
	return mcpCursor{}, fmt.Errorf("error at parse cursor %q: unsupported format", cursor)
}

// mcpGroupByValues は group_by が受理する値。順序は doc 用。
var mcpGroupByValues = []string{"month", "day", "week_of_day", "hour", "data_type", "rep_name", "create_app", "update_app", "url_domain", "file_extension"}

func isValidMCPGroupBy(groupBy string) bool {
	return slices.Contains(mcpGroupByValues, groupBy)
}

// mcpIDFKindValues は idf_kinds が受理する値。
var mcpIDFKindValues = []string{"image", "video", "audio", "zip", "other"}

// mcpBucketLimit は group_by の最大バケット数。超過分は "(other)" へ合算する。
// url_domain のような自由値キーの爆発から応答サイズを守る（max_size_mb は group_by に適用されないため）。
const mcpBucketLimit = 1000

// applyMCPDataTypesFilter は DTO の data_type 文字列（mi_create / claude_conversation 等）の
// 許可リストで結果を絞る。nil=未使用、非nil空=0件（FindQuery の null 意味論に揃える）。
// FindQuery に足さずリクエストレベルなのは、和集合仕様の rep_types（Web の Mi 画面が依存）を
// 触らずに Mi/MiReKyou/プラグインを直接絞る口を作るため（外部監査 S5/A2）。
func applyMCPDataTypesFilter(kyous []reps.Kyou, dataTypes []string) []reps.Kyou {
	if dataTypes == nil {
		return kyous
	}
	allow := make(map[string]struct{}, len(dataTypes))
	for _, dataType := range dataTypes {
		allow[dataType] = struct{}{}
	}
	out := kyous[:0]
	for _, kyou := range kyous {
		if _, ok := allow[kyou.DataType]; ok {
			out = append(out, kyou)
		}
	}
	return out
}

// applyMCPCreateAppsFilter は「どのアプリが書いた記録か」で絞る。
// nil=未使用、非nil空=0件（FindQuery の null 意味論に揃える）。
//
// gkill_kftl / gkill_mcp_readwrite / gkill_mcp_write / urlog_bookmarklet / git などの値が
// 全レコードに入っているのに、引く手段だけが無かった（2026-08-24 の再監査）。
//
// **FindQuery に足していないのは意図的**。ReKyou / MiReKyou のワード委譲は
// 利用者のクエリをそのまま下位検索へ流すので、この条件が SQL まで降りると
// 「MCPで作ったリポストだが、リポスト先の記録はブラウザで作った」が
// エラーも警告も無しに消える。rep名をSQLへ降ろすのを否決したのと同じ失敗クラス。
// data_types が同じ理由でリクエストレベルに居る（外部監査 S5/A2）。
func applyMCPCreateAppsFilter(kyous []reps.Kyou, createApps []string) []reps.Kyou {
	if createApps == nil {
		return kyous
	}
	allow := make(map[string]struct{}, len(createApps))
	for _, createApp := range createApps {
		allow[createApp] = struct{}{}
	}
	out := kyous[:0]
	for _, kyou := range kyous {
		if _, ok := allow[kyou.CreateApp]; ok {
			out = append(out, kyou)
		}
	}
	return out
}

// applyMCPUpdateAppsFilter は「どのアプリが最後に更新した記録か」で絞る。
// 意味論は applyMCPCreateAppsFilter と同じ。
func applyMCPUpdateAppsFilter(kyous []reps.Kyou, updateApps []string) []reps.Kyou {
	if updateApps == nil {
		return kyous
	}
	allow := make(map[string]struct{}, len(updateApps))
	for _, updateApp := range updateApps {
		allow[updateApp] = struct{}{}
	}
	out := kyous[:0]
	for _, kyou := range kyous {
		if _, ok := allow[kyou.UpdateApp]; ok {
			out = append(out, kyou)
		}
	}
	return out
}

// applyMCPNumFilter は数値ペイロードの範囲絞り込み。
// 対象は kc.num_value / nlog.amount / lantana.mood のみで、有効時は数値を持たない
// 種別の行は結果から外れる（「数値で絞る」の意味論）。typed の取得は集約 FindXxx 経由
// なので findChunkedByIDs のバインド上限対策を素通しで受ける。
func applyMCPNumFilter(ctx context.Context, repositories *reps.GkillRepositories, kyous []reps.Kyou, numMin *float64, numMax *float64) ([]reps.Kyou, []string, error) {
	warnings := []string{}

	kcIDs := []string{}
	nlogIDs := []string{}
	lantanaIDs := []string{}
	for _, kyou := range kyous {
		switch payloadKindOfDataType(kyou.DataType) {
		case "kc":
			kcIDs = append(kcIDs, kyou.ID)
		case "nlog":
			nlogIDs = append(nlogIDs, kyou.ID)
		case "lantana":
			lantanaIDs = append(lantanaIDs, kyou.ID)
		}
	}
	if len(kcIDs)+len(nlogIDs)+len(lantanaIDs) == 0 {
		warnings = append(warnings, "num filter matched no numeric records (it applies to kc / nlog / lantana only); all records were filtered out")
		return kyous[:0], warnings, nil
	}

	valueByID := map[string]float64{}
	unparsableCount := 0
	if len(kcIDs) != 0 {
		kcs, err := repositories.KCReps.FindKC(ctx, &find.FindQuery{IDs: kcIDs, OnlyLatestData: true})
		if err != nil {
			return nil, warnings, fmt.Errorf("error at find kc for num filter: %w", err)
		}
		for _, kc := range kcs {
			value, err := kc.NumValue.Float64()
			if err != nil {
				unparsableCount++
				continue
			}
			valueByID[kc.ID] = value
		}
	}
	if len(nlogIDs) != 0 {
		nlogs, err := repositories.NlogReps.FindNlog(ctx, &find.FindQuery{IDs: nlogIDs, OnlyLatestData: true})
		if err != nil {
			return nil, warnings, fmt.Errorf("error at find nlog for num filter: %w", err)
		}
		for _, nlog := range nlogs {
			value, err := nlog.Amount.Float64()
			if err != nil {
				unparsableCount++
				continue
			}
			valueByID[nlog.ID] = value
		}
	}
	if len(lantanaIDs) != 0 {
		lantanas, err := repositories.LantanaReps.FindLantana(ctx, &find.FindQuery{IDs: lantanaIDs, OnlyLatestData: true})
		if err != nil {
			return nil, warnings, fmt.Errorf("error at find lantana for num filter: %w", err)
		}
		for _, lantana := range lantanas {
			valueByID[lantana.ID] = float64(lantana.Mood)
		}
	}
	if unparsableCount > 0 {
		warnings = append(warnings, fmt.Sprintf("num filter skipped %d record(s) whose numeric value could not be parsed", unparsableCount))
	}

	out := kyous[:0]
	for _, kyou := range kyous {
		value, ok := valueByID[kyou.ID]
		if !ok {
			continue
		}
		if numMin != nil && value < *numMin {
			continue
		}
		if numMax != nil && value > *numMax {
			continue
		}
		out = append(out, kyou)
	}
	return out, warnings, nil
}

// classifyIDFKind は IDF の種別を image/video/audio/zip/other のどれかへ寄せる。
func classifyIDFKind(idfKyou reps.IDFKyou) string {
	switch {
	case idfKyou.IsImage:
		return "image"
	case idfKyou.IsVideo:
		return "video"
	case idfKyou.IsAudio:
		return "audio"
	case idfKyou.IsZip:
		return "zip"
	}
	return "other"
}

// applyMCPIDFKindsFilter は idf(ファイル)の種別絞り込み。
// nil=未使用、非nil空=0件。有効時は idf 以外の行は結果から外れる。
// 未知の種別値は警告に積む（黙って0件へ落とすと綴り違いに気付けない）。
func applyMCPIDFKindsFilter(ctx context.Context, repositories *reps.GkillRepositories, kyous []reps.Kyou, idfKinds []string) ([]reps.Kyou, []string, error) {
	warnings := []string{}
	allow := make(map[string]struct{}, len(idfKinds))
	for _, kind := range idfKinds {
		if !slices.Contains(mcpIDFKindValues, kind) {
			warnings = append(warnings, fmt.Sprintf("unknown idf_kind %q (valid: %s)", kind, strings.Join(mcpIDFKindValues, ", ")))
			continue
		}
		allow[kind] = struct{}{}
	}

	idfIDs := []string{}
	for _, kyou := range kyous {
		if payloadKindOfDataType(kyou.DataType) == "idf" {
			idfIDs = append(idfIDs, kyou.ID)
		}
	}
	kindByID := map[string]string{}
	if len(idfIDs) != 0 {
		idfKyous, err := repositories.IDFKyouReps.FindIDFKyou(ctx, &find.FindQuery{IDs: idfIDs, OnlyLatestData: true})
		if err != nil {
			return nil, warnings, fmt.Errorf("error at find idf kyou for idf_kinds filter: %w", err)
		}
		for _, idfKyou := range idfKyous {
			kindByID[idfKyou.ID] = classifyIDFKind(idfKyou)
		}
	}

	out := kyous[:0]
	for _, kyou := range kyous {
		kind, ok := kindByID[kyou.ID]
		if !ok {
			continue
		}
		if _, allowed := allow[kind]; allowed {
			out = append(out, kyou)
		}
	}
	return out, warnings, nil
}

// mcpWeekOfDayKeys は group_by:"week_of_day" のバケットキー（time.Weekday の並び）。
var mcpWeekOfDayKeys = [7]string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}

// bucketizeMCPKyous は group_by の集計本体。
// 時刻系のキーはサーバのローカルタイムゾーンで作る（related_time の表示と揃える）。
// url_domain / file_extension は typed の一括取得を伴い、対象種別の行だけを数える
// （それ以外の行が混ざっていた場合は除外し、その旨を警告で返す）。
func bucketizeMCPKyous(ctx context.Context, repositories *reps.GkillRepositories, kyous []reps.Kyou, groupBy string) ([]req_res.KyouCountBucketMCPDTO, []string, error) {
	warnings := []string{}
	counts := map[string]int{}
	timeKeyed := false

	switch groupBy {
	case "month":
		timeKeyed = true
		for _, kyou := range kyous {
			counts[kyou.RelatedTime.In(time.Local).Format("2006-01")]++
		}
	case "day":
		timeKeyed = true
		for _, kyou := range kyous {
			counts[kyou.RelatedTime.In(time.Local).Format("2006-01-02")]++
		}
	case "hour":
		timeKeyed = true
		for _, kyou := range kyous {
			counts[kyou.RelatedTime.In(time.Local).Format("15")]++
		}
	case "week_of_day":
		timeKeyed = true
		for _, kyou := range kyous {
			counts[mcpWeekOfDayKeys[int(kyou.RelatedTime.In(time.Local).Weekday())]]++
		}
	case "data_type":
		for _, kyou := range kyous {
			counts[kyou.DataType]++
		}
	case "rep_name":
		for _, kyou := range kyous {
			key := kyou.RepName
			if key == "" {
				// 追加直後の行はキャッシュ表の REP_NAME が空のことがある
				key = "(unknown)"
			}
			counts[key]++
		}
	case "create_app", "update_app":
		// create_apps / update_apps で絞れるのに、**有効値を知る手段が無かった**。
		// ツール説明にベタ書きした一覧しか手掛かりが無く、0件のときに
		// 「存在しない」のか「名前違い」のかを呼び出し側が判別できない
		// （2026-08-24 の実利用レビュー）。
		for _, kyou := range kyous {
			key := kyou.CreateApp
			if groupBy == "update_app" {
				key = kyou.UpdateApp
			}
			if key == "" {
				key = "(unknown)"
			}
			counts[key]++
		}
	case "url_domain":
		urlogIDs := []string{}
		for _, kyou := range kyous {
			if payloadKindOfDataType(kyou.DataType) == "urlog" {
				urlogIDs = append(urlogIDs, kyou.ID)
			}
		}
		if excluded := len(kyous) - len(urlogIDs); excluded > 0 {
			warnings = append(warnings, fmt.Sprintf("group_by url_domain covers only urlog records; %d non-urlog record(s) were excluded from buckets", excluded))
		}
		if len(urlogIDs) != 0 {
			findQuery := &find.FindQuery{IDs: urlogIDs, OnlyLatestData: true, ExcludeURLogThumbnailImage: true}
			urlogs, err := repositories.URLogReps.FindURLog(ctx, findQuery)
			if err != nil {
				return nil, warnings, fmt.Errorf("error at find urlog for group_by url_domain: %w", err)
			}
			for _, urlog := range urlogs {
				parsed, err := url.Parse(urlog.URL)
				if err != nil || parsed.Hostname() == "" {
					counts["(invalid)"]++
					continue
				}
				counts[strings.ToLower(parsed.Hostname())]++
			}
		}
	case "file_extension":
		idfIDs := []string{}
		for _, kyou := range kyous {
			if payloadKindOfDataType(kyou.DataType) == "idf" {
				idfIDs = append(idfIDs, kyou.ID)
			}
		}
		if excluded := len(kyous) - len(idfIDs); excluded > 0 {
			warnings = append(warnings, fmt.Sprintf("group_by file_extension covers only idf records; %d non-idf record(s) were excluded from buckets", excluded))
		}
		if len(idfIDs) != 0 {
			idfKyous, err := repositories.IDFKyouReps.FindIDFKyou(ctx, &find.FindQuery{IDs: idfIDs, OnlyLatestData: true})
			if err != nil {
				return nil, warnings, fmt.Errorf("error at find idf kyou for group_by file_extension: %w", err)
			}
			for _, idfKyou := range idfKyous {
				ext := strings.ToLower(filepath.Ext(idfKyou.TargetFile))
				if ext == "" {
					ext = "(none)"
				}
				counts[ext]++
			}
		}
	default:
		return nil, warnings, fmt.Errorf("unknown group_by %q", groupBy)
	}

	buckets := make([]req_res.KyouCountBucketMCPDTO, 0, len(counts))
	for key, count := range counts {
		buckets = append(buckets, req_res.KyouCountBucketMCPDTO{Key: key, Count: count})
	}
	if timeKeyed {
		if groupBy == "week_of_day" {
			// 曜日は日〜土の固定順（辞書順に並べると曜日の並びが壊れる）
			order := map[string]int{}
			for i, key := range mcpWeekOfDayKeys {
				order[key] = i
			}
			slices.SortFunc(buckets, func(a, b req_res.KyouCountBucketMCPDTO) int {
				return order[a.Key] - order[b.Key]
			})
		} else {
			slices.SortFunc(buckets, func(a, b req_res.KyouCountBucketMCPDTO) int {
				return strings.Compare(a.Key, b.Key)
			})
		}
	} else {
		// カテゴリ系は件数降順・同数はキー昇順
		slices.SortFunc(buckets, func(a, b req_res.KyouCountBucketMCPDTO) int {
			if a.Count != b.Count {
				return b.Count - a.Count
			}
			return strings.Compare(a.Key, b.Key)
		})
	}

	// バケット爆発から応答サイズを守る（url_domain は自由値キーのため）
	if len(buckets) > mcpBucketLimit {
		otherCount := 0
		for _, bucket := range buckets[mcpBucketLimit-1:] {
			otherCount += bucket.Count
		}
		folded := len(buckets) - (mcpBucketLimit - 1)
		buckets = append(buckets[:mcpBucketLimit-1], req_res.KyouCountBucketMCPDTO{Key: "(other)", Count: otherCount})
		warnings = append(warnings, fmt.Sprintf("group_by produced more than %d buckets; %d bucket(s) were folded into \"(other)\"", mcpBucketLimit, folded))
	}

	return buckets, warnings, nil
}

// knownMCPDataTypes は data_types の検証に使う既知集合。
// 組み込みの基底型 + 射影名 + プラグイン manifest の DataType。
//
// Kyouを1件も出さないプラグイン（emits_kyou=false）のDataTypeは**入れない**。
// manifestの必須項目なので値は存在するが、そのdata_typeを持つKyouは存在しないので、
// 既知として通すと「警告も出ないのに必ず0件」になる（2026-08-24 の実利用報告）。
func knownMCPDataTypes(repositories *reps.GkillRepositories) map[string]struct{} {
	known := map[string]struct{}{}
	for _, dataType := range []string{
		"kmemo", "kc", "urlog", "nlog", "lantana", "rekyou", "idf", "git_commit_log",
		"timeis", "timeis_start", "timeis_end",
		"mi", "mi_create", "mi_check", "mi_limit", "mi_start", "mi_end",
		"mirekyou", "mirekyou_create", "mirekyou_check", "mirekyou_limit", "mirekyou_start", "mirekyou_end",
	} {
		known[dataType] = struct{}{}
	}
	for _, pluginRep := range repositories.PluginReps {
		manifest := pluginRep.GetManifest()
		if !manifest.EmitsKyouOrDefault() {
			continue
		}
		known[manifest.DataType] = struct{}{}
	}
	return known
}

// emptyPluginIndexHint は「登録済みプラグインの data_type だが索引が空」のときの案内を返す。
// 該当しなければ空文字。
//
// 索引の状態は非ブロッキングで読める（プラグインへは行かない）。
// **失敗理由の本文は返さない** —— プラグインの生 stderr と索引構築のエラー文は
// 利用者の端末のディレクトリ構成を含むため（ADR-0707）。
// 「何か書かれている」ことと読む先だけ伝えれば、診断としては足りる。
func emptyPluginIndexHint(repositories *reps.GkillRepositories, dataType string) string {
	for _, pluginRep := range repositories.PluginReps {
		manifest := pluginRep.GetManifest()
		if manifest.DataType != dataType || !manifest.EmitsKyouOrDefault() {
			continue
		}
		typedIndex := pluginRep.TypedIndex()
		if typedIndex == nil {
			// provides を宣言していないプラグインは索引を持たない。
			// 取り込み状況を知る手段が無いので、断定しない。
			return ""
		}
		stats := typedIndex.Stats()
		if stats.State == reps.PluginTypedIndexStateOK && stats.RecordCount > 0 {
			return ""
		}
		hint := fmt.Sprintf("data_type %q belongs to plugin %q, whose index is %s with %d record(s), so this filter matches nothing",
			dataType, manifest.Name, stats.State, stats.RecordCount)
		if stats.LastBuildError != "" {
			hint += "; the last index build failed (the reason is withheld because it describes the user's own machine — read it from get_plugin_list on the server console)"
		}
		return hint + ". Check gkill_get_plugin_list"
	}
	return ""
}

// nonKyouPluginHint は「Kyouを出さないプラグイン」の rep名 / data_type に一致したときの
// 案内文を返します。一致しなければ空文字を返します。
//
// 汎用の「綴りを確かめろ」では直しようがない ―― 綴りは合っており、
// そのプラグインが原理的にKyouを出さないことが原因なので、読む先を名指しする。
func nonKyouPluginHint(repositories *reps.GkillRepositories, value string) string {
	for _, pluginRep := range repositories.PluginReps {
		manifest := pluginRep.GetManifest()
		if manifest.EmitsKyouOrDefault() {
			continue
		}
		if manifest.RepName != value && manifest.DataType != value {
			continue
		}
		provides := make([]string, 0, len(manifest.Provides))
		for _, kind := range manifest.Provides {
			provides = append(provides, string(kind))
		}
		hint := fmt.Sprintf("plugin %q emits no kyou", manifest.Name)
		if len(provides) != 0 {
			hint += fmt.Sprintf(" (provides: %s)", strings.Join(provides, ", "))
		}
		if manifest.ProvidesKind(gkill_plugin.PluginProvidesGPSLog) {
			hint += "; read its data with get_gps_log"
		}
		return hint
	}
	return ""
}

// localTime / localTimePtr は応答へ載せる時刻をサーバのローカルタイムゾーンへ揃える。
//
// 保存層は "2006-01-02T15:04:05-07:00" で読み書きするので、**書いた側のタイムゾーンが
// そのまま往復する**。KFTL(Go)はローカルで書き、MCP と Web は new Date().toISOString()
// なので UTC で書く。その結果 Mi の create_time は +09:00、MiReKyou の create_time は
// 同じ瞬間なのに Z、というように**同じ1レコードの中で上位フィールドと payload の
// オフセットが食い違って**いた（2026-08-24 の実利用レビュー）。
// RelatedTime / UpdateTime は既に揃えてあるので、payload 側も同じ扱いにする。DBは触らない。
func localTime(t time.Time) time.Time {
	return t.In(time.Local)
}

func localTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	local := t.In(time.Local)
	return &local
}

// forMiWithoutProjectionWarning は「for_mi を立てたのに include_*_mi を1つも立てていない」
// ときの案内を返す。該当しなければ空文字。
//
// この形は **0件が返るのに警告が1行も出ない**。filterMiForMi は MiReps.FindMi /
// MiReKyouReps.FindMiReKyou の結果に含まれる ID しか残さないが、5フラグが全て false だと
// SQL の射影が1本も組み立てられず空になるため（mi_repository_sqlite3_impl.go の
// sqlSegments が空）。スキーマには「最低1つ立てろ」と書いてあるが、
// 実利用の AI は 0件を「先週はタスクが無かった」と読んだ（2026-08-25 のレビュー）。
//
// rep_types の綴り違いには有効値つきの警告が出るのに、こちらだけ無言なのは非対称でもある。
func forMiWithoutProjectionWarning(query *find.FindQuery) string {
	if !query.ForMi {
		return ""
	}
	if query.IncludeCreateMi || query.IncludeCheckMi || query.IncludeLimitMi ||
		query.IncludeStartMi || query.IncludeEndMi {
		return ""
	}
	return "query.for_mi is set but none of include_create_mi / include_check_mi / include_limit_mi / " +
		"include_start_mi / include_end_mi is true, so this search can only return zero entries. " +
		"The five flags choose which time projection supplies rows; they do not narrow an existing result set. " +
		"Set at least one (include_create_mi:true is the usual starting point). " +
		"A zero count from this query means nothing about whether tasks exist"
}

// miSortTypeIgnoredWarning は「mi_sort_type を指定したのに、それが指す射影を
// 立てていない」ときの案内を返す。該当しなければ空文字。
//
// mi_sort_type は並び順の名前をしているが、実際は **カレンダー範囲・時間帯・曜日を
// 照合する時刻軸** も決める(find_filter.go の refilterOverriddenKyousForMi)。
// 対応する include_*_mi が立っていないとその軸には切り替わらず、指定は黙って無視され、
// **件数だけが変わる**。実測では同じ1週間が limit_time 指定で15件、
// estimate_start_time 指定で9件になった(2026-08-25 のレビュー)。
// 週次の集計をこれで作ると、間違いに気づく手がかりが一つも無い。
func miSortTypeIgnoredWarning(query *find.FindQuery) string {
	if !query.ForMi || query.MiSortType == "" {
		return ""
	}
	// mi_sort_type が要求する射影と、それを立てるフラグの対応。
	required := map[string]struct {
		flag  bool
		field string
	}{
		"create_time":         {query.IncludeCreateMi, "include_create_mi"},
		"estimate_start_time": {query.IncludeStartMi, "include_start_mi"},
		"estimate_end_time":   {query.IncludeEndMi, "include_end_mi"},
		"limit_time":          {query.IncludeLimitMi, "include_limit_mi"},
	}
	needed, ok := required[string(query.MiSortType)]
	if !ok || needed.flag {
		return ""
	}
	return fmt.Sprintf(
		"query.mi_sort_type is %q but query.%s is not true, so the sort type is ignored. mi_sort_type also decides which timestamp the calendar range, the time-of-day window and the weekday filter are matched against, so ignoring it changes the number of entries returned, not just their order. Set %s:true to use that axis",
		string(query.MiSortType), needed.field, needed.field)
}

// miProjectionDataTypes は Mi / MiReKyou の射影名の全集合。
// knownMCPDataTypes が既知として通す値のうち、for_mi を立てないと出てこないもの。
var miProjectionDataTypes = []string{
	"mi_create", "mi_check", "mi_limit", "mi_start", "mi_end",
	"mirekyou_create", "mirekyou_check", "mirekyou_limit", "mirekyou_start", "mirekyou_end",
}

// miProjectionWarning は「Mi の射影名で絞ったのに for_mi を立てていない」ときの案内を返す。
// 該当しなければ空文字。
//
// Mi の5射影は for_mi を立てたときだけ残る（find_filter.go の
// `isMiData := strings.HasPrefix(DataType, "mi") && ForMi`）。立てないと
// compareKyouEntryPriority が代表1件へ潰し、_start 優先→DataType辞書昇順なので
// mi_check < mi_create。しかも MI.IS_CHECKED は NOT NULL で mi_check 行は全 Mi に
// 必ず存在するため、**mi_create はめったに生き残らない**。
// ただし「絶対に出ない」ではない —— CREATE_TIME が窓の中で UPDATE_TIME が窓の外にある Mi
// （作ったあと窓の外で更新したもの）は mi_create が代表として残る。実測でも素の日付検索に
// mi_create が出る（2026-08-25、1週間で1件）。**「0件だから作っていない」と読ませないこと。**
// data_types は検索後の後段フィルタなので、既知の値なのに警告ゼロで件数が激減していた
// （2026-08-24 の実利用レビュー）。
//
// 潰し込み自体は外さない（外すと素の検索で1つの Mi が最大5件に増え、
// 全クライアントの件数とページングが変わる）。ForMi を勝手に立てることもしない
// （ForMi は検索対象を Mi/MiReKyou へ**限定**するので、
// data_types:["mi_create","kmemo"] のような指定が黙って壊れる）。
func miProjectionWarning(query *find.FindQuery, dataTypes []string) string {
	if query.ForMi {
		return ""
	}
	// 既知の射影名だけを見る。接頭辞で判定すると、プラグインが名乗る
	// "mi" 始まりの data_type まで巻き込む。
	projections := []string{}
	for _, dataType := range dataTypes {
		if slices.Contains(miProjectionDataTypes, dataType) {
			projections = append(projections, dataType)
		}
	}
	if len(projections) == 0 {
		return ""
	}
	return fmt.Sprintf(
		"data_types has Mi projection(s) %s but query.for_mi is not set. "+
			"Without for_mi the five Mi projections collapse to a single representative per record "+
			"(mi_start wins, then mi_check), so mi_create rarely survives and this filter returns far fewer rows "+
			"than the number of tasks that actually match — a low count here does NOT mean no tasks. "+
			"Set query.for_mi=true together with at least one of include_create_mi / include_check_mi / "+
			"include_limit_mi / include_start_mi / include_end_mi. "+
			"Note for_mi also restricts the search to Mi and MiReKyou, so query it separately from other data_types",
		strings.Join(projections, ", "))
}

// collectMCPUnknownValueWarnings は「綴り違いのフィルタ値が黙って0件になる」問題（外部監査 S7）への
// 防御。rep_types / tags / hide_tags / timeis_tags / reps / data_types の各値を既知集合と照合し、
// 見つからなかった値を警告として列挙する。エラーにはしない（実在するが期間内に該当が無い、
// という正当な0件と同じ経路を壊さないため）。照合用一覧の取得に失敗したときも
// リクエストは失敗させず、検証をスキップした旨だけ警告する。
func collectMCPUnknownValueWarnings(ctx context.Context, repositories *reps.GkillRepositories, query *find.FindQuery, dataTypes []string) []string {
	warnings := []string{}

	if warning := miProjectionWarning(query, dataTypes); warning != "" {
		warnings = append(warnings, warning)
	}

	if warning := miSortTypeIgnoredWarning(query); warning != "" {
		warnings = append(warnings, warning)
	}
	if warning := forMiWithoutProjectionWarning(query); warning != "" {
		warnings = append(warnings, warning)
	}

	for _, repType := range query.RepTypes {
		if !find.IsKyouRepType(repType) {
			warnings = append(warnings, fmt.Sprintf("unknown rep_type %q (valid values: %s; plugin records are matched via query.reps or data_types, not rep_types)", repType, strings.Join(find.KyouRepTypes, ", ")))
		}
	}

	tagNamesToCheck := 0
	tagNamesToCheck += len(query.Tags) + len(query.HideTags) + len(query.TimeIsTags)
	if tagNamesToCheck > 0 {
		// 検証には対象の生死を問わない一覧を使う。GetAllTagNames は対象が削除済みの
		// タグを落とすので、include_deleted_data で削除済みを開いた検索に
		// 「未知のタグ」という誤った警告が出てしまう。
		allTagNames, err := repositories.GetAllTagNamesIncludingDeletedTargets(ctx)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("could not validate tag names against the tag list: %v", err))
		} else {
			knownTags := make(map[string]struct{}, len(allTagNames))
			for _, tagName := range allTagNames {
				knownTags[strings.ToLower(tagName)] = struct{}{}
			}
			checkTagField := func(fieldName string, tagNames []string, allowNoTags bool) {
				for _, tagName := range tagNames {
					if allowNoTags && tagName == api.NoTags {
						continue
					}
					if _, ok := knownTags[strings.ToLower(tagName)]; !ok {
						warnings = append(warnings, fmt.Sprintf("unknown tag %q in %s (no tag with this name exists)", tagName, fieldName))
					}
				}
			}
			checkTagField("query.tags", query.Tags, true)
			checkTagField("query.hide_tags", query.HideTags, false)
			checkTagField("query.timeis_tags", query.TimeIsTags, true)
		}
	}

	if len(query.Reps) > 0 {
		allRepNames, err := repositories.GetAllRepNames(ctx)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("could not validate rep names against the rep list: %v", err))
		} else {
			knownReps := make(map[string]struct{}, len(allRepNames))
			for _, repName := range allRepNames {
				knownReps[repName] = struct{}{}
			}
			for _, repName := range query.Reps {
				if _, ok := knownReps[repName]; ok {
					continue
				}
				if hint := nonKyouPluginHint(repositories, repName); hint != "" {
					warnings = append(warnings, fmt.Sprintf("unknown rep %q in query.reps: %s", repName, hint))
					continue
				}
				warnings = append(warnings, fmt.Sprintf("unknown rep %q in query.reps (rep names are case-sensitive; list them with get_rep_infos)", repName))
			}
		}
	}

	if len(dataTypes) > 0 {
		known := knownMCPDataTypes(repositories)
		for _, dataType := range dataTypes {
			if _, ok := known[dataType]; ok {
				// 既知でも、そのプラグインがまだ1件も取り込めていなければ0件は確定する。
				// 「該当なし」と「プラグインが動いていない」が区別できないままだった
				// （Claude.ai プラグインがデータソース欠如で失敗中に、
				// data_types:["claude_conversation"] が0件・警告なしで返っていた。
				// 2026-08-24 の実利用レビュー）。
				if hint := emptyPluginIndexHint(repositories, dataType); hint != "" {
					warnings = append(warnings, hint)
				}
				continue
			}
			if hint := nonKyouPluginHint(repositories, dataType); hint != "" {
				warnings = append(warnings, fmt.Sprintf("unknown data_type %q: %s", dataType, hint))
				continue
			}
			warnings = append(warnings, fmt.Sprintf("unknown data_type %q (built-in projections plus plugin data types; list plugin types with get_plugin_list)", dataType))
		}
	}

	return warnings
}

// livePlayingTimeIsCandidates は付随 TimeIs の母集合から削除済みを落とす。
//
// **削除済みの除外はここでしか行われない。** FindTimeIs は rep の直叩きで、
// GenerateFindSQLCommon も TimeIsRepositories も IS_DELETED を一切見ない。
// gkill で削除済みを落としているのは find_filter.go の Kyou 集約だけで、
// 付随 TimeIs の経路はそこを通らない（Web はクライアントが playing_time 検索を投げ、
// 共有ページは FindFilter.FindKyous を通るので、どちらも落ちている。独自走査はMCPだけ）。
//
// 落とさないと「終了記録ごと消した未終了の打刻」が、開始時刻以降のあらゆる記録へ
// 永久に付く。timeIsCoversMoment が EndTime==nil を「まだ走っている」と扱うためで、
// 削除されたことは EndTime には現れない。
//
// find_filter.go の「IsDeleted は使わないこと」は *FindQuery の旗として* 使うなという話
// （git_commit_log が IsDeleted=true を「削除済みのみ検索」の逆の意味で読むため）。
// レコードの .IsDeleted を読むのは同ファイルの削除済み除外が現にやっていること。
// 呼び出し側は OnlyLatestData:true で引くので、ID ごと最新版1件へ畳んだ後を受け取る。
func livePlayingTimeIsCandidates(found []reps.TimeIs) []reps.TimeIs {
	live := make([]reps.TimeIs, 0, len(found))
	for _, timeis := range found {
		if timeis.IsDeleted {
			continue
		}
		live = append(live, timeis)
	}
	return live
}

// timeIsCoversMoment は、その瞬間にこの打刻が走っていたかを返す。
// EndTime が nil の打刻は「まだ終わっていない」ので開始時刻より後をすべて覆う。
// 判定の意味は playing_time の SQL（START_TIME <= ? AND (? <= END_TIME OR END_TIME IS NULL)）
// と揃えてある。違うのは削除済みの扱いだけで、それは呼び出し前に
// livePlayingTimeIsCandidates が落とす。
func timeIsCoversMoment(timeis reps.TimeIs, moment time.Time) bool {
	// SQL は両端を含む(>= と <=)ので、ここも含める。排他にすると、
	// 打刻と同じ時刻に書かれた記録(KFTL で打刻と本文を一度に書いたときなど)が
	// playing_time では出るのにここでは付かない、という食い違いになる。
	if moment.Before(timeis.StartTime) {
		return false
	}
	if timeis.EndTime == nil {
		return true
	}
	return !moment.After(*timeis.EndTime)
}
