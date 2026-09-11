package common

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
	"github.com/mt3hr/gkill/src/server/gkill/api/req_res"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

func TestAddTagCmdNotNil(t *testing.T) {
	if AddTagCmd == nil {
		t.Fatal("AddTagCmd should not be nil")
	}
	if AddTagCmd.Use != "add_tag" {
		t.Errorf("AddTagCmd.Use = %q, want %q", AddTagCmd.Use, "add_tag")
	}
	for _, flagName := range []string{"rule", "rules_file", "dry_run"} {
		if AddTagCmd.Flags().Lookup(flagName) == nil {
			t.Errorf("flag --%s should be defined", flagName)
		}
	}
	// 旧 auto_tag のフラグは残さない(残すと「指定したのに効かない」が静かに起きる)
	for _, flagName := range []string{"tag_by_rep_prefix", "tag_by_rep_name"} {
		if AddTagCmd.Flags().Lookup(flagName) != nil {
			t.Errorf("flag --%s should be removed", flagName)
		}
	}
}

// ── ルール JSON の復号 ──

// parseRuleJSON はテスト用に1件のルール JSON を復号・検証する。
func parseRuleJSON(t *testing.T, raw string) (addTagRule, error) {
	t.Helper()
	rules, err := parseAddTagRulesJSON([]byte(raw), "test")
	if err != nil {
		return addTagRule{}, err
	}
	if len(rules) != 1 {
		t.Fatalf("rules length: got %d, want 1", len(rules))
	}
	return rules[0], nil
}

