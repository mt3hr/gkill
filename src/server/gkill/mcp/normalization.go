package mcp

// 編集前に読む: .claude/skills/gkill-mcp/SKILL.md
//
// 読み取りツールの引数正規化（旧 normalization.mjs）。
// 応答と gkill へ送る本文を旧実装とバイト単位で揃えるため、
// 検証の順序・文言・正規化後のキー順をそのまま写している。

import (
	"regexp"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// Pad2 は String(value).padStart(2, "0")。
func Pad2(value any) string {
	return padStart(jsString(value), 2, "0")
}

// FormatLocalRfc3339 はローカル時刻の RFC 3339（秒精度、オフセット付き）。
func FormatLocalRfc3339(t time.Time) string {
	local := t.In(time.Local)
	_, offsetSeconds := local.Zone()
	offsetMinutes := offsetSeconds / 60
	sign := "+"
	if offsetMinutes < 0 {
		sign = "-"
		offsetMinutes = -offsetMinutes
	}
	return local.Format("2006-01-02T15:04:05") + sign + Pad2(offsetMinutes/60) + ":" + Pad2(offsetMinutes%60)
}

// NormalizeDateOnlyToRfc3339 は YYYY-MM-DD をその日の始まり（または終わり）のローカル RFC 3339 にする。
// 実在しない日付（2026-02-30 等）は false。
func NormalizeDateOnlyToRfc3339(value string, endOfDay bool) (string, bool) {
	match := DateOnlyRegex.FindString(value)
	if match == "" {
		return "", false
	}
	year := atoiDigits(match[0:4])
	month := atoiDigits(match[5:7])
	day := atoiDigits(match[8:10])
	var date time.Time
	if endOfDay {
		date = time.Date(year, time.Month(month), day, 23, 59, 59, 0, time.Local)
	} else {
		date = time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
	}
	if date.Year() != year || int(date.Month()) != month || date.Day() != day {
		return "", false
	}
	return FormatLocalRfc3339(date), true
}

func atoiDigits(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}

// ForMiDefaultProjectionNote は for_mi 単独の検索に include_create_mi を補ったときに warnings へ足す1行。
const ForMiDefaultProjectionNote = "query.for_mi was set without any include_*_mi flag, so include_create_mi:true was assumed (the five flags choose " +
	"which time projection supplies rows; with none of them a for_mi search returns nothing). Set include_check_mi / " +
	"include_limit_mi / include_start_mi / include_end_mi explicitly to look at other timestamps"

// オフセットの無い日時（2026-09-18T00:00:00 / 2026-09-18T00:00）。RFC 3339 としては不完全だが
// ISO-8601 としては妥当なので、足りないもの（オフセット）を名指しする。
var datetimeWithoutOffsetRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}(?::\d{2}(?:\.\d{1,9})?)?$`)

// DateTimeOptions は NormalizeDateTimeString の振る舞い。
type DateTimeOptions struct {
	AllowDateOnly bool
	EndOfDay      bool
}

// NormalizeDateTimeString は RFC 3339（オフセット必須）か、許せば YYYY-MM-DD を受けて RFC 3339 にする。
func NormalizeDateTimeString(value any, field string, opts DateTimeOptions) (string, error) {
	trimmed, err := AssertTrimmedString(value, field)
	if err != nil {
		return "", err
	}
	if RFC3339Regex.MatchString(trimmed) {
		if _, ok := jsDateParse(trimmed); ok {
			return trimmed, nil
		}
	}
	if opts.AllowDateOnly && DateOnlyRegex.MatchString(trimmed) {
		if normalized, ok := NormalizeDateOnlyToRfc3339(trimmed, opts.EndOfDay); ok {
			return normalized, nil
		}
	}
	if datetimeWithoutOffsetRegex.MatchString(trimmed) {
		dateOnlyPart := ""
		if opts.AllowDateOnly {
			dateOnlyPart = " or " + DateOnlyDesc + " (the day is expanded locally)"
		}
		return "", InvalidArgument(
			field,
			"has no timezone offset: append one (e.g. "+jsSlice(trimmed, 0, 10)+"T"+jsSlice(padEnd(jsSlice(trimmed, 11, 19), 8, ":00"), 0, 8)+"+09:00, or Z for UTC). "+
				"Accepted forms: "+ISODateTimeDesc+dateOnlyPart+"; seconds are required",
			value,
		)
	}
	allowedFormat := ISODateTimeDesc
	if opts.AllowDateOnly {
		allowedFormat = ISODateTimeDesc + " or " + DateOnlyDesc
	}
	return "", InvalidArgument(field, "must be "+allowedFormat, value)
}

// legacyUseFlagValueKeys は旧 use_X フラグと、そのフラグが束ねていた値キーの対応表。
var legacyUseFlagValueKeys = map[string][]string{
	"use_tags":           {"tags"},
	"use_reps":           {"reps"},
	"use_rep_types":      {"rep_types"},
	"use_ids":            {"ids"},
	"use_include_id":     {},
	"use_mi_sort_type":   {},
	"use_mi_check_state": {},
	"use_words":          {"words", "not_words"},
	"use_timeis":         {"timeis_words", "timeis_not_words", "timeis_tags"},
	"use_timeis_tags":    {"timeis_tags"},
	"use_calendar":       {"calendar_start_date", "calendar_end_date"},
	"use_map":            {"map_latitude", "map_longitude", "map_radius"},
	"use_playing":        {"playing_time"},
	"use_update_time":    {"update_time"},
	"use_mi_board_name":  {"mi_board_name"},
	"use_period_of_time": {"period_of_time_start_time_second", "period_of_time_end_time_second", "period_of_time_week_of_days"},
}

var noHiddenKeys = NewStringSet()

// NormalizeKyouQuery は gkill_get_kyous の query を検証して正規化する。
func NormalizeKyouQuery(query any) (*jsonobj.Object, error) {
	source, err := AssertObject(query, "query")
	if err != nil {
		return nil, err
	}
	normalized := jsonobj.New()
	disabledLegacyFlags := []string{}

	// 未知キーは全部集めて1回で投げる。廃止済み（only_latest_data / use_*）は受理するが allowed には載せない（ADR-0620）。
	if err := AssertKnownKeys(source, KyousQueryAllFields, "query", DeprecatedQueryFields.Union(LegacyUseFlagKeys)); err != nil {
		return nil, err
	}

	for _, key := range source.Keys() {
		value := source.Value(key)
		field := "query." + key
		// null はキー欠落と同義（フィルタ未使用）。
		if value == nil {
			continue
		}
		if LegacyUseFlagKeys.Has(key) {
			enabled, err := AssertBoolean(value, field)
			if err != nil {
				return nil, err
			}
			if !enabled {
				disabledLegacyFlags = append(disabledLegacyFlags, key)
			}
			continue
		}
		if KyousQueryBooleanFields.Has(key) {
			b, err := AssertBoolean(value, field)
			if err != nil {
				return nil, err
			}
			normalized.Set(key, b)
			continue
		}
		if KyousQueryStringArrayFields.Has(key) {
			items, err := AssertStringArray(value, field)
			if err != nil {
				return nil, err
			}
			normalized.Set(key, jsonobj.Strings(items...))
			continue
		}
		if KyousQueryNumberFields.Has(key) {
			f, err := AssertNumber(value, field, NumberRange{})
			if err != nil {
				return nil, err
			}
			normalized.Set(key, f)
			continue
		}
		if spec, ok := kyousQueryIntegerField(key); ok {
			n, err := AssertInteger(value, field, IntRange{Min: I(spec.Min), Max: I(spec.Max)})
			if err != nil {
				return nil, err
			}
			normalized.Set(key, n)
			continue
		}
		if spec, ok := kyousQueryDateTimeField(key); ok {
			if value == "" {
				continue // 未使用欄を "" で送るクライアントがある
			}
			if s, isString := value.(string); key == "playing_time" && isString && jsTrim(s) == "now" {
				normalized.Set(key, FormatLocalRfc3339(timeNow()))
				continue
			}
			normalizedValue, err := NormalizeDateTimeString(value, field, DateTimeOptions{AllowDateOnly: spec.AllowDateOnly, EndOfDay: spec.EndOfDay})
			if err != nil {
				return nil, err
			}
			normalized.Set(key, normalizedValue)
			continue
		}
		if key == "period_of_time_week_of_days" {
			items, err := AssertIntegerArray(value, field, IntRange{Min: I(0), Max: I(6)})
			if err != nil {
				return nil, err
			}
			normalized.Set(key, int64sToAny(items))
			continue
		}
		if key == "mi_board_name" {
			s, err := AssertTrimmedString(value, field)
			if err != nil {
				return nil, err
			}
			normalized.Set(key, s)
			continue
		}
		if key == "mi_check_state" {
			state, err := AssertTrimmedString(value, field)
			if err != nil {
				return nil, err
			}
			if !MiCheckStates.Has(state) {
				return nil, InvalidArgument(field, "must be one of: "+MiCheckStates.Join(", "), value)
			}
			normalized.Set(key, state)
			continue
		}
		if key == "mi_sort_type" {
			sortType, err := AssertTrimmedString(value, field)
			if err != nil {
				return nil, err
			}
			if !MiSortTypes.Has(sortType) {
				return nil, InvalidArgument(field, "must be one of: "+MiSortTypes.Join(", "), value)
			}
			normalized.Set(key, sortType)
			continue
		}
		// ここへ来るキーは無い（AssertKnownKeys がループの前に全部弾く）。万一の取りこぼしは同じ文言で。
		return nil, InvalidArgument(field, UnknownKeyMessage(), value, "allowed", jsonobj.Strings(KyousQueryAllFields.Sorted()...))
	}

	// 旧 use_X:false は「そのグループを使わない」だったので、束ねていた値キーを取り除く。
	for _, flagKey := range disabledLegacyFlags {
		for _, valueKey := range legacyUseFlagValueKeys[flagKey] {
			normalized.Delete(valueKey)
		}
	}

	// TimeIs タグだけの指定でもサーバ側のゲート（timeis_words の非 null 存在）を満たす。
	if normalized.Has("timeis_tags") && !normalized.Has("timeis_words") && !normalized.Has("timeis_not_words") {
		normalized.Set("timeis_words", jsonobj.Arr())
	}

	// 逆さまの期間は gkill 側で0件になるだけで警告も出ないので、入口で弾く。
	if normalized.Has("calendar_start_date") && normalized.Has("calendar_end_date") {
		start, _ := jsDateParseMillis(jsString(normalized.Value("calendar_start_date")))
		end, _ := jsDateParseMillis(jsString(normalized.Value("calendar_end_date")))
		if start > end {
			return nil, InvalidArgument(
				"query.calendar_start_date",
				"must not be after query.calendar_end_date ("+jsString(normalized.Value("calendar_end_date"))+"); an inverted range silently matches nothing",
				normalized.Value("calendar_start_date"),
			)
		}
	}

	if err := assertMapFilterComplete(normalized); err != nil {
		return nil, err
	}

	normalized.Set("only_latest_data", true)
	return normalized, nil
}

func int64sToAny(items []int64) []any {
	out := make([]any, len(items))
	for i, n := range items {
		out[i] = n
	}
	return out
}

// assertMapFilterComplete は地図条件の3値が揃っていることと値の範囲を検査する（ADR-0625）。
func assertMapFilterComplete(normalized *jsonobj.Object) error {
	fields := []string{"map_latitude", "map_longitude", "map_radius"}
	present := []string{}
	missing := []string{}
	for _, field := range fields {
		if normalized.Defined(field) {
			present = append(present, field)
		} else {
			missing = append(missing, field)
		}
	}
	if len(present) == 0 {
		return nil
	}
	if len(present) != len(fields) {
		qualified := make([]string, len(present))
		for i, field := range present {
			qualified[i] = "query." + field
		}
		return InvalidArgument(
			"query."+missing[0],
			"is required when "+strings.Join(qualified, " / ")+" is set: the map filter activates only "+
				"with all three of map_latitude, map_longitude and map_radius (meters); with any of them missing gkill silently "+
				"ignores the location condition and returns the unfiltered result. Missing: "+strings.Join(missing, ", "),
			jsonobj.Undefined,
		)
	}
	latitude, _ := jsonobj.ToFloat(normalized.Value("map_latitude"))
	longitude, _ := jsonobj.ToFloat(normalized.Value("map_longitude"))
	radius, _ := jsonobj.ToFloat(normalized.Value("map_radius"))
	if latitude < -90 || latitude > 90 {
		return InvalidArgument("query.map_latitude", "must be between -90 and 90 (degrees)", normalized.Value("map_latitude"))
	}
	if longitude < -180 || longitude > 180 {
		return InvalidArgument("query.map_longitude", "must be between -180 and 180 (degrees)", normalized.Value("map_longitude"))
	}
	if !(radius > 0) {
		return InvalidArgument(
			"query.map_radius",
			"must be greater than 0 (meters); gkill silently skips the map filter for a radius of 0 or less",
			normalized.Value("map_radius"),
		)
	}
	return nil
}

// parseCanonicalJSONValue は「正規の JSON 符号化の文字列」に限り本来の型へ復元する。曖昧な文字列は復元しない。
func parseCanonicalJSONValue(value string, kind string) (any, bool) {
	trimmed := jsTrim(value)
	if trimmed == "" {
		return nil, false
	}
	parsed, err := jsonobj.Unmarshal([]byte(trimmed))
	if err != nil {
		return nil, false
	}
	switch kind {
	case "boolean":
		if b, ok := parsed.(bool); ok {
			return b, true
		}
	case "number":
		if f, ok := jsonobj.ToFloat(parsed); ok && !isInfOrNaN(f) {
			if _, isBool := parsed.(bool); !isBool {
				if _, isString := parsed.(string); !isString {
					return parsed, true
				}
			}
		}
	case "string_array":
		if items, ok := jsonobj.AsArray(parsed); ok {
			for _, item := range items {
				if _, isString := item.(string); !isString {
					return nil, false
				}
			}
			return items, true
		}
	case "object_array":
		if items, ok := jsonobj.AsArray(parsed); ok {
			for _, item := range items {
				if !IsPlainObject(item) {
					return nil, false
				}
			}
			return items, true
		}
	}
	return nil, false
}

// reviveStaleSchemaArgs は救済表に載った引数が文字列で届いていたら型を復元した複製を返す（無ければそのまま）。
func reviveStaleSchemaArgs(source *jsonobj.Object, kinds []StaleSchemaArgKind) *jsonobj.Object {
	var revived *jsonobj.Object
	for _, entry := range kinds {
		if !source.Has(entry.Key) {
			continue
		}
		value, isString := source.Value(entry.Key).(string)
		if !isString {
			continue
		}
		parsed, ok := parseCanonicalJSONValue(value, entry.Kind)
		if !ok {
			continue
		}
		if revived == nil {
			revived = source.Clone()
		}
		revived.Set(entry.Key, parsed)
	}
	if revived == nil {
		return source
	}
	return revived
}

func kinds(pairs ...string) []StaleSchemaArgKind {
	out := make([]StaleSchemaArgKind, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		out = append(out, StaleSchemaArgKind{Key: pairs[i], Kind: pairs[i+1]})
	}
	return out
}

// StaleSchemaArgKind は「後付け引数の名前と、古いスキーマから文字列で届いたときの型」。
type StaleSchemaArgKind struct {
	Key  string
	Kind string
}

// StaleSchemaToolKinds はツール1本ぶんの救済表。
type StaleSchemaToolKinds struct {
	Tool  string
	Kinds []StaleSchemaArgKind
}

// v2 (ADR-0604) で追加したトップレベル引数。string 型の group_by / cursor は旧スキーマ経由でも壊れないので対象外。
var kyousStaleSchemaArgKinds = kinds(
	"count_only", "boolean",
	"data_types", "string_array",
	"create_apps", "string_array",
	"update_apps", "string_array",
	"num_min", "number",
	"num_max", "number",
	"idf_kinds", "string_array",
	"include_file_size", "boolean",
	"include_attached_ids", "boolean",
	"include_file_urls", "boolean",
)

var gpsStaleSchemaArgKinds = kinds("limit", "number", "count_only", "boolean")

var appConfigStaleSchemaArgKinds = kinds("fields", "string_array", "include_ui_state", "boolean", "compact", "boolean", "max_size_mb", "number")

var idfStaleSchemaArgKinds = kinds("is_video", "boolean")

var repNamesStaleSchemaArgKinds = kinds("limit", "number")

var tagNamesStaleSchemaArgKinds = kinds("limit", "number")

var repInfosStaleSchemaArgKinds = kinds(
	"fields", "string_array",
	"data_kinds", "string_array",
	"writable_only", "boolean",
	"rep_types", "string_array",
	"rep_names", "string_array",
)

var deleteTargetsStaleSchemaArgKinds = kinds("targets", "object_array")

var kyouHistoryStaleSchemaArgKinds = kinds("limit", "number", "offset", "number")

var addUrlogStaleSchemaArgKinds = kinds("fetch_metadata", "boolean", "fetch_favicon", "boolean")

var miBoardGuardStaleSchemaArgKinds = kinds("allow_create_board", "boolean")

// StaleSchemaArgKindsByTool はツール名から救済表を引く表。
// **スキーマへ非 string 型の引数を足したら、対応する表とここの両方へ載せること。**
var StaleSchemaArgKindsByTool = []StaleSchemaToolKinds{
	{Tool: "gkill_get_kyous", Kinds: kyousStaleSchemaArgKinds},
	{Tool: "gkill_get_gps_log", Kinds: gpsStaleSchemaArgKinds},
	{Tool: "gkill_get_application_config", Kinds: appConfigStaleSchemaArgKinds},
	{Tool: "gkill_get_idf_file", Kinds: idfStaleSchemaArgKinds},
	{Tool: "gkill_get_all_rep_names", Kinds: repNamesStaleSchemaArgKinds},
	{Tool: "gkill_get_all_tag_names", Kinds: tagNamesStaleSchemaArgKinds},
	{Tool: "gkill_get_rep_infos", Kinds: repInfosStaleSchemaArgKinds},
	{Tool: "gkill_get_kyou_history", Kinds: kyouHistoryStaleSchemaArgKinds},
	{Tool: "gkill_delete_kyou", Kinds: deleteTargetsStaleSchemaArgKinds},
	{Tool: "gkill_restore_kyou", Kinds: deleteTargetsStaleSchemaArgKinds},
	{Tool: "gkill_add_urlog", Kinds: addUrlogStaleSchemaArgKinds},
	{Tool: "gkill_add_mi", Kinds: miBoardGuardStaleSchemaArgKinds},
	{Tool: "gkill_update_mi", Kinds: miBoardGuardStaleSchemaArgKinds},
}

func staleSchemaArgKindsFor(tool string) ([]StaleSchemaArgKind, bool) {
	for _, entry := range StaleSchemaArgKindsByTool {
		if entry.Tool == tool {
			return entry.Kinds, true
		}
	}
	return nil, false
}

// DeprecatedTopLevelArgs は gkill_get_kyous の廃止済みトップレベル引数（受理はするが公開しない。ADR-0620）。
var DeprecatedTopLevelArgs = NewStringSet("include_id", "include_rep_name")

// DeprecatedQueryFields は query の廃止済みキー（受理はするが公開しない）。
var DeprecatedQueryFields = NewStringSet("only_latest_data")

// StaleSchemaSignals は「古いツールスキーマで呼ばれた証拠」。
type StaleSchemaSignals struct {
	Revived    []string
	Deprecated []string
}

// DetectStaleSchemaSignals は args に古スキーマの痕跡があれば返す。無ければ nil。
func DetectStaleSchemaSignals(name string, args any) *StaleSchemaSignals {
	if !IsPlainObject(args) {
		return nil
	}
	source := args.(*jsonobj.Object)
	revived := []string{}
	if kindsByKey, ok := staleSchemaArgKindsFor(name); ok {
		for _, entry := range kindsByKey {
			if !source.Has(entry.Key) {
				continue
			}
			value, isString := source.Value(entry.Key).(string)
			if !isString {
				continue
			}
			if _, ok := parseCanonicalJSONValue(value, entry.Kind); ok {
				revived = append(revived, entry.Key)
			}
		}
	}
	deprecated := []string{}
	for _, key := range DeprecatedTopLevelArgs.Values() {
		if source.Has(key) {
			deprecated = append(deprecated, key)
		}
	}
	if query, ok := source.Object("query"); ok && query != nil {
		for _, key := range query.Keys() {
			if DeprecatedQueryFields.Has(key) || LegacyUseFlagKeys.Has(key) {
				deprecated = append(deprecated, "query."+key)
			}
		}
	}
	if len(revived) == 0 && len(deprecated) == 0 {
		return nil
	}
	return &StaleSchemaSignals{Revived: revived, Deprecated: deprecated}
}

// StaleSchemaWarning は DetectStaleSchemaSignals の結果を1行の警告文にする。
func StaleSchemaWarning(signals *StaleSchemaSignals) string {
	evidence := []string{}
	if len(signals.Revived) != 0 {
		evidence = append(evidence, "arguments arrived as JSON strings: "+strings.Join(signals.Revived, ", "))
	}
	if len(signals.Deprecated) != 0 {
		evidence = append(evidence, "deprecated arguments were sent: "+strings.Join(signals.Deprecated, ", "))
	}
	return "this MCP client's tool schema snapshot looks stale (" + strings.Join(evidence, "; ") + "). " +
		"Tool schemas are fetched once per client session, so a server-side fix stays invisible until the " +
		"client reconnects. Reconnect the MCP client to pick up the current schema — features you may not be " +
		"seeing include gkill_get_gps_log cursor/limit/count_only/group_by, the top-level data_types filter " +
		"on gkill_get_kyous, and the fields projection on gkill_get_application_config. " +
		"gkill_status returns the server's current schema_revision; the gkill_status description in your tool list " +
		"ends with the revision you were given, so the two can be compared directly."
}

// AppendStaleSchemaWarning は古いスキーマの証拠があるときだけ payload.warnings へ1行足す（複製を返す）。
func AppendStaleSchemaWarning(payload *jsonobj.Object, name string, args any) *jsonobj.Object {
	if payload == nil {
		return payload
	}
	signals := DetectStaleSchemaSignals(name, args)
	if signals == nil {
		return payload
	}
	warnings := []any{}
	if existing, ok := payload.Array("warnings"); ok {
		warnings = append(warnings, existing...)
	}
	warnings = append(warnings, StaleSchemaWarning(signals))
	out := payload.Clone()
	out.Set("warnings", warnings)
	return out
}

// argsObject は `args == null ? {} : assertObject(args, "arguments")`。
func argsObject(args any) (*jsonobj.Object, error) {
	if args == nil || jsonobj.IsUndefined(args) {
		return jsonobj.New(), nil
	}
	return AssertObject(args, "arguments")
}

// NormalizeKyouArgs は gkill_get_kyous の引数。
func NormalizeKyouArgs(args any) (*jsonobj.Object, error) {
	base, err := argsObject(args)
	if err != nil {
		return nil, err
	}
	source := reviveStaleSchemaArgs(base, kyousStaleSchemaArgKinds)
	if err := AssertKnownKeys(source, KyousTopLevelFields, "arguments", DeprecatedTopLevelArgs); err != nil {
		return nil, err
	}

	var queryInput any = jsonobj.New()
	if source.Has("query") {
		queryInput = source.Value("query")
	}
	query, err := NormalizeKyouQuery(queryInput)
	if err != nil {
		return nil, err
	}
	normalized := jsonobj.Obj(
		"query", query,
		"limit", int64(DefaultKyousLimit),
		"max_size_mb", float64(DefaultKyousMaxSizeMB),
		"is_include_timeis", DefaultKyousIncludeTimeIs,
		"include_plugin_content", DefaultIncludePluginContent,
		"plugin_content_max_text_length", int64(DefaultInlinePluginContentMaxTextLength),
		"plugin_content_format", DefaultPluginContentFormat,
		"include_attached_ids", false,
		"include_file_urls", false,
	)

	// for_mi:true で include_*_mi が1つも立っていない検索は必ず0件なので include_create_mi を補う（ADR-0627）。
	if query.Value("for_mi") == true {
		anyFlag := false
		for _, field := range MiProjectionFlagFields {
			if query.Value(field) == true {
				anyFlag = true
				break
			}
		}
		if !anyFlag {
			query.Set("include_create_mi", true)
			normalized.Set("notes", jsonobj.Strings(ForMiDefaultProjectionNote))
		}
	}

	if source.Defined("locale_name") {
		s, err := AssertTrimmedString(source.Value("locale_name"), "locale_name")
		if err != nil {
			return nil, err
		}
		normalized.Set("locale_name", s)
	}
	if source.Defined("limit") {
		n, err := AssertInteger(source.Value("limit"), "limit", IntRange{Min: I(1), Max: I(1000)})
		if err != nil {
			return nil, err
		}
		normalized.Set("limit", n)
	}
	if source.Defined("cursor") {
		cursor, err := AssertTrimmedString(source.Value("cursor"), "cursor")
		if err != nil {
			return nil, err
		}
		if jsLength(cursor) > MaxCursorLength {
			return nil, Errorf("cursor is too long (%d > %d)", jsLength(cursor), MaxCursorLength)
		}
		cursorTime := cursor
		if idx := strings.Index(cursor, "::"); idx >= 0 {
			cursorTime = cursor[:idx]
		}
		if _, ok := jsDateParse(cursorTime); !ok {
			return nil, InvalidArgument(
				"cursor",
				"must be the next_cursor value from a previous response, passed back verbatim "+
					`(an RFC3339 time, optionally followed by "::" and the entry id)`,
				source.Value("cursor"),
			)
		}
		normalized.Set("cursor", cursor)
	}
	if source.Defined("max_size_mb") {
		f, err := AssertNumber(source.Value("max_size_mb"), "max_size_mb", NumberRange{MinExclusive: F(0)})
		if err != nil {
			return nil, err
		}
		normalized.Set("max_size_mb", f)
	}
	if source.Defined("is_include_timeis") {
		b, err := AssertBoolean(source.Value("is_include_timeis"), "is_include_timeis")
		if err != nil {
			return nil, err
		}
		normalized.Set("is_include_timeis", b)
	}
	// include_id / include_rep_name は v2 で廃止。受理はするが値は使わない（型検証だけ行う）。
	if source.Defined("include_id") {
		if _, err := AssertBoolean(source.Value("include_id"), "include_id"); err != nil {
			return nil, err
		}
	}
	if source.Defined("include_rep_name") {
		if _, err := AssertBoolean(source.Value("include_rep_name"), "include_rep_name"); err != nil {
			return nil, err
		}
	}
	// ---- v2 (ADR-0604) ----
	if source.Defined("count_only") {
		b, err := AssertBoolean(source.Value("count_only"), "count_only")
		if err != nil {
			return nil, err
		}
		normalized.Set("count_only", b)
	}
	if source.Defined("group_by") {
		groupBy, err := AssertTrimmedString(source.Value("group_by"), "group_by")
		if err != nil {
			return nil, err
		}
		if !KyousGroupByValues.Has(groupBy) {
			return nil, Errorf("Invalid group_by: %s (valid: %s)", jsonobj.MarshalString(groupBy), KyousGroupByValues.Join(", "))
		}
		normalized.Set("group_by", groupBy)
	}
	for _, key := range []string{"create_apps", "update_apps", "data_types"} {
		if source.Defined(key) {
			items, err := AssertStringArray(source.Value(key), key)
			if err != nil {
				return nil, err
			}
			normalized.Set(key, jsonobj.Strings(items...))
		}
	}
	for _, key := range []string{"num_min", "num_max"} {
		if source.Defined(key) {
			f, err := AssertNumber(source.Value(key), key, NumberRange{})
			if err != nil {
				return nil, err
			}
			normalized.Set(key, f)
		}
	}
	if source.Defined("idf_kinds") {
		idfKinds, err := AssertStringArray(source.Value("idf_kinds"), "idf_kinds")
		if err != nil {
			return nil, err
		}
		for _, kind := range idfKinds {
			if !KyousIdfKindValues.Has(kind) {
				return nil, Errorf("Invalid idf_kind: %s (valid: %s)", jsonobj.MarshalString(kind), KyousIdfKindValues.Join(", "))
			}
		}
		normalized.Set("idf_kinds", jsonobj.Strings(idfKinds...))
	}
	for _, key := range []string{"include_file_size", "include_attached_ids", "include_file_urls", "include_plugin_content"} {
		if source.Defined(key) {
			b, err := AssertBoolean(source.Value(key), key)
			if err != nil {
				return nil, err
			}
			normalized.Set(key, b)
		}
	}
	if source.Defined("plugin_content_max_text_length") {
		n, err := AssertInteger(source.Value("plugin_content_max_text_length"), "plugin_content_max_text_length", IntRange{Min: I(1), Max: I(MaxPluginContentMaxTextLength)})
		if err != nil {
			return nil, err
		}
		normalized.Set("plugin_content_max_text_length", n)
	}
	if source.Defined("plugin_content_format") {
		format, err := AssertTrimmedString(source.Value("plugin_content_format"), "plugin_content_format")
		if err != nil {
			return nil, err
		}
		format = strings.ToLower(format)
		if !PluginContentFormats.Has(format) {
			return nil, InvalidArgument("plugin_content_format", "must be one of: "+PluginContentFormats.Join(", "), source.Value("plugin_content_format"))
		}
		normalized.Set("plugin_content_format", format)
	}

	if err := AssertAggregationNotCombinedWithCursor(normalized); err != nil {
		return nil, err
	}
	return normalized, nil
}

// AssertAggregationNotCombinedWithCursor は count_only / group_by と cursor の併用、
// および count_only と group_by の併用を弾く（get_kyous と GPS で同じ規則。ADR-0611）。
func AssertAggregationNotCombinedWithCursor(args *jsonobj.Object) error {
	if args == nil {
		return nil
	}
	if jsTruthy(args.Value("cursor")) {
		for _, field := range []string{"count_only", "group_by"} {
			if jsTruthy(args.Value(field)) {
				return InvalidArgument(
					field,
					"cannot be combined with cursor: it counts everything the query matches, "+
						"while a cursor resumes partway through. Drop the cursor to aggregate, "+
						"or drop count_only/group_by to page",
					args.Value(field),
				)
			}
		}
	}
	if jsTruthy(args.Value("count_only")) && jsTruthy(args.Value("group_by")) {
		return InvalidArgument(
			"count_only",
			"cannot be combined with group_by: group_by already returns only counts "+
				"(buckets plus total_count, no entries). Drop count_only to get the buckets, "+
				"or drop group_by to get the single total",
			args.Value("count_only"),
		)
	}
	return nil
}

// setTrimmedIfDefined は `if (hasOwnProperty(key) && value !== undefined) normalized[key] = assertTrimmedString(...)`。
func setTrimmedIfDefined(normalized, source *jsonobj.Object, key string) error {
	if !source.Defined(key) {
		return nil
	}
	s, err := AssertTrimmedString(source.Value(key), key)
	if err != nil {
		return err
	}
	normalized.Set(key, s)
	return nil
}

func setBooleanIfDefined(normalized, source *jsonobj.Object, key string) error {
	if !source.Defined(key) {
		return nil
	}
	b, err := AssertBoolean(source.Value(key), key)
	if err != nil {
		return err
	}
	normalized.Set(key, b)
	return nil
}

func setStringArrayIfDefined(normalized, source *jsonobj.Object, key string) error {
	if !source.Defined(key) {
		return nil
	}
	items, err := AssertStringArray(source.Value(key), key)
	if err != nil {
		return err
	}
	normalized.Set(key, jsonobj.Strings(items...))
	return nil
}

// NormalizeRepInfosArgs は gkill_get_rep_infos の引数を検証する。
func NormalizeRepInfosArgs(args any) (*jsonobj.Object, error) {
	base, err := argsObject(args)
	if err != nil {
		return nil, err
	}
	source := reviveStaleSchemaArgs(base, repInfosStaleSchemaArgKinds)
	if err := AssertKnownKeys(source, NewStringSet("locale_name", "fields", "data_kinds", "writable_only", "rep_types", "rep_names", "contains"), "arguments", noHiddenKeys); err != nil {
		return nil, err
	}
	normalized := jsonobj.New()
	if err := setTrimmedIfDefined(normalized, source, "locale_name"); err != nil {
		return nil, err
	}
	if source.Defined("fields") {
		fields, err := AssertStringArray(source.Value("fields"), "fields")
		if err != nil {
			return nil, err
		}
		for _, field := range fields {
			if !RepInfosFields.Has(field) {
				return nil, InvalidArgument("fields", "must be one of: "+RepInfosFields.Join(", "), field)
			}
		}
		normalized.Set("fields", jsonobj.Strings(fields...))
	}
	if err := setBooleanIfDefined(normalized, source, "writable_only"); err != nil {
		return nil, err
	}
	if err := setStringArrayIfDefined(normalized, source, "rep_types"); err != nil {
		return nil, err
	}
	if err := setStringArrayIfDefined(normalized, source, "rep_names"); err != nil {
		return nil, err
	}
	if err := setTrimmedIfDefined(normalized, source, "contains"); err != nil {
		return nil, err
	}
	if source.Defined("data_kinds") {
		dataKinds, err := AssertStringArray(source.Value("data_kinds"), "data_kinds")
		if err != nil {
			return nil, err
		}
		for _, dataKind := range dataKinds {
			if !AttachedDataKinds.Has(dataKind) {
				return nil, InvalidArgument("data_kinds", "must be one of: "+AttachedDataKinds.Join(", "), dataKind)
			}
		}
		normalized.Set("data_kinds", jsonobj.Strings(dataKinds...))
	}
	return normalized, nil
}

// normalizeNameListArgs は名前一覧ツール（rep 名 / タグ名）の共通の引数検証。
func normalizeNameListArgs(args any, staleKinds []StaleSchemaArgKind, defaultLimit, maxLimit int64) (*jsonobj.Object, error) {
	base, err := argsObject(args)
	if err != nil {
		return nil, err
	}
	source := reviveStaleSchemaArgs(base, staleKinds)
	if err := AssertKnownKeys(source, NewStringSet("locale_name", "contains", "limit"), "arguments", noHiddenKeys); err != nil {
		return nil, err
	}
	normalized := jsonobj.Obj("limit", defaultLimit)
	if err := setTrimmedIfDefined(normalized, source, "locale_name"); err != nil {
		return nil, err
	}
	if err := setTrimmedIfDefined(normalized, source, "contains"); err != nil {
		return nil, err
	}
	if source.Defined("limit") {
		n, err := AssertInteger(source.Value("limit"), "limit", IntRange{Min: I(1), Max: I(maxLimit)})
		if err != nil {
			return nil, err
		}
		normalized.Set("limit", n)
	}
	return normalized, nil
}

// NormalizeRepNamesArgs は gkill_get_all_rep_names の引数を検証する。
func NormalizeRepNamesArgs(args any) (*jsonobj.Object, error) {
	return normalizeNameListArgs(args, repNamesStaleSchemaArgKinds, DefaultRepNamesLimit, MaxRepNamesLimit)
}

// NormalizeTagNamesArgs は gkill_get_all_tag_names の引数を検証する。
func NormalizeTagNamesArgs(args any) (*jsonobj.Object, error) {
	return normalizeNameListArgs(args, tagNamesStaleSchemaArgKinds, DefaultTagNamesLimit, MaxTagNamesLimit)
}

// NormalizeMcpHelpArgs は gkill_get_mcp_help の引数を検証する。topic は省略可（index）。
func NormalizeMcpHelpArgs(args any) (*jsonobj.Object, error) {
	source, err := argsObject(args)
	if err != nil {
		return nil, err
	}
	if err := AssertKnownKeys(source, NewStringSet("topic"), "arguments", noHiddenKeys); err != nil {
		return nil, err
	}
	normalized := jsonobj.New()
	if source.Defined("topic") && source.Value("topic") != nil {
		topic, err := AssertTrimmedString(source.Value("topic"), "topic")
		if err != nil {
			return nil, err
		}
		if !containsStringItem(HelpTopicNames, topic) {
			return nil, InvalidArgument("topic", "must be one of: "+strings.Join(HelpTopicNames, ", "), topic)
		}
		normalized.Set("topic", topic)
	}
	return normalized, nil
}

// NormalizeStatusArgs は gkill_status の引数を検証する。引数は1つも取らない。
func NormalizeStatusArgs(args any) (*jsonobj.Object, error) {
	source, err := argsObject(args)
	if err != nil {
		return nil, err
	}
	if err := AssertKnownKeys(source, NewStringSet(), "arguments", noHiddenKeys); err != nil {
		return nil, err
	}
	return jsonobj.New(), nil
}

// NormalizeLocaleOnlyArgs は locale_name だけを受けるツールの引数。
func NormalizeLocaleOnlyArgs(args any) (*jsonobj.Object, error) {
	source, err := argsObject(args)
	if err != nil {
		return nil, err
	}
	if err := AssertKnownKeys(source, NewStringSet("locale_name"), "arguments", noHiddenKeys); err != nil {
		return nil, err
	}
	if !source.Defined("locale_name") {
		return jsonobj.New(), nil
	}
	s, err := AssertTrimmedString(source.Value("locale_name"), "locale_name")
	if err != nil {
		return nil, err
	}
	return jsonobj.Obj("locale_name", s), nil
}

// NormalizeGpsArgs は gkill_get_gps_log の引数を検証する。
func NormalizeGpsArgs(args any) (*jsonobj.Object, error) {
	base, err := argsObject(args)
	if err != nil {
		return nil, err
	}
	source := reviveStaleSchemaArgs(base, gpsStaleSchemaArgKinds)
	if err := AssertKnownKeys(source, NewStringSet("start_date", "end_date", "locale_name", "limit", "cursor", "count_only", "group_by"), "arguments", noHiddenKeys); err != nil {
		return nil, err
	}
	startDate, err := NormalizeDateTimeString(source.Value("start_date"), "start_date", DateTimeOptions{AllowDateOnly: true, EndOfDay: false})
	if err != nil {
		return nil, err
	}
	endDate, err := NormalizeDateTimeString(source.Value("end_date"), "end_date", DateTimeOptions{AllowDateOnly: true, EndOfDay: true})
	if err != nil {
		return nil, err
	}
	normalized := jsonobj.Obj("start_date", startDate, "end_date", endDate)
	if err := setTrimmedIfDefined(normalized, source, "locale_name"); err != nil {
		return nil, err
	}
	start, _ := jsDateParseMillis(startDate)
	end, _ := jsDateParseMillis(endDate)
	if start > end {
		return nil, InvalidArgument("start_date", "must not be after end_date ("+endDate+"); an inverted range silently matches nothing", startDate)
	}
	normalized.Set("limit", int64(DefaultGpsLimit))
	if source.Defined("limit") {
		n, err := AssertInteger(source.Value("limit"), "limit", IntRange{Min: I(1), Max: I(MaxGpsLimit)})
		if err != nil {
			return nil, err
		}
		normalized.Set("limit", n)
	}
	if source.Defined("cursor") {
		cursor, err := AssertTrimmedString(source.Value("cursor"), "cursor")
		if err != nil {
			return nil, err
		}
		if jsLength(cursor) > MaxCursorLength {
			return nil, Errorf("cursor is too long (%d > %d)", jsLength(cursor), MaxCursorLength)
		}
		// GPS カーソルは get_kyous とは別方式（base64url の JSON {t,n}）。判定は発行側と同じ gps_cursor.go。
		if !IsValidGpsCursor(cursor) {
			return nil, InvalidArgument(
				"cursor",
				"must be the next_cursor value from a previous gkill_get_gps_log response, passed back verbatim "+
					"(an opaque token — do not construct or edit it)",
				source.Value("cursor"),
			)
		}
		normalized.Set("cursor", cursor)
	}
	if err := setBooleanIfDefined(normalized, source, "count_only"); err != nil {
		return nil, err
	}
	if source.Defined("group_by") {
		groupBy, err := AssertTrimmedString(source.Value("group_by"), "group_by")
		if err != nil {
			return nil, err
		}
		if !GpsGroupByValues.Has(groupBy) {
			return nil, Errorf("Invalid group_by: %s (valid: %s)", jsonobj.MarshalString(groupBy), GpsGroupByValues.Join(", "))
		}
		normalized.Set("group_by", groupBy)
	}
	return normalized, nil
}

// NormalizeAppConfigArgs は gkill_get_application_config の引数を検証する。
func NormalizeAppConfigArgs(args any) (*jsonobj.Object, error) {
	base, err := argsObject(args)
	if err != nil {
		return nil, err
	}
	source := reviveStaleSchemaArgs(base, appConfigStaleSchemaArgKinds)
	if err := AssertKnownKeys(source, NewStringSet("locale_name", "fields", "include_ui_state", "compact", "contains", "max_size_mb"), "arguments", noHiddenKeys); err != nil {
		return nil, err
	}
	normalized := jsonobj.New()
	if err := setTrimmedIfDefined(normalized, source, "locale_name"); err != nil {
		return nil, err
	}
	if source.Defined("fields") {
		fields, err := AssertStringArray(source.Value("fields"), "fields")
		if err != nil {
			return nil, err
		}
		for _, field := range fields {
			if !AppConfigFields.Has(field) {
				return nil, Errorf("Invalid field: %s (valid: %s)", jsonobj.MarshalString(field), AppConfigFields.Join(", "))
			}
		}
		normalized.Set("fields", jsonobj.Strings(fields...))
	}
	normalized.Set("compact", true)
	if err := setBooleanIfDefined(normalized, source, "compact"); err != nil {
		return nil, err
	}
	if err := setTrimmedIfDefined(normalized, source, "contains"); err != nil {
		return nil, err
	}
	normalized.Set("max_size_mb", float64(DefaultKyousMaxSizeMB))
	if source.Defined("max_size_mb") {
		f, err := AssertNumber(source.Value("max_size_mb"), "max_size_mb", NumberRange{MinExclusive: F(0)})
		if err != nil {
			return nil, err
		}
		normalized.Set("max_size_mb", f)
	}
	normalized.Set("include_ui_state", false)
	if err := setBooleanIfDefined(normalized, source, "include_ui_state"); err != nil {
		return nil, err
	}
	return normalized, nil
}

// NormalizeIdfFileArgs は gkill_get_idf_file の引数を検証する。
func NormalizeIdfFileArgs(args any) (*jsonobj.Object, error) {
	base, err := argsObject(args)
	if err != nil {
		return nil, err
	}
	source := reviveStaleSchemaArgs(base, idfStaleSchemaArgKinds)
	if err := AssertKnownKeys(source, NewStringSet("rep_name", "file_name", "thumb", "is_video", "locale_name"), "arguments", noHiddenKeys); err != nil {
		return nil, err
	}
	repName, err := AssertTrimmedString(source.Value("rep_name"), "rep_name")
	if err != nil {
		return nil, err
	}
	fileName, err := AssertTrimmedString(source.Value("file_name"), "file_name")
	if err != nil {
		return nil, err
	}
	normalized := jsonobj.Obj("rep_name", repName, "file_name", fileName)
	if source.Defined("thumb") {
		thumb, err := AssertTrimmedString(source.Value("thumb"), "thumb")
		if err != nil {
			return nil, err
		}
		if !ThumbQueryRegex.MatchString(thumb) {
			return nil, InvalidArgument("thumb", `must be "<width>x<height>", e.g. "1024x1024"`, source.Value("thumb"))
		}
		parts := strings.SplitN(thumb, "x", 2)
		if atoiDigits(parts[0]) > MaxThumbSize || atoiDigits(parts[1]) > MaxThumbSize {
			return nil, InvalidArgument("thumb", "must not exceed "+itoa(MaxThumbSize)+" per side (larger values make the server return the original instead)", source.Value("thumb"))
		}
		normalized.Set("thumb", thumb)
	}
	if err := setBooleanIfDefined(normalized, source, "is_video"); err != nil {
		return nil, err
	}
	if normalized.Value("is_video") == true && !normalized.Defined("thumb") {
		return nil, InvalidArgument("is_video", "requires thumb (a video frame is only extracted when a thumbnail size is given)", source.Value("is_video"))
	}
	if err := setTrimmedIfDefined(normalized, source, "locale_name"); err != nil {
		return nil, err
	}
	return normalized, nil
}

// NormalizeKyouHistoryArgs は gkill_get_kyou_history の引数を検証する。data_type は必須。
func NormalizeKyouHistoryArgs(args any) (*jsonobj.Object, error) {
	base, err := AssertObjectAllowUndefined(args, "arguments")
	if err != nil {
		return nil, err
	}
	if base == nil {
		base = jsonobj.New()
	}
	source := reviveStaleSchemaArgs(base, kyouHistoryStaleSchemaArgKinds)
	if err := AssertKnownKeys(source, NewStringSet("id", "data_type", "limit", "offset", "locale_name"), "arguments", noHiddenKeys); err != nil {
		return nil, err
	}
	id, err := AssertTrimmedString(source.Value("id"), "id")
	if err != nil {
		return nil, err
	}
	rawDataType, err := AssertTrimmedString(source.Value("data_type"), "data_type")
	if err != nil {
		return nil, err
	}
	dataType := ToEntityDataType(rawDataType)
	if !containsStringItem(EntityDataTypeValues, dataType) {
		return nil, InvalidArgument("data_type", "must be one of: "+strings.Join(EntityDataTypeValues, ", "), dataType)
	}
	limit := int64(DefaultKyouHistoryLimit)
	if source.Defined("limit") {
		limit, err = AssertInteger(source.Value("limit"), "limit", IntRange{Min: I(1), Max: I(MaxKyouHistoryLimit)})
		if err != nil {
			return nil, err
		}
	}
	offset := int64(0)
	if source.Defined("offset") {
		offset, err = AssertInteger(source.Value("offset"), "offset", IntRange{Min: I(0)})
		if err != nil {
			return nil, err
		}
	}
	var localeName any = jsonobj.Undefined
	if source.Defined("locale_name") {
		localeName, err = AssertTrimmedString(source.Value("locale_name"), "locale_name")
		if err != nil {
			return nil, err
		}
	}
	return jsonobj.Obj("id", id, "data_type", dataType, "limit", limit, "offset", offset, "locale_name", localeName), nil
}

func containsStringItem(items []string, needle string) bool {
	for _, item := range items {
		if item == needle {
			return true
		}
	}
	return false
}

func isInfOrNaN(f float64) bool {
	return f != f || f > 1.7976931348623157e308 || f < -1.7976931348623157e308
}
