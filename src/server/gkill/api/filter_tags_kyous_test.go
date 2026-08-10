package api

// タグAND絞り込みの回帰テスト。
//
// 以前は「前のタグまででマッチしたKyou」と「今回のタグにマッチしたKyou」を
// 二重ループで総当たり比較していた（map参照なら O(1) のところ）。
// タグAND5個 × Kyou 5,000件で約1.25億回の文字列比較になっていた。
//
// map参照へ置き換えても結果は変わらないはずなので、
// 「全部のタグを持つKyouだけが残る」という結果そのものを固定する。
//
// 同じ filterTagsKyous を別観点で見るテストが find_filter_tags_test.go にある。
// あちらはAND/ORの意味論のエッジケース（"no tags"・大小無視・0件タグ名・空タグ指定）担当で、
// こちらはAND交差の性能改修に対する結果不変の担保。ヘルパ（kyouForTagFilter / tagMapOf /
// tagFor / keysOfKyouMap）はこのファイルで定義し、あちらからも使う。

import (
	"context"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
)

func kyouForTagFilter(id string) []reps.Kyou {
	return []reps.Kyou{{
		ID:          id,
		RelatedTime: time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC),
		UpdateTime:  time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC),
	}}
}

func tagMapOf(tags ...reps.Tag) map[string]reps.Tag {
	m := map[string]reps.Tag{}
	for _, tag := range tags {
		m[tag.ID] = tag
	}
	return m
}

func tagFor(targetID string, tagName string) reps.Tag {
	return reps.Tag{
		ID:       targetID + "-" + tagName,
		TargetID: targetID,
		Tag:      tagName,
	}
}

func TestFilterTagsKyous_And_KeepsOnlyKyousHavingAllTags(t *testing.T) {
	ctx := context.Background()

	// kyou-all は両方のタグを持つ / kyou-a と kyou-b は片方だけ / kyou-none はタグ無し
	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{
			TagsAnd: true,
			Tags:    []string{"tagA", "tagB"},
		},
		MatchKyousCurrent: map[string][]reps.Kyou{
			"kyou-all":  kyouForTagFilter("kyou-all"),
			"kyou-a":    kyouForTagFilter("kyou-a"),
			"kyou-b":    kyouForTagFilter("kyou-b"),
			"kyou-none": kyouForTagFilter("kyou-none"),
		},
		MatchTags: tagMapOf(
			tagFor("kyou-all", "tagA"),
			tagFor("kyou-all", "tagB"),
			tagFor("kyou-a", "tagA"),
			tagFor("kyou-b", "tagB"),
		),
		RelatedTagIDs: map[string]struct{}{
			"kyou-all": {},
			"kyou-a":   {},
			"kyou-b":   {},
		},
	}

	f := &FindFilter{}
	if _, err := f.filterTagsKyous(ctx, findCtx); err != nil {
		t.Fatalf("filterTagsKyous failed: %v", err)
	}

	if len(findCtx.MatchKyousCurrent) != 1 {
		t.Fatalf("全タグを持つKyouだけが残るはず: %v", keysOfKyouMap(findCtx.MatchKyousCurrent))
	}
	if _, exist := findCtx.MatchKyousCurrent["kyou-all"]; !exist {
		t.Errorf("kyou-all が残っていない: %v", keysOfKyouMap(findCtx.MatchKyousCurrent))
	}
}

// タグが1つだけのときも、AND経路で正しく絞れること
func TestFilterTagsKyous_And_SingleTag(t *testing.T) {
	ctx := context.Background()

	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{
			TagsAnd: true,
			Tags:    []string{"tagA"},
		},
		MatchKyousCurrent: map[string][]reps.Kyou{
			"kyou-a":     kyouForTagFilter("kyou-a"),
			"kyou-other": kyouForTagFilter("kyou-other"),
		},
		MatchTags: tagMapOf(
			tagFor("kyou-a", "tagA"),
			tagFor("kyou-other", "tagZ"),
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

	if len(findCtx.MatchKyousCurrent) != 1 {
		t.Fatalf("tagAを持つKyouだけが残るはず: %v", keysOfKyouMap(findCtx.MatchKyousCurrent))
	}
	if _, exist := findCtx.MatchKyousCurrent["kyou-a"]; !exist {
		t.Errorf("kyou-a が残っていない: %v", keysOfKyouMap(findCtx.MatchKyousCurrent))
	}
}

// どのKyouも全タグは持たない場合は空になること
func TestFilterTagsKyous_And_NoKyouHasAllTags(t *testing.T) {
	ctx := context.Background()

	findCtx := &FindKyouContext{
		ParsedFindQuery: &find.FindQuery{
			TagsAnd: true,
			Tags:    []string{"tagA", "tagB"},
		},
		MatchKyousCurrent: map[string][]reps.Kyou{
			"kyou-a": kyouForTagFilter("kyou-a"),
			"kyou-b": kyouForTagFilter("kyou-b"),
		},
		MatchTags: tagMapOf(
			tagFor("kyou-a", "tagA"),
			tagFor("kyou-b", "tagB"),
		),
		RelatedTagIDs: map[string]struct{}{
			"kyou-a": {},
			"kyou-b": {},
		},
	}

	f := &FindFilter{}
	if _, err := f.filterTagsKyous(ctx, findCtx); err != nil {
		t.Fatalf("filterTagsKyous failed: %v", err)
	}

	if len(findCtx.MatchKyousCurrent) != 0 {
		t.Errorf("全タグを持つKyouは無いので空になるはず: %v", keysOfKyouMap(findCtx.MatchKyousCurrent))
	}
}

func keysOfKyouMap(m map[string][]reps.Kyou) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