func TestParseAddTagRulesJSON_MinimalSidebarRule(t *testing.T) {
	// 運用スクリプトが書く最小形。rep_types_in_sidebar だけで通り、Go 側のフィルタは全部未使用のまま
	rule, err := parseRuleJSON(t, `{"tag": "autolog_screenshot", "query": {"rep_types_in_sidebar": ["AutoScreenshot"]}}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if rule.Tag != "autolog_screenshot" {
		t.Errorf("Tag = %q", rule.Tag)
	}
	if !slices.Equal(rule.RepTypesInSidebar, []string{"AutoScreenshot"}) {
		t.Errorf("RepTypesInSidebar = %#v", rule.RepTypesInSidebar)
	}
	if rule.DevicesInSidebar != nil {
		t.Errorf("DevicesInSidebar should stay nil (= all devices): %#v", rule.DevicesInSidebar)
	}
	if !isFilterlessFindQuery(&rule.Query) {
		t.Errorf("Go 側の FindQuery は未使用のままのはず: %s", formatEffectiveQuery(&rule.Query))
	}
	// 1件だけのときは何件目かを添えない
	if rule.Source != "test" {
		t.Errorf("Source = %q", rule.Source)
	}
}

func TestParseAddTagRulesJSON_ArrayAndSingleObject(t *testing.T) {
	array := `[
		{"tag": "a", "query": {"rep_types_in_sidebar": ["A"]}},
		{"tag": "b", "query": {"reps": ["B_dev_20260101"]}}
	]`
	rules, err := parseAddTagRulesJSON([]byte(array), "rules.json")
	if err != nil {
		t.Fatalf("array: %v", err)
	}
	if len(rules) != 2 || rules[0].Tag != "a" || rules[1].Tag != "b" {
		t.Errorf("array rules: %#v", rules)
	}
	if rules[1].Source != "rules.json #2" {
		t.Errorf("Source = %q, want rules.json #2", rules[1].Source)
	}

	single, err := parseAddTagRulesJSON([]byte(`{"tag": "a", "query": {"rep_types_in_sidebar": ["A"]}}`), "--rule #1")
	if err != nil {
		t.Fatalf("single: %v", err)
	}
	if len(single) != 1 || single[0].Source != "--rule #1" {
		t.Errorf("single rules: %#v", single)
	}

	for _, raw := range []string{`[]`, `"tag"`, `123`, ``, `   `} {
		if _, err := parseAddTagRulesJSON([]byte(raw), "x"); err == nil {
			t.Errorf("%q should fail", raw)
		}
	}
}

func TestParseAddTagRulesJSON_RejectsUnknownKey(t *testing.T) {
	// "rep" の綴り誤りを黙って落とすと、reps が未使用になって全件が対象になる。
	// どのキーが悪いかがエラー文で分かること
	_, err := parseRuleJSON(t, `{"tag": "a", "query": {"rep": ["x"]}}`)
	if err == nil {
		t.Fatal("unknown key should fail")
	}
	if !strings.Contains(err.Error(), `"rep"`) {
		t.Errorf("エラー文に未知のキー名が入っていない: %v", err)
	}

	// ルール階層の未知キーも同じ(query の外に tags を書いた等)
	if _, err := parseRuleJSON(t, `{"tag": "a", "query": {"reps": ["x"]}, "tags": ["y"]}`); err == nil {
		t.Error("unknown key at rule level should fail")
	}
}

func TestParseAddTagRulesJSON_RejectsTrailingGarbage(t *testing.T) {
	for _, raw := range []string{
		`{"tag": "a", "query": {"reps": ["x"]}}{"tag": "b", "query": {"reps": ["y"]}}`,
		`[{"tag": "a", "query": {"reps": ["x"]}}] x`,
		`{"tag": "a", "query": {"reps": ["x"]}} ,`,
	} {
		if _, err := parseAddTagRulesJSON([]byte(raw), "x"); err == nil {
			t.Errorf("%q should fail (trailing content)", raw)
		}
	}
}

func TestParseAddTagRulesJSON_StripsBOMAndMigratesLegacyUseFlags(t *testing.T) {
	// BOM 付き(Windows のエディタが付けがち)でも読める
	withBOM := "\xEF\xBB\xBF" + `{"tag": "a", "query": {"reps": ["x"]}}`
	if _, err := parseRuleJSON(t, withBOM); err != nil {
		t.Errorf("BOM 付きでも読めるべき: %v", err)
	}

	// 旧形式 use_* は現行形式へ直してから復号する(use_reps=false → reps は未使用)。
	// 復号後の形は decodeAddTagRulesStrict で確かめる(検証まで通すと reps:[] が0件で弾かれる)
	wires, err := decodeAddTagRulesStrict([]byte(`{"tag": "a", "query": {"use_reps": false, "reps": ["x"], "use_tags": true, "tags": null}}`))
	if err != nil {
		t.Fatalf("legacy: %v", err)
	}
	if wires[0].Query.Reps != nil {
		t.Errorf("use_reps=false なら reps は nil のはず: %#v", wires[0].Query.Reps)
	}
	if wires[0].Query.Tags == nil || len(wires[0].Query.Tags) != 0 {
		t.Errorf("use_tags=true + null なら tags は非nil空のはず: %#v", wires[0].Query.Tags)
	}
}

// ── ルールの検証 ──

func TestBuildAddTagRule_RejectsMissingTagOrQuery(t *testing.T) {
	for _, raw := range []string{
		`{"tag": "", "query": {"reps": ["x"]}}`,
		`{"tag": "   ", "query": {"reps": ["x"]}}`,
		`{"query": {"reps": ["x"]}}`,
		`{"tag": "a"}`,
		`{"tag": "a", "query": null}`,
	} {
		if _, err := parseRuleJSON(t, raw); err == nil {
			t.Errorf("%q should fail", raw)
		}
	}
}

func TestBuildAddTagRule_RejectsFilterlessQuery(t *testing.T) {
	// 絞り込みが無い query は全件が対象になる。修飾子だけでは絞り込みにならない
	for _, query := range []string{
		`{}`,
		`{"hide_tags": []}`,
		`{"tags_and": true, "words_and": true}`,
		`{"mi_board_name": "board"}`,
		`{"mi_check_state": "checked"}`,
		`{"mi_sort_type": "estimate_start_time"}`,
		`{"include_end_timeis": true, "include_create_mi": true}`,
		`{"only_latest_data": true, "include_deleted_data": true}`,
		`{"query_id": "q1", "is_focus_kyou_in_list_view": true}`,
	} {
		_, err := parseRuleJSON(t, `{"tag": "a", "query": `+query+`}`)
		if err == nil {
			t.Errorf("query %s should fail (filterless)", query)
		} else if !strings.Contains(err.Error(), "検索条件が空") {
			t.Errorf("query %s: unexpected error: %v", query, err)
		}
	}
}

func TestBuildAddTagRule_EachFilterMakesQueryNonEmpty(t *testing.T) {
	// 絞り込みになるフィールドは単独で通る(全件拒否に巻き込まれない)
	for _, query := range []string{
		`{"reps": ["x"]}`,
		`{"tags": ["x"]}`,
		`{"rep_types": ["kmemo"]}`,
		`{"ids": ["id1"]}`,
		`{"words": ["foo"]}`,
		`{"not_words": ["foo"]}`,
		`{"words": []}`,
		`{"timeis_words": []}`,
		`{"timeis_words": ["foo"], "timeis_tags": ["t"]}`,
		`{"calendar_start_date": "2026-09-01T00:00:00+09:00"}`,
		`{"calendar_end_date": "2026-09-01T00:00:00+09:00"}`,
		`{"map_latitude": 35.0, "map_longitude": 135.0, "map_radius": 1.5}`,
		`{"playing_time": "2026-09-01T00:00:00+09:00"}`,
		`{"update_time": "2026-09-01T00:00:00+09:00"}`,
		`{"is_image_only": true}`,
		`{"for_mi": true, "mi_board_name": "board", "mi_check_state": "checked"}`,
		`{"period_of_time_start_time_second": 0}`,
		`{"period_of_time_end_time_second": 3600}`,
		`{"period_of_time_week_of_days": [0, 6]}`,
		`{"hide_tags": ["x"]}`,
		`{"rep_types_in_sidebar": ["A"]}`,
		`{"rep_types_in_sidebar": ["A"], "devices_in_sidebar": ["dev"]}`,
		`{"rep_types_in_sidebar": ["A"], "reps": []}`,
	} {
		if _, err := parseRuleJSON(t, `{"tag": "a", "query": `+query+`}`); err != nil {
			t.Errorf("query %s should pass: %v", query, err)
		}
	}
}

func TestBuildAddTagRule_RejectsAlwaysEmptySpecs(t *testing.T) {
	// 非nilの空配列は「0件指定」で、対象が常に0件になる
	for _, query := range []string{
		`{"tags": []}`,
		`{"rep_types": []}`,
		`{"ids": []}`,
		`{"reps": []}`,
		`{"period_of_time_week_of_days": []}`,
		`{"reps": ["x"], "tags": []}`,
	} {
		_, err := parseRuleJSON(t, `{"tag": "a", "query": `+query+`}`)
		if err == nil {
			t.Errorf("query %s should fail (always empty)", query)
		} else if !strings.Contains(err.Error(), "常に0件") {
			t.Errorf("query %s: unexpected error: %v", query, err)
		}
	}
}

func TestBuildAddTagRule_RejectsPartialMapAndOrphanTimeIsTags(t *testing.T) {
	// 揃っていないと HasMapFilter / HasTimeIsFilter が偽になり、書いたのに黙って無視される
	for _, query := range []string{
		`{"reps": ["x"], "map_radius": 1.0}`,
		`{"reps": ["x"], "map_latitude": 35.0, "map_longitude": 135.0}`,
		`{"reps": ["x"], "timeis_tags": ["t"]}`,
	} {
		if _, err := parseRuleJSON(t, `{"tag": "a", "query": `+query+`}`); err == nil {
			t.Errorf("query %s should fail (silently ignored combination)", query)
		}
	}
}

func TestBuildAddTagRule_RejectsKeywords(t *testing.T) {
	// keywords は入力欄の生文字列。CLI では解析しないので、非空なら words を書くよう促す
	for _, query := range []string{
		`{"reps": ["x"], "keywords": "foo"}`,
		`{"reps": ["x"], "keywords": "foo", "words": ["foo"]}`,
		`{"reps": ["x"], "timeis_keywords": "foo", "timeis_words": ["foo"]}`,
	} {
		_, err := parseRuleJSON(t, `{"tag": "a", "query": `+query+`}`)
		if err == nil {
			t.Errorf("query %s should fail (keywords)", query)
		} else if !strings.Contains(err.Error(), "keywords") {
			t.Errorf("query %s: unexpected error: %v", query, err)
		}
	}
	// 空文字列(コンストラクタ既定)は通る
	if _, err := parseRuleJSON(t, `{"tag": "a", "query": {"reps": ["x"], "keywords": "", "timeis_keywords": ""}}`); err != nil {
		t.Errorf("empty keywords should pass: %v", err)
	}
}

func TestBuildAddTagRule_RejectsDevicesWithoutRepTypes(t *testing.T) {
	_, err := parseRuleJSON(t, `{"tag": "a", "query": {"reps": ["x"], "devices_in_sidebar": ["dev"]}}`)
	if err == nil {
		t.Error("devices_in_sidebar without rep_types_in_sidebar should fail")
	}
}

func TestBuildAddTagRule_RejectsUpdateCache(t *testing.T) {
	_, err := parseRuleJSON(t, `{"tag": "a", "query": {"reps": ["x"], "update_cache": true}}`)
	if err == nil {
		t.Error("update_cache: true should fail")
	} else if !strings.Contains(err.Error(), "update_cache") {
		t.Errorf("unexpected error: %v", err)
	}
	if _, err := parseRuleJSON(t, `{"tag": "a", "query": {"reps": ["x"], "update_cache": false}}`); err != nil {
		t.Errorf("update_cache: false should pass: %v", err)
	}
}

// クライアントの FindKyouQuery が持つ全キーを CLI が既知として受けること(ドリフト防止)。
//
// 復号は DisallowUnknownFields なので、クライアント側にキーが1つ増えるだけで
// 画面からコピーした検索条件が「未知のキー」で読めなくなる。TS のソースから
// `this.<key> =` を集めて、それぞれを含む JSON が復号できることを確かめる。
func TestAddTagQueryJSONAcceptsEveryClientFindKyouQueryKey(t *testing.T) {
	// このファイルは src/server/gkill/main/common/ にあるので、4つ上が src/
	tsPath := filepath.Join("..", "..", "..", "..", "client", "classes", "api", "find_query", "find-kyou-query.ts")
	content, err := os.ReadFile(tsPath)
	if err != nil {
		t.Fatalf("クライアントの FindKyouQuery が読めない: %v", err)
	}
	keyRe := regexp.MustCompile(`(?m)^\s*this\.([a-z_]+)\s*=`)
	keys := map[string]struct{}{}
	for _, mt := range keyRe.FindAllStringSubmatch(string(content), -1) {
		keys[mt[1]] = struct{}{}
	}
	if len(keys) < 30 {
		t.Fatalf("キーの抽出が少なすぎる(走査の壊れ): %d", len(keys))
	}

	for key := range keys {
		raw := fmt.Sprintf(`{"tag": "a", "query": {%q: null}}`, key)
		_, err := decodeAddTagRulesStrict([]byte(raw))
		if err != nil {
			t.Errorf("クライアントのキー %q を CLI が受けない(addTagQueryJSON に足すこと): %v", key, err)
		}
	}
}

// ── rep_types_in_sidebar の展開 ──

func TestSplitRepNameLikeClient(t *testing.T) {
	cases := []struct {
		repName string
		repType string
		device  string
	}{
		{"AutoScreenshot_dev1_20260101", "AutoScreenshot", "dev1"},
		{"gkill", "gkill", "なし"},
		{"a_b", "a_b", "なし"},
		{"a_b_c_d", "a_b_c_d", "なし"},
		{"Kmemo_dev1_202601", "Kmemo", "dev1"},
		// JS の split と同じく空要素も数える
		{"a__b", "a", ""},
	}
	for _, c := range cases {
		repType, device := splitRepNameLikeClient(c.repName)
		if repType != c.repType || device != c.device {
			t.Errorf("splitRepNameLikeClient(%q) = (%q, %q), want (%q, %q)", c.repName, repType, device, c.repType, c.device)
		}
	}
}

func TestExpandSidebarReps(t *testing.T) {
	allRepNames := []string{
		"AutoScreenshot_dev2_20260201",
		"AutoScreenshot_dev1_20260101",
		"AutoScreenshot_dev1_20260101", // 重複
		"AutoAudio_dev1_20260101",
		"Kmemo_dev1_202601",
		"gkill",
		"AutoScreenshotX_dev1_20260101", // 前方一致では当たらない
	}

	got := expandSidebarReps(allRepNames, []string{"AutoScreenshot"}, nil)
	want := []string{"AutoScreenshot_dev1_20260101", "AutoScreenshot_dev2_20260201"}
	if !slices.Equal(got, want) {
		t.Errorf("all devices: got %#v, want %#v", got, want)
	}

	got = expandSidebarReps(allRepNames, []string{"AutoScreenshot", "AutoAudio"}, []string{"dev1"})
	want = []string{"AutoAudio_dev1_20260101", "AutoScreenshot_dev1_20260101"}
	if !slices.Equal(got, want) {
		t.Errorf("dev1 only: got %#v, want %#v", got, want)
	}

	// 非dvnf名は種別=名前全体・端末="なし"(クライアントと同じ)
	got = expandSidebarReps(allRepNames, []string{"gkill"}, nil)
	if !slices.Equal(got, []string{"gkill"}) {
		t.Errorf("non-dvnf: got %#v", got)
	}

	// 空指定は0件だが非nil(呼び出し側が「当たらなかった」と判定できる)
	for _, c := range []struct {
		name     string
		repTypes []string
		devices  []string
	}{
		{"devices []", []string{"AutoScreenshot"}, []string{}},
		{"rep types []", []string{}, nil},
		{"no match", []string{"Nothing"}, nil},
	} {
		got := expandSidebarReps(allRepNames, c.repTypes, c.devices)
		if got == nil || len(got) != 0 {
			t.Errorf("%s: got %#v, want empty non-nil", c.name, got)
		}
	}
}

func TestAddTagRuleEffectiveQueryOverridesReps(t *testing.T) {
	allRepNames := []string{"AutoScreenshot_dev1_20260101", "Kmemo_dev1_202601"}
	start := "2026-09-01T00:00:00+09:00"

	rule, err := parseRuleJSON(t, `{"tag": "a", "query": {"rep_types_in_sidebar": ["AutoScreenshot"], "reps": ["手書き"], "calendar_start_date": "`+start+`"}}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	query := rule.effectiveQuery(allRepNames)
	if !slices.Equal(query.Reps, []string{"AutoScreenshot_dev1_20260101"}) {
		t.Errorf("Reps should be replaced by the expansion: %#v", query.Reps)
	}
	if query.CalendarStartDate == nil {
		t.Error("other filters should be kept")
	}
	// 元のルールは書き換えない(ユーザごとに使い回す)
	if !slices.Equal(rule.Query.Reps, []string{"手書き"}) {
		t.Errorf("rule.Query.Reps should stay: %#v", rule.Query.Reps)
	}

	// 展開しないルールは reps をそのまま送る(rep 名一覧は使わないので nil で呼べる)
	plain, err := parseRuleJSON(t, `{"tag": "a", "query": {"reps": ["gkill"]}}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := plain.effectiveQuery(nil).Reps; !slices.Equal(got, []string{"gkill"}) {
		t.Errorf("plain Reps = %#v", got)
	}
}

func TestFormatEffectiveQueryShowsOnlySpecifiedFields(t *testing.T) {
	query := &find.FindQuery{Reps: []string{"a"}, Tags: []string{}, TagsAnd: true}
	got := formatEffectiveQuery(query)
	var fields map[string]any
	if err := json.Unmarshal([]byte(got), &fields); err != nil {
		t.Fatalf("not JSON: %q", got)
	}
	if _, ok := fields["reps"]; !ok {
		t.Errorf("reps should be shown: %s", got)
	}
	if _, ok := fields["tags"]; !ok {
		t.Errorf("empty array is meaningful and should be shown: %s", got)
	}
	if _, ok := fields["tags_and"]; !ok {
		t.Errorf("true should be shown: %s", got)
	}
	for _, key := range []string{"words", "words_and", "calendar_start_date", "mi_board_name", "mi_check_state"} {
		if _, ok := fields[key]; ok {
			t.Errorf("unspecified %s should be hidden: %s", key, got)
		}
	}
}

// ── 付与予定とタグ ID ──

func TestPutAddTagTargetDoesNotDuplicateSameTag(t *testing.T) {
	targets := map[string]*addTagTarget{}
	kyou := reps.Kyou{ID: "kyou1", RepName: "gkill"}

	putAddTagTarget(targets, kyou, "gkill")
	putAddTagTarget(targets, kyou, "gkill")
	putAddTagTarget(targets, kyou, "autolog_screenshot")

	if len(targets) != 1 {
		t.Fatalf("targets length: got %d, want 1", len(targets))
	}
	if got := targets["kyou1"].Tags; len(got) != 2 {
		t.Errorf("tags: got %#v, want 2 entries", got)
	}
}

func TestAddTagIDIsStableForSameTargetAndTag(t *testing.T) {
	// IDが変わると過去に付与したぶんと食い違い、全件が付け直しになる。
	// 冪等性はこのIDとサーバ側のAlreadyExistTagErrorだけで担保している
	first := addTagID("kyou1", "gkill")
	if first != addTagID("kyou1", "gkill") {
		t.Error("tag id should be stable")
	}
	if first == addTagID("kyou1", "gkill_autolog") {
		t.Error("different tag names should get different ids")
	}
	if first == addTagID("kyou2", "gkill") {
		t.Error("different targets should get different ids")
	}
	if _, err := uuid.Parse(first); err != nil {
		t.Errorf("tag id should be a uuid: %v", err)
	}
	// 区切りが無いと "ab"+"c" と "a"+"bc" が同じIDになってしまう
	if addTagID("ab", "c") == addTagID("a", "bc") {
		t.Error("tag id should not collide across the target/tag boundary")
	}
}

func TestAddTagIDMatchesPreviouslyIssuedID(t *testing.T) {
	// auto_tag と呼ばれていた頃(さらに前は独立バイナリ)に付与したタグと同じIDになること。
	// 名前空間の文字列を変えると全件が付け直しになるので、値で固定しておく。
	// サブコマンドを add_tag へ改名しても "gkill_auto_tag" のままであることがこの検査の要
	want := uuid.NewSHA1(
		uuid.NewSHA1(uuid.NameSpaceOID, []byte("github.com/mt3hr/gkill/gkill_auto_tag")),
		[]byte("kyou1\x00gkill"),
	).String()
	if got := addTagID("kyou1", "gkill"); got != want {
		t.Errorf("addTagID = %q, want %q", got, want)
	}
	if addTagAppName != "gkill_auto_tag" {
		t.Errorf("addTagAppName = %q, want gkill_auto_tag (CREATE_APP に刻まれた過去の値と揃える)", addTagAppName)
	}
}

func TestShouldRefreshAddTagSession(t *testing.T) {
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
		if got := shouldRefreshAddTagSession(c.added); got != c.want {
			t.Errorf("shouldRefreshAddTagSession(%d) = %v, want %v", c.added, got, c.want)
		}
	}
}

func TestFindTaggedKyouIDsQueryUsesTagsAnd(t *testing.T) {
	// 単一タグの「付いているものだけ」をANDで表現していることを固定する
	// (現在のfind_filterはOR/ANDとも完全一致照合なので結果は同じだが、意図の直接表現)
	start := queryTime(t, "2026-09-01T00:00:00+09:00")
	base := &find.FindQuery{
		RepTypes:          []string{"git_commit_log"},
		Reps:              []string{"gkill"},
		HideTags:          []string{"hidden"},
		CalendarStartDate: &start,
	}
	client := &addTagAPIClient{}
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
	// タグ以外の条件はそのまま残る。残らないと「条件そのまま」との差分が意味を失う
	if !slices.Equal(query.RepTypes, []string{"git_commit_log"}) || !slices.Equal(query.Reps, []string{"gkill"}) ||
		!slices.Equal(query.HideTags, []string{"hidden"}) || query.CalendarStartDate == nil {
		t.Errorf("base query should be kept: %s", formatEffectiveQuery(query))
	}
	// 呼び出し元のクエリを書き換えてはいけない（同じ条件を使い回すため）
	if base.Tags != nil || base.TagsAnd {
		t.Error("base query should not be modified")
	}
}

// queryTime は RFC3339 の文字列を time.Time にする(テストの検索条件用)。
func queryTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("time: %v", err)
	}
	return parsed
}

// ── HTTP クライアント ──

// newAddTagTestClient は httptest.Server を宛先にした addTagAPIClient を作る。
// ResolveLocalServerEndpoint は設定DB(server_config.db)が要るので、テストでは宛先を直接組み立てる。
func newAddTagTestClient(server *httptest.Server) *addTagAPIClient {
	return &addTagAPIClient{
		Endpoint: &LocalServerEndpoint{
			BaseURL: server.URL,
			Device:  "test_device",
			Client:  server.Client(),
		},
		SessionID: "test_session",
	}
}

// post は応答本文の読み取りに失敗したら、ステータス付きの「読み取り失敗」エラーを返す。
//
// サーバがContent-Lengthぶんの本文を送りきらずに接続を切ると、
// クライアント側の io.ReadAll が途中で unexpected EOF になる(実装が実際に踏む形)。
// デコード失敗とは別のエラー文で、どの段階で壊れたかが分かることを固定する。
func TestAddTagPost_ReadBodyFailureReturnsReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 宣言した長さより短い本文を書いてハンドラを終える。
		// net/httpサーバは書き足りないまま接続を閉じるので、クライアントは読み取り途中で失敗する
		w.Header().Set("Content-Length", "4096")
		_, _ = w.Write([]byte(`{"messages":null,"errors":null`))
	}))
	defer server.Close()

	client := newAddTagTestClient(server)
	response := &req_res.GetAllRepNamesResponse{}
	err := client.post(context.Background(), "/api/get_all_rep_names", &req_res.GetAllRepNamesRequest{SessionID: client.SessionID, LocaleName: addTagLocaleName}, response)
	if err == nil {
		t.Fatal("本文の読み取り失敗はエラーになるべき")
	}
	if !strings.Contains(err.Error(), "error at read response of") {
		t.Errorf("読み取り失敗のエラー文ではない: %v", err)
	}
	if !strings.Contains(err.Error(), "status = 200") {
		t.Errorf("エラー文にステータスが入っていない: %v", err)
	}
}

// 上限(MaxResponseBodyBytes)を超える本文は上限で切り詰められ、デコード失敗として現れる。
//
// io.LimitReader は上限超過をエラーにせず黙って打ち切るので、
// 「本文が大きすぎる」はこの実装では読み取り失敗ではなく、
// 途中で切れたJSONのデコード失敗になる。巨大応答でも読むのは上限まで、
// エラー文へ入る断片は1024バイトまでで、どちらも際限なく膨らまないことを固定する。
// 既定の1GiBをテストで送るわけにいかないので、上限を小さくして確かめる。
func TestAddTagPost_HugeBodyIsCappedAndFailsDecode(t *testing.T) {
	const cap = 64 * 1024
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"messages":null,"errors":null,"rep_names":["`))
		_, _ = w.Write(bytes.Repeat([]byte("a"), cap))
		_, _ = w.Write([]byte(`"]}`))
	}))
	defer server.Close()

	client := newAddTagTestClient(server)
	client.MaxResponseBodyBytes = cap
	response := &req_res.GetAllRepNamesResponse{}
	err := client.post(context.Background(), "/api/get_all_rep_names", &req_res.GetAllRepNamesRequest{SessionID: client.SessionID, LocaleName: addTagLocaleName}, response)
	if err == nil {
		t.Fatal("上限を超えて切り詰められた本文はエラーになるべき")
	}
	if !strings.Contains(err.Error(), "error at decode response of") {
		t.Errorf("デコード失敗のエラー文ではない: %v", err)
	}
	if len(err.Error()) > 4096 {
		t.Errorf("エラー文が長すぎる(断片が1024バイトで切られていない): %d bytes", len(err.Error()))
	}

	// 上限未設定(0)なら既定値が使われ、同じ本文が読める
	client.MaxResponseBodyBytes = 0
	response = &req_res.GetAllRepNamesResponse{}
	if err := client.post(context.Background(), "/api/get_all_rep_names", &req_res.GetAllRepNamesRequest{SessionID: client.SessionID, LocaleName: addTagLocaleName}, response); err != nil {
		t.Errorf("既定の上限では読めるべき: %v", err)
	}
}

