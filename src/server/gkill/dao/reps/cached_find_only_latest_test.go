package reps

// cached実装の FindNlog / FindKC / FindTimeIs が OnlyLatestData を正しく見ていなかった回帰テスト。
//
//   - nlog:   `var onlyLatestData bool` が宣言のみで未代入（常にfalse）だった
//   - kc:     `onlyLatestData := false` がハードコードされていた
//   - timeis: end分岐だけ `onlyLatestData = true` 固定で、start分岐と非対称だった
//
// いずれも非cached実装（query.OnlyLatestData 依存）と食い違い、
// nlog/kc は一覧に編集履歴の全版が重複して出て、
// timeis は履歴表示で終了時刻の行だけ旧版が消えていた。

import (
	"context"
	"testing"
	"time"
)

func TestCachedNlog_FindNlogOnlyLatestVersion(t *testing.T) {
	repo := newCachedNlogRepo(t)
	ctx := context.Background()

	v1 := makeNlog("nlog-latest-001", "旧タイトル", "店", 100)
	if err := repo.AddNlogInfo(ctx, v1); err != nil {
		t.Fatalf("AddNlogInfo(v1) failed: %v", err)
	}

	v2 := makeNlog("nlog-latest-001", "新タイトル", "店", 100)
	v2.UpdateTime = v1.UpdateTime.Add(time.Hour)
	if err := repo.AddNlogInfo(ctx, v2); err != nil {
		t.Fatalf("AddNlogInfo(v2) failed: %v", err)
	}

	results, err := repo.FindNlog(ctx, makeDefaultFindQuery())
	if err != nil {
		t.Fatalf("FindNlog failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("OnlyLatestData=true では最新版1件だけが返るはず: got %d件", len(results))
	}
	if results[0].Title != "新タイトル" {
		t.Errorf("最新版のタイトルが返るはず: got %q", results[0].Title)
	}
}

func TestCachedKC_FindKCOnlyLatestVersion(t *testing.T) {
	repo := newCachedKCRepo(t)
	ctx := context.Background()

	v1 := makeKC("kc-latest-001", "旧タイトル", 42.5)
	if err := repo.AddKCInfo(ctx, v1); err != nil {
		t.Fatalf("AddKCInfo(v1) failed: %v", err)
	}

	v2 := makeKC("kc-latest-001", "新タイトル", 43.5)
	v2.UpdateTime = v1.UpdateTime.Add(time.Hour)
	if err := repo.AddKCInfo(ctx, v2); err != nil {
		t.Fatalf("AddKCInfo(v2) failed: %v", err)
	}

	results, err := repo.FindKC(ctx, makeDefaultFindQuery())
	if err != nil {
		t.Fatalf("FindKC failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("OnlyLatestData=true では最新版1件だけが返るはず: got %d件", len(results))
	}
	if results[0].Title != "新タイトル" {
		t.Errorf("最新版のタイトルが返るはず: got %q", results[0].Title)
	}
}

// cached実装のTimeIsでも、end分岐が query.OnlyLatestData に従うこと。
// 非cachedと同じ検査を同じヘルパで通す（片方だけ直すとrepの構成次第で挙動が変わる）。
func TestCachedTimeIs_FindEndBranchHonorsOnlyLatestData(t *testing.T) {
	assertTimeIsEndBranchHonorsOnlyLatestData(t, newCachedTimeIsRepo(t), "timeis-cached-endlatest-001")
}
