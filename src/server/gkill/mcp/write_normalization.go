package mcp

// 書き込みツールの引数正規化（旧 write-normalization.mjs）。
//
// 追加(add)と更新(update)は同じ手順で、型ごとに違うのはフィールド名・種別・追加時に必須かだけなので、
// 表（entityFieldSpecs）から作る（ADR-0611）。

import (
	"regexp"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// DeleteDataTypes は gkill_delete_kyou / gkill_restore_kyou が畳んだ後に受理する data_type。
var DeleteDataTypes = NewStringSet(EntityDataTypeValues...)

// 日付だけを渡されたとき「その日の終わり」へ展開するフィールド（締切・見積終了は「その日じゅう」）。
var endOfDayDatetimeFields = NewStringSet("limit_time", "estimate_end_time")

// optionalDatetime は省略可能な日時引数を検証する。無ければ Undefined。
func optionalDatetime(args *jsonobj.Object, field string) (any, error) {
	value := args.Value(field)
	if value == nil || jsonobj.IsUndefined(value) {
		return jsonobj.Undefined, nil
	}
	return NormalizeDateTimeString(value, field, DateTimeOptions{AllowDateOnly: true, EndOfDay: endOfDayDatetimeFields.Has(field)})
}

// assertArgs は args が素のオブジェクトであることを検査する。
func assertArgs(args any) (*jsonobj.Object, error) {
	if !IsPlainObject(args) {
		return nil, InvalidArgument("arguments", "must be an object", args)
	}
	return args.(*jsonobj.Object), nil
}

// URL からスキームが抜けていると gkill はページ取得すら試みず、title が空のまま保存されるので入口で弾く。
var urlSchemeRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*://`)

func assertURLWithScheme(value any, field string) (string, error) {
	url, err := AssertTrimmedString(value, field)
	if err != nil {
		return "", err
	}
	if !urlSchemeRegex.MatchString(url) {
		return "", InvalidArgument(field, `must include a scheme, e.g. "https://example.com/page"`, value)
	}
	return url, nil
}

// assertTimeIsOrder は終わりが始まりより前の TimeIs を弾く。null は「終了を消す」なので比較しない。
func assertTimeIsOrder(startTime, endTime any) error {
	if jsonobj.IsUndefined(startTime) || jsonobj.IsUndefined(endTime) || endTime == nil {
		return nil
	}
	start, _ := jsDateParseMillis(jsString(startTime))
	end, _ := jsDateParseMillis(jsString(endTime))
	if start > end {
		return InvalidArgument("end_time", "must not be before start_time ("+jsString(startTime)+"); the interval would have a negative length", endTime)
	}
	return nil
}

// entityField はエンティティのフィールド1つの宣言。
type entityField struct {
	Name                string
	Kind                string // string / url / integer / number / boolean / datetime
	RequiredOnAdd       bool
	DefaultOnAdd        any // add で未指定のときに入れる値（nil = 無し）
	AddOnly             bool
	NullClears          bool
	RevivesStaleBoolean bool
	Range               *IntRange
}

type entitySpec struct {
	Fields []entityField
	After  func(normalized *jsonobj.Object) error
}

// entityFieldSpecs はエンティティのフィールド表（旧 ENTITY_FIELD_SPECS）。
var entityFieldSpecs = map[string]entitySpec{
	"kmemo": {Fields: []entityField{
		{Name: "content", Kind: "string", RequiredOnAdd: true},
		{Name: "related_time", Kind: "datetime"},
	}},
	"urlog": {Fields: []entityField{
		{Name: "url", Kind: "url", RequiredOnAdd: true},
		{Name: "title", Kind: "string"},
		{Name: "related_time", Kind: "datetime"},
		// 外向き取得の抑止。add 専用（update 経路には抑止すべき取得が無い）。
		{Name: "fetch_metadata", Kind: "boolean", DefaultOnAdd: true, AddOnly: true, RevivesStaleBoolean: true},
		{Name: "fetch_favicon", Kind: "boolean", DefaultOnAdd: true, AddOnly: true, RevivesStaleBoolean: true},
	}},
	"nlog": {Fields: []entityField{
		{Name: "title", Kind: "string", RequiredOnAdd: true},
		{Name: "amount", Kind: "integer", RequiredOnAdd: true},
		{Name: "shop", Kind: "string"},
		{Name: "related_time", Kind: "datetime"},
	}},
	"lantana": {Fields: []entityField{
		{Name: "mood", Kind: "integer", RequiredOnAdd: true, Range: &IntRange{Min: I(0), Max: I(10)}},
		{Name: "related_time", Kind: "datetime"},
	}},
	"timeis": {
		Fields: []entityField{
			{Name: "title", Kind: "string", RequiredOnAdd: true},
			{Name: "start_time", Kind: "datetime"},
			// end_time だけは null に意味がある（終了を取り消して進行中へ戻す）。未指定 = 触らない、null = 消す、値 = その時刻。
			{Name: "end_time", Kind: "datetime", NullClears: true},
		},
		After: func(normalized *jsonobj.Object) error {
			return assertTimeIsOrder(normalized.Value("start_time"), normalized.Value("end_time"))
		},
	},
	"mi": {Fields: []entityField{
		{Name: "title", Kind: "string", RequiredOnAdd: true},
		// board_name は add でも省略可（呼び出し側が mi_default_board で埋める）。
		{Name: "board_name", Kind: "string"},
		{Name: "is_checked", Kind: "boolean", DefaultOnAdd: false},
		{Name: "limit_time", Kind: "datetime", NullClears: true},
		{Name: "estimate_start_time", Kind: "datetime", NullClears: true},
		{Name: "estimate_end_time", Kind: "datetime", NullClears: true},
		{Name: "allow_create_board", Kind: "boolean", DefaultOnAdd: true, RevivesStaleBoolean: true},
	}},
	"kc": {Fields: []entityField{
		{Name: "title", Kind: "string", RequiredOnAdd: true},
		{Name: "num_value", Kind: "number", RequiredOnAdd: true},
		{Name: "related_time", Kind: "datetime"},
	}},
	"tag": {Fields: []entityField{
		{Name: "tag", Kind: "string", RequiredOnAdd: true},
		{Name: "target_id", Kind: "string", RequiredOnAdd: true, AddOnly: true},
	}},
	"text": {Fields: []entityField{
		{Name: "text", Kind: "string", RequiredOnAdd: true},
		{Name: "target_id", Kind: "string", RequiredOnAdd: true, AddOnly: true},
	}},
}

func assertFieldValue(field entityField, value any) (any, error) {
	switch field.Kind {
	case "string":
		return AssertTrimmedString(value, field.Name)
	case "url":
		return assertURLWithScheme(value, field.Name)
	case "integer":
		r := IntRange{}
		if field.Range != nil {
			r = *field.Range
		}
		return AssertInteger(value, field.Name, r)
	case "number":
		return AssertNumber(value, field.Name, NumberRange{})
	case "boolean":
		return AssertBoolean(value, field.Name)
	}
	return nil, Errorf("unknown field kind: %s", field.Kind)
}

// normalizeEntityArgs は add / update の引数を entityFieldSpecs に従って検証する。
// mode="add" は RequiredOnAdd を必須にし、mode="update" は id だけ必須の patch。
func normalizeEntityArgs(dataType string, args any, mode string) (*jsonobj.Object, error) {
	spec := entityFieldSpecs[dataType]
	source, err := assertArgs(args)
	if err != nil {
		return nil, err
	}
	fields := []entityField{}
	for _, field := range spec.Fields {
		if mode == "add" || !field.AddOnly {
			fields = append(fields, field)
		}
	}
	allowedKeys := NewStringSet("locale_name")
	if mode == "update" {
		allowedKeys.Add("id")
	}
	for _, field := range fields {
		allowedKeys.Add(field.Name)
	}
	if err := AssertKnownKeys(source, allowedKeys, "", noHiddenKeys); err != nil {
		return nil, err
	}

	normalized := jsonobj.New()
	if mode == "update" {
		id, err := AssertTrimmedString(source.Value("id"), "id")
		if err != nil {
			return nil, err
		}
		normalized.Set("id", id)
	}
	for _, field := range fields {
		if field.Kind == "datetime" {
			if mode == "update" && field.NullClears && source.Has(field.Name) && source.Value(field.Name) == nil {
				normalized.Set(field.Name, nil)
				continue
			}
			value, err := optionalDatetime(source, field.Name)
			if err != nil {
				return nil, err
			}
			normalized.Set(field.Name, value)
			continue
		}
		value := source.Value(field.Name)
		// 後付けの boolean 引数は古いスキーマのクライアントから "true" / "false" で届くので型を復元する。
		if field.RevivesStaleBoolean {
			if s, isString := value.(string); isString {
				trimmed := jsTrim(s)
				if trimmed == "true" || trimmed == "false" {
					value = trimmed == "true"
				}
			}
		}
		if mode == "add" && field.RequiredOnAdd {
			checked, err := assertFieldValue(field, value)
			if err != nil {
				return nil, err
			}
			normalized.Set(field.Name, checked)
			continue
		}
		if jsonobj.IsUndefined(value) {
			if mode == "add" && field.DefaultOnAdd != nil {
				normalized.Set(field.Name, field.DefaultOnAdd)
			} else {
				normalized.Set(field.Name, jsonobj.Undefined)
			}
			continue
		}
		checked, err := assertFieldValue(field, value)
		if err != nil {
			return nil, err
		}
		normalized.Set(field.Name, checked)
	}
	if source.Defined("locale_name") {
		localeName, err := AssertTrimmedString(source.Value("locale_name"), "locale_name")
		if err != nil {
			return nil, err
		}
		normalized.Set("locale_name", localeName)
	} else {
		normalized.Set("locale_name", jsonobj.Undefined)
	}
	if spec.After != nil {
		if err := spec.After(normalized); err != nil {
			return nil, err
		}
	}
	return normalized, nil
}

func NormalizeKmemoArgs(args any) (*jsonobj.Object, error) {
	return normalizeEntityArgs("kmemo", args, "add")
}
func NormalizeUrlogArgs(args any) (*jsonobj.Object, error) {
	return normalizeEntityArgs("urlog", args, "add")
}
func NormalizeNlogArgs(args any) (*jsonobj.Object, error) {
	return normalizeEntityArgs("nlog", args, "add")
}
func NormalizeLantanaArgs(args any) (*jsonobj.Object, error) {
	return normalizeEntityArgs("lantana", args, "add")
}
func NormalizeTimeIsArgs(args any) (*jsonobj.Object, error) {
	return normalizeEntityArgs("timeis", args, "add")
}
func NormalizeMiArgs(args any) (*jsonobj.Object, error) {
	return normalizeEntityArgs("mi", args, "add")
}
func NormalizeKcArgs(args any) (*jsonobj.Object, error) {
	return normalizeEntityArgs("kc", args, "add")
}
func NormalizeTagArgs(args any) (*jsonobj.Object, error) {
	return normalizeEntityArgs("tag", args, "add")
}
func NormalizeTextArgs(args any) (*jsonobj.Object, error) {
	return normalizeEntityArgs("text", args, "add")
}

func NormalizeUpdateKmemoArgs(args any) (*jsonobj.Object, error) {
	return normalizeEntityArgs("kmemo", args, "update")
}
func NormalizeUpdateUrlogArgs(args any) (*jsonobj.Object, error) {
	return normalizeEntityArgs("urlog", args, "update")
}
func NormalizeUpdateNlogArgs(args any) (*jsonobj.Object, error) {
	return normalizeEntityArgs("nlog", args, "update")
}
func NormalizeUpdateLantanaArgs(args any) (*jsonobj.Object, error) {
	return normalizeEntityArgs("lantana", args, "update")
}
func NormalizeUpdateTimeIsArgs(args any) (*jsonobj.Object, error) {
	return normalizeEntityArgs("timeis", args, "update")
}
func NormalizeUpdateMiArgs(args any) (*jsonobj.Object, error) {
	return normalizeEntityArgs("mi", args, "update")
}
func NormalizeUpdateKcArgs(args any) (*jsonobj.Object, error) {
	return normalizeEntityArgs("kc", args, "update")
}
func NormalizeUpdateTagArgs(args any) (*jsonobj.Object, error) {
	return normalizeEntityArgs("tag", args, "update")
}
func NormalizeUpdateTextArgs(args any) (*jsonobj.Object, error) {
	return normalizeEntityArgs("text", args, "update")
}

// NormalizeKftlArgs は gkill_submit_kftl の引数。
func NormalizeKftlArgs(args any) (*jsonobj.Object, error) {
	source, err := assertArgs(args)
	if err != nil {
		return nil, err
	}
	if err := AssertKnownKeys(source, NewStringSet("kftl_text", "locale_name", "idempotency_key"), "", noHiddenKeys); err != nil {
		return nil, err
	}
	kftlText, err := AssertTrimmedString(source.Value("kftl_text"), "kftl_text")
	if err != nil {
		return nil, err
	}
	normalized := jsonobj.Obj("kftl_text", kftlText, "locale_name", jsonobj.Undefined, "idempotency_key", jsonobj.Undefined)
	if source.Defined("locale_name") {
		localeName, err := AssertTrimmedString(source.Value("locale_name"), "locale_name")
		if err != nil {
			return nil, err
		}
		normalized.Set("locale_name", localeName)
	}
	// 同じ鍵で再送すれば二重登録にならない（ADR-0510）。
	if source.Defined("idempotency_key") {
		key, err := AssertTrimmedString(source.Value("idempotency_key"), "idempotency_key")
		if err != nil {
			return nil, err
		}
		normalized.Set("idempotency_key", key)
	}
	return normalized, nil
}

// normalizeDeleteTargets は削除・復活の「対象」引数を1本にまとめる。
// 単件は {id, data_type}、一括は {targets:[{id, data_type}, ...]}。
func normalizeDeleteTargets(args *jsonobj.Object, verb string) ([]any, error) {
	hasSingle := args.Defined("id") || args.Defined("data_type")
	hasBatch := args.Defined("targets")
	if hasSingle && hasBatch {
		return nil, InvalidArgument("targets", "cannot be combined with id / data_type; pass one form or the other", args.Value("targets"))
	}
	if !hasSingle && !hasBatch {
		return nil, InvalidArgument("id", "is required (or pass targets:[{id, data_type}] to "+verb+" several entries)", args.Value("id"))
	}
	if hasSingle {
		id, err := AssertTrimmedString(args.Value("id"), "id")
		if err != nil {
			return nil, err
		}
		rawDataType, err := AssertTrimmedString(args.Value("data_type"), "data_type")
		if err != nil {
			return nil, err
		}
		dataType := ToEntityDataType(rawDataType)
		if !DeleteDataTypes.Has(dataType) {
			return nil, InvalidArgument("data_type", "must be one of: "+DeleteDataTypes.Join(", "), dataType)
		}
		return jsonobj.Arr(jsonobj.Obj("id", id, "data_type", dataType)), nil
	}
	targets, isArray := jsonobj.AsArray(args.Value("targets"))
	if !isArray {
		return nil, InvalidArgument("targets", "must be an array of {id, data_type}", args.Value("targets"))
	}
	if len(targets) == 0 {
		return nil, InvalidArgument("targets", "must not be empty", args.Value("targets"))
	}
	if len(targets) > MaxDeleteTargets {
		return nil, InvalidArgument("targets", "must have at most "+itoa(MaxDeleteTargets)+" entries", int64(len(targets)))
	}
	out := make([]any, 0, len(targets))
	for index, target := range targets {
		prefix := "targets[" + itoa(index) + "]"
		if !IsPlainObject(target) {
			return nil, InvalidArgument(prefix, "must be an object with id and data_type", target)
		}
		entry := target.(*jsonobj.Object)
		// gkill_submit_kftl の created[] をそのまま渡すと updated / related_time が未知キーになる。変換の仕方を言う。
		if entry.Has("updated") || entry.Has("related_time") {
			return nil, InvalidArgument(
				prefix,
				"looks like a gkill_submit_kftl created[] entry: pass only {id, data_type} (drop updated / related_time), and skip "+
					"entries with updated:true — those are pre-existing records the submission updated (a timeis it ended), not records "+
					"it created, so deleting them is not an undo. Use created.filter(c => !c.updated).map(({id, data_type}) => ({id, data_type}))",
				target,
			)
		}
		if err := AssertKnownKeys(entry, NewStringSet("id", "data_type"), prefix, noHiddenKeys); err != nil {
			return nil, err
		}
		id, err := AssertTrimmedString(entry.Value("id"), prefix+".id")
		if err != nil {
			return nil, err
		}
		rawDataType, err := AssertTrimmedString(entry.Value("data_type"), prefix+".data_type")
		if err != nil {
			return nil, err
		}
		dataType := ToEntityDataType(rawDataType)
		if !DeleteDataTypes.Has(dataType) {
			return nil, InvalidArgument(prefix+".data_type", "must be one of: "+DeleteDataTypes.Join(", "), dataType)
		}
		out = append(out, jsonobj.Obj("id", id, "data_type", dataType))
	}
	return out, nil
}

func normalizeSoftDeleteArgs(args any, verb string) (*jsonobj.Object, error) {
	base, err := assertArgs(args)
	if err != nil {
		return nil, err
	}
	source := reviveStaleSchemaArgs(base, deleteTargetsStaleSchemaArgKinds)
	if err := AssertKnownKeys(source, NewStringSet("id", "data_type", "targets", "locale_name"), "", noHiddenKeys); err != nil {
		return nil, err
	}
	targets, err := normalizeDeleteTargets(source, verb)
	if err != nil {
		return nil, err
	}
	var localeName any = jsonobj.Undefined
	if source.Defined("locale_name") {
		localeName, err = AssertTrimmedString(source.Value("locale_name"), "locale_name")
		if err != nil {
			return nil, err
		}
	}
	return jsonobj.Obj("targets", targets, "batch", source.Defined("targets"), "locale_name", localeName), nil
}

// NormalizeRestoreArgs は gkill_restore_kyou の引数。
func NormalizeRestoreArgs(args any) (*jsonobj.Object, error) {
	return normalizeSoftDeleteArgs(args, "restore")
}

// NormalizeDeleteArgs は gkill_delete_kyou の引数。
func NormalizeDeleteArgs(args any) (*jsonobj.Object, error) {
	return normalizeSoftDeleteArgs(args, "delete")
}
