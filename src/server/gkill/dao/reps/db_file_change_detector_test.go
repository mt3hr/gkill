package reps

// キャッシュrepのフルリビルドを「実DBファイルが変わったときだけ」に絞れているかの回帰テスト。
//
// ここが常に true を返していたせいで、メモを1つ保存するだけで13種すべての
// キャッシュが全消し＋全行再INSERTされ、その間ずっと共有の書き込みロックを握るため
// 全種類の検索が止まっていた。

import (
	"context"
	"testing"
)

// 変更検知に対応させた全repが、同じ約束を守っていることを確認する。
//   - 初回は「変更あり」（起動直後にキャッシュが構築される）
//   - CommitCacheRebuild を呼ぶまでは「変更あり」のまま（再構築失敗時に取りこぼさない）
//   - Commit後、ファイルが変わっていなければ「変更なし」（フルリビルドを飛ばせる）
func TestAllCachedReps_LastUpdateCacheChangedIsNotHardcoded(t *testing.T) {
	type trackable interface {
		UpdateCache(ctx context.Context) error
		LastUpdateCacheChanged() bool
	}

	cases := []struct {
		name string
		repo func(*testing.T) trackable
	}{
		{"kmemo", func(t *testing.T) trackable { return newTempKmemoRepo(t) }},
		{"kc", func(t *testing.T) trackable { return newTempKCRepo(t) }},
		{"urlog", func(t *testing.T) trackable { return newTempURLogRepo(t) }},
		{"nlog", func(t *testing.T) trackable { return newTempNlogRepo(t) }},
		{"lantana", func(t *testing.T) trackable { return newTempLantanaRepo(t) }},
		{"mi", func(t *testing.T) trackable { return newTempMiRepo(t) }},
		{"timeis", func(t *testing.T) trackable { return newTempTimeIsRepo(t) }},
		{"notification", func(t *testing.T) trackable { return newTempNotificationRepo(t) }},
		{"tag", func(t *testing.T) trackable { return newTempTagRepo(t) }},
		{"text", func(t *testing.T) trackable { return newTempTextRepo(t) }},
	}

	ctx := context.Background()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo := c.repo(t)

			if err := repo.UpdateCache(ctx); err != nil {
				t.Fatalf("UpdateCache failed: %v", err)
			}
			if !repo.LastUpdateCacheChanged() {
				t.Fatal("初回は「変更あり」であるべき")
			}

			committer, ok := repo.(cacheRebuildCommitter)
			if !ok {
				t.Fatalf("%s rep が cacheRebuildCommitter を実装していない（フルリビルドを飛ばせない）", c.name)
			}
			committer.CommitCacheRebuild()

			if err := repo.UpdateCache(ctx); err != nil {
				t.Fatalf("UpdateCache failed: %v", err)
			}
			if repo.LastUpdateCacheChanged() {
				t.Error("ファイルが変わっていないのに「変更あり」（毎回フルリビルドされ、その間ずっと全検索が止まる）")
			}
		})
	}
}

func TestKmemoRepository_LastUpdateCacheChanged_TracksFileChanges(t *testing.T) {
	ctx := context.Background()
	repo := newTempKmemoRepo(t)

	// 起動直後は基準が無いので必ず「変更あり」＝キャッシュを構築させる
	if err := repo.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache failed: %v", err)
	}
	if !repo.LastUpdateCacheChanged() {
		t.Fatal("初回は「変更あり」であるべき（キャッシュが構築されなくなる）")
	}

	// 再構築が成功していない間は、何度聞いても「変更あり」のまま
	if err := repo.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache failed: %v", err)
	}
	if !repo.LastUpdateCacheChanged() {
		t.Fatal("commit前は「変更あり」のままであるべき（再構築失敗時に古いキャッシュが残る）")
	}

	// 再構築成功を通知
	committer, ok := repo.(cacheRebuildCommitter)
	if !ok {
		t.Fatal("kmemo rep が cacheRebuildCommitter を実装していない")
	}
	committer.CommitCacheRebuild()

	// 以降、ファイルが変わっていなければ「変更なし」＝フルリビルドを飛ばせる
	if err := repo.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache failed: %v", err)
	}
	if repo.LastUpdateCacheChanged() {
		t.Error("ファイルが変わっていないのに「変更あり」になっている（毎回フルリビルドされる）")
	}

	// 書き込むと「変更あり」に戻る
	if err := repo.AddKmemoInfo(ctx, makeKmemo("kmemo-change-001", "hello")); err != nil {
		t.Fatalf("AddKmemoInfo failed: %v", err)
	}
	if err := repo.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache failed: %v", err)
	}
	if !repo.LastUpdateCacheChanged() {
		t.Error("書き込み後は「変更あり」であるべき（キャッシュが古いまま残る）")
	}
}
