package find

import (
	"bytes"
	"encoding/json"
	"log/slog"
)

// legacyUseFlagKeys は旧形式の FindQuery JSON が持っていた use_* フラグキー。
// サーバ既知の14個に加えて、クライアント専用の use_mi_sort_type / use_mi_check_state
// （値の mi_sort_type / mi_check_state は残す。フラグは全コードベースで読み取りゼロの死にフラグ）も
// 削除対象に含める。移行後の JSON に use_* が一切残らないことで、
// クライアント側正規化の「レガシーキー検出 fast path」が効くようにする。
var legacyUseFlagKeys = []string{
	"use_tags",
	"use_reps",
	"use_rep_types",
	"use_ids",
	"use_include_id",
	"use_words",
	"use_timeis",
	"use_timeis_tags",
	"use_calendar",
	"use_map",
	"use_plaing",
	"use_update_time",
	"use_mi_board_name",
	"use_period_of_time",
	"use_mi_sort_type",
	"use_mi_check_state",
}

// MigrateLegacyFindQueryJSON は use_* フラグ入りの旧形式 FindQuery JSON を
// null 判定の新形式へ書き換える。
//
// JSON 全体を再帰走査し、use_* キーを1つ以上持つオブジェクトだけを変換するため、
// RYUU 設定のようにネストの奥へ FindQuery を抱える JSON にもそのまま使える。冪等。
//
// 変換規則:
//   - use_X=false → 対応する値フィールドを null にする（フィルタ未使用）
//   - use_X=true  → 値を維持する。配列系の値が null/欠落なら [] を、mi_board_name なら "" を
//     物質化して旧挙動（空指定=0件、空板名比較）を保存する
//   - use_timeis=false のときは timeis_tags も null にする
//     （旧ゲートは UseTimeIs && UseTimeIsTags の複合だったため）
//   - use_include_id / use_mi_sort_type / use_mi_check_state はキー削除のみ
//   - 最後に use_* キー自体を削除する
//
// 数値は json.Number のまま保持するため精度は落ちない。
// 旧形式が見つからなければ raw をそのまま返す（changed=false）。
func MigrateLegacyFindQueryJSON(raw []byte) (migrated []byte, changed bool, err error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return raw, false, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	err = decoder.Decode(&value)
	if err != nil {
		return nil, false, err
	}

	changed = migrateLegacyFindQueryValue(value)
	if !changed {
		return raw, false, nil
	}

	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	err = encoder.Encode(value)
	if err != nil {
		return nil, false, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), true, nil
}

// migrateLegacyFindQueryValue は JSON 値を再帰走査し、
// use_* キーを持つオブジェクトを見つけしだい変換する。変換が起きたら true を返す。
func migrateLegacyFindQueryValue(value any) bool {
	changed := false
	switch typedValue := value.(type) {
	case map[string]any:
		if hasLegacyUseFlagKey(typedValue) {
			migrateLegacyFindQueryObject(typedValue)
			changed = true
		}
		for _, child := range typedValue {
			if migrateLegacyFindQueryValue(child) {
				changed = true
			}
		}
	case []any:
		for _, child := range typedValue {
			if migrateLegacyFindQueryValue(child) {
				changed = true
			}
		}
	}
	return changed
}

func hasLegacyUseFlagKey(obj map[string]any) bool {
	for _, key := range legacyUseFlagKeys {
		if _, exist := obj[key]; exist {
			return true
		}
	}
	return false
}

// legacyFlagValue はフラグキーの値を返す。
// bool 以外（null 等）は旧サーバの json.Unmarshal と同じく false 相当として扱う。
func legacyFlagValue(obj map[string]any, key string) (enabled bool, has bool) {
	value, exist := obj[key]
	if !exist {
		return false, false
	}
	boolValue, _ := value.(bool)
	return boolValue, true
}

// materializeLegacyArray は値が null/欠落の配列フィールドに [] を物質化する。
// use_X=true のとき「有効だが空指定」（=0件などの旧挙動）を新形式でも保存するため。
func materializeLegacyArray(obj map[string]any, key string) {
	if value, exist := obj[key]; !exist || value == nil {
		obj[key] = []any{}
	}
}

