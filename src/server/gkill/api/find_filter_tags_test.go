package api

// タグ絞り込み（filterTagsKyous / filterTagsTimeIs）の回帰テスト。
//
// 修正対象のバグ:
//   - TimeIsタグAND + "no tags" で tagNameMap[NoTags] が nil のまま代入され panic
//   - KyouタグAND + "no tags" で内側mapを毎ループ作り直し、最後の1件しか残らない
//   - AND検索でヒット0件のタグ名が交差から脱落し、ANDが緩む
//   - AND分岐だけタグ名照合が大文字小文字を区別する（OR分岐のSQLは大小無視）
//   - use_tags=true + tags空 の挙動がOR/ANDで割れる（タグ付き全件 vs 0件）
//
// あわせて、もともと正しいOR分岐の挙動（タグ一致∪タグ無し、非表示タグ削除）を固定する。
//
// 同じ filterTagsKyous を別観点で見るテストが filter_tags_kyous_test.go にある。
// あちらはAND交差を二重ループからmap参照へ置き換えた性能改修に対する結果不変の担保で、
// こちらはAND/ORの意味論のエッジケース担当。
// 共用ヘルパ（kyouForTagFilter / tagMapOf / tagFor / keysOfKyouMap）はあちらで定義している。

import (
	"context"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

// TimeIsタグAND + "no tags" で panic せず、タグ無しTimeIsが全件残ること
func TestFilterTagsTimeIs_And_NoTags_KeepsAllUntagged(t *testing.T) {
	ctx := context.Background()

	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{
			TimeIsWords:   []string{},
			TimeIsTagsAnd: true,
			TimeIsTags:    []string{NoTags},
		},
		MatchTimeIssAtFindTimeIs: map[string]reps.TimeIs{
			"timeis-1": {ID: "timeis-1"},
			"timeis-2": {ID: "timeis-2"},
		},
		MatchTimeIssAtFilterTags:         map[string]reps.TimeIs{},
		MatchTimeIsTags:                  map[string]reps.Tag{},
		RelatedTagIDs:                    map[string]struct{}{},
		MatchHideTagsWhenUncheckedTimeIs: map[string]reps.Tag{},
	}

	f := &FindFilter{}
	if _, err := f.filterTagsTimeIs(ctx, findCtx); err != nil {
		t.Fatalf("filterTagsTimeIs failed: %v", err)
	}

	if len(findCtx.MatchTimeIssAtFilterTags) != 2 {
		t.Errorf("タグ無しTimeIsが全件残るはず: got %d件", len(findCtx.MatchTimeIssAtFilterTags))
	}
}

// KyouタグAND + "no tags" でタグ無しKyouが全件残ること
func TestFilterTagsKyous_And_NoTags_KeepsAllUntagged(t *testing.T) {
	ctx := context.Background()

	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{
			TagsAnd: true,
			Tags:    []string{NoTags},
		},
		MatchKyousCurrent: map[string][]reps.Kyou{
			"kyou-1": kyouForTagFilter("kyou-1"),
			"kyou-2": kyouForTagFilter("kyou-2"),
			"kyou-3": kyouForTagFilter("kyou-3"),
		},
		MatchTags:     map[string]reps.Tag{},
		RelatedTagIDs: map[string]struct{}{},
	}

	f := &FindFilter{}
	if _, err := f.filterTagsKyous(ctx, findCtx); err != nil {
		t.Fatalf("filterTagsKyous failed: %v", err)
	}

	if len(findCtx.MatchKyousCurrent) != 3 {
		t.Errorf("タグ無しKyouが全件残るはず: got %v", keysOfKyouMap(findCtx.MatchKyousCurrent))
	}
}

// AND検索でクエリ中のタグ名が1件もヒットしなければ結果は空になること
// （存在しないタグ名が黙って無視されてANDが緩まないこと）
func TestFilterTagsKyous_And_MissingTagNameYieldsEmpty(t *testing.T) {
	ctx := context.Background()

	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{
			TagsAnd: true,
			Tags:    []string{"tagA", "存在しないタグ"},
		},
		MatchKyousCurrent: map[string][]reps.Kyou{
			"kyou-a": kyouForTagFilter("kyou-a"),
		},
		MatchTags: tagMapOf(
			tagFor("kyou-a", "tagA"),
		),
		RelatedTagIDs: map[string]struct{}{
			"kyou-a": {},
		},
	}

	f := &FindFilter{}
	if _, err := f.filterTagsKyous(ctx, findCtx); err != nil {
		t.Fatalf("filterTagsKyous failed: %v", err)
	}

	if len(findCtx.MatchKyousCurrent) != 0 {
		t.Errorf("存在しないタグ名とのANDは空になるはず: got %v", keysOfKyouMap(findCtx.MatchKyousCurrent))
	}
}