// 本文がJSONですらないとき(プロキシのHTMLエラーページ、TLSサーバへ平文で繋いだ等)は、
// ステータスと本文の断片を添えたエラーになる。断片は1024バイトで打ち切る。
func TestAddTagPost_NonJSONBodyReturnsSnippetError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadGateway)
		// 1024バイト目より後ろにマーカーを置き、断片へ入らないことを確かめる
		_, _ = w.Write([]byte("<html><body>Bad Gateway " + strings.Repeat("x", 1024) + "TAIL_MARKER_BEYOND_CAP</body></html>"))
	}))
	defer server.Close()

	client := newAddTagTestClient(server)
	response := &req_res.GetAllRepNamesResponse{}
	err := client.post(context.Background(), "/api/get_all_rep_names", &req_res.GetAllRepNamesRequest{SessionID: client.SessionID, LocaleName: addTagLocaleName}, response)
	if err == nil {
		t.Fatal("非JSON本文はエラーになるべき")
	}
	if !strings.Contains(err.Error(), "error at decode response of") {
		t.Errorf("デコード失敗のエラー文ではない: %v", err)
	}
	if !strings.Contains(err.Error(), "status = 502") {
		t.Errorf("エラー文にステータスが入っていない: %v", err)
	}
	if !strings.Contains(err.Error(), "Bad Gateway") {
		t.Errorf("エラー文に本文の断片が入っていない: %v", err)
	}
	if strings.Contains(err.Error(), "TAIL_MARKER_BEYOND_CAP") {
		t.Errorf("断片が1024バイトで切られていない: %v", err)
	}
}

