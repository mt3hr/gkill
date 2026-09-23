package mcp

// 編集前に読む: .claude/skills/gkill-mcp/SKILL.md
//
// 書き込みツールのディスパッチと要約（旧 write-handlers.mjs）。write / readwrite の2サーバが共有する。
// 実装はこの1箇所が正本で、サーバ側は IsWriteToolName / HandleWriteToolCall / SummarizeWriteToolPayload へ委譲するだけ。

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// WriteDevice は書き込み時に記録する端末名。サーバ種別によらず共通。
// アプリ名 (create_app / update_app) はサーバごとに違うので ctx.AppName で受け取る。
const WriteDevice = "mcp"

// newUUID は新しい記録の ID。テストとゴールデン再生で差し替える。
var newUUID = func() string { return uuid.New().String() }

// IsWriteToolName は name が書き込みツールか。
func IsWriteToolName(name string) bool {
	return findTool(WriteTools, name) != nil
}

// HandleWriteToolCall は書き込みツール1件を処理する。
// ディスパッチ本体を包んで、古いツールスキーマを掴んだクライアントへの警告を**1箇所で**足す（読み取り側と同じ形）。
func HandleWriteToolCall(ctx *CallContext, name string, args any) (*jsonobj.Object, error) {
	payload, err := dispatchWriteToolCall(ctx, name, args)
	if err != nil {
		return nil, err
	}
	return AppendStaleSchemaWarning(payload, name, args), nil
}

// resolveDefaultBoardName は gkill_add_mi の board_name 未指定時に使う既定板名を返す。
// ApplicationConfig の mi_default_board が取れなければ "Inbox" へ落とす。
func resolveDefaultBoardName(ctx *CallContext, localeName any) string {
	body := jsonobj.New()
	if !jsonobj.IsUndefined(localeName) {
		body.Set("locale_name", localeName)
	}
	response, err := ctx.callApi("/api/get_application_config", body)
	if err == nil {
		if config, ok := response.Object("application_config"); ok && config != nil {
			if configured, ok := config.Value("mi_default_board").(string); ok && jsTrim(configured) != "" {
				return configured
			}
		}
	}
	// 既定板が引けないことを理由にタスク作成そのものを失敗させない
	return "Inbox"
}

// assertBoardExists は board_name が実在の板名か照合し、無ければ呼び出し側エラーで弾く（完全一致）。
func assertBoardExists(ctx *CallContext, boardName string, localeName any) error {
	body := jsonobj.New()
	if !jsonobj.IsUndefined(localeName) {
		body.Set("locale_name", localeName)
	}
	response, err := ctx.callApi("/api/get_mi_board_list", body)
	if err != nil {
		return err
	}
	for _, board := range arrayOrEmpty(response.Value("boards")) {
		if board == boardName {
			return nil
		}
	}
	return InvalidArgument(
		"board_name",
		"unknown board "+jsonobj.MarshalString(boardName)+" — gkill would create a new board with this name. "+
			"Pass allow_create_board:true to allow that, or pick an existing board (gkill_get_mi_board_list)",
		boardName,
	)
}

// stripUrlogImages は URLog の応答から画像の base64 を落とす（読み返し側には元から載っていない）。
func stripUrlogImages(urlog any) any {
	if !jsTruthy(urlog) {
		return nil
	}
	o, ok := urlog.(*jsonobj.Object)
	if !ok {
		return urlog
	}
	rest := o.Clone()
	rest.Delete("favicon_image")
	rest.Delete("thumbnail_image")
	return rest
}

// mergeStored は「サーバが保存した版」を手元の値に重ねる。応答が部分的でも手元のフィールドを落とさない。
func mergeStored(current *jsonobj.Object, stored any) *jsonobj.Object {
	if !jsTruthy(stored) {
		return current
	}
	storedObject, ok := stored.(*jsonobj.Object)
	if !ok {
		return current
	}
	return current.Clone().Merge(storedObject)
}

// nextUpdateTime は「現在値より必ず後」になる更新時刻を返す。
// UPDATE_TIME は1秒解像度で保存されるので、実時刻が現在値と同じ秒に落ちるときだけ1秒進める。
func nextUpdateTime(current *jsonobj.Object) string {
	now := timeNow()
	updateTime := ""
	if current != nil {
		if s, ok := current.Value("update_time").(string); ok {
			updateTime = s
		}
	}
	previous, ok := jsDateParseMillis(updateTime)
	if !ok {
		return jsISOString(now)
	}
	previousSecond := (previous / 1000) * 1000
	nowSecond := (now.UnixMilli() / 1000) * 1000
	if nowSecond > previousSecond {
		return jsISOString(now)
	}
	return jsISOString(time.UnixMilli(previousSecond + 1000))
}

