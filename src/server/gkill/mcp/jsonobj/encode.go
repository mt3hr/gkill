package jsonobj

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"unicode/utf8"
)

// Marshal は JSON.stringify(v) と同じバイト列を返す（空白なし）。
//
// JS との一致のために encoding/json と違うところ:
//   - キーは挿入順（配列添字風のキーだけ先頭に昇順）
//   - Undefined の値を持つキーは省き、配列の中の Undefined は null
//   - 文字列は < > & と U+2028 / U+2029 をエスケープしない。\b \f は短形式
//   - NaN / ±Inf は null
//   - 数値は ES6 の Number#toString と同じ形（1e21 以上と 1e-7 未満だけ指数表記）
func Marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := encodeValue(&buf, v, "", ""); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// MarshalString は Marshal の string 版。失敗は panic（値の集合は閉じているので実運用では起きない）。
func MarshalString(v any) string {
	b, err := Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// MarshalIndent は JSON.stringify(v, null, indent) と同じ形（indent は "  " など）。
func MarshalIndent(v any, indent string) ([]byte, error) {
	var buf bytes.Buffer
	if err := encodeValue(&buf, v, indent, ""); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// MarshalIndentString は MarshalIndent の string 版。
func MarshalIndentString(v any, indent string) string {
	b, err := MarshalIndent(v, indent)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// ByteLength は Marshal した UTF-8 のバイト数（Buffer.byteLength(JSON.stringify(v))）。
func ByteLength(v any) int {
	b, err := Marshal(v)
	if err != nil {
		return 0
	}
	return len(b)
}

func encodeValue(buf *bytes.Buffer, v any, indent, prefix string) error {
	switch x := v.(type) {
	case nil:
		buf.WriteString("null")
	case UndefinedType:
		// 配列の要素として来たときだけここへ来る（オブジェクトのキーは呼び出し側で省く）
		buf.WriteString("null")
	case bool:
		if x {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case string:
		encodeString(buf, x)
	case json.Number:
		f, err := strconv.ParseFloat(string(x), 64)
		if err != nil {
			return fmt.Errorf("jsonobj: invalid number %q", string(x))
		}
		encodeFloat(buf, f)
	case float64:
		encodeFloat(buf, x)
	case float32:
		encodeFloat(buf, float64(x))
	case int:
		buf.WriteString(strconv.FormatInt(int64(x), 10))
	case int8:
		buf.WriteString(strconv.FormatInt(int64(x), 10))
	case int16:
		buf.WriteString(strconv.FormatInt(int64(x), 10))
	case int32:
		buf.WriteString(strconv.FormatInt(int64(x), 10))
	case int64:
		buf.WriteString(strconv.FormatInt(x, 10))
	case uint:
		buf.WriteString(strconv.FormatUint(uint64(x), 10))
	case uint8:
		buf.WriteString(strconv.FormatUint(uint64(x), 10))
	case uint16:
		buf.WriteString(strconv.FormatUint(uint64(x), 10))
	case uint32:
		buf.WriteString(strconv.FormatUint(uint64(x), 10))
	case uint64:
		buf.WriteString(strconv.FormatUint(x, 10))
	case *Object:
		if x == nil {
			buf.WriteString("null")
			return nil
		}
		return encodeObject(buf, x, indent, prefix)
	case []any:
		return encodeArray(buf, x, indent, prefix)
	case []string:
		return encodeArray(buf, Strings(x...), indent, prefix)
	case []*Object:
		arr, _ := AsArray(x)
		return encodeArray(buf, arr, indent, prefix)
	case map[string]any:
		// Go の map は順序を持たないので、encoding/json と同じく辞書順で出す
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		o := New()
		for _, k := range keys {
			o.Set(k, x[k])
		}
		return encodeObject(buf, o, indent, prefix)
	default:
		// struct 等は encoding/json で一度 JSON にしてから順序つきの値に戻して出し直す
		// （HTML エスケープと数値の形をこのパッケージの規則に揃えるため）
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Ptr && rv.IsNil() {
			buf.WriteString("null")
			return nil
		}
		var tmp bytes.Buffer
		enc := json.NewEncoder(&tmp)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(v); err != nil {
			return err
		}
		decoded, err := Unmarshal(bytes.TrimRight(tmp.Bytes(), "\n"))
		if err != nil {
			return err
		}
		return encodeValue(buf, decoded, indent, prefix)
	}
	return nil
}

func encodeObject(buf *bytes.Buffer, o *Object, indent, prefix string) error {
	keys := o.Keys()
	// undefined の値は省く（JSON.stringify と同じ）
	present := make([]string, 0, len(keys))
	for _, k := range keys {
		if IsUndefined(o.vals[k]) {
			continue
		}
		present = append(present, k)
	}
	if len(present) == 0 {
		buf.WriteString("{}")
		return nil
	}
	buf.WriteByte('{')
	inner := prefix + indent
	for i, k := range present {
		if i > 0 {
			buf.WriteByte(',')
		}
		if indent != "" {
			buf.WriteByte('\n')
			buf.WriteString(inner)
		}
		encodeString(buf, k)
		buf.WriteByte(':')
		if indent != "" {
			buf.WriteByte(' ')
		}
		if err := encodeValue(buf, o.vals[k], indent, inner); err != nil {
			return err
		}
	}
	if indent != "" {
		buf.WriteByte('\n')
		buf.WriteString(prefix)
	}
	buf.WriteByte('}')
	return nil
}

func encodeArray(buf *bytes.Buffer, arr []any, indent, prefix string) error {
	if len(arr) == 0 {
		buf.WriteString("[]")
		return nil
	}
	buf.WriteByte('[')
	inner := prefix + indent
	for i, item := range arr {
		if i > 0 {
			buf.WriteByte(',')
		}
		if indent != "" {
			buf.WriteByte('\n')
			buf.WriteString(inner)
		}
		if err := encodeValue(buf, item, indent, inner); err != nil {
			return err
		}
	}
	if indent != "" {
		buf.WriteByte('\n')
		buf.WriteString(prefix)
	}
	buf.WriteByte(']')
	return nil
}

const hexDigits = "0123456789abcdef"

// encodeString は JSON.stringify の文字列エスケープ。
func encodeString(buf *bytes.Buffer, s string) {
	buf.WriteByte('"')
	start := 0
	for i := 0; i < len(s); {
		b := s[i]
		if b < utf8.RuneSelf {
			if b >= 0x20 && b != '"' && b != '\\' {
				i++
				continue
			}
			if start < i {
				buf.WriteString(s[start:i])
			}
			switch b {
			case '"':
				buf.WriteString(`\"`)
			case '\\':
				buf.WriteString(`\\`)
			case '\b':
				buf.WriteString(`\b`)
			case '\f':
				buf.WriteString(`\f`)
			case '\n':
				buf.WriteString(`\n`)
			case '\r':
				buf.WriteString(`\r`)
			case '\t':
				buf.WriteString(`\t`)
			default:
				buf.WriteString(`\u00`)
				buf.WriteByte(hexDigits[b>>4])
				buf.WriteByte(hexDigits[b&0xF])
			}
			i++
			start = i
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			// 不正な UTF-8 は U+FFFD にする（JS の文字列は常に妥当なので、ここは Go 側だけの縁）
			if start < i {
				buf.WriteString(s[start:i])
			}
			buf.WriteString("�")
			i += size
			start = i
			continue
		}
		i += size
	}
	if start < len(s) {
		buf.WriteString(s[start:])
	}
	buf.WriteByte('"')
}

// encodeFloat は ES6 の Number#toString と同じ表記（encoding/json の規則と同じ）。
func encodeFloat(buf *bytes.Buffer, f float64) {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		// JSON.stringify(NaN) / (Infinity) は "null"
		buf.WriteString("null")
		return
	}
	if f == 0 {
		// JS では -0 も "0"
		buf.WriteByte('0')
		return
	}
	abs := math.Abs(f)
	format := byte('f')
	if abs < 1e-6 || abs >= 1e21 {
		format = 'e'
	}
	b := strconv.AppendFloat(nil, f, format, -1, 64)
	if format == 'e' {
		// "1e-07" → "1e-7"、"1e+21" はそのまま
		n := len(b)
		if n >= 4 && b[n-4] == 'e' && b[n-3] == '-' && b[n-2] == '0' {
			b[n-2] = b[n-1]
			b = b[:n-1]
		}
	}
	buf.Write(b)
}