// HTTP 200でも本文のerrorsに中身があれば失敗として扱う。
//
// gkillは2026-08より前は異常時も常に200を返していたし、今もエラーの中身は
// 本文のerrors配列が正なので、ステータスだけを見て成功と判定してはいけない。
func TestAddTagGetAllRepNames_ErrorsWithHTTP200IsFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"messages":null,"errors":[{"error_code":%q,"error_message":"管理者権限がありません"}],"rep_names":null}`, message.AccountNotHasAdminError)
	}))
	defer server.Close()

	client := newAddTagTestClient(server)
	repNames, err := client.GetAllRepNames(context.Background())
	if err == nil {
		t.Fatalf("200 + errorsあり は失敗になるべき: rep names = %#v", repNames)
	}
	if !strings.Contains(err.Error(), message.AccountNotHasAdminError) {
		t.Errorf("エラー文にerror_codeが入っていない: %v", err)
	}
	if !strings.Contains(err.Error(), "管理者権限がありません") {
		t.Errorf("エラー文にerror_messageが入っていない: %v", err)
	}
}

// 非2xxでも本文のerrorsがデコードできるなら、error_code/error_messageを伝える。
//
// gkillは2026-08から異常時に4xx/5xxを返すが、エラーの中身は今までどおり
// 本文のerrors配列にしか入っていない。ステータスで打ち切ると「HTTP 401」しか
// 分からず、セッション切れなのか権限不足なのか判別できなくなる
// (本文のerrorsを優先する判断はMCPのgkill-client.mjsと同じ)。
func TestAddTagGetAllRepNames_Non2xxCarriesErrorMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"messages":null,"errors":[{"error_code":%q,"error_message":"セッションが見つかりませんでした"}],"rep_names":null}`, message.AccountSessionNotFoundError)
	}))
	defer server.Close()

	client := newAddTagTestClient(server)
	_, err := client.GetAllRepNames(context.Background())
	if err == nil {
		t.Fatal("401 + errorsあり は失敗になるべき")
	}
	if !strings.Contains(err.Error(), message.AccountSessionNotFoundError) {
		t.Errorf("エラー文にerror_codeが入っていない: %v", err)
	}
	if !strings.Contains(err.Error(), "セッションが見つかりませんでした") {
		t.Errorf("エラー文にerror_messageが伝わっていない: %v", err)
	}
}

