package jsonobj

import (
	"encoding/json"
	"math"
	"testing"
)

func TestObjectKeepsInsertionOrder(t *testing.T) {
	o := Obj("b", 1, "a", 2)
	o.Set("c", 3)
	o.Set("b", 9) // 既存キーは位置を保つ
	got := MarshalString(o)
	if got != `{"b":9,"a":2,"c":3}` {
		t.Fatalf("got %s", got)
	}
	o.Delete("a")
	if got := MarshalString(o); got != `{"b":9,"c":3}` {
		t.Fatalf("after delete: %s", got)
	}
}

func TestObjectKeysOrderArrayIndexKeysFirst(t *testing.T) {
	// JS: Object.keys({b:1, "10":1, "2":1, a:1}) → ["2","10","b","a"]
	o := Obj("b", 1, "10", 1, "2", 1, "a", 1)
	keys := o.Keys()
	want := []string{"2", "10", "b", "a"}
	if len(keys) != len(want) {
		t.Fatalf("keys=%v", keys)
	}
	for i := range want {
		if keys[i] != want[i] {
			t.Fatalf("keys=%v want %v", keys, want)
		}
	}
	// "01" や "-1" は添字ではない
	o2 := Obj("01", 1, "-1", 1, "0", 1)
	if got := MarshalString(o2); got != `{"0":1,"01":1,"-1":1}` {
		t.Fatalf("got %s", got)
	}
}

func TestMarshalOmitsUndefinedKeysAndNullsUndefinedInArrays(t *testing.T) {
	o := Obj("a", Undefined, "b", 1, "c", Arr(1, Undefined, nil))
	if got := MarshalString(o); got != `{"b":1,"c":[1,null,null]}` {
		t.Fatalf("got %s", got)
	}
	if o.Len() != 3 || !o.Has("a") || o.Defined("a") {
		t.Fatalf("undefined key must still exist: len=%d has=%v defined=%v", o.Len(), o.Has("a"), o.Defined("a"))
	}
}

