package gkill_server_api

// HandleGetKyousMCP (v2) のリクエストレベルの補助群。
// 複合カーソル・data_types/num/idf_kinds フィルタ・group_by バケット化・未知値警告。
// 契約の全体と却下案: documents/adr/0053-mcp-composite-cursor-strict-limits.md

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
func encodeMCPCursor(t time.Time, id string) string {
	return t.Format(time.RFC3339Nano) + mcpCursorSeparator + id
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
var mcpGroupByValues = []string{"month", "day", "week_of_day", "hour", "data_type", "rep_name", "url_domain", "file_extension"}

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
		known[pluginRep.GetManifest().DataType] = struct{}{}
	}
	return known
}

// collectMCPUnknownValueWarnings は「綴り違いのフィルタ値が黙って0件になる」問題（外部監査 S7）への
// 防御。rep_types / tags / hide_tags / timeis_tags / reps / data_types の各値を既知集合と照合し、
// 見つからなかった値を警告として列挙する。エラーにはしない（実在するが期間内に該当が無い、
// という正当な0件と同じ経路を壊さないため）。照合用一覧の取得に失敗したときも
// リクエストは失敗させず、検証をスキップした旨だけ警告する。
func collectMCPUnknownValueWarnings(ctx context.Context, repositories *reps.GkillRepositories, query *find.FindQuery, dataTypes []string) []string {
	warnings := []string{}

	for _, repType := range query.RepTypes {
		if !find.IsKyouRepType(repType) {
			warnings = append(warnings, fmt.Sprintf("unknown rep_type %q (valid values: %s; plugin records are matched via query.reps or data_types, not rep_types)", repType, strings.Join(find.KyouRepTypes, ", ")))
		}
	}

	tagNamesToCheck := 0
	tagNamesToCheck += len(query.Tags) + len(query.HideTags) + len(query.TimeIsTags)
	if tagNamesToCheck > 0 {
		allTagNames, err := repositories.GetAllTagNames(ctx)
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
				if _, ok := knownReps[repName]; !ok {
					warnings = append(warnings, fmt.Sprintf("unknown rep %q in query.reps (rep names are case-sensitive; list them with get_rep_infos)", repName))
				}
			}
		}
	}

	if len(dataTypes) > 0 {
		known := knownMCPDataTypes(repositories)
		for _, dataType := range dataTypes {
			if _, ok := known[dataType]; !ok {
				warnings = append(warnings, fmt.Sprintf("unknown data_type %q (built-in projections plus plugin data types; list plugin types with get_plugin_list)", dataType))
			}
		}
	}

	return warnings
}
