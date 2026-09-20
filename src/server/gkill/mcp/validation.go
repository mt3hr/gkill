package mcp

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/mt3hr/gkill/src/server/gkill/mcp/jsonobj"
)

// StringSet は挿入順を保つ文字列の集合（JS の Set）。
// 「must be one of: a, b, c」のような文言は挿入順に依存するので、map では代用できない。
type StringSet struct {
	items []string
	index map[string]struct{}
}

// NewStringSet は集合を作る。重複は最初の1つだけ残る。
func NewStringSet(items ...string) *StringSet {
	s := &StringSet{index: map[string]struct{}{}}
	for _, item := range items {
		s.Add(item)
	}
	return s
}

// Add は末尾へ足す（既にあれば何もしない）。
func (s *StringSet) Add(item string) {
	if _, ok := s.index[item]; ok {
		return
	}
	s.index[item] = struct{}{}
	s.items = append(s.items, item)
}

// Has は含むか。nil の集合は何も含まない。
func (s *StringSet) Has(item string) bool {
	if s == nil {
		return false
	}
	_, ok := s.index[item]
	return ok
}

// Values は挿入順の一覧（コピー）。
func (s *StringSet) Values() []string {
	if s == nil {
		return nil
	}
	out := make([]string, len(s.items))
	copy(out, s.items)
	return out
}

// Sorted は辞書順の一覧。
func (s *StringSet) Sorted() []string {
	out := s.Values()
	sort.Strings(out)
	return out
}

// Len は要素数。
func (s *StringSet) Len() int {
	if s == nil {
		return 0
	}
	return len(s.items)
}

// Union は自分と others を合わせた新しい集合（JS の new Set([...a, ...b])）。
func (s *StringSet) Union(others ...*StringSet) *StringSet {
	out := NewStringSet(s.Values()...)
	for _, other := range others {
		for _, item := range other.Values() {
			out.Add(item)
		}
	}
	return out
}

// Join は挿入順に区切り文字で連結する。
func (s *StringSet) Join(sep string) string {
	return strings.Join(s.Values(), sep)
}

// NumberRange は AssertNumber の範囲指定（nil は未指定）。
type NumberRange struct {
	MinExclusive *float64
	Min          *float64
	Max          *float64
}

// IntRange は AssertInteger の範囲指定（nil は未指定）。
type IntRange struct {
	Min *int64
	Max *int64
}

// F は *float64 の短い書き方。
func F(v float64) *float64 { return &v }

// I は *int64 の短い書き方。
func I(v int64) *int64 { return &v }

// AssertObject は value が素のオブジェクトであることを検査して返す。
func AssertObject(value any, field string) (*jsonobj.Object, error) {
	if !IsPlainObject(value) {
		return nil, InvalidArgument(field, "must be an object", value)
	}
	return value.(*jsonobj.Object), nil
}

// AssertObjectAllowUndefined は undefined を許す AssertObject（undefined なら nil, nil）。
func AssertObjectAllowUndefined(value any, field string) (*jsonobj.Object, error) {
	if jsonobj.IsUndefined(value) {
		return nil, nil
	}
	return AssertObject(value, field)
}

// AssertBoolean は bool であることを検査する。
func AssertBoolean(value any, field string) (bool, error) {
	b, ok := value.(bool)
	if !ok {
		return false, InvalidArgument(field, "must be a boolean", value)
	}
	return b, nil
}

// AssertNumber は有限の数値であることと範囲を検査する。
func AssertNumber(value any, field string, r NumberRange) (float64, error) {
	f, ok := jsonobj.ToFloat(value)
	if !ok || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, InvalidArgument(field, "must be a finite number", value)
	}
	if r.MinExclusive != nil && f <= *r.MinExclusive {
		return 0, InvalidArgument(field, fmt.Sprintf("must be greater than %s", jsonobj.MarshalString(*r.MinExclusive)), value)
	}
	if r.Min != nil && f < *r.Min {
		return 0, InvalidArgument(field, fmt.Sprintf("must be greater than or equal to %s", jsonobj.MarshalString(*r.Min)), value)
	}
	if r.Max != nil && f > *r.Max {
		return 0, InvalidArgument(field, fmt.Sprintf("must be less than or equal to %s", jsonobj.MarshalString(*r.Max)), value)
	}
	return f, nil
}

// AssertInteger は整数であること・安全な範囲・min/max を検査する。
func AssertInteger(value any, field string, r IntRange) (int64, error) {
	f, ok := jsonobj.ToFloat(value)
	if !ok || !jsonobj.IsIntegral(f) {
		return 0, InvalidArgument(field, "must be an integer", value)
	}
	// 2^53 を超える整数は JSON を往復するだけで別の値に化ける。受理すると
	// 「保存した金額と読み戻した金額が違う」が例外もエラーも無しに起きる。
	if !jsonobj.IsSafeInteger(f) {
		return 0, InvalidArgument(
			field,
			fmt.Sprintf("must be within the safe integer range (at most %s)", jsonobj.MarshalString(jsonobj.MaxSafeInteger)),
			value,
		)
	}
	n := int64(f)
	if r.Min != nil && n < *r.Min {
		return 0, InvalidArgument(field, fmt.Sprintf("must be greater than or equal to %d", *r.Min), value)
	}
	if r.Max != nil && n > *r.Max {
		return 0, InvalidArgument(field, fmt.Sprintf("must be less than or equal to %d", *r.Max), value)
	}
	return n, nil
}

