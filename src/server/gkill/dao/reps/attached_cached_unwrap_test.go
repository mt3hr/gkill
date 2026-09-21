package reps

// 付随データ（タグ・テキスト）のキャッシュrepが UnWrapTyped で leaf まで降りることを固定する。
//
// 本番はキャッシュ有効（gkill_options.IsCacheInMemory の既定が true）で、
// TagReps / TextReps は集約をキャッシュrepで包んだ形になる。その UnWrapTyped が
// 1段しか剥がしていなかったため、集約自身が leaf として返り、
// GetRepName() が "TagReps" / "TextReps" という**実在しないリテラル**を返していた。
//
// これは「gkill_add_tag はどこへ書くのか」を書く前に知るための一覧
// (get_rep_infos_mcp の attached_data_reps) に漏れており、
// 実際の書き込み先 "Tag" と食い違う値が案内されていた（実利用レビュー）。
// 同じ注意は GetLatestDataRepositoryAddress のコメント（ADR-0210）にあったが、
// UnWrapTyped 側では守られていなかった。notification は元から再帰していて正しい。

import (
	"context"
	"strings"
	"sync"
	"testing"
)

func TestTagRepositoryCached_UnWrapTypedReachesLeafNames(t *testing.T) {
	ctx := context.Background()
	leaf := newTempTagRepo(t)
	// 本番と同じ配置: 集約をキャッシュrepで包む
	aggregated := TagRepositories{leaf}
	cached, err := NewTagRepositoryCachedSQLite3Impl(ctx, aggregated, openMemoryDB(t), &sync.RWMutex{}, "TAG_CACHE")
	if err != nil {
		t.Fatalf("failed to create cached tag repo: %v", err)
	}
	t.Cleanup(func() { cached.Close(ctx) })

	unwrapped, err := cached.UnWrapTyped()
	if err != nil {
		t.Fatalf("UnWrapTyped: %v", err)
	}
	if len(unwrapped) == 0 {
		t.Fatal("UnWrapTyped が空を返した")
	}
	for _, rep := range unwrapped {
		name, err := rep.GetRepName(ctx)
		if err != nil {
			t.Fatalf("GetRepName: %v", err)
		}
		if name == "TagReps" {
			t.Errorf("集約の名前 %q が leaf として返っている（実在しない rep 名）", name)
		}
		// leaf はファイル名から拡張子を落とした実名になる
		if !strings.EqualFold(name, "tag") {
			t.Errorf("rep 名 = %q, want \"tag\" 相当の実名", name)
		}
	}
}

func TestTextRepositoryCached_UnWrapTypedReachesLeafNames(t *testing.T) {
	ctx := context.Background()
	leaf := newTempTextRepo(t)
	aggregated := TextRepositories{leaf}
	cached, err := NewTextRepositoryCachedSQLite3Impl(ctx, aggregated, openMemoryDB(t), &sync.RWMutex{}, "TEXT_CACHE")
	if err != nil {
		t.Fatalf("failed to create cached text repo: %v", err)
	}
	t.Cleanup(func() { cached.Close(ctx) })

	unwrapped, err := cached.UnWrapTyped()
	if err != nil {
		t.Fatalf("UnWrapTyped: %v", err)
	}
	if len(unwrapped) == 0 {
		t.Fatal("UnWrapTyped が空を返した")
	}
	for _, rep := range unwrapped {
		name, err := rep.GetRepName(ctx)
		if err != nil {
			t.Fatalf("GetRepName: %v", err)
		}
		if name == "TextReps" {
			t.Errorf("集約の名前 %q が leaf として返っている（実在しない rep 名）", name)
		}
		if !strings.EqualFold(name, "text") {
			t.Errorf("rep 名 = %q, want \"text\" 相当の実名", name)
		}
	}
}
