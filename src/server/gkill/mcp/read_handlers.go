package mcp

// 編集前に読む: .claude/skills/gkill-mcp/SKILL.md
//
// 読み取りツールのディスパッチと要約（旧 read-handlers.mjs）。read / readwrite / write の3サーバが共有する。
// 実装はこの1箇所が正本で、サーバ側は IsReadToolName / HandleReadToolCall / SummarizeReadToolPayload へ委譲するだけ。

import (
	"encoding/base64"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// IsReadToolName は name が読み取りツールか。
func IsReadToolName(name string) bool {
	return findTool(ReadTools, name) != nil
}

// HandleReadToolCall は読み取りツール1件を処理する。
// ディスパッチ本体を包んで、古いツールスキーマを掴んだクライアントへの警告を**1箇所で**足す。
func HandleReadToolCall(ctx *CallContext, name string, args any) (*jsonobj.Object, error) {
	payload, err := dispatchReadToolCall(ctx, name, args)
	if err != nil {
		return nil, err
	}
	return AppendStaleSchemaWarning(payload, name, args), nil
}

func (c *CallContext) callApi(pathname string, body *jsonobj.Object) (*jsonobj.Object, error) {
	return c.Client.CallApi(c.context(), pathname, body, true, c.SID)
}

// arrayOrEmpty は `Array.isArray(v) ? v : []`。
func arrayOrEmpty(v any) []any {
	if items, ok := jsonobj.AsArray(v); ok {
		return items
	}
	return []any{}
}

// localeOnlyBody は `locale_name === undefined ? {} : { locale_name }`。
func localeOnlyBody(normalized *jsonobj.Object) *jsonobj.Object {
	if normalized.Defined("locale_name") {
		return jsonobj.Obj("locale_name", normalized.Value("locale_name"))
	}
	return jsonobj.New()
}

func dispatchReadToolCall(ctx *CallContext, name string, args any) (*jsonobj.Object, error) {
	switch name {
	case "gkill_status":
		if _, err := NormalizeStatusArgs(args); err != nil {
			return nil, err
		}
		return buildStatusPayload(ctx), nil

	case "gkill_get_mcp_help":
		// 静的な本文。gkill へは往復しない（ADR-0622）。
		normalized, err := NormalizeMcpHelpArgs(args)
		if err != nil {
			return nil, err
		}
		return BuildHelpPayload(jsString(normalized.Value("topic"))), nil

	case "gkill_get_kyous":
		return handleGetKyous(ctx, args)

	case "gkill_get_mi_board_list":
		normalized, err := NormalizeLocaleOnlyArgs(args)
		if err != nil {
			return nil, err
		}
		response, err := ctx.callApi("/api/get_mi_board_list", normalized)
		if err != nil {
			return nil, err
		}
		return jsonobj.Obj("boards", arrayOrEmpty(response.Value("boards"))), nil

	case "gkill_get_all_tag_names":
		// 絞り込みは MCP 側。gkill は全件を返すので、contains / limit は送らない。
		normalized, err := NormalizeTagNamesArgs(args)
		if err != nil {
			return nil, err
		}
		response, err := ctx.callApi("/api/get_all_tag_names", localeOnlyBody(normalized))
		if err != nil {
			return nil, err
		}
		return PaginateTagNames(arrayOrEmpty(response.Value("tag_names")), normalized), nil

	case "gkill_get_all_rep_names":
		normalized, err := NormalizeRepNamesArgs(args)
		if err != nil {
			return nil, err
		}
		response, err := ctx.callApi("/api/get_all_rep_names", localeOnlyBody(normalized))
		if err != nil {
			return nil, err
		}
		return PaginateRepNames(arrayOrEmpty(response.Value("rep_names")), normalized), nil

	case "gkill_get_gps_log":
		normalized, err := NormalizeGpsArgs(args)
		if err != nil {
			return nil, err
		}
		response, err := ctx.callApi("/api/get_gps_log", jsonobj.Obj(
			"start_date", normalized.Value("start_date"),
			"end_date", normalized.Value("end_date"),
			"locale_name", normalized.Value("locale_name"),
		))
		if err != nil {
			return nil, err
		}
		// ページングは MCP 側実装。gkill は期間内の全点を（時刻降順・重複排除済みで）返す。
		return PaginateGpsLogs(arrayOrEmpty(response.Value("gps_logs")), normalized)

	case "gkill_get_rep_infos":
		return handleGetRepInfos(ctx, args)

	case "gkill_get_application_config":
		return handleGetApplicationConfig(ctx, args)

	case "gkill_get_idf_file":
		return handleGetIdfFile(ctx, args)

	case "gkill_get_kyou_history":
		return handleGetKyouHistory(ctx, args)

	case "gkill_get_skill_list":
		return handleGetSkillList(ctx, args)

	case "gkill_get_skill":
		return handleGetSkill(ctx, args)
	}
	return nil, Errorf("Unknown read tool: %s", name)
}

func handleGetKyous(ctx *CallContext, args any) (*jsonobj.Object, error) {
	normalized, err := NormalizeKyouArgs(args)
	if err != nil {
		return nil, err
	}
	boolOrFalse := func(key string) bool {
		return jsTruthy(normalized.Value(key))
	}
	response, err := ctx.callApi("/api/get_kyous_mcp", jsonobj.Obj(
		"query", normalized.Value("query"),
		"locale_name", normalized.Value("locale_name"),
		"limit", normalized.Value("limit"),
		"cursor", normalized.Value("cursor"),
		"max_size_mb", normalized.Value("max_size_mb"),
		"is_include_timeis", normalized.Value("is_include_timeis"),
		// ---- v2 (ADR-0604)。include_id / include_rep_name は廃止（常時付与） ----
		"count_only", boolOrFalse("count_only"),
		"group_by", normalized.Value("group_by"),
		"data_types", normalized.Value("data_types"),
		"create_apps", normalized.Value("create_apps"),
		"update_apps", normalized.Value("update_apps"),
		"num_min", normalized.Value("num_min"),
		"num_max", normalized.Value("num_max"),
		"idf_kinds", normalized.Value("idf_kinds"),
		"include_file_size", boolOrFalse("include_file_size"),
		// tag_entities / text_entities は頼まれたときだけ組ませる（ADR-0629）
		"include_attached_ids", boolOrFalse("include_attached_ids"),
	))
	if err != nil {
		return nil, err
	}
	// 入口で補った既定（for_mi の include_create_mi）は Go の warnings と同じ列に並べる。
	warnings := []any{}
	warnings = append(warnings, arrayOrEmpty(normalized.Value("notes"))...)
	warnings = append(warnings, arrayOrEmpty(response.Value("warnings"))...)

	payload := jsonobj.Obj("kyous", arrayOrEmpty(response.Value("kyous")))
	// v2: total_count は cursor 無し応答にのみ入る。カーソルページで嘘の0を作らない。
	if response.Defined("total_count") && response.Value("total_count") != nil {
		payload.Set("total_count", response.Value("total_count"))
	}
	payload.Set("returned_count", nullish(response.Value("returned_count"), int64(0)))
	payload.Set("remaining_count", nullish(response.Value("remaining_count"), int64(0)))
	payload.Set("has_more", jsTruthy(response.Value("has_more")))
	if jsTruthy(response.Value("next_cursor")) {
		payload.Set("next_cursor", response.Value("next_cursor"))
	}
	if buckets, ok := jsonobj.AsArray(response.Value("buckets")); ok {
		payload.Set("buckets", buckets)
	}
	// プラグインの説明は各 Kyou へ焼き込まず、応答トップレベルへ rep 名ごと1回だけ載せる。
	if plugins, ok := jsonobj.AsArray(response.Value("plugins")); ok && len(plugins) > 0 {
		payload.Set("plugins", plugins)
	}
	// 警告は partial に限らず常設。partial は付随データ欠落専用の印として従来の意味を保つ (M-05)。
	if jsTruthy(response.Value("partial")) {
		payload.Set("partial", true)
	}
	if len(warnings) > 0 {
		payload.Set("warnings", warnings)
	}
	if jsTruthy(normalized.Value("include_plugin_content")) {
		maxTextLength, _ := normalized.Int("plugin_content_max_text_length")
		stats := InlinePluginContents(func(pathname string, body *jsonobj.Object) (*jsonobj.Object, error) {
			return ctx.callApi(pathname, body)
		}, arrayOrEmpty(payload.Value("kyous")), InlineOptions{
			MaxTextLength: int(maxTextLength),
			Format:        jsString(normalized.Value("plugin_content_format")),
			LocaleName:    definedString(normalized, "locale_name"),
		})
		payload.Set("plugin_content", inlineStatsObject(stats))
		// 本文を足した後の実サイズで max_size_mb を守り直す（ADR-0624）
		maxSizeMb, _ := normalized.Float("max_size_mb")
		EnforceKyousSizeBudget(payload, maxSizeMb)
	}
	if jsTruthy(normalized.Value("include_file_urls")) {
		// HTTP のときだけ BuildToolResult が公開 URL を鋳造する印（JSON には出ない。ADR-0630）
		payload.SetMeta(MintFileLinksMark, true)
	}
	return payload, nil
}

// nullish は `value ?? fallback`。
func nullish(value any, fallback any) any {
	if value == nil || jsonobj.IsUndefined(value) {
		return fallback
	}
	return value
}

func definedString(o *jsonobj.Object, key string) string {
	if !o.Defined(key) {
		return ""
	}
	return jsString(o.Value(key))
}

// inlineStatsObject は InlineStats を応答の plugin_content ブロックにする（キー順は旧実装と同じ）。
func inlineStatsObject(stats InlineStats) *jsonobj.Object {
	return jsonobj.Obj(
		"requested", int64(stats.Requested),
		"inlined", int64(stats.Inlined),
		"truncated", int64(stats.Truncated),
		"skipped", int64(stats.Skipped),
		"errors", int64(stats.Errors),
		"total_text_length", int64(stats.TotalTextLength),
	)
}

func inlineStatsFromObject(o *jsonobj.Object) *InlineStats {
	if o == nil {
		return nil
	}
	intOf := func(key string) int {
		n, _ := o.Int(key)
		return int(n)
	}
	return &InlineStats{
		Requested:       intOf("requested"),
		Inlined:         intOf("inlined"),
		Truncated:       intOf("truncated"),
		Skipped:         intOf("skipped"),
		Errors:          intOf("errors"),
		TotalTextLength: intOf("total_text_length"),
	}
}

func handleGetRepInfos(ctx *CallContext, args any) (*jsonobj.Object, error) {
	normalized, err := NormalizeRepInfosArgs(args)
	if err != nil {
		return nil, err
	}
	response, err := ctx.callApi("/api/get_rep_infos_mcp", localeOnlyBody(normalized))
	if err != nil {
		return nil, err
	}
	full := jsonobj.Obj(
		"rep_infos", arrayOrEmpty(response.Value("rep_infos")),
		"canonical_rep_types", arrayOrEmpty(response.Value("canonical_rep_types")),
		"plugins", arrayOrEmpty(response.Value("plugins")),
		// タグ・テキスト・通知・GPSログの格納先。query.reps へ渡すと静かに0件になる。
		"attached_data_reps", arrayOrEmpty(response.Value("attached_data_reps")),
	)
	// data_kinds 絞り込み。
	if normalized.Defined("data_kinds") {
		wanted := NewStringSet(anyToStrings(arrayOrEmpty(normalized.Value("data_kinds")))...)
		full.Set("attached_data_reps", filterObjects(arrayOrEmpty(full.Value("attached_data_reps")), func(rep *jsonobj.Object) bool {
			return wanted.Has(jsString(rep.Value("data_kind")))
		}))
	}
	if err := applyRepInfosRowFilters(full, normalized); err != nil {
		return nil, err
	}
	// fields 射影。
	if !normalized.Defined("fields") {
		return full, nil
	}
	projected := jsonobj.New()
	for _, field := range anyToStrings(arrayOrEmpty(normalized.Value("fields"))) {
		if !RepInfosFields.Has(field) {
			continue
		}
		projected.Set(field, full.Value(field))
	}
	return projected, nil
}

func anyToStrings(items []any) []string {
	out := make([]string, len(items))
	for i, item := range items {
		out[i] = jsString(item)
	}
	return out
}

// filterObjects は配列の各要素（オブジェクトのみ判定に掛ける。非オブジェクトは keep(nil) の結果に従う）を絞る。
func filterObjects(items []any, keep func(rep *jsonobj.Object) bool) []any {
	out := []any{}
	for _, item := range items {
		rep, _ := item.(*jsonobj.Object)
		if keep(rep) {
			out = append(out, item)
		}
	}
	return out
}

// applyRepInfosRowFilters は gkill_get_rep_infos の行絞り込み（rep_types / rep_names / contains / writable_only）を
// 4配列へ**その場で**適用する。fields 射影の前に呼ぶ。
func applyRepInfosRowFilters(full *jsonobj.Object, normalized *jsonobj.Object) error {
	if normalized.Defined("rep_types") {
		canonicalList := anyToStrings(arrayOrEmpty(full.Value("canonical_rep_types")))
		canonical := NewStringSet(canonicalList...)
		if canonical.Len() > 0 {
			for _, repType := range anyToStrings(arrayOrEmpty(normalized.Value("rep_types"))) {
				if !canonical.Has(repType) {
					return InvalidArgument("rep_types", "must be one of the canonical rep types: "+strings.Join(canonicalList, ", "), repType)
				}
			}
		}
		wanted := NewStringSet(anyToStrings(arrayOrEmpty(normalized.Value("rep_types")))...)
		full.Set("rep_infos", filterObjects(arrayOrEmpty(full.Value("rep_infos")), func(rep *jsonobj.Object) bool {
			return wanted.Has(jsString(rep.Value("rep_type")))
		}))
	}
	if normalized.Defined("rep_names") {
		wanted := NewStringSet(anyToStrings(arrayOrEmpty(normalized.Value("rep_names")))...)
		keep := func(rep *jsonobj.Object) bool { return wanted.Has(jsString(rep.Value("rep_name"))) }
		for _, key := range []string{"rep_infos", "plugins", "attached_data_reps"} {
			full.Set(key, filterObjects(arrayOrEmpty(full.Value(key)), keep))
		}
	}
	if normalized.Defined("contains") {
		needle := strings.ToLower(jsString(normalized.Value("contains")))
		keep := func(rep *jsonobj.Object) bool {
			name, ok := rep.Value("rep_name").(string)
			return ok && strings.Contains(strings.ToLower(name), needle)
		}
		for _, key := range []string{"rep_infos", "plugins", "attached_data_reps"} {
			full.Set(key, filterObjects(arrayOrEmpty(full.Value(key)), keep))
		}
	}
	if jsTruthy(normalized.Value("writable_only")) {
		keep := func(rep *jsonobj.Object) bool { return rep.Value("use_to_write") == true }
		full.Set("rep_infos", filterObjects(arrayOrEmpty(full.Value("rep_infos")), keep))
		full.Set("attached_data_reps", filterObjects(arrayOrEmpty(full.Value("attached_data_reps")), keep))
		// プラグインは書き込み先にならない
		full.Set("plugins", []any{})
	}
	return nil
}

func handleGetApplicationConfig(ctx *CallContext, args any) (*jsonobj.Object, error) {
	normalized, err := NormalizeAppConfigArgs(args)
	if err != nil {
		return nil, err
	}
	response, err := ctx.callApi("/api/get_application_config", localeOnlyBody(normalized))
	if err != nil {
		return nil, err
	}
	config, ok := response.Object("application_config")
	if !ok || config == nil {
		config = jsonobj.New()
	}
	full := jsonobj.Obj(
		// 接続先の識別。gkill は元から返しているのに、以前の射影が捨てていた。
		"user_id", config.Value("user_id"),
		"device", config.Value("device"),
		"tag_struct", config.Value("tag_struct"),
		"mi_board_struct", config.Value("mi_board_struct"),
		"rep_struct", config.Value("rep_struct"),
		"rep_type_struct", config.Value("rep_type_struct"),
		"device_struct", config.Value("device_struct"),
		"kftl_template_struct", config.Value("kftl_template_struct"),
		"mi_default_board", config.Value("mi_default_board"),
		"show_tags_in_list", config.Value("show_tags_in_list"),
	)
	// fields 射影（実測で全量193.6k字＝応答上限超過）。許可リストの照合は動的な書き込みの直前でも行う。
	projected := full
	if normalized.Defined("fields") {
		projected = jsonobj.New()
		for _, field := range anyToStrings(arrayOrEmpty(normalized.Value("fields"))) {
			if !AppConfigFields.Has(field) {
				continue
			}
			// descriptions は full に無い仮想欄（ツリーから組む）。full.Value で Undefined を入れると
			// encode で黙って消えるので、ここで組み立てる。fields を省いた既定の全量には載せない。
			if field == AppConfigDescriptionsField {
				projected.Set(field, BuildAppConfigDescriptions(full))
				continue
			}
			projected.Set(field, full.Value(field))
		}
	}
	// UI 状態キー（ツリーエディタの一時状態）を既定で剥がす。
	if !jsTruthy(normalized.Value("include_ui_state")) {
		projected = StripAppConfigUiState(projected).(*jsonobj.Object)
	}
	// contains で葉を刈り、compact で既定値の欄を落とし、max_size_mb に必ず収める（ADR-0629）。
	if normalized.Defined("contains") {
		projected = FilterAppConfigStructs(projected, jsString(normalized.Value("contains")))
		projected = FilterAppConfigDescriptions(projected, jsString(normalized.Value("contains")))
	}
	if jsTruthy(normalized.Value("compact")) {
		projected = CompactAppConfigStructs(projected)
	}
	maxSizeMb, _ := normalized.Float("max_size_mb")
	return CapAppConfigSize(projected, maxSizeMb), nil
}

func handleGetIdfFile(ctx *CallContext, args any) (*jsonobj.Object, error) {
	normalized, err := NormalizeIdfFileArgs(args)
	if err != nil {
		return nil, err
	}
	repName := jsString(normalized.Value("rep_name"))
	fileName := jsString(normalized.Value("file_name"))
	// クエリの形は他の実装と揃える: ?is_video=true&thumb=WxH
	fileQuery := []string{}
	if jsTruthy(normalized.Value("is_video")) {
		fileQuery = append(fileQuery, "is_video=true")
	}
	if normalized.Defined("thumb") {
		fileQuery = append(fileQuery, "thumb="+jsString(normalized.Value("thumb")))
	}
	segments := strings.Split(fileName, "/")
	for i, segment := range segments {
		segments[i] = encodeURIComponent(segment)
	}
	filePath := "/files/" + encodeURIComponent(repName) + "/" + strings.Join(segments, "/")
	if len(fileQuery) != 0 {
		filePath += "?" + strings.Join(fileQuery, "&")
	}
	fileSid := ctx.SID
	if fileSid == "" {
		fileSid, err = ctx.Client.Login(ctx.context())
		if err != nil {
			return nil, err
		}
	}
	file, err := ctx.Client.FetchFile(ctx.context(), filePath, fileSid)
	if err != nil {
		// 404 の大半は rep 名か file 名の取り違えだが、HTTP の生の文言からはそれが読めない。
		if apiErr, ok := AsGkillApiError(err); ok && apiErr.Detail != nil {
			if status, ok := jsonobj.ToFloat(apiErr.Detail.Value("status")); ok && status == 404 {
				return nil, NewGkillApiError(
					"File not found: rep_name="+jsonobj.MarshalString(repName)+" "+
						"file_name="+jsonobj.MarshalString(fileName)+". Both come from the IDF payload of "+
						"gkill_get_kyous (payload.rep_name and payload.file_name) — rep_name is the repository, "+
						"not the entry's data_type. List valid repository names with gkill_get_rep_infos. "+
						"The file may also have been removed from the repository.",
					apiErr.Detail,
				)
			}
		}
		return nil, err
	}
	mimeType := NormalizeMimeType(file.ContentType)
	// base64 は JSON-RPC レスポンスに素で載るので、青天井にすると数百MBの動画で応答が破裂する
	if int64(len(file.Buffer)) > MaxIDFFileBytes {
		originalURLField := "file_url"
		if strings.HasPrefix(mimeType, "image/") {
			originalURLField = "file_url_full"
		}
		return nil, NewGkillApiError(
			"File is too large to return through MCP: "+itoa(len(file.Buffer))+" bytes (limit "+jsString(MaxIDFFileBytes)+"). "+
				`If this is an image or a video, retry with thumb (e.g. thumb:"1024x1024", plus `+
				"is_video:true for a video) to get a downscaled JPEG that fits. "+
				"On stdio clients you can instead read the IDF payload's file_path directly — no size limit. "+
				"Otherwise hand the user the payload's "+originalURLField+", which is served from /files/ "+
				"with no size limit.",
			jsonobj.Obj(
				"file_name", fileName,
				"file_size_bytes", int64(len(file.Buffer)),
				"max_bytes", MaxIDFFileBytes,
			),
		)
	}
	responseFileName := fileName
	if normalized.Defined("thumb") {
		// thumb のときは中身が JPEG なので、名前の拡張子もそれに合わせる。
		responseFileName = thumbFileName(fileName, mimeType)
	}
	payload := jsonobj.Obj(
		"file_name", responseFileName,
		"mime_type", mimeType,
		"file_size_bytes", int64(len(file.Buffer)),
		"is_image", strings.HasPrefix(mimeType, "image/"),
	)
	if normalized.Defined("thumb") {
		// 縮小して取ったときだけ載せる。原寸と取り違えないための印。
		payload.Set("thumb", normalized.Value("thumb"))
	}
	payload.Set("file_content_base64", base64.StdEncoding.EncodeToString(file.Buffer))
	return payload, nil
}

var thumbExtensionRegex = regexp.MustCompile(`\.[^./\\]*$`)

// thumbFileName は縮小版の名前。thumb を頼むとサーバは JPEG を返すので、拡張子もそれに合わせる。
func thumbFileName(fileName string, mimeType string) string {
	if mimeType != "image/jpeg" {
		return fileName
	}
	return thumbExtensionRegex.ReplaceAllString(fileName, "") + ".jpg"
}

func handleGetKyouHistory(ctx *CallContext, args any) (*jsonobj.Object, error) {
	normalized, err := NormalizeKyouHistoryArgs(args)
	if err != nil {
		return nil, err
	}
	id := jsString(normalized.Value("id"))
	dataType := jsString(normalized.Value("data_type"))
	target, _ := EntityTargetOf(dataType)
	body := jsonobj.Obj("id", id)
	if normalized.Defined("locale_name") {
		body.Set("locale_name", normalized.Value("locale_name"))
	}
	// 型別エンドポイントの histories は IS_DELETED で絞らないので、削除済みの版もそのまま返る。
	response, err := ctx.callApi(target.GetEndpoint, body)
	if err != nil {
		return nil, err
	}
	histories, ok := jsonobj.AsArray(response.Value(target.HistoriesKey))
	if !ok || len(histories) == 0 {
		return nil, NewGkillApiError(EntityNotFoundMessage(id, dataType), nil)
	}
	offset64, _ := normalized.Int("offset")
	limit64, _ := normalized.Int("limit")
	offset := int(offset64)
	limit := int(limit64)
	// 各版の data_type は射影名で返ってくるので、この口が受理する語彙（エンティティ名）に揃える。
	versions := []any{}
	for _, version := range jsSliceArray(histories, offset, offset+limit) {
		if o, isObject := version.(*jsonobj.Object); isObject && o != nil {
			copied := o.Clone()
			copied.Set("data_type", dataType)
			versions = append(versions, copied)
			continue
		}
		versions = append(versions, version)
	}
	hasMore := offset+len(versions) < len(histories)
	latestIsDeleted := false
	if first, isObject := histories[0].(*jsonobj.Object); isObject && first != nil {
		latestIsDeleted = jsTruthy(first.Value("is_deleted"))
	}
	payload := jsonobj.Obj(
		"id", id,
		"data_type", dataType,
		"latest_is_deleted", latestIsDeleted,
		"version_count", int64(len(histories)),
		"offset", int64(offset),
		"returned_count", int64(len(versions)),
		"has_more", hasMore,
	)
	if hasMore {
		// 続きは next_offset を offset に渡す（ADR-0626）
		payload.Set("next_offset", int64(offset+len(versions)))
	}
	payload.Set("versions", versions)
	return payload, nil
}

// jsSliceArray は Array#slice(start, end)（範囲外は切り詰める）。
func jsSliceArray(items []any, start, end int) []any {
	if start < 0 {
		start = 0
	}
	if end > len(items) {
		end = len(items)
	}
	if start >= end {
		return []any{}
	}
	return items[start:end]
}

// buildStatusPayload は gkill_status の応答を組む。
// gkill へ届かないときも失敗にしない —— 「MCP は生きているが gkill が落ちている」を区別できるのがこのツールの仕事。
// 届かない理由の本文は載せない（接続先 URL を含みうる。ADR-0707）。
func buildStatusPayload(ctx *CallContext) *jsonobj.Object {
	server := ctx.Server
	var kind, name, version, revision, toolCount, startedAt, uptime any
	transport := "http"
	if ctx.IsLocalTransport {
		transport = "stdio"
	}
	if server != nil {
		kind = server.ServerKind
		name = server.ServerName
		version = server.ServerVersion
		revision = server.SchemaRevision
		toolCount = int64(len(server.Tools))
		transport = server.Transport()
		if !server.StartedAt.IsZero() {
			startedAt = FormatLocalRfc3339(server.StartedAt)
			seconds := timeNow().Sub(server.StartedAt).Milliseconds() / 1000
			if seconds < 0 {
				seconds = 0
			}
			uptime = seconds
		}
	}
	payload := jsonobj.Obj(
		"server_kind", kind,
		"server_name", name,
		"server_version", version,
		"schema_revision", revision,
		"tool_count", toolCount,
		"transport", transport,
		"started_at", startedAt,
		"uptime_seconds", uptime,
		"gkill_reachable", false,
	)
	// 接続先とビルドは ApplicationConfig にしか載っていない（専用 API は無い）。
	response, err := ctx.callApi("/api/get_application_config", jsonobj.New())
	if err != nil {
		payload.Set("gkill_error", classifyGkillFailure(err))
		ctx.Log.Warn("status_gkill_unreachable", "error", err.Error())
		return payload
	}
	config, ok := response.Object("application_config")
	if !ok || config == nil {
		config = jsonobj.New()
	}
	payload.Set("gkill_reachable", true)
	payload.Set("account", jsonobj.Obj("user_id", nullish(config.Value("user_id"), nil), "device", nullish(config.Value("device"), nil)))
	payload.Set("gkill", jsonobj.Obj(
		"version", nullish(config.Value("version"), nil),
		"commit_hash", nullish(config.Value("commit_hash"), nil),
		"build_time", nullish(config.Value("build_time"), nil),
	))
	// 利用者が AI 向けに書いたスキルの名前と説明（ADR-0634）。AI が作業の最初に呼ぶのはこのツールなので、
	// ここに載せておくと gkill_get_skill_list を知らなくても気づける。取れなくても status は失敗にしない。
	skills, err := fetchSkillList(ctx, jsonobj.New())
	if err != nil {
		payload.Set("skills_error", classifyGkillFailure(err))
		ctx.Log.Warn("status_skill_list_failed", "error", err.Error())
		return payload
	}
	brief := []any{}
	for _, item := range skills {
		skill, ok := item.(*jsonobj.Object)
		if !ok || skill == nil {
			continue
		}
		brief = append(brief, jsonobj.Obj("name", skill.Value("name"), "description", skill.Value("description")))
	}
	payload.Set("skills", brief)
	return payload
}

// classifyGkillFailure は gkill へ届かなかった理由を、端末固有の情報を含まない語に畳む。
func classifyGkillFailure(err error) string {
	apiErr, ok := AsGkillApiError(err)
	if !ok {
		return "unreachable"
	}
	detail := apiErr.Detail
	codeSuffix := ""
	var status any = jsonobj.Undefined
	if detail != nil {
		if errs, ok := detail.Array("errors"); ok && len(errs) > 0 {
			if first, isObject := errs[0].(*jsonobj.Object); isObject && first != nil {
				if code, isString := first.Value("error_code").(string); isString && code != "" {
					codeSuffix = " (" + code + ")"
				}
			}
		}
		status = detail.Value("status")
	}
	if strings.HasPrefix(apiErr.Message, "Login failed") {
		return "login_failed" + codeSuffix + " — do not retry: the credentials this MCP server holds are rejected, and every attempt counts against gkill's per-IP login rate limit"
	}
	if f, ok := jsonobj.ToFloat(status); ok {
		if _, isBool := status.(bool); !isBool {
			if _, isString := status.(string); !isString {
				return "HTTP " + jsNumberString(f)
			}
		}
	}
	if codeSuffix != "" {
		return "api_error" + codeSuffix
	}
	return "unreachable"
}

// SummarizeReadToolPayload は読み取りツールの結果要約。対象外のツールは false。
// 古スキーマの印の付け方は payload.go が正本（書き込み側と同じ文言にするため）。
func SummarizeReadToolPayload(name string, payload *jsonobj.Object) (string, bool) {
	summary, ok := summarizeReadToolPayloadBody(name, payload)
	if !ok {
		return "", false
	}
	return AppendStaleSchemaNoteToSummary(summary, payload), true
}

func lengthOfArray(v any) int {
	if items, ok := jsonobj.AsArray(v); ok {
		return len(items)
	}
	return 0
}

func summarizeReadToolPayloadBody(name string, payload *jsonobj.Object) (string, bool) {
	if payload == nil {
		payload = jsonobj.New()
	}
	if summary, ok := summarizeSkillPayload(name, payload); ok {
		return summary, true
	}
	switch name {
	case "gkill_status":
		kind := jsString(nullish(payload.Value("server_kind"), "unknown"))
		revision := jsString(nullish(payload.Value("schema_revision"), "unknown"))
		uptime := jsString(nullish(payload.Value("uptime_seconds"), int64(0)))
		if !jsTruthy(payload.Value("gkill_reachable")) {
			return "MCP " + kind + " server is up (schema_revision " + revision + ", up " + uptime + "s) but gkill is NOT reachable (" + jsString(nullish(payload.Value("gkill_error"), "unreachable")) + ").", true
		}
		account, _ := payload.Object("account")
		userID := jsString(nullish(account.Value("user_id"), "unknown"))
		device := jsString(nullish(account.Value("device"), "unknown"))
		summary := "Connected to " + userID + "@" + device + " via " + kind + " server (schema_revision " + revision + ", up " + uptime + "s)."
		if count := lengthOfArray(payload.Value("skills")); count > 0 {
			summary += " " + itoa(count) + " skill(s) stored — read the one matching the task with gkill_get_skill."
		}
		return summary, true
	case "gkill_get_mcp_help":
		length := 0
		if text, ok := payload.Value("text").(string); ok {
			length = jsLength(text)
		}
		return `Help topic "` + jsString(nullish(payload.Value("topic"), "index")) + `": ` + jsString(nullish(payload.Value("title"), "")) + " (" + itoa(length) + " chars).", true
	case "gkill_get_kyous":
		// v2: total_count は cursor 無し応答にのみ入る。残量の真実は remaining_count。
		if buckets, ok := jsonobj.AsArray(payload.Value("buckets")); ok {
			return "Aggregated " + itoa(len(buckets)) + " buckets (" + jsString(nullish(payload.Value("total_count"), int64(0))) + " entries).", true
		}
		returnedCount := nullish(payload.Value("returned_count"), int64(0))
		remaining := nullish(payload.Value("remaining_count"), int64(0))
		returnedZero, _ := jsonobj.ToFloat(returnedCount)
		if lengthOfArray(payload.Value("kyous")) == 0 && payload.Defined("total_count") && returnedZero == 0 && !jsTruthy(payload.Value("has_more")) {
			// 0件のときは count_only の有無で文言が割れないようにする。
			if total, ok := jsonobj.ToFloat(payload.Value("total_count")); ok && total == 0 {
				return "No entries matched.", true
			}
			return "Counted " + jsString(payload.Value("total_count")) + " entries.", true
		}
		pluginContent, _ := payload.Object("plugin_content")
		pluginSuffix := SummarizeInlinePluginContent(inlineStatsFromObject(pluginContent))
		totalPart := ""
		if payload.Defined("total_count") {
			totalPart = " of " + jsString(payload.Value("total_count"))
		}
		if jsTruthy(payload.Value("has_more")) && jsTruthy(payload.Value("next_cursor")) {
			return "Returned " + jsString(returnedCount) + totalPart + " kyou entries (" + jsString(remaining) + " remaining). Next page: cursor=\"" + jsString(payload.Value("next_cursor")) + "\"." + pluginSuffix, true
		}
		return "Returned " + jsString(returnedCount) + totalPart + " kyou entries (0 remaining)." + pluginSuffix, true
	case "gkill_get_mi_board_list":
		return "Fetched " + itoa(lengthOfArray(payload.Value("boards"))) + " Mi boards.", true
	case "gkill_get_all_tag_names":
		returned := lengthOfArray(payload.Value("tag_names"))
		if jsTruthy(payload.Value("truncated")) {
			return "Fetched " + itoa(returned) + " of " + jsString(payload.Value("total_count")) + " matching tag names (truncated — narrow with contains or raise limit).", true
		}
		return "Fetched " + itoa(returned) + " tag names.", true
	case "gkill_get_all_rep_names":
		returned := lengthOfArray(payload.Value("rep_names"))
		if jsTruthy(payload.Value("truncated")) {
			return "Fetched " + itoa(returned) + " of " + jsString(payload.Value("total_count")) + " matching repository names (truncated — narrow with contains or raise limit).", true
		}
		return "Fetched " + itoa(returned) + " repository names.", true
	case "gkill_get_gps_log":
		if buckets, ok := jsonobj.AsArray(payload.Value("buckets")); ok {
			return "Aggregated " + itoa(len(buckets)) + " daily buckets (" + jsString(nullish(payload.Value("total_count"), int64(0))) + " GPS points).", true
		}
		returned := lengthOfArray(payload.Value("gps_logs"))
		if returned == 0 && payload.Defined("total_count") && !jsTruthy(payload.Value("has_more")) {
			if total, ok := jsonobj.ToFloat(payload.Value("total_count")); ok && total == 0 {
				return "No GPS points matched.", true
			}
			return "Counted " + jsString(payload.Value("total_count")) + " GPS points.", true
		}
		if jsTruthy(payload.Value("has_more")) && jsTruthy(payload.Value("next_cursor")) {
			return "Returned " + itoa(returned) + " GPS points (" + jsString(nullish(payload.Value("remaining_count"), int64(0))) + " remaining). Next page: cursor=\"" + jsString(payload.Value("next_cursor")) + "\".", true
		}
		return "Returned " + itoa(returned) + " GPS points (0 remaining).", true
	case "gkill_get_application_config":
		return "Fetched application configuration (" + itoa(len(payload.Keys())) + " fields).", true
	case "gkill_get_kyou_history":
		total := jsString(nullish(payload.Value("version_count"), int64(0)))
		shown := jsString(nullish(payload.Value("returned_count"), int64(0)))
		deleted := ""
		if jsTruthy(payload.Value("latest_is_deleted")) {
			deleted = " — latest version is DELETED"
		}
		offset := ""
		if jsTruthy(payload.Value("offset")) {
			offset = " from offset " + jsString(payload.Value("offset"))
		}
		more := ""
		if jsTruthy(payload.Value("has_more")) {
			more = " (more available: pass offset:" + jsString(payload.Value("next_offset")) + ")"
		}
		return "Returned " + shown + " of " + total + " versions" + offset + more + deleted + ".", true
	case "gkill_get_rep_infos":
		// fields で rep_infos を外した呼び出しに「Fetched 0 repositories」と言わない。
		parts := []string{}
		if repInfos, ok := jsonobj.AsArray(payload.Value("rep_infos")); ok {
			writable := 0
			hasWriteFlag := false
			for _, item := range repInfos {
				rep, _ := item.(*jsonobj.Object)
				if rep == nil {
					continue
				}
				if rep.Value("use_to_write") == true {
					writable++
				}
				if _, isBool := rep.Value("use_to_write").(bool); isBool {
					hasWriteFlag = true
				}
			}
			if hasWriteFlag {
				parts = append(parts, itoa(len(repInfos))+" repositories ("+itoa(writable)+" writable)")
			} else {
				parts = append(parts, itoa(len(repInfos))+" repositories")
			}
		} else {
			parts = append(parts, "repositories omitted by fields")
		}
		if canonical, ok := jsonobj.AsArray(payload.Value("canonical_rep_types")); ok {
			parts = append(parts, itoa(len(canonical))+" canonical rep types")
		} else {
			parts = append(parts, "canonical rep types omitted by fields")
		}
		return "Fetched " + strings.Join(parts, ", ") + ".", true
	case "gkill_get_idf_file":
		return "Retrieved file: " + jsString(payload.Value("file_name")) + " (" + jsString(payload.Value("file_size_bytes")) + " bytes, " + jsString(payload.Value("mime_type")) + ")", true
	}
	return "", false
}

// ApplicationConfig の struct ツリーを持つキー。
var appConfigStructKeys = []string{"tag_struct", "mi_board_struct", "rep_struct", "rep_type_struct", "device_struct", "kftl_template_struct"}

// 葉の「識別欄」。name がこれと同じ値なら name を落とせる（compact）。contains の照合対象でもある。
// 語彙は Web の TS クラス（src/client/classes/datas/config/*-struct-element-data.ts）が正本で、
// 実データは tag_name / rep_name / rep_type_name / device_name / board_name / title（KFTL テンプレート）。
// 以前は偽 gkill のフィクスチャに合わせて tag / device / rep_type と書かれており、実環境では
// tag / device / rep_type の3ツリーで name 落としも contains の照合も効いていなかった（ADR-0632）。
var structIdentityKeys = []string{"tag_name", "rep_name", "rep_type_name", "device_name", "board_name", "title"}

// AppConfigDescriptionsField は fields で明示したときだけ組む仮想欄（ADR-0632）。
const AppConfigDescriptionsField = "descriptions"

// structIdentityOf は葉の識別欄の値（無ければ name）。
func structIdentityOf(node *jsonobj.Object) string {
	for _, key := range structIdentityKeys {
		if s, ok := node.Value(key).(string); ok && s != "" {
			return s
		}
	}
	return jsString(node.Value("name"))
}

// BuildAppConfigDescriptions は 6 ツリーを深さ優先で歩き、description が非空のノードだけを
// {struct, name, path, is_dir(true のときだけ), description} の平坦な配列にする。
// name は葉なら識別欄・フォルダなら表示名、path は祖先の表示名を "/" で連結したもの。
// ルート（深さ 0）は path にも一覧にも含めない —— ルート名は保存時に "__root__" が補われるが
// 古い設定では "" のこともあるので、リテラル比較ではなく深さで除く。
// ツリーが未設定（null / Undefined）なら飛ばし、配列ルート（旧形）は各要素を深さ 0 として歩く。
func BuildAppConfigDescriptions(full *jsonobj.Object) []any {
	out := []any{}
	var walk func(structKey string, node any, depth int, ancestors []string)
	walk = func(structKey string, node any, depth int, ancestors []string) {
		if items, ok := jsonobj.AsArray(node); ok {
			for _, item := range items {
				walk(structKey, item, depth, ancestors)
			}
			return
		}
		if !IsPlainObject(node) {
			return
		}
		o := node.(*jsonobj.Object)
		isDir := o.Value("is_dir") == true
		name := jsString(o.Value("name"))
		if !isDir {
			name = structIdentityOf(o)
		}
		if depth > 0 {
			if description, ok := o.Value("description").(string); ok && description != "" {
				entry := jsonobj.Obj(
					"struct", structKey,
					"name", name,
					"path", strings.Join(append(append([]string{}, ancestors...), name), "/"),
				)
				if isDir {
					entry.Set("is_dir", true)
				}
				entry.Set("description", description)
				out = append(out, entry)
			}
		}
		children, ok := jsonobj.AsArray(o.Value("children"))
		if !ok {
			return
		}
		next := ancestors
		if depth > 0 {
			next = append(append([]string{}, ancestors...), name)
		}
		for _, child := range children {
			walk(structKey, child, depth+1, next)
		}
	}
	for _, key := range appConfigStructKeys {
		value := full.Value(key)
		if value == nil || jsonobj.IsUndefined(value) {
			continue
		}
		walk(key, value, 0, nil)
	}
	return out
}

// FilterAppConfigDescriptions は contains を descriptions 一覧に掛ける（name / path / description の大小無視の部分一致）。
// ツリー用の filterStructNodes に流すと葉扱いで識別欄しか見ないので、一覧は別に刈る。
func FilterAppConfigDescriptions(projected *jsonobj.Object, needle string) *jsonobj.Object {
	items, ok := jsonobj.AsArray(projected.Value(AppConfigDescriptionsField))
	if !ok {
		return projected
	}
	lowered := strings.ToLower(needle)
	kept := []any{}
	for _, item := range items {
		if !IsPlainObject(item) {
			continue
		}
		entry := item.(*jsonobj.Object)
		for _, key := range []string{"name", "path", "description"} {
			if s, isString := entry.Value(key).(string); isString && strings.Contains(strings.ToLower(s), lowered) {
				kept = append(kept, entry)
				break
			}
		}
	}
	out := projected.Clone()
	out.Set(AppConfigDescriptionsField, kept)
	return out
}

// jsStrictEquals は `===`（オブジェクトは同一性、数値は値、Undefined / null はそれぞれ同士だけ）。
func jsStrictEquals(a, b any) bool {
	if jsonobj.IsUndefined(a) || jsonobj.IsUndefined(b) {
		return jsonobj.IsUndefined(a) && jsonobj.IsUndefined(b)
	}
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	switch x := a.(type) {
	case string:
		y, ok := b.(string)
		return ok && x == y
	case bool:
		y, ok := b.(bool)
		return ok && x == y
	case *jsonobj.Object:
		y, ok := b.(*jsonobj.Object)
		return ok && x == y
	}
	if fa, ok := jsonobj.ToFloat(a); ok {
		fb, ok := jsonobj.ToFloat(b)
		return ok && fa == fb
	}
	return false
}

// compactStructNode は既定値の欄を落とす（ADR-0629）。
func compactStructNode(node any) any {
	if items, ok := jsonobj.AsArray(node); ok {
		out := make([]any, len(items))
		for i, child := range items {
			out[i] = compactStructNode(child)
		}
		return out
	}
	if !IsPlainObject(node) {
		return node
	}
	o := node.(*jsonobj.Object)
	out := jsonobj.New()
	for _, key := range o.Keys() {
		value := o.Value(key)
		if key == "children" {
			if value == nil {
				continue
			}
			if items, ok := jsonobj.AsArray(value); ok && len(items) == 0 {
				continue
			}
			out.Set(key, compactStructNode(value))
			continue
		}
		if (key == "is_dir" || key == "ignore_check_rep_rykv") && value == false {
			continue
		}
		// description は利用者の運用メモ。空文字は「書いていない」なので落とし、非空は必ず残す（ADR-0632）
		if key == "description" && value == "" {
			continue
		}
		if key == "name" {
			same := false
			for _, idKey := range structIdentityKeys {
				if jsStrictEquals(o.Value(idKey), value) {
					same = true
					break
				}
			}
			if same {
				continue
			}
		}
		out.Set(key, value)
	}
	return out
}

// CompactAppConfigStructs は応答（fields 射影後）の struct ツリーだけを compact する。
func CompactAppConfigStructs(projected *jsonobj.Object) *jsonobj.Object {
	out := projected.Clone()
	for _, key := range appConfigStructKeys {
		if out.Has(key) && out.Value(key) != nil && !jsonobj.IsUndefined(out.Value(key)) {
			out.Set(key, compactStructNode(out.Value(key)))
		}
	}
	return out
}

// filterStructNodes は葉の識別欄か name に needle（大小無視の部分一致）を含む葉だけを残し、
// 葉が1つも残らない入れ物（children を持つノード）ごと落とす。
func filterStructNodes(nodes any, needle string) any {
	items, ok := jsonobj.AsArray(nodes)
	if !ok {
		return nodes
	}
	lowered := strings.ToLower(needle)
	matches := func(node *jsonobj.Object) bool {
		for _, key := range append(append([]string{}, structIdentityKeys...), "name") {
			if s, isString := node.Value(key).(string); isString && strings.Contains(strings.ToLower(s), lowered) {
				return true
			}
		}
		return false
	}
	out := []any{}
	for _, item := range items {
		if !IsPlainObject(item) {
			continue
		}
		node := item.(*jsonobj.Object)
		if _, isArray := jsonobj.AsArray(node.Value("children")); isArray {
			children, _ := jsonobj.AsArray(filterStructNodes(node.Value("children"), needle))
			if len(children) > 0 {
				copied := node.Clone()
				copied.Set("children", children)
				out = append(out, copied)
			}
			continue
		}
		if matches(node) {
			out = append(out, node)
		}
	}
	return out
}

// FilterAppConfigStructs は contains で struct ツリーを刈る。ツリー以外の欄はそのまま。
// 実データのツリーはルート 1 オブジェクト（{name:"__root__", children:[...], is_dir:true}）なので、
// ルートの children を刈って同じルートに戻す（1 件も残らなければ children:[]）。配列ルート（旧形）もそのまま受ける。
// 以前は配列しか見ておらず、実環境では contains が何も刈らなかった（ADR-0632）。
func FilterAppConfigStructs(projected *jsonobj.Object, needle string) *jsonobj.Object {
	out := projected.Clone()
	for _, key := range appConfigStructKeys {
		if !out.Has(key) {
			continue
		}
		value := out.Value(key)
		if _, isArray := jsonobj.AsArray(value); isArray {
			out.Set(key, filterStructNodes(value, needle))
			continue
		}
		if !IsPlainObject(value) {
			continue
		}
		root := value.(*jsonobj.Object)
		if _, hasChildren := jsonobj.AsArray(root.Value("children")); !hasChildren {
			continue
		}
		copied := root.Clone()
		copied.Set("children", filterStructNodes(root.Value("children"), needle))
		out.Set(key, copied)
	}
	return out
}

// CapAppConfigSize は応答の JSON バイト数を max_size_mb に必ず収める。
// 超えていたら struct 欄を大きい順に {omitted_bytes} へ置き換え、warnings で fields / contains を案内する。
func CapAppConfigSize(projected *jsonobj.Object, maxSizeMb float64) *jsonobj.Object {
	maxBytes := int(maxSizeMb * 1024 * 1024)
	if jsonobj.ByteLength(projected) <= maxBytes {
		return projected
	}
	out := projected.Clone()
	warnings := []any{}
	type candidate struct {
		key   string
		bytes int
	}
	candidates := []candidate{}
	for _, key := range appConfigStructKeys {
		if out.Has(key) && !jsonobj.IsUndefined(out.Value(key)) {
			candidates = append(candidates, candidate{key: key, bytes: jsonobj.ByteLength(out.Value(key))})
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].bytes > candidates[j].bytes })
	for _, c := range candidates {
		if jsonobj.ByteLength(out) <= maxBytes {
			break
		}
		out.Set(c.key, jsonobj.Obj("omitted_bytes", int64(c.bytes)))
		warnings = append(warnings,
			c.key+" ("+itoa(c.bytes)+" bytes) was omitted to keep the response under max_size_mb ("+itoa(maxBytes)+" bytes): "+
				"narrow it with contains, request only the fields you need, or raise max_size_mb")
	}
	merged := []any{}
	merged = append(merged, arrayOrEmpty(out.Value("warnings"))...)
	merged = append(merged, warnings...)
	out.Set("warnings", merged)
	return out
}

