package common

import (
	"testing"

	"github.com/google/uuid"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

func TestAutoTagCmdNotNil(t *testing.T) {
	if AutoTagCmd == nil {
		t.Fatal("AutoTagCmd should not be nil")
	}
	if AutoTagCmd.Use != "auto_tag" {
		t.Errorf("AutoTagCmd.Use = %q, want %q", AutoTagCmd.Use, "auto_tag")
	}
	for _, flagName := range []string{"tag_by_rep_prefix", "tag_by_rep_name", "dry_run"} {
		if AutoTagCmd.Flags().Lookup(flagName) == nil {
			t.Errorf("AutoTagCmd should have --%s", flagName)
		}
	}
}

func TestParseAutoTagRules(t *testing.T) {
	prefixRules, repTypeRules, err := parseAutoTagRules(
		[]string{"AutoScreenshot_=autolog_screenshot", "Box_=box"},
		[]string{"git_commit_log"},
	)
	if err != nil {
		t.Fatalf("parseAutoTagRules: %v", err)
	}
	if len(prefixRules) != 2 {
		t.Fatalf("prefixRules length: got %d, want 2", len(prefixRules))
	}
	if prefixRules[0].Prefix != "AutoScreenshot_" || prefixRules[0].Tag != "autolog_screenshot" {
		t.Errorf("prefixRules[0]: got %#v", prefixRules[0])
	}
	if len(repTypeRules) != 1 || repTypeRules[0] != "git_commit_log" {
		t.Errorf("repTypeRules: got %#v", repTypeRules)
	}
}

func TestParseAutoTagRulesRejectsBrokenPrefixRule(t *testing.T) {
	for _, arg := range []string{"AutoScreenshot_", "=autolog_screenshot", "AutoScreenshot_="} {
		if _, _, err := parseAutoTagRules([]string{arg}, nil); err == nil {
			t.Errorf("parseAutoTagRules(%q) should fail", arg)
		}
	}
	if _, _, err := parseAutoTagRules(nil, []string{""}); err == nil {
		t.Error("empty rep type should fail")
	}
}

func TestAddAutoTagTargetDoesNotDuplicateSameTag(t *testing.T) {
	targets := map[string]*autoTagTarget{}
	kyou := reps.Kyou{ID: "kyou1", RepName: "gkill"}

	addAutoTagTarget(targets, kyou, "gkill")
	addAutoTagTarget(targets, kyou, "gkill")
	addAutoTagTarget(targets, kyou, "autolog_screenshot")

	if len(targets) != 1 {
		t.Fatalf("targets length: got %d, want 1", len(targets))
	}
	if got := targets["kyou1"].Tags; len(got) != 2 {
		t.Errorf("tags: got %#v, want 2 entries", got)
	}
}

func TestAutoTagIDIsStableForSameTargetAndTag(t *testing.T) {
	// IDが変わると過去に付与したぶんと食い違い、全件が付け直しになる。
	// 冪等性はこのIDとサーバ側のAlreadyExistTagErrorだけで担保している
	first := autoTagID("kyou1", "gkill")
	if first != autoTagID("kyou1", "gkill") {
		t.Error("tag id should be stable")
	}
	if first == autoTagID("kyou1", "gkill_autolog") {
		t.Error("different tag names should get different ids")
	}
	if first == autoTagID("kyou2", "gkill") {
		t.Error("different targets should get different ids")
	}
	if _, err := uuid.Parse(first); err != nil {
		t.Errorf("tag id should be a uuid: %v", err)
	}
	// 区切りが無いと "ab"+"c" と "a"+"bc" が同じIDになってしまう
	if autoTagID("ab", "c") == autoTagID("a", "bc") {
		t.Error("tag id should not collide across the target/tag boundary")
	}
}

func TestAutoTagIDMatchesPreviouslyIssuedID(t *testing.T) {
	// 独立バイナリだった頃に付与したタグと同じIDになること。
	// 名前空間の文字列を変えると全件が付け直しになるので、値で固定しておく
	want := uuid.NewSHA1(
		uuid.NewSHA1(uuid.NameSpaceOID, []byte("github.com/mt3hr/gkill/gkill_auto_tag")),
		[]byte("kyou1\x00gkill"),
	).String()
	if got := autoTagID("kyou1", "gkill"); got != want {
		t.Errorf("autoTagID = %q, want %q", got, want)
	}
}

func TestShouldRefreshAutoTagSession(t *testing.T) {
	// 500件ごと(進捗印字と同じ区切り)でだけ true。0件では延長しない。
	cases := []struct {
		added int
		want  bool
	}{
		{0, false},
		{1, false},
		{499, false},
		{500, true},
		{501, false},
		{999, false},
		{1000, true},
		{1500, true},
	}
	for _, c := range cases {
		if got := shouldRefreshAutoTagSession(c.added); got != c.want {
			t.Errorf("shouldRefreshAutoTagSession(%d) = %v, want %v", c.added, got, c.want)
		}
	}
}

func TestFindTaggedKyouIDsQueryUsesTagsAnd(t *testing.T) {
	// 単一タグの「付いているものだけ」をANDで表現していることを固定する
	// (現在のfind_filterはOR/ANDとも完全一致照合なので結果は同じだが、意図の直接表現)
	base := &find.FindQuery{RepTypes: []string{"git_commit_log"}}
	client := &autoTagAPIClient{}
	query := client.buildTaggedQuery(base, "gkill")

	if query.Tags == nil {
		t.Error("Tags should be non-nil (tag filter enabled)")
	}
	if !query.TagsAnd {
		t.Error("TagsAnd should be true")
	}
	if len(query.Tags) != 1 || query.Tags[0] != "gkill" {
		t.Errorf("Tags: got %#v, want [gkill]", query.Tags)
	}
	if query.RepTypes == nil || len(query.RepTypes) != 1 || query.RepTypes[0] != "git_commit_log" {
		t.Errorf("base query should be kept: got %#v", query)
	}
	// 呼び出し元のクエリを書き換えてはいけない（rep名ごとに使い回すため）
	if base.Tags != nil {
		t.Error("base query should not be modified")
	}
}