// AssertTrimmedString は空でない文字列であることを検査し、前後の空白を落として返す。
func AssertTrimmedString(value any, field string) (string, error) {
	s, ok := value.(string)
	if !ok {
		return "", InvalidArgument(field, "must be a string", value)
	}
	trimmed := jsTrim(s)
	if trimmed == "" {
		return "", InvalidArgument(field, "must not be empty", value)
	}
	return trimmed, nil
}

// AssertStringArray は文字列の配列であることを検査し、各要素を trim して返す。
func AssertStringArray(value any, field string) ([]string, error) {
	arr, ok := jsonobj.AsArray(value)
	if !ok {
		return nil, InvalidArgument(field, "must be an array of strings", value)
	}
	out := make([]string, 0, len(arr))
	for index, item := range arr {
		s, err := AssertTrimmedString(item, fmt.Sprintf("%s[%d]", field, index))
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

// AssertIntegerArray は整数の配列であることと各要素の範囲を検査する。
func AssertIntegerArray(value any, field string, r IntRange) ([]int64, error) {
	arr, ok := jsonobj.AsArray(value)
	if !ok {
		return nil, InvalidArgument(field, "must be an array of integers", value)
	}
	out := make([]int64, 0, len(arr))
	for index, item := range arr {
		n, err := AssertInteger(item, fmt.Sprintf("%s[%d]", field, index), r)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

// UnknownKeyMessage は「その名前の引数は無い」の文言。**未知キーを弾く全箇所がこれを使う。**
//
// 呼び出し側の書き間違いと、クライアントが握っている古いツール一覧（改名・削除前の
// 名前を今も載せている）とは、サーバからは区別できない。ツール一覧はクライアントの
// セッション寿命で固定されるので、サーバを直しても生きているセッションには届かない
// （2026-09-14 のレビューで、ChatGPT が改名前の検索条件名を一覧どおりに送って
// 「is not supported」だけを受け取り、行き止まりになった）。
// EntityNotFoundMessage と同じく**区別できないことを言う**: 一覧にその名前があるなら
// 一覧が古い、と両方の可能性を示し、再接続と gkill_status の照合を案内する。
// 改名前の名前そのものは書かない（旧綴りは verify_docs が禁止する。ADR-0806）。
func UnknownKeyMessage() string {
	return "is not supported (see detail.allowed for the accepted names). Either the name is misspelled, or the tool list " +
		"your client holds is stale: tool lists are fetched once per client session, so a field renamed or removed on the " +
		"server stays in your list until you reconnect. If your tool schema lists this name, reconnect the MCP client; " +
		"gkill_status reports the server's current schema_revision to compare against the one in its description"
}

// AssertKnownKeys は未知のキーを弾く。
// field は「どのオブジェクトの中か」を示す接頭辞で、空なら "arguments"（ツール引数の直下）。
//
// 未知のキーは**全部集めて1回で**投げる（detail.unknown）。1件ずつ返すと、キーを3つ
// 間違えた呼び出しは3往復になる（2026-09-18 の実利用報告。KFTL は全行まとめて返す）。
// detail.field は先頭の未知キー（既存の呼び出し側とテストは1件の形を前提にしている）。
// hidden は「受理はするが detail.allowed に載せない」キー（廃止済み引数。公開スキーマにも無いので、
// 一覧に出すと「only_latest_data:false なら過去版が読めるのか」と考える余地を作るだけ。ADR-0620）。
func AssertKnownKeys(value *jsonobj.Object, allowed *StringSet, field string, hidden *StringSet) error {
	if field == "" {
		field = "arguments"
	}
	var unknown []string
	for _, key := range value.Keys() {
		if allowed.Has(key) || hidden.Has(key) {
			continue
		}
		unknown = append(unknown, key)
	}
	if len(unknown) == 0 {
		return nil
	}
	suffix := ""
	if len(unknown) > 1 {
		suffix = fmt.Sprintf(" (%d unknown names in this object: %s)", len(unknown), strings.Join(unknown, ", "))
	}
	var allowedList []string
	for _, key := range allowed.Values() {
		if hidden.Has(key) {
			continue
		}
		allowedList = append(allowedList, key)
	}
	sort.Strings(allowedList)
	if allowedList == nil {
		allowedList = []string{}
	}
	return InvalidArgument(
		field+"."+unknown[0],
		UnknownKeyMessage()+suffix,
		value.Value(unknown[0]),
		"allowed", jsonobj.Strings(allowedList...),
		"unknown", jsonobj.Strings(unknown...),
	)
}

// jsTrim は JS の String#trim（空白と改行に加え BOM や全角空白も落とす）。
func jsTrim(s string) string {
	return strings.TrimFunc(s, isJSWhitespace)
}

func isJSWhitespace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\v', '\f', '\r', 0x00A0, 0x1680, 0x2028, 0x2029, 0x202F, 0x205F, 0x3000, 0xFEFF:
		return true
	}
	return r >= 0x2000 && r <= 0x200A
}