// EnforceKyousSizeBudget は include_plugin_content で本文を足した後の kyous[] を max_size_mb に収め直す（ADR-0624）。
// 規則は Go 本体と同じ: 2件目以降は足す前に判定、先頭1件は超えても返して警告。押し出した分は次頁へ。
func EnforceKyousSizeBudget(payload *jsonobj.Object, maxSizeMb float64) {
	if payload == nil {
		return
	}
	kyous, ok := jsonobj.AsArray(payload.Value("kyous"))
	if !ok || len(kyous) == 0 {
		return
	}
	maxBytes := int(maxSizeMb * 1024 * 1024)
	kept := []any{}
	running := 0
	firstAlone := -1
	for _, kyou := range kyous {
		bytes := jsonobj.ByteLength(kyou)
		if len(kept) > 0 && running+bytes > maxBytes {
			break
		}
		if len(kept) == 0 && bytes > maxBytes {
			firstAlone = bytes
		}
		running += bytes
		kept = append(kept, kyou)
	}
	heldBack := len(kyous) - len(kept)
	warnings := []any{}
	warnings = append(warnings, arrayOrEmpty(payload.Value("warnings"))...)
	if firstAlone >= 0 {
		warnings = append(warnings,
			"single record with plugin content ("+itoa(firstAlone)+" bytes) exceeds max_size_mb ("+itoa(maxBytes)+" bytes); returned anyway to keep pagination progressing — lower plugin_content_max_text_length to shrink it")
	}
	if heldBack > 0 {
		last, _ := kept[len(kept)-1].(*jsonobj.Object)
		payload.Set("kyous", kept)
		payload.Set("returned_count", int64(len(kept)))
		remaining, _ := jsonobj.ToFloat(nullish(payload.Value("remaining_count"), int64(0)))
		payload.Set("remaining_count", int64(remaining)+int64(heldBack))
		payload.Set("has_more", true)
		payload.Set("next_cursor", jsString(last.Value("related_time"))+"::"+jsString(last.Value("id")))
		warnings = append(warnings,
			"plugin content pushed the page over max_size_mb ("+itoa(maxBytes)+" bytes): "+itoa(heldBack)+" of "+itoa(len(kept)+heldBack)+" entries were held back and will be returned on the next cursor page (next_cursor is set)")
		if previous, ok := payload.Object("plugin_content"); ok && previous != nil {
			payload.Set("plugin_content", recountInlinePluginContent(kept, previous))
		}
	}
	if len(warnings) > 0 {
		payload.Set("warnings", warnings)
	}
}