func TestMarshalStringEscapesLikeJavaScript(t *testing.T) {
	cases := map[string]string{
		"a<b>&c":     `"a<b>&c"`,
		"  ":         "\"  \"",
		"\b\f\n\r\t": `"\b\f\n\r\t"`,
		"\x01\x1f":   `"\u0001\u001f"`,
		"\x7f":       "\"\x7f\"",
		`q"b\`:       `"q\"b\\"`,
		"日本語😀":       `"日本語😀"`,
		"bad\xffutf": "\"bad�utf\"",
	}
	for in, want := range cases {
		if got := MarshalString(in); got != want {
			t.Errorf("%q: got %s want %s", in, got, want)
		}
	}
}

func TestMarshalNumbersLikeES6(t *testing.T) {
	// 定数の畳み込みを避ける（0.1 + 0.2 をリテラルで書くとコンパイル時に正確な 0.3 になる）
	tenth, twoTenths := 0.1, 0.2
	cases := []struct {
		in   any
		want string
	}{
		{0.25, "0.25"},
		{float64(20), "20"},
		{int64(-1500), "-1500"},
		{json.Number("1.0"), "1"},
		{json.Number("1e2"), "100"},
		{1e21, "1e+21"},
		{1e-7, "1e-7"},
		{5e-7, "5e-7"},
		{0.000001, "0.000001"},
		{123456789012345680000.0, "123456789012345680000"},
		{math.NaN(), "null"},
		{math.Inf(1), "null"},
		{math.Copysign(0, -1), "0"},
		{tenth + twoTenths, "0.30000000000000004"},
		{72.5, "72.5"},
		{float64(9007199254740991), "9007199254740991"},
	}
	for _, c := range cases {
		if got := MarshalString(c.in); got != c.want {
			t.Errorf("%v: got %s want %s", c.in, got, c.want)
		}
	}
}

func TestMarshalIndentMatchesJSONStringifyWithTwoSpaces(t *testing.T) {
	o := Obj("a", 1, "b", Arr(1, Obj("c", "x")), "d", Obj(), "e", Arr(), "f", nil)
	want := "{\n  \"a\": 1,\n  \"b\": [\n    1,\n    {\n      \"c\": \"x\"\n    }\n  ],\n  \"d\": {},\n  \"e\": [],\n  \"f\": null\n}"
	if got := MarshalIndentString(o, "  "); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
}

func TestUnmarshalPreservesOrderAndNumbers(t *testing.T) {
	v := MustUnmarshal(`{"z":1,"a":[true,null,"s",2.5],"m":{"k":"v"}}`)
	o, ok := v.(*Object)
	if !ok {
		t.Fatalf("not an object: %T", v)
	}
	if got := MarshalString(o); got != `{"z":1,"a":[true,null,"s",2.5],"m":{"k":"v"}}` {
		t.Fatalf("round trip: %s", got)
	}
	if n, ok := o.Int("z"); !ok || n != 1 {
		t.Fatalf("Int z: %v %v", n, ok)
	}
	if _, err := Unmarshal([]byte(`{} x`)); err == nil {
		t.Fatal("trailing data must be rejected")
	}
	if _, err := Unmarshal([]byte(`{bad`)); err == nil {
		t.Fatal("invalid JSON must be rejected")
	}
}

func TestEqualSemantics(t *testing.T) {
	if !Equal(Obj("a", 1, "u", Undefined), Obj("a", json.Number("1"))) {
		t.Fatal("undefined keys are ignored and numbers compare by value")
	}
	if Equal(Obj("a", 1), Obj("a", 1, "b", nil)) {
		t.Fatal("null is a value, not undefined")
	}
	if !Equal(Arr("x", 2), Strings("x", "2")) == false {
		// Strings は "2" を文字列にするので等しくない
		t.Fatal("string \"2\" must not equal number 2")
	}
	if !Equal([]string{"a"}, Arr("a")) {
		t.Fatal("[]string equals []any of strings")
	}
	if !Equal(nil, (*Object)(nil)) {
		t.Fatal("nil object equals null")
	}
	if !Contains(Obj("a", 1, "b", Obj("c", 2, "d", 3)), Obj("b", Obj("c", 2))) {
		t.Fatal("Contains must match nested subsets")
	}
}

func TestFromGoConvertsStructsThroughEncodingJSON(t *testing.T) {
	type sample struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
		Skip  string `json:"skip,omitempty"`
	}
	v, err := FromGo(sample{Name: "<x>", Count: 2})
	if err != nil {
		t.Fatal(err)
	}
	if got := MarshalString(v); got != `{"name":"<x>","count":2}` {
		t.Fatalf("got %s", got)
	}
	if got := MarshalString(sample{Name: "y"}); got != `{"name":"y","count":0}` {
		t.Fatalf("direct struct: %s", got)
	}
}

func TestPathAndTypedAccessors(t *testing.T) {
	o := MustUnmarshal(`{"inputSchema":{"properties":{"topic":{"enum":["index","mi"]}}},"flag":true,"n":3.5}`).(*Object)
	enum, ok := AsArray(o.Path("inputSchema", "properties", "topic", "enum"))
	if !ok || len(enum) != 2 || enum[1] != "mi" {
		t.Fatalf("path: %v %v", enum, ok)
	}
	if !IsUndefined(o.Path("inputSchema", "nope", "x")) {
		t.Fatal("missing path must be Undefined")
	}
	if b, ok := o.Bool("flag"); !ok || !b {
		t.Fatal("Bool")
	}
	if _, ok := o.Int("n"); ok {
		t.Fatal("3.5 is not an integer")
	}
	if TypeOf(o) != "object" || TypeOf(Arr()) != "array" || TypeOf(nil) != "null" || TypeOf(Undefined) != "undefined" || TypeOf(json.Number("1")) != "number" {
		t.Fatal("TypeOf")
	}
}
