// Package jsonobj は JavaScript のオブジェクト意味論をそのまま持つ JSON の値モデル。
//
// MCP サーバは Node.js 実装（src/mcp、2026-09-20 に Go へ置き換え）から移したもので、
// 応答のバイト列を旧実装と一致させるには「キーの挿入順」「undefined と null の区別」
// 「JSON.stringify のエスケープ規則」を Go 側でも持つ必要がある。
// encoding/json の map はキーを辞書順に並べ替え、struct は undefined を表せないので、
// ここに順序つきの Object と JSON.stringify 互換の Marshal を置く。
//
// 値の集合: nil(null) / bool / string / json.Number / int / int64 / float64 / *Object / []any / Undefined。
// Undefined はキーが存在するのに値が undefined である状態（JS の `{a: undefined}`）で、
// Marshal では省かれ、Equal では「無いキー」と同じに扱う（vitest の toEqual と同じ）。
package jsonobj

import (
	"encoding/json"
	"math"
	"reflect"
	"strconv"
)

// UndefinedType は JS の undefined を表す型。
type UndefinedType struct{}

// Undefined は JS の undefined。キーが無いことの明示や、undefined 値の保持に使う。
var Undefined = UndefinedType{}

// IsUndefined は v が Undefined か。
func IsUndefined(v any) bool {
	_, ok := v.(UndefinedType)
	return ok
}

// Object は挿入順を保つ JSON オブジェクト。
//
// meta は JS の Symbol プロパティ相当で、JSON には出ず Keys にも載らない
// （payload.mjs の MINT_FILE_LINKS のような「印」を運ぶ）。
type Object struct {
	keys []string
	vals map[string]any
	meta map[string]any
}

// New は空の Object を返す。
func New() *Object {
	return &Object{vals: map[string]any{}}
}

// Obj はキーと値を交互に並べて Object を作る（テストと定数表の記述用）。
//
//	Obj("name", "x", "count", 3)
//
// 引数が奇数個、またはキーが string でなければ panic する（定数の書き間違いは起動時に落とす）。
func Obj(kv ...any) *Object {
	if len(kv)%2 != 0 {
		panic("jsonobj.Obj: odd number of arguments")
	}
	o := New()
	for i := 0; i < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			panic("jsonobj.Obj: key is not a string")
		}
		o.Set(key, kv[i+1])
	}
	return o
}

// Arr は []any を作る（テストと定数表の記述用）。
func Arr(items ...any) []any {
	out := make([]any, len(items))
	copy(out, items)
	return out
}

// Strings は []string を []any にする。
func Strings(items ...string) []any {
	out := make([]any, len(items))
	for i, s := range items {
		out[i] = s
	}
	return out
}

// Get はキーの値を返す。第2戻り値はキーの存在（値が Undefined でも true）。
func (o *Object) Get(key string) (any, bool) {
	if o == nil {
		return nil, false
	}
	v, ok := o.vals[key]
	return v, ok
}

// Value はキーの値を返す。キーが無ければ Undefined（JS の obj.key と同じ）。
func (o *Object) Value(key string) any {
	if o == nil {
		return Undefined
	}
	v, ok := o.vals[key]
	if !ok {
		return Undefined
	}
	return v
}

// Has はキーが存在するか（hasOwnProperty。値が Undefined でも true）。
func (o *Object) Has(key string) bool {
	if o == nil {
		return false
	}
	_, ok := o.vals[key]
	return ok
}

// Defined はキーが存在し、かつ値が Undefined でないか（JS の `obj.key !== undefined`）。
func (o *Object) Defined(key string) bool {
	if o == nil {
		return false
	}
	v, ok := o.vals[key]
	return ok && !IsUndefined(v)
}