// recountInlinePluginContent は残した kyous[] から plugin_content の件数を数え直す。
func recountInlinePluginContent(kyous []any, previous *jsonobj.Object) *jsonobj.Object {
	stats := previous.Clone()
	requested, inlined, truncated, skipped, errorsCount, totalTextLength := 0, 0, 0, 0, 0, 0
	for _, item := range kyous {
		kyou, _ := item.(*jsonobj.Object)
		if kyou == nil {
			continue
		}
		payload, ok := kyou.Object("payload")
		if !ok || payload == nil || payload.Value("kind") != "plugin" {
			continue
		}
		requested++
		switch payload.Value("content_status") {
		case "ok":
			inlined++
		case "truncated":
			inlined++
			truncated++
		case "skipped":
			skipped++
		case "error":
			errorsCount++
		}
		if text, ok := payload.Value("content_text").(string); ok {
			totalTextLength += jsLength(text)
		}
		if html, ok := payload.Value("content_html").(string); ok {
			totalTextLength += jsLength(html)
		}
	}
	stats.Set("requested", int64(requested))
	stats.Set("inlined", int64(inlined))
	stats.Set("truncated", int64(truncated))
	stats.Set("skipped", int64(skipped))
	stats.Set("errors", int64(errorsCount))
	stats.Set("total_text_length", int64(totalTextLength))
	return stats
}