// updateTarget は gkill_update_* 9ツールの「型ごとに違うところ」だけを持つ表（旧 UPDATE_TARGETS）。
type updateTarget struct {
	Normalize   func(args any) (*jsonobj.Object, error)
	PatchFields []string
	PreUpdate   func(ctx *CallContext, normalized *jsonobj.Object) error
}

var updateTargets = map[string]updateTarget{
	"kmemo":   {Normalize: NormalizeUpdateKmemoArgs, PatchFields: []string{"content", "related_time"}},
	"urlog":   {Normalize: NormalizeUpdateUrlogArgs, PatchFields: []string{"url", "title", "related_time"}},
	"nlog":    {Normalize: NormalizeUpdateNlogArgs, PatchFields: []string{"title", "amount", "shop", "related_time"}},
	"lantana": {Normalize: NormalizeUpdateLantanaArgs, PatchFields: []string{"mood", "related_time"}},
	"timeis":  {Normalize: NormalizeUpdateTimeIsArgs, PatchFields: []string{"title", "start_time", "end_time"}},
	"mi": {
		Normalize:   NormalizeUpdateMiArgs,
		PatchFields: []string{"title", "board_name", "is_checked", "limit_time", "estimate_start_time", "estimate_end_time"},
		// allow_create_board:false のときだけ、移動先の板名を実在の板と照合する（patchFields ではない）。
		PreUpdate: func(ctx *CallContext, normalized *jsonobj.Object) error {
			if normalized.Value("allow_create_board") == false && normalized.Defined("board_name") {
				return assertBoardExists(ctx, jsString(normalized.Value("board_name")), normalized.Value("locale_name"))
			}
			return nil
		},
	},
	"kc":   {Normalize: NormalizeUpdateKcArgs, PatchFields: []string{"title", "num_value", "related_time"}},
	"tag":  {Normalize: NormalizeUpdateTagArgs, PatchFields: []string{"tag"}},
	"text": {Normalize: NormalizeUpdateTextArgs, PatchFields: []string{"text"}},
}

// updateTargetOrder は要約器の表を旧実装と同じ順で組むためのキー順。
var updateTargetOrder = []string{"kmemo", "urlog", "nlog", "lantana", "timeis", "mi", "kc", "tag", "text"}

// runUpdate は gkill_update_* 1件を処理する。
// gkill に部分更新の API は無いので、現在値を取って渡された欄だけ上書きし、同じ型の更新 API へ送り直す（patch）。
func runUpdate(ctx *CallContext, dataType string, args any) (*jsonobj.Object, error) {
	spec, hasSpec := updateTargets[dataType]
	target, hasTarget := EntityTargetOf(dataType)
	if !hasSpec || !hasTarget {
		return nil, Errorf("Unsupported data_type for update: %s", dataType)
	}
	normalized, err := spec.Normalize(args)
	if err != nil {
		return nil, err
	}
	// 型固有の事前検証（現状は mi の板名照合だけ）。取得より前に置いて fail-fast にする。
	if spec.PreUpdate != nil {
		if err := spec.PreUpdate(ctx, normalized); err != nil {
			return nil, err
		}
	}
	id := jsString(normalized.Value("id"))
	getResponse, err := ctx.callApi(target.GetEndpoint, jsonobj.Obj("id", id))
	if err != nil {
		return nil, err
	}
	histories, ok := jsonobj.AsArray(getResponse.Value(target.HistoriesKey))
	if !ok || len(histories) == 0 {
		return nil, NewGkillApiError(EntityNotFoundMessage(id, dataType), nil)
	}
	current, _ := histories[0].(*jsonobj.Object)
	if current == nil {
		current = jsonobj.New()
	}
	// 未指定 = 触らない。null は「消す」の意味を持つ欄があるので !== undefined で見る。
	patchedFields := []string{}
	for _, field := range spec.PatchFields {
		if normalized.Defined(field) {
			patchedFields = append(patchedFields, field)
		}
	}
	if len(patchedFields) == 0 {
		return nil, Errorf("No fields to update for %s %s (nothing changed; pass at least one of %s).", dataType, id, strings.Join(spec.PatchFields, ", "))
	}
	// 指定された欄が全部いまの値と同じでも弾く（ADR-0628）。
	allSame := true
	for _, field := range patchedFields {
		if !valuesEquivalent(current.Value(field), normalized.Value(field)) {
			allSame = false
			break
		}
	}
	if allSame {
		return nil, Errorf("No effective change for %s %s: every specified field (%s) "+
			"already has that value, so nothing was written (gkill is append-only and an identical version would only "+
			"add noise to the history). Pass a different value to update.", dataType, id, strings.Join(patchedFields, ", "))
	}
	for _, field := range patchedFields {
		current.Set(field, normalized.Value(field))
	}
	current.Set("update_time", nextUpdateTime(current))
	current.Set("update_app", ctx.AppName)
	current.Set("update_device", WriteDevice)
	current.Set("update_user", ctx.UserID)
	response, err := ctx.callApi(target.UpdateEndpoint, jsonobj.Obj(
		target.RequestKey, current,
		"want_response_kyou", true,
		"locale_name", normalized.Value("locale_name"),
	))
	if err != nil {
		return nil, err
	}
	return jsonobj.Obj(
		target.ResponseKey, withEntityDataType(mergeStored(current, response.Value(target.ResponseKey)), dataType),
		"updated_kyou", nullValue(response.Value("updated_kyou")),
	), nil
}