// 既存IDのタグはHTTP 409 + ERR000056で届き、AddTagは「既に付いている」(スキップ)として飲む。
//
// add_tagの冪等性はこの経路が要で、「付いているか」の判定を取りこぼしても
// サーバが同じIDを弾いて二重登録にならず、手で消したタグも同じIDで弾かれて復活しない。
// 2026-08からERR000056はHTTP 409で届くようになったため、ステータスで
// 打ち切るとこのスキップに到達できず、冪等なはずの再実行が失敗になってしまう。
func TestAddTagAddTag_AlreadyExistOver409IsSkip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		fmt.Fprintf(w, `{"messages":null,"errors":[{"error_code":%q,"error_message":"すでに存在するタグです"}],"added_tag":null}`, message.AlreadyExistTagError)
	}))
	defer server.Close()

	client := newAddTagTestClient(server)
	alreadyExist, err := client.AddTag(context.Background(), reps.Tag{
		ID:       addTagID("kyou1", "gkill"),
		TargetID: "kyou1",
		Tag:      "gkill",
	})
	if err != nil {
		t.Fatalf("409 + ERR000056 はスキップ扱いのはず: %v", err)
	}
	if !alreadyExist {
		t.Error("alreadyExist = false, want true")
	}
}

// 非2xxで本文にエラーの中身が無いなら、ステータスを唯一の手掛かりとしてエラーにする。
//
// ここでnilを返すと「4xxなのに成功・0件」になり、静かに壊れる。
// errors:[null](中身なしの要素だけ)も同じ扱い。
func TestAddTagPost_Non2xxWithoutErrorContentFailsWithStatus(t *testing.T) {
	for _, body := range []string{
		`{"messages":null,"errors":null,"rep_names":null}`,
		`{"messages":null,"errors":[null],"rep_names":null}`,
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(body))
		}))

		client := newAddTagTestClient(server)
		response := &req_res.GetAllRepNamesResponse{}
		err := client.post(context.Background(), "/api/get_all_rep_names", &req_res.GetAllRepNamesRequest{SessionID: client.SessionID, LocaleName: addTagLocaleName}, response)
		server.Close()

		if err == nil {
			t.Errorf("body %q: 500でerrorsに中身が無くてもエラーになるべき", body)
			continue
		}
		if !strings.Contains(err.Error(), "error at post") || !strings.Contains(err.Error(), "status = 500") {
			t.Errorf("body %q: ステータスを手掛かりにしたエラー文ではない: %v", body, err)
		}
	}
}