// StripAppConfigUiState は struct ツリーから UI 状態キーを再帰的に剥がす。
func StripAppConfigUiState(value any) any {
	if items, ok := jsonobj.AsArray(value); ok {
		out := make([]any, len(items))
		for i, item := range items {
			out[i] = StripAppConfigUiState(item)
		}
		return out
	}
	if IsPlainObject(value) {
		o := value.(*jsonobj.Object)
		out := jsonobj.New()
		for _, key := range o.Keys() {
			if AppConfigUIStateKeys.Has(key) {
				continue
			}
			out.Set(key, StripAppConfigUiState(o.Value(key)))
		}
		return out
	}
	return value
}

// paginateNameList は名前一覧に contains / limit を適用する共通部分。
func paginateNameList(names []any, options *jsonobj.Object) ([]any, int) {
	matched := names
	if options.Defined("contains") && options.Value("contains") != "" {
		needle := strings.ToLower(jsString(options.Value("contains")))
		matched = []any{}
		for _, name := range names {
			if strings.Contains(strings.ToLower(jsString(name)), needle) {
				matched = append(matched, name)
			}
		}
	}
	limit, _ := jsonobj.ToFloat(options.Value("limit"))
	page := jsSliceArray(matched, 0, int(limit))
	return page, len(matched)
}