// Set は値を入れる。既存のキーは位置を保ち、新しいキーは末尾に付く（JS と同じ）。
func (o *Object) Set(key string, v any) *Object {
	if _, ok := o.vals[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.vals[key] = v
	return o
}

// Delete はキーを消す。
func (o *Object) Delete(key string) {
	if o == nil {
		return
	}
	if _, ok := o.vals[key]; !ok {
		return
	}
	delete(o.vals, key)
	for i, k := range o.keys {
		if k == key {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			break
		}
	}
}

// Keys は JS の Object.keys と同じ順序でキーを返す:
// 配列添字とみなせるキー（"0" "1" … 2^32-2 の正準表記）を昇順に、そのあと残りを挿入順に。
func (o *Object) Keys() []string {
	if o == nil {
		return nil
	}
	var indexKeys []string
	var otherKeys []string
	for _, k := range o.keys {
		if isArrayIndexKey(k) {
			indexKeys = append(indexKeys, k)
		} else {
			otherKeys = append(otherKeys, k)
		}
	}
	if len(indexKeys) > 1 {
		sortIndexKeys(indexKeys)
	}
	out := make([]string, 0, len(o.keys))
	out = append(out, indexKeys...)
	out = append(out, otherKeys...)
	return out
}

// Len はキー数（Undefined 値のキーも数える。JS の Object.keys(o).length と同じ）。
func (o *Object) Len() int {
	if o == nil {
		return 0
	}
	return len(o.keys)
}

// Clone は浅いコピー（JS の `{...obj}`）。meta はコピーしない（Symbol も spread で写るが、
// 印を運ぶのは元の payload だけでよい）。
func (o *Object) Clone() *Object {
	out := New()
	if o == nil {
		return out
	}
	for _, k := range o.keys {
		out.Set(k, o.vals[k])
	}
	return out
}

// Merge は other のキーを o へ上書きで写す（JS の `{...o, ...other}` の other 側）。
func (o *Object) Merge(other *Object) *Object {
	if other == nil {
		return o
	}
	for _, k := range other.keys {
		o.Set(k, other.vals[k])
	}
	return o
}

// Meta は Symbol 相当の印を返す。
func (o *Object) Meta(key string) (any, bool) {
	if o == nil || o.meta == nil {
		return nil, false
	}
	v, ok := o.meta[key]
	return v, ok
}

// SetMeta は Symbol 相当の印を付ける（JSON には出ない）。
func (o *Object) SetMeta(key string, v any) {
	if o.meta == nil {
		o.meta = map[string]any{}
	}
	o.meta[key] = v
}

// String は string 値を返す（JS の `typeof v === "string"`）。
func (o *Object) String(key string) (string, bool) {
	v, ok := o.Get(key)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// Bool は bool 値を返す。
func (o *Object) Bool(key string) (bool, bool) {
	v, ok := o.Get(key)
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

// Float は数値を float64 で返す（JS の `typeof v === "number"` に相当。NaN/Inf も数値）。
func (o *Object) Float(key string) (float64, bool) {
	v, ok := o.Get(key)
	if !ok {
		return 0, false
	}
	return ToFloat(v)
}

// Int は整数値を int64 で返す（JS の Number.isInteger）。
func (o *Object) Int(key string) (int64, bool) {
	f, ok := o.Float(key)
	if !ok || !IsIntegral(f) {
		return 0, false
	}
	return int64(f), true
}

// Array は配列値を返す。
func (o *Object) Array(key string) ([]any, bool) {
	v, ok := o.Get(key)
	if !ok {
		return nil, false
	}
	return AsArray(v)
}

// Object は入れ子の Object を返す。
func (o *Object) Object(key string) (*Object, bool) {
	v, ok := o.Get(key)
	if !ok {
		return nil, false
	}
	child, ok := v.(*Object)
	if !ok || child == nil {
		return nil, false
	}
	return child, true
}

// Path はキーの列をたどる（JS の `a.b.c`）。途中で切れたら Undefined。
func (o *Object) Path(keys ...string) any {
	var cur any = o
	for _, k := range keys {
		obj, ok := cur.(*Object)
		if !ok || obj == nil {
			return Undefined
		}
		cur = obj.Value(k)
	}
	return cur
}

// AsArray は v が配列なら []any で返す（[]string / []*Object も受ける）。
func AsArray(v any) ([]any, bool) {
	switch a := v.(type) {
	case []any:
		return a, true
	case []string:
		return Strings(a...), true
	case []*Object:
		out := make([]any, len(a))
		for i, item := range a {
			out[i] = item
		}
		return out, true
	case nil:
		return nil, false
	}
	// []int64 / []int / []float64 など任意のスライスも配列として扱う（[]byte は除く）
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice || rv.Type().Elem().Kind() == reflect.Uint8 {
		return nil, false
	}
	out := make([]any, rv.Len())
	for i := range out {
		out[i] = rv.Index(i).Interface()
	}
	return out, true
}

// IsNumber は v が JS の number に相当するか。
func IsNumber(v any) bool {
	_, ok := ToFloat(v)
	return ok
}

// ToFloat は数値を float64 にする。数値でなければ false。
func ToFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case json.Number:
		f, err := strconv.ParseFloat(string(n), 64)
		if err != nil {
			return 0, false
		}
		return f, true
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	default:
		return 0, false
	}
}

// IsIntegral は JS の Number.isInteger。
func IsIntegral(f float64) bool {
	return !math.IsNaN(f) && !math.IsInf(f, 0) && f == math.Trunc(f)
}

// MaxSafeInteger は JS の Number.MAX_SAFE_INTEGER。
const MaxSafeInteger = float64(9007199254740991)

// IsSafeInteger は JS の Number.isSafeInteger。
func IsSafeInteger(f float64) bool {
	return IsIntegral(f) && math.Abs(f) <= MaxSafeInteger
}

// TypeOf は JS の typeof に相当する分類（describeValueType の材料）。
// "null" / "array" / "object" / "string" / "number" / "boolean" / "undefined" / "unknown"。
func TypeOf(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case UndefinedType:
		return "undefined"
	case string:
		return "string"
	case bool:
		return "boolean"
	case *Object:
		if v.(*Object) == nil {
			return "null"
		}
		return "object"
	case []any, []string, []*Object:
		return "array"
	}
	if IsNumber(v) {
		return "number"
	}
	return "unknown"
}

// isArrayIndexKey は JS が「配列添字」とみなすキーか（正準な 10 進で 0..2^32-2）。
func isArrayIndexKey(k string) bool {
	if k == "" || len(k) > 10 {
		return false
	}
	if k == "0" {
		return true
	}
	if k[0] == '0' {
		return false
	}
	for i := 0; i < len(k); i++ {
		if k[i] < '0' || k[i] > '9' {
			return false
		}
	}
	n, err := strconv.ParseUint(k, 10, 64)
	if err != nil {
		return false
	}
	return n <= 4294967294
}

func sortIndexKeys(keys []string) {
	// 件数は小さいので単純な挿入ソートで十分
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0; j-- {
			a, _ := strconv.ParseUint(keys[j-1], 10, 64)
			b, _ := strconv.ParseUint(keys[j], 10, 64)
			if a <= b {
				break
			}
			keys[j-1], keys[j] = keys[j], keys[j-1]
		}
	}
}
