package gkill_plugin

import (
	"encoding/json"
	"testing"
)

// TestEmitsKyouOrDefault_DefaultsToTrue は、emits_kyou を書いていない
// 既存のプラグインが従来どおり Kyou を返す扱いになることを確認する。
//
// ここが false 側に倒れると、manifest.json を書き換えていない全プラグインが
// 「記録保管場所」の一覧から消え、検索結果にも出なくなる。
func TestEmitsKyouOrDefault_DefaultsToTrue(t *testing.T) {
	cases := map[string]bool{
		`{"name":"p"}`:                    true,
		`{"name":"p","emits_kyou":null}`:  true,
		`{"name":"p","emits_kyou":true}`:  true,
		`{"name":"p","emits_kyou":false}`: false,
	}
	for body, want := range cases {
		var manifest PluginManifest
		if err := json.Unmarshal([]byte(body), &manifest); err != nil {
			t.Fatalf("unmarshal %s: %v", body, err)
		}
		if got := manifest.EmitsKyouOrDefault(); got != want {
			t.Errorf("%s → EmitsKyouOrDefault() = %v, want %v", body, got, want)
		}
	}
}

// TestEmitsKyou_IsOmittedWhenUnset は、既定のまま書き出したmanifestに
// emits_kyou が現れないことを確認する。
//
// 既定を明示的な true として書き出すと、それを読む古いgkillでは
// 未知のキーになる。省略のままにしておく。
func TestEmitsKyou_IsOmittedWhenUnset(t *testing.T) {
	encoded, err := json.Marshal(PluginManifest{Name: "p"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, exist := decoded["emits_kyou"]; exist {
		t.Errorf("未指定なのに emits_kyou が書き出されている: %s", encoded)
	}
}