// migrateLegacyFindQueryObject は use_* キーを持つオブジェクト1つを新形式へ変換する。
func migrateLegacyFindQueryObject(obj map[string]any) {
	// 配列系グループ: use=false で null、use=true で null/欠落を [] に物質化する。
	// hide_tags は触らない（サーバの適用ゲートが tags に従属するため挙動が変わらない。
	// クライアントは hide_tags を常に非nullで持つ）
	applyLegacyArrayGroup(obj, "use_words", "words", "not_words")
	applyLegacyArrayGroup(obj, "use_tags", "tags")
	applyLegacyArrayGroup(obj, "use_reps", "reps")
	applyLegacyArrayGroup(obj, "use_rep_types", "rep_types")
	applyLegacyArrayGroup(obj, "use_ids", "ids")

	// TimeIs グループ: 旧ゲートは UseTimeIs && UseTimeIsTags の複合。
	// use_timeis=false ならタグ側の値も null にして表現を揃える
	useTimeIs, hasUseTimeIs := legacyFlagValue(obj, "use_timeis")
	useTimeIsTags, hasUseTimeIsTags := legacyFlagValue(obj, "use_timeis_tags")
	if hasUseTimeIs {
		if useTimeIs {
			materializeLegacyArray(obj, "timeis_words")
			materializeLegacyArray(obj, "timeis_not_words")
		} else {
			obj["timeis_words"] = nil
			obj["timeis_not_words"] = nil
			obj["timeis_tags"] = nil
		}
	}
	if hasUseTimeIsTags && (useTimeIs || !hasUseTimeIs) {
		if useTimeIsTags {
			materializeLegacyArray(obj, "timeis_tags")
		} else {
			obj["timeis_tags"] = nil
		}
	}

	// use=false で値を null にするだけのグループ（値の物質化は不要）
	applyLegacyNullableGroup(obj, "use_calendar", "calendar_start_date", "calendar_end_date")
	applyLegacyNullableGroup(obj, "use_map", "map_radius", "map_latitude", "map_longitude")
	applyLegacyNullableGroup(obj, "use_update_time", "update_time")

	// Plaing: 旧サーバは「use_plaing=true かつ時刻ゼロ値/null は現在時刻」と解釈していた。
	// 新形式では表現できないため null のまま残す（永続化データでは実質未使用を確認済み）
	if usePlaing, has := legacyFlagValue(obj, "use_plaing"); has {
		if usePlaing {
			if value, exist := obj["plaing_time"]; !exist || value == nil {
				slog.Warn("旧形式FindQueryのuse_plaing=trueかつplaing_time未指定を検出しました。「現在時刻」の意味は新形式で保存できないため、フィルタ未使用(null)として移行します")
				obj["plaing_time"] = nil
			}
		} else {
			obj["plaing_time"] = nil
		}
	}

	// Mi板名: use=true で値が null/欠落なら "" を物質化（旧「空板名との等値比較」を保存）
	if useMiBoardName, has := legacyFlagValue(obj, "use_mi_board_name"); has {
		if useMiBoardName {
			if value, exist := obj["mi_board_name"]; !exist || value == nil {
				obj["mi_board_name"] = ""
			}
		} else {
			obj["mi_board_name"] = nil
		}
	}

	// 時間帯: use=true では week_of_days のみ [] を物質化（旧 nil→0件挙動の保存。
	// 新形式では nil=曜日制限なし / []=0件 と意味が分かれるため）。start/end はそのまま
	if usePeriodOfTime, has := legacyFlagValue(obj, "use_period_of_time"); has {
		if usePeriodOfTime {
			materializeLegacyArray(obj, "period_of_time_week_of_days")
		} else {
			obj["period_of_time_start_time_second"] = nil
			obj["period_of_time_end_time_second"] = nil
			obj["period_of_time_week_of_days"] = nil
		}
	}

	// use_include_id / use_mi_sort_type / use_mi_check_state は対応する値の変換なし
	// （use_include_id は両側で死にフィールド。mi_sort_type / mi_check_state の値は常時有効のまま残る）

	for _, key := range legacyUseFlagKeys {
		delete(obj, key)
	}
}

func applyLegacyArrayGroup(obj map[string]any, flagKey string, valueKeys ...string) {
	enabled, has := legacyFlagValue(obj, flagKey)
	if !has {
		return
	}
	for _, valueKey := range valueKeys {
		if enabled {
			materializeLegacyArray(obj, valueKey)
		} else {
			obj[valueKey] = nil
		}
	}
}

func applyLegacyNullableGroup(obj map[string]any, flagKey string, valueKeys ...string) {
	enabled, has := legacyFlagValue(obj, flagKey)
	if !has || enabled {
		return
	}
	for _, valueKey := range valueKeys {
		obj[valueKey] = nil
	}
}