// AND分岐のタグ名照合が大文字小文字を無視すること（OR分岐のSQL LOWER()= と同じ意味論）
func TestFilterTagsKyous_And_TagNameCaseInsensitive(t *testing.T) {
	ctx := context.Background()

	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{
			TagsAnd: true,
			Tags:    []string{"work"},
		},
		MatchKyousCurrent: map[string][]reps.Kyou{
			"kyou-a": kyouForTagFilter("kyou-a"),
		},
		MatchTags: tagMapOf(
			tagFor("kyou-a", "Work"),
		),
		RelatedTagIDs: map[string]struct{}{
			"kyou-a": {},
		},
	}

	f := &FindFilter{}
	if _, err := f.filterTagsKyous(ctx, findCtx); err != nil {
		t.Fatalf("filterTagsKyous failed: %v", err)
	}

	if _, exist := findCtx.MatchKyousCurrent["kyou-a"]; !exist {
		t.Errorf("タグ名照合は大文字小文字を無視するはず: got %v", keysOfKyouMap(findCtx.MatchKyousCurrent))
	}
}

// tags=[]（フィルタ有効だがタグのチェックが0個）なら、OR/ANDとも0件になること
func TestFilterTagsKyous_EmptyTags_YieldsEmpty(t *testing.T) {
	for _, tagsAnd := range []bool{false, true} {
		ctx := context.Background()

		findCtx := &FindKyouContext{
			ParsedFindQuery: &find.FindQuery{
				TagsAnd: tagsAnd,
				Tags:    []string{},
			},
			MatchKyousCurrent: map[string][]reps.Kyou{
				"kyou-tagged":   kyouForTagFilter("kyou-tagged"),
				"kyou-untagged": kyouForTagFilter("kyou-untagged"),
			},
			// findTags は tags空だと全タグを返すので、それを模す
			MatchTags: tagMapOf(
				tagFor("kyou-tagged", "tagA"),
			),
			RelatedTagIDs: map[string]struct{}{
				"kyou-tagged": {},
			},
		}

		f := &FindFilter{}
		if _, err := f.filterTagsKyous(ctx, findCtx); err != nil {
			t.Fatalf("filterTagsKyous failed (tagsAnd=%v): %v", tagsAnd, err)
		}

		if len(findCtx.MatchKyousCurrent) != 0 {
			t.Errorf("tags空なら0件のはず (tagsAnd=%v): got %v", tagsAnd, keysOfKyouMap(findCtx.MatchKyousCurrent))
		}
	}
}

// OR分岐の基本挙動: クエリ中のタグにマッチしたKyou ∪ タグ無しKyou（"no tags"指定時）が残り、
// クエリに無いタグしか持たないKyouは落ちること
func TestFilterTagsKyous_Or_TagOrNoTagsUnion(t *testing.T) {
	ctx := context.Background()

	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{
			TagsAnd: false,
			Tags:    []string{"tagA", NoTags},
		},
		MatchKyousCurrent: map[string][]reps.Kyou{
			"kyou-a":        kyouForTagFilter("kyou-a"),
			"kyou-untagged": kyouForTagFilter("kyou-untagged"),
			"kyou-other":    kyouForTagFilter("kyou-other"),
		},
		// findTags はクエリ中のタグ名（tagA）に完全一致したタグだけを返す
		MatchTags: tagMapOf(
			tagFor("kyou-a", "tagA"),
		),
		RelatedTagIDs: map[string]struct{}{
			"kyou-a":     {},
			"kyou-other": {},
		},
	}

	f := &FindFilter{}
	if _, err := f.filterTagsKyous(ctx, findCtx); err != nil {
		t.Fatalf("filterTagsKyous failed: %v", err)
	}

	if len(findCtx.MatchKyousCurrent) != 2 {
		t.Fatalf("tagA付きとタグ無しの2件が残るはず: got %v", keysOfKyouMap(findCtx.MatchKyousCurrent))
	}
	if _, exist := findCtx.MatchKyousCurrent["kyou-a"]; !exist {
		t.Errorf("kyou-a が残っていない")
	}
	if _, exist := findCtx.MatchKyousCurrent["kyou-untagged"]; !exist {
		t.Errorf("kyou-untagged が残っていない")
	}
}

// OR分岐の非表示タグ: 非表示タグの対象Kyouが結果から消えること
func TestFilterTagsKyous_Or_HideTagRemoves(t *testing.T) {
	ctx := context.Background()

	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{
			TagsAnd: false,
			Tags:    []string{"tagA"},
		},
		MatchKyousCurrent: map[string][]reps.Kyou{
			"kyou-a": kyouForTagFilter("kyou-a"),
			"kyou-b": kyouForTagFilter("kyou-b"),
		},
		MatchTags: tagMapOf(
			tagFor("kyou-a", "tagA"),
			tagFor("kyou-b", "tagA"),
		),
		RelatedTagIDs: map[string]struct{}{
			"kyou-a": {},
			"kyou-b": {},
		},
		MatchHideTagsWhenUncheckedKyou: tagMapOf(
			tagFor("kyou-b", "非表示タグ"),
		),
	}

	f := &FindFilter{}
	if _, err := f.filterTagsKyous(ctx, findCtx); err != nil {
		t.Fatalf("filterTagsKyous failed: %v", err)
	}

	if _, exist := findCtx.MatchKyousCurrent["kyou-b"]; exist {
		t.Errorf("非表示タグの対象 kyou-b は消えるはず: got %v", keysOfKyouMap(findCtx.MatchKyousCurrent))
	}
	if _, exist := findCtx.MatchKyousCurrent["kyou-a"]; !exist {
		t.Errorf("kyou-a は残るはず")
	}
}