// nullValue は `value || null`。
func nullValue(value any) any {
	if !jsTruthy(value) {
		return nil
	}
	return value
}

// valuesEquivalent は「パッチの値がいまの値と同じか」。日時はオフセット違いを同じ瞬間として比べ、
// 数値は Number で、null（消す）はいまの値が空なら同じとみなす。
var rfc3339LikeRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?(?:Z|[+-]\d{2}:\d{2})$`)

func valuesEquivalent(current, next any) bool {
	if jsStrictEquals(current, next) {
		return true
	}
	currentEmpty := current == nil || jsonobj.IsUndefined(current) || current == ""
	if next == nil || jsonobj.IsUndefined(next) {
		return currentEmpty
	}
	if currentEmpty {
		return false
	}
	nextString, nextIsString := next.(string)
	currentString, currentIsString := current.(string)
	if nextIsString && currentIsString {
		if rfc3339LikeRegex.MatchString(currentString) && rfc3339LikeRegex.MatchString(nextString) {
			a, okA := jsDateParseMillis(currentString)
			b, okB := jsDateParseMillis(nextString)
			return okA && okB && a == b
		}
		return currentString == nextString
	}
	nextNumber, nextIsNumber := jsonobj.ToFloat(next)
	currentNumber, currentIsNumber := jsonobj.ToFloat(current)
	if _, isBool := next.(bool); isBool {
		nextIsNumber = false
	}
	if _, isBool := current.(bool); isBool {
		currentIsNumber = false
	}
	if nextIsNumber || currentIsNumber {
		// Number(x) が NaN になる組み合わせは等しくない
		if !nextIsNumber || !currentIsNumber {
			return jsNumberCoerceEqual(current, next)
		}
		return currentNumber == nextNumber
	}
	return false
}

// jsNumberCoerceEqual は Number(current) === Number(next) を、片方が数値でないときに評価する。
func jsNumberCoerceEqual(current, next any) bool {
	coerce := func(v any) (float64, bool) {
		if f, ok := jsonobj.ToFloat(v); ok {
			return f, true
		}
		switch x := v.(type) {
		case bool:
			if x {
				return 1, true
			}
			return 0, true
		case string:
			trimmed := jsTrim(x)
			if trimmed == "" {
				return 0, true
			}
			if f, err := strconv.ParseFloat(trimmed, 64); err == nil {
				return f, true
			}
		case nil:
			return 0, true
		}
		return 0, false
	}
	a, okA := coerce(current)
	b, okB := coerce(next)
	return okA && okB && a == b
}

// withEntityDataType は応答の実体の data_type をこの口の語彙（エンティティ名）に揃える。
func withEntityDataType(entity any, dataType string) any {
	o, ok := entity.(*jsonobj.Object)
	if !ok || o == nil {
		return entity
	}
	out := o.Clone()
	out.Set("data_type", dataType)
	return out
}

// describeTargetNotFound は add_tag / add_text の「対象の記録が無い」（error_kind not_found）を、
// target_id を名指しした文言で包み直す。それ以外はそのまま返す。
func describeTargetNotFound(err error, targetID string, kind string) error {
	apiErr, ok := AsGkillApiError(err)
	if !ok {
		return err
	}
	notFound := false
	if apiErr.Detail != nil {
		for _, entry := range arrayOrEmpty(apiErr.Detail.Value("errors")) {
			if o, isObject := entry.(*jsonobj.Object); isObject && o != nil && o.Value("error_kind") == "not_found" {
				notFound = true
				break
			}
		}
	}
	if !notFound && strings.Contains(apiErr.Message, "kind=not_found") {
		notFound = true
	}
	if !notFound {
		return err
	}
	return NewGkillApiError(
		"target_id "+jsonobj.MarshalString(targetID)+" matched no kyou, so the "+kind+" was not added. target_id must be the id of an "+
			"entry (kmemo / mi / timeis / urlog ...), not a tag or text id, and the entry must not be deleted (check it with "+
			"gkill_get_kyou_history or gkill_get_kyous query.ids). Server said: "+apiErr.Message,
		apiErr.Detail,
	)
}

// softDeleteOne は1件の is_deleted を切り替える。削除と復活で共通（現在値を取って更新 API へ送る patch 方式）。
func softDeleteOne(ctx *CallContext, entry *jsonobj.Object, deleting bool, localeName any) (*jsonobj.Object, error) {
	dataType := jsString(entry.Value("data_type"))
	id := jsString(entry.Value("id"))
	target, ok := EntityTargetOf(dataType)
	if !ok {
		verb := "restore"
		if deleting {
			verb = "delete"
		}
		return nil, Errorf("Unsupported data_type for %s: %s", verb, dataType)
	}
	// 1. 現在値を取る（データ欄を落とさずに patch するため）
	getResponse, err := ctx.callApi(target.GetEndpoint, jsonobj.Obj("id", id))
	if err != nil {
		return nil, err
	}
	histories, isArray := jsonobj.AsArray(getResponse.Value(target.HistoriesKey))
	if !isArray || len(histories) == 0 {
		return nil, NewGkillApiError(EntityNotFoundMessage(id, dataType), nil)
	}
	current, _ := histories[0].(*jsonobj.Object)
	if current == nil {
		current = jsonobj.New()
	}
	// 無意味な版を積まない。
	if deleting && jsTruthy(current.Value("is_deleted")) {
		return nil, Errorf("Entity is already deleted: %s (nothing changed; read it with gkill_get_kyou_history or undo with gkill_restore_kyou)", id)
	}
	if !deleting && !jsTruthy(current.Value("is_deleted")) {
		return nil, Errorf("Entity is already active (not deleted): %s", id)
	}
	// 2. is_deleted と更新メタデータを差し替える
	current.Set("is_deleted", deleting)
	current.Set("update_time", nextUpdateTime(current))
	current.Set("update_app", ctx.AppName)
	current.Set("update_device", WriteDevice)
	current.Set("update_user", ctx.UserID)
	// 3. 更新 API へ送る
	response, err := ctx.callApi(target.UpdateEndpoint, jsonobj.Obj(
		target.RequestKey, current,
		"want_response_kyou", true,
		"locale_name", localeName,
	))
	if err != nil {
		return nil, err
	}
	result := jsonobj.New()
	// 復活はキー名だけ restored_ に付け替える（サーバ側の応答キーは updated_ のまま）
	responseKey := "restored_" + dataType
	if deleting {
		responseKey = target.ResponseKey
	}
	result.Set(responseKey, withEntityDataType(mergeStored(current, response.Value(target.ResponseKey)), dataType))
	if jsTruthy(response.Value("updated_kyou")) {
		result.Set("updated_kyou", response.Value("updated_kyou"))
	}
	return result, nil
}

// runSoftDeleteTargets は単件と一括の両方を捌く。一括は直列に回し、途中で失敗しても止めない。
func runSoftDeleteTargets(ctx *CallContext, normalized *jsonobj.Object, deleting bool) (*jsonobj.Object, error) {
	targets := arrayOrEmpty(normalized.Value("targets"))
	localeName := normalized.Value("locale_name")
	if !jsTruthy(normalized.Value("batch")) {
		entry, _ := targets[0].(*jsonobj.Object)
		return softDeleteOne(ctx, entry, deleting, localeName)
	}
	results := []any{}
	succeeded := 0
	for _, item := range targets {
		entry, _ := item.(*jsonobj.Object)
		_, err := softDeleteOne(ctx, entry, deleting, localeName)
		if err != nil {
			results = append(results, jsonobj.Obj("id", entry.Value("id"), "data_type", entry.Value("data_type"), "ok", false, "error", err.Error()))
			continue
		}
		results = append(results, jsonobj.Obj("id", entry.Value("id"), "data_type", entry.Value("data_type"), "ok", true))
		succeeded++
	}
	return jsonobj.Obj(
		"results", results,
		"succeeded_count", int64(succeeded),
		"failed_count", int64(len(results)-succeeded),
	), nil
}

// auditFields は追加時の作成・更新メタデータ（キー順は旧実装と同じ）。
func auditFields(entity *jsonobj.Object, ctx *CallContext, now string) *jsonobj.Object {
	entity.Set("create_time", now)
	entity.Set("create_app", ctx.AppName)
	entity.Set("create_device", WriteDevice)
	entity.Set("create_user", ctx.UserID)
	entity.Set("update_time", now)
	entity.Set("update_app", ctx.AppName)
	entity.Set("update_device", WriteDevice)
	entity.Set("update_user", ctx.UserID)
	entity.Set("is_deleted", false)
	return entity
}

// orNow は `normalized.related_time || now`。
func orNow(value any, now string) any {
	if jsTruthy(value) {
		return value
	}
	return now
}

// orNull は `value || null`。
func orNull(value any) any {
	if jsTruthy(value) {
		return value
	}
	return nil
}

// orEmpty は `value || ""`。
func orEmpty(value any) any {
	if jsTruthy(value) {
		return value
	}
	return ""
}

func addResult(response *jsonobj.Object, key string) *jsonobj.Object {
	return jsonobj.Obj(key, nullValue(response.Value(key)), "added_kyou", nullValue(response.Value("added_kyou")))
}

func dispatchWriteToolCall(ctx *CallContext, name string, args any) (*jsonobj.Object, error) {
	switch name {
	case "gkill_add_kmemo":
		normalized, err := NormalizeKmemoArgs(args)
		if err != nil {
			return nil, err
		}
		now := jsISOString(timeNow())
		kmemo := auditFields(jsonobj.Obj(
			"id", newUUID(),
			"rep_name", "",
			"related_time", orNow(normalized.Value("related_time"), now),
			"content", normalized.Value("content"),
			"data_type", "kmemo",
		), ctx, now)
		response, err := ctx.callApi("/api/add_kmemo", jsonobj.Obj("kmemo", kmemo, "want_response_kyou", true, "locale_name", normalized.Value("locale_name")))
		if err != nil {
			return nil, err
		}
		return addResult(response, "added_kmemo"), nil

	case "gkill_add_urlog":
		normalized, err := NormalizeUrlogArgs(args)
		if err != nil {
			return nil, err
		}
		now := jsISOString(timeNow())
		urlog := auditFields(jsonobj.Obj(
			"id", newUUID(),
			"rep_name", "",
			"related_time", orNow(normalized.Value("related_time"), now),
			"url", normalized.Value("url"),
			"title", orEmpty(normalized.Value("title")),
			"image_base64", "",
			"data_type", "urlog",
		), ctx, now)
		response, err := ctx.callApi("/api/add_urlog", jsonobj.Obj(
			"urlog", urlog,
			"want_response_kyou", true,
			"locale_name", normalized.Value("locale_name"),
			// fetch_metadata / fetch_favicon (既定 true) を Go 側の抑止フラグへ反転して写す。
			"skip_fetch_metadata", normalized.Value("fetch_metadata") == false,
			"skip_fetch_favicon", normalized.Value("fetch_favicon") == false,
		))
		if err != nil {
			return nil, err
		}
		return jsonobj.Obj("added_urlog", stripUrlogImages(response.Value("added_urlog")), "added_kyou", nullValue(response.Value("added_kyou"))), nil

	case "gkill_add_nlog":
		normalized, err := NormalizeNlogArgs(args)
		if err != nil {
			return nil, err
		}
		now := jsISOString(timeNow())
		nlog := auditFields(jsonobj.Obj(
			"id", newUUID(),
			"rep_name", "",
			"related_time", orNow(normalized.Value("related_time"), now),
			"shop", orEmpty(normalized.Value("shop")),
			"title", normalized.Value("title"),
			"amount", normalized.Value("amount"),
			"data_type", "nlog",
		), ctx, now)
		response, err := ctx.callApi("/api/add_nlog", jsonobj.Obj("nlog", nlog, "want_response_kyou", true, "locale_name", normalized.Value("locale_name")))
		if err != nil {
			return nil, err
		}
		return addResult(response, "added_nlog"), nil

	case "gkill_add_lantana":
		normalized, err := NormalizeLantanaArgs(args)
		if err != nil {
			return nil, err
		}
		now := jsISOString(timeNow())
		lantana := auditFields(jsonobj.Obj(
			"id", newUUID(),
			"rep_name", "",
			"related_time", orNow(normalized.Value("related_time"), now),
			"mood", normalized.Value("mood"),
			"data_type", "lantana",
		), ctx, now)
		response, err := ctx.callApi("/api/add_lantana", jsonobj.Obj("lantana", lantana, "want_response_kyou", true, "locale_name", normalized.Value("locale_name")))
		if err != nil {
			return nil, err
		}
		return addResult(response, "added_lantana"), nil

	case "gkill_add_timeis":
		normalized, err := NormalizeTimeIsArgs(args)
		if err != nil {
			return nil, err
		}
		now := jsISOString(timeNow())
		timeis := auditFields(jsonobj.Obj(
			"id", newUUID(),
			"rep_name", "",
			"title", normalized.Value("title"),
			"start_time", orNow(normalized.Value("start_time"), now),
			"end_time", orNull(normalized.Value("end_time")),
			"data_type", "timeis",
		), ctx, now)
		response, err := ctx.callApi("/api/add_timeis", jsonobj.Obj("timeis", timeis, "want_response_kyou", true, "locale_name", normalized.Value("locale_name")))
		if err != nil {
			return nil, err
		}
		return addResult(response, "added_timeis"), nil

	case "gkill_add_mi":
		normalized, err := NormalizeMiArgs(args)
		if err != nil {
			return nil, err
		}
		// allow_create_board:false のときだけ、指定された板名を実在の板と照合する。
		if normalized.Value("allow_create_board") == false && normalized.Defined("board_name") {
			if err := assertBoardExists(ctx, jsString(normalized.Value("board_name")), normalized.Value("locale_name")); err != nil {
				return nil, err
			}
		}
		// board_name 未指定ならアカウントの既定板へ入れる（空文字のまま送ると名前の無い板に積まれる）。
		var boardName any
		if normalized.Defined("board_name") {
			boardName = normalized.Value("board_name")
		} else {
			boardName = resolveDefaultBoardName(ctx, normalized.Value("locale_name"))
		}
		now := jsISOString(timeNow())
		mi := auditFields(jsonobj.Obj(
			"id", newUUID(),
			"rep_name", "",
			"title", normalized.Value("title"),
			"is_checked", normalized.Value("is_checked"),
			"board_name", boardName,
			"limit_time", orNull(normalized.Value("limit_time")),
			"estimate_start_time", orNull(normalized.Value("estimate_start_time")),
			"estimate_end_time", orNull(normalized.Value("estimate_end_time")),
			"data_type", "mi",
		), ctx, now)
		response, err := ctx.callApi("/api/add_mi", jsonobj.Obj("mi", mi, "want_response_kyou", true, "locale_name", normalized.Value("locale_name")))
		if err != nil {
			return nil, err
		}
		return addResult(response, "added_mi"), nil

	case "gkill_add_kc":
		normalized, err := NormalizeKcArgs(args)
		if err != nil {
			return nil, err
		}
		now := jsISOString(timeNow())
		kc := auditFields(jsonobj.Obj(
			"id", newUUID(),
			"rep_name", "",
			"related_time", orNow(normalized.Value("related_time"), now),
			"title", normalized.Value("title"),
			"num_value", normalized.Value("num_value"),
			"data_type", "kc",
		), ctx, now)
		response, err := ctx.callApi("/api/add_kc", jsonobj.Obj("kc", kc, "want_response_kyou", true, "locale_name", normalized.Value("locale_name")))
		if err != nil {
			return nil, err
		}
		return addResult(response, "added_kc"), nil

	case "gkill_add_tag":
		normalized, err := NormalizeTagArgs(args)
		if err != nil {
			return nil, err
		}
		now := jsISOString(timeNow())
		tag := auditFields(jsonobj.Obj(
			"id", newUUID(),
			"rep_name", "",
			"target_id", normalized.Value("target_id"),
			"tag", normalized.Value("tag"),
			// 送らないと Go のゼロ値 (0001-01-01) がそのまま保存される
			"related_time", now,
			"data_type", "tag",
		), ctx, now)
		response, err := ctx.callApi("/api/add_tag", jsonobj.Obj("tag", tag, "want_response_kyou", true, "locale_name", normalized.Value("locale_name")))
		if err != nil {
			return nil, describeTargetNotFound(err, jsString(normalized.Value("target_id")), "tag")
		}
		// AddTagResponse に added_kyou は無い
		return jsonobj.Obj("added_tag", nullValue(response.Value("added_tag"))), nil

	case "gkill_add_text":
		normalized, err := NormalizeTextArgs(args)
		if err != nil {
			return nil, err
		}
		now := jsISOString(timeNow())
		text := auditFields(jsonobj.Obj(
			"id", newUUID(),
			"rep_name", "",
			"target_id", normalized.Value("target_id"),
			"text", normalized.Value("text"),
			"related_time", now,
			"data_type", "text",
		), ctx, now)
		response, err := ctx.callApi("/api/add_text", jsonobj.Obj("text", text, "want_response_kyou", true, "locale_name", normalized.Value("locale_name")))
		if err != nil {
			return nil, describeTargetNotFound(err, jsString(normalized.Value("target_id")), "text")
		}
		return jsonobj.Obj("added_text", nullValue(response.Value("added_text"))), nil

	case "gkill_submit_kftl":
		normalized, err := NormalizeKftlArgs(args)
		if err != nil {
			return nil, err
		}
		response, err := ctx.callApi("/api/submit_kftl_text", jsonobj.Obj(
			"kftl_text", normalized.Value("kftl_text"),
			"locale_name", normalized.Value("locale_name"),
			"idempotency_key", normalized.Value("idempotency_key"),
		))
		if err != nil {
			return nil, err
		}
		// created は「実際に書かれたもの」。replayed:true は冪等キーで畳んだ再送（ADR-0510）。
		messages := response.Value("messages")
		if !jsTruthy(messages) {
			messages = []any{}
		}
		created := response.Value("created")
		if !jsTruthy(created) {
			created = []any{}
		}
		return jsonobj.Obj("messages", messages, "created", created, "replayed", jsTruthy(response.Value("replayed"))), nil

	case "gkill_delete_kyou":
		normalized, err := NormalizeDeleteArgs(args)
		if err != nil {
			return nil, err
		}
		return runSoftDeleteTargets(ctx, normalized, true)

	case "gkill_restore_kyou":
		normalized, err := NormalizeRestoreArgs(args)
		if err != nil {
			return nil, err
		}
		return runSoftDeleteTargets(ctx, normalized, false)

	case "gkill_update_kmemo", "gkill_update_urlog", "gkill_update_nlog", "gkill_update_lantana",
		"gkill_update_timeis", "gkill_update_mi", "gkill_update_kc", "gkill_update_tag", "gkill_update_text":
		return runUpdate(ctx, strings.TrimPrefix(name, "gkill_update_"), args)

	case "gkill_add_skill":
		return handleAddSkill(ctx, args)

	case "gkill_update_skill":
		return handleUpdateSkill(ctx, args)

		// gkill_delete_skill は公開しない（skill_delete_tool.go の冒頭）。一覧から外しても case があると
		// 呼べてしまうので case もコメントアウトしてある。公開するときは WriteTools の登録行と一緒に戻す。
		// case "gkill_delete_skill":
		// 	return handleDeleteSkill(ctx, args)
	}
	return nil, NewGkillApiError(UnknownToolMessage(name), nil)
}

// batchSoftDeleteSummary は一括削除・一括復活の1行サマリ。失敗件数を出さないと failed_count:1 でも completed と読める。
func batchSoftDeleteSummary(verb string, payload *jsonobj.Object) string {
	succeeded, _ := jsonobj.ToFloat(nullish(payload.Value("succeeded_count"), int64(0)))
	failed, _ := jsonobj.ToFloat(nullish(payload.Value("failed_count"), int64(0)))
	total := succeeded + failed
	if failed == 0 {
		return verb + ": " + jsNumberString(succeeded) + "/" + jsNumberString(total) + " entries."
	}
	return verb + ": " + jsNumberString(succeeded) + "/" + jsNumberString(total) + " entries — " + jsNumberString(failed) + " FAILED (see results[] for the reason of each)."
}

// addSummaryVerbs: 動詞が "Added" なのは tag / text だけ。付随データは既存の記録へ「付ける」。
var addSummaryVerbs = map[string]string{"tag": "Added", "text": "Added"}

func idOrUnknown(payload *jsonobj.Object, key string) string {
	if entity, ok := payload.Object(key); ok && entity != nil {
		if id := entity.Value("id"); jsTruthy(id) {
			return jsString(id)
		}
	}
	return "unknown"
}

// entitySummarizer は gkill_add_* / gkill_update_* の1行要約を表から作る（ADR-0611）。
func entitySummarizer(name string) (func(payload *jsonobj.Object) string, bool) {
	for _, dataType := range updateTargetOrder {
		if name == "gkill_add_"+dataType {
			verb := addSummaryVerbs[dataType]
			if verb == "" {
				verb = "Created"
			}
			return func(payload *jsonobj.Object) string {
				return verb + " " + dataType + ": " + idOrUnknown(payload, "added_"+dataType)
			}, true
		}
		if name == "gkill_update_"+dataType {
			return func(payload *jsonobj.Object) string {
				return "Updated " + dataType + ": " + idOrUnknown(payload, "updated_"+dataType)
			}, true
		}
	}
	return nil, false
}

// SummarizeWriteToolPayload は書き込みツールの結果要約。対象外のツールは false。
func SummarizeWriteToolPayload(name string, payload *jsonobj.Object) (string, bool) {
	if payload == nil {
		payload = jsonobj.New()
	}
	if summarizer, ok := entitySummarizer(name); ok {
		return AppendStaleSchemaNoteToSummary(summarizer(payload), payload), true
	}
	summary, ok := summarizeWriteToolPayloadBody(name, payload)
	if !ok {
		return "", false
	}
	return AppendStaleSchemaNoteToSummary(summary, payload), true
}

func summarizeWriteToolPayloadBody(name string, payload *jsonobj.Object) (string, bool) {
	if summary, ok := summarizeSkillPayload(name, payload); ok {
		return summary, true
	}
	switch name {
	case "gkill_submit_kftl":
		created := arrayOrEmpty(payload.Value("created"))
		if jsTruthy(payload.Value("replayed")) {
			// 畳んだ再送。created は元の送信の控えで、今回は1件も書いていない。
			return "KFTL replay folded: " + itoa(len(created)) + " record(s) of the original submission returned again (replayed:true, nothing written this time).", true
		}
		if len(created) == 0 {
			return "KFTL submitted: nothing was written (blank lines write nothing).", true
		}
		kinds := map[string]int{}
		order := []string{}
		for _, item := range created {
			record, _ := item.(*jsonobj.Object)
			if record == nil {
				record = jsonobj.New()
			}
			kind := jsString(record.Value("data_type"))
			if jsTruthy(record.Value("updated")) {
				kind += " (updated)"
			}
			if _, seen := kinds[kind]; !seen {
				order = append(order, kind)
			}
			kinds[kind]++
		}
		parts := []string{}
		for _, kind := range order {
			if kinds[kind] == 1 {
				parts = append(parts, kind)
			} else {
				parts = append(parts, kind+" x"+itoa(kinds[kind]))
			}
		}
		return "KFTL submitted: wrote " + itoa(len(created)) + " record(s) — " + strings.Join(parts, ", ") + ".", true
	case "gkill_restore_kyou":
		if _, isArray := jsonobj.AsArray(payload.Value("results")); isArray {
			return batchSoftDeleteSummary("Restored", payload), true
		}
		return "Restored: " + describeSingleSoftDelete(payload, "restored_"), true
	case "gkill_delete_kyou":
		if _, isArray := jsonobj.AsArray(payload.Value("results")); isArray {
			return batchSoftDeleteSummary("Deleted (soft)", payload), true
		}
		return "Deleted (soft): " + describeSingleSoftDelete(payload, "updated_"), true
	}
	return "", false
}

// describeSingleSoftDelete は単件の削除・復活の要約に「型 id」を出す。
func describeSingleSoftDelete(payload *jsonobj.Object, prefix string) string {
	parts := []string{}
	for _, key := range payload.Keys() {
		if !strings.HasPrefix(key, prefix) || key == prefix+"kyou" {
			continue
		}
		id := "(id unknown)"
		if entity, ok := payload.Object(key); ok && entity != nil && entity.Defined("id") && entity.Value("id") != nil {
			id = jsString(entity.Value("id"))
		}
		parts = append(parts, strings.TrimPrefix(key, prefix)+" "+id)
	}
	if len(parts) == 0 {
		return "completed"
	}
	return strings.Join(parts, ", ")
}
