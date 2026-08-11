package main

import (
	"encoding/json"
	"testing"
)

// TestManifest_DoesNotEmitKyou は、このプラグインが「記録保管場所」の一覧に
// 出ないことを確認する。
//
// このプラグインは位置情報だけを提供し、Kyou を1件も返さない。
// emits_kyou を落とすと rykv の絞り込みツリーに GoogleLocation が並び、
// 選んでも0件しか出ない項目になる。
//
// GPSログの受け渡しは rep の選択状態と無関係に常に効く
// （gkill側の消費者は GPSLogReps を素通しで舐めており、FindQuery.Reps を見ない）。
func TestManifest_DoesNotEmitKyou(t *testing.T) {
	var manifest struct {
		RepName   string   `json:"rep_name"`
		Provides  []string `json:"provides"`
		EmitsKyou *bool    `json:"emits_kyou"`
	}
	if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
		t.Fatalf("manifest.json を読めない: %v", err)
	}
	if manifest.EmitsKyou == nil {
		t.Fatal("emits_kyou が書かれていない（省略すると true 扱いになり Rep 一覧に出る）")
	}
	if *manifest.EmitsKyou {
		t.Error("emits_kyou が true になっている")
	}
	if manifest.RepName != repName {
		t.Errorf("rep_name = %q, want %q", manifest.RepName, repName)
	}
	if len(manifest.Provides) != 1 || manifest.Provides[0] != "gpslog" {
		t.Errorf("provides = %v, want [gpslog]", manifest.Provides)
	}
}
