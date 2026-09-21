package main

import (
	"reflect"
	"testing"

	sdk "github.com/mt3hr/gkill/src/server/gkill/plugin/sdk"
)

// configOf の secondary_data_sources は「キーが無い＝既定」「空配列＝全部合算」を区別する。
// nil と [] を同じに扱うと、古い config.json（キー無し）が全部合算へ戻って歩数が2倍になる。
func TestConfigOfSecondaryDataSourcesDistinguishesMissingFromEmpty(t *testing.T) {
	pluginDir := t.TempDir() // config.json が無いので cfg がそのまま使われる

	missing := configOf(pluginDir, sdk.Config{})
	if !reflect.DeepEqual(missing.SecondaryDataSources, defaultSecondaryDataSources()) {
		t.Errorf("キー無し = %v, want 既定 %v", missing.SecondaryDataSources, defaultSecondaryDataSources())
	}

	empty := configOf(pluginDir, sdk.Config{configKeySecondaryDataSources: []any{}})
	if empty.SecondaryDataSources == nil || len(empty.SecondaryDataSources) != 0 {
		t.Errorf("空配列 = %#v, want 空（全部合算）", empty.SecondaryDataSources)
	}

	listed := configOf(pluginDir, sdk.Config{configKeySecondaryDataSources: []any{" Phone Health Connect ", "", "Google Health App"}})
	if !reflect.DeepEqual(listed.SecondaryDataSources, []string{"Phone Health Connect", "Google Health App"}) {
		t.Errorf("配列 = %v, want 空行を落として前後の空白を切った2件", listed.SecondaryDataSources)
	}

	newlines := configOf(pluginDir, sdk.Config{configKeySecondaryDataSources: "A\r\nB\n"})
	if !reflect.DeepEqual(newlines.SecondaryDataSources, []string{"A", "B"}) {
		t.Errorf("改行区切り = %v, want [A B]", newlines.SecondaryDataSources)
	}
}

// foldRule は畳み直しの規則を1つの文字列にする。大小と前後の空白の違いだけなら同じ規則
// （設定画面を開いて保存し直すたびに全日を畳み直さない）。並びが違えば別の規則（優先順が変わる）。
func TestFoldRuleNormalizesCaseAndWhitespace(t *testing.T) {
	base := pluginConfig{SecondaryDataSources: []string{"Phone Health Connect", "Google Health App"}}
	sameRule := pluginConfig{SecondaryDataSources: []string{"  phone health connect", "GOOGLE HEALTH APP  "}}
	if base.foldRule() != sameRule.foldRule() {
		t.Errorf("大小・空白違いで規則が変わる: %q vs %q", base.foldRule(), sameRule.foldRule())
	}
	reordered := pluginConfig{SecondaryDataSources: []string{"Google Health App", "Phone Health Connect"}}
	if base.foldRule() == reordered.foldRule() {
		t.Errorf("優先順が違うのに同じ規則: %q", base.foldRule())
	}
	sumAll := pluginConfig{SecondaryDataSources: []string{}}
	if sumAll.foldRule() == base.foldRule() || sumAll.foldRule() != "secondary=" {
		t.Errorf("全部合算の規則 = %q", sumAll.foldRule())
	}
}