// 200 + errors:null が今までどおり成功として通ること(退行防止)。
// あわせて、リクエストが指定パスへJSONで届いていることも確かめる。
func TestAddTagGetAllRepNames_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/get_all_rep_names" {
			t.Errorf("path = %q, want /api/get_all_rep_names", r.URL.Path)
		}
		request := &req_res.GetAllRepNamesRequest{}
		if err := json.NewDecoder(r.Body).Decode(request); err != nil {
			t.Errorf("リクエスト本文がJSONとして読めない: %v", err)
		} else if request.SessionID != "test_session" {
			t.Errorf("session id = %q, want test_session", request.SessionID)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"messages":null,"errors":null,"rep_names":["gkill","gkill_autolog"]}`))
	}))
	defer server.Close()

	client := newAddTagTestClient(server)
	repNames, err := client.GetAllRepNames(context.Background())
	if err != nil {
		t.Fatalf("GetAllRepNames: %v", err)
	}
	if len(repNames) != 2 || repNames[0] != "gkill" || repNames[1] != "gkill_autolog" {
		t.Errorf("rep names: got %#v, want [gkill gkill_autolog]", repNames)
	}
}

// GetKyous は送った検索条件をそのまま /api/get_kyous へ届け、応答の messages
// (読み込めなかった rep の警告)を成功扱いのまま標準出力へ出す。
// 黙って捨てると、その rep の記録が対象から消えていることに気付けない。
func TestAddTagGetKyous_PassesQueryAndPrintsWarningMessages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/get_kyous" {
			t.Errorf("path = %q, want /api/get_kyous", r.URL.Path)
		}
		request := &req_res.GetKyousRequest{}
		if err := json.NewDecoder(r.Body).Decode(request); err != nil {
			t.Errorf("リクエスト本文がJSONとして読めない: %v", err)
		} else if request.Query == nil || !slices.Equal(request.Query.Reps, []string{"AutoScreenshot_dev1_20260101"}) {
			t.Errorf("query が届いていない: %#v", request.Query)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"messages":[{"message_code":%q,"message":"検索完了"},{"message_code":%q,"message":"読み込めなかった記録保管場所があります (broken_rep)"}],"errors":null,"kyous":[{"id":"kyou1","rep_name":"AutoScreenshot_dev1_20260101"}]}`,
			message.GetKyousSuccessMessage, message.FindKyousRepLoadWarningMessage)
	}))
	defer server.Close()

	stdout := captureStdout(t, func() {
		client := newAddTagTestClient(server)
		kyous, err := client.GetKyous(context.Background(), &find.FindQuery{Reps: []string{"AutoScreenshot_dev1_20260101"}})
		if err != nil {
			t.Fatalf("GetKyous: %v", err)
		}
		if len(kyous) != 1 || kyous[0].ID != "kyou1" {
			t.Errorf("kyous: %#v", kyous)
		}
	})
	if !strings.Contains(stdout, "broken_rep") {
		t.Errorf("警告メッセージが標準出力に出ていない: %q", stdout)
	}
	// 成功メッセージ(検索完了)は毎回載るので出さない
	if strings.Contains(stdout, "検索完了") {
		t.Errorf("成功メッセージまで印字している: %q", stdout)
	}
}

// captureStdout は fn の間の標準出力を文字列で返す。
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = writer
	done := make(chan string)
	go func() {
		buf := &bytes.Buffer{}
		_, _ = buf.ReadFrom(reader)
		done <- buf.String()
	}()
	func() {
		defer func() {
			os.Stdout = original
			_ = writer.Close()
		}()
		fn()
	}()
	return <-done
}