// PaginateRepNames は rep 名の全一覧に contains / limit を適用する。total_count は絞り込み後・limit 適用前の件数。
func PaginateRepNames(repNames []any, options *jsonobj.Object) *jsonobj.Object {
	page, total := paginateNameList(repNames, options)
	return jsonobj.Obj(
		"rep_names", page,
		"total_count", int64(total),
		"returned_count", int64(len(page)),
		"truncated", len(page) < total,
	)
}

// PaginateTagNames は PaginateRepNames と同じ規則をタグ名へ当てる。
func PaginateTagNames(tagNames []any, options *jsonobj.Object) *jsonobj.Object {
	page, total := paginateNameList(tagNames, options)
	return jsonobj.Obj(
		"tag_names", page,
		"total_count", int64(total),
		"returned_count", int64(len(page)),
		"truncated", len(page) < total,
	)
}

func relatedTimeOf(point any) string {
	if o, ok := point.(*jsonobj.Object); ok && o != nil {
		return jsString(o.Value("related_time"))
	}
	return "undefined"
}

// PaginateGpsLogs は取得済みの全点列に limit / cursor / count_only / group_by を適用する。
func PaginateGpsLogs(gpsLogs []any, options *jsonobj.Object) (*jsonobj.Object, error) {
	// 規則の正本は normalization.go。get_kyous と同じ文言で弾く。
	if err := AssertAggregationNotCombinedWithCursor(options); err != nil {
		return nil, err
	}
	if jsTruthy(options.Value("count_only")) {
		return jsonobj.Obj("gps_logs", []any{}, "total_count", int64(len(gpsLogs)), "returned_count", int64(0), "remaining_count", int64(0), "has_more", false), nil
	}
	if options.Value("group_by") == "day" {
		// 日別カバレッジ（キーは点の related_time のローカル日付）
		counts := map[string]int{}
		keys := []string{}
		for _, point := range gpsLogs {
			at, _ := jsDateParse(relatedTimeOf(point))
			local := at.In(time.Local)
			key := local.Format("2006-01-02")
			if _, seen := counts[key]; !seen {
				keys = append(keys, key)
			}
			counts[key]++
		}
		sort.Strings(keys)
		buckets := []any{}
		for _, key := range keys {
			buckets = append(buckets, jsonobj.Obj("key", key, "count", int64(counts[key])))
		}
		return jsonobj.Obj("gps_logs", []any{}, "buckets", buckets, "total_count", int64(len(gpsLogs)), "returned_count", int64(0), "remaining_count", int64(0), "has_more", false), nil
	}

	startIndex := 0
	if jsTruthy(options.Value("cursor")) {
		cursor, err := DecodeGpsCursor(jsString(options.Value("cursor")))
		if err != nil {
			return nil, err
		}
		sameTimeSeen := 0
		startIndex = len(gpsLogs)
		cursorMillis, _ := jsDateParseMillis(cursor.T)
		for i, point := range gpsLogs {
			t := relatedTimeOf(point)
			if t == cursor.T {
				sameTimeSeen++
				if sameTimeSeen > cursor.N {
					startIndex = i
					break
				}
				continue
			}
			// サーバは時刻降順なので、カーソル時刻より古い点が最初に現れた位置から再開
			if millis, _ := jsDateParseMillis(t); millis < cursorMillis {
				startIndex = i
				break
			}
		}
	}

	batch := jsSliceArray(gpsLogs, startIndex, len(gpsLogs))
	limit, _ := jsonobj.ToFloat(options.Value("limit"))
	page := jsSliceArray(batch, 0, int(limit))
	remaining := len(batch) - len(page)
	payload := jsonobj.Obj(
		"gps_logs", page,
		"returned_count", int64(len(page)),
		"remaining_count", int64(remaining),
		"has_more", remaining > 0,
	)
	if !jsTruthy(options.Value("cursor")) {
		payload.Set("total_count", int64(len(gpsLogs)))
	}
	if remaining > 0 && len(page) > 0 {
		lastTime := relatedTimeOf(page[len(page)-1])
		n := 0
		for i := startIndex; i < startIndex+len(page); i++ {
			if relatedTimeOf(gpsLogs[i]) == lastTime {
				n++
			}
		}
		// ランがページ境界をまたぐとき、前ページで消費した分も数えないと重複して返す
		priorSameTime := 0
		for i := startIndex - 1; i >= 0; i-- {
			if relatedTimeOf(gpsLogs[i]) == lastTime {
				priorSameTime++
			} else {
				break
			}
		}
		payload.Set("next_cursor", EncodeGpsCursor(lastTime, priorSameTime+n))
	}
	return payload, nil
}
