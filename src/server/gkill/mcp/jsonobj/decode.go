package jsonobj

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Unmarshal は JSON を順序つきの値へ復元する（JSON.parse）。
//
// オブジェクトは *Object（キーは出現順）、配列は []any、数値は json.Number、
// 文字列 / bool / null はそのまま。JSON.parse と同じく、重複キーは後勝ちで位置は最初の出現。
func Unmarshal(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	v, err := decodeValue(dec)
	if err != nil {
		return nil, err
	}
	// 末尾のゴミ（"{} x"）は JSON.parse と同じく拒否する
	if _, err := dec.Token(); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("jsonobj: unexpected trailing data")
		}
		return nil, err
	}
	return v, nil
}

// MustUnmarshal は Unmarshal の失敗を panic にする（テスト・定数用）。
func MustUnmarshal(data string) any {
	v, err := Unmarshal([]byte(data))
	if err != nil {
		panic(err)
	}
	return v
}

// UnmarshalObject は JSON がオブジェクトであることまで確かめて *Object を返す。
func UnmarshalObject(data []byte) (*Object, error) {
	v, err := Unmarshal(data)
	if err != nil {
		return nil, err
	}
	o, ok := v.(*Object)
	if !ok {
		return nil, fmt.Errorf("jsonobj: not a JSON object")
	}
	return o, nil
}

func decodeValue(dec *json.Decoder) (any, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	return decodeFromToken(dec, tok)
}

func decodeFromToken(dec *json.Decoder, tok json.Token) (any, error) {
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			o := New()
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyTok.(string)
				if !ok {
					return nil, fmt.Errorf("jsonobj: object key is not a string")
				}
				val, err := decodeValue(dec)
				if err != nil {
					return nil, err
				}
				o.Set(key, val)
			}
			if _, err := dec.Token(); err != nil { // '}'
				return nil, err
			}
			return o, nil
		case '[':
			arr := []any{}
			for dec.More() {
				val, err := decodeValue(dec)
				if err != nil {
					return nil, err
				}
				arr = append(arr, val)
			}
			if _, err := dec.Token(); err != nil { // ']'
				return nil, err
			}
			return arr, nil
		default:
			return nil, fmt.Errorf("jsonobj: unexpected delimiter %q", string(t))
		}
	case nil:
		return nil, nil
	default:
		// string / bool / json.Number
		return t, nil
	}
}

// DeepClone は値を再帰的に複製する（structuredClone）。
func DeepClone(v any) any {
	switch x := v.(type) {
	case *Object:
		if x == nil {
			return (*Object)(nil)
		}
		out := New()
		for _, k := range x.keys {
			out.Set(k, DeepClone(x.vals[k]))
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = DeepClone(item)
		}
		return out
	default:
		return v
	}
}

// FromGo は Go の値（struct / map / slice）を順序つきの値へ変換する（encoding/json の写像を経由）。
func FromGo(v any) (any, error) {
	switch v.(type) {
	case nil, *Object, []any, string, bool, json.Number, float64, int, int64:
		return v, nil
	}
	b, err := Marshal(v)
	if err != nil {
		return nil, err
	}
	return Unmarshal(b)
}
