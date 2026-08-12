package reps

// --cache_reps_local=true（本番の gkill_server はこれで動いている）のときに、
// ローカルキャッシュrepが「毎回コピーし直して毎回“変更あり”を返す」状態へ戻っていないかの回帰テスト。
//
// UpdateCache が
//
//	閉じる → os.Remove → os.Stat（消したので必ず失敗）→ 要コピー判定 → コピー → 開き直す
//
// の順で書かれていると、mtime+サイズの判定が意味を持たなくなり、
// LastUpdateCacheChanged() が常に true を返す。
// すると上位のキャッシュrepのフルリビルド抑止（db_file_change_detector.go）が丸ごと効かなくなり、
// 実データでは update_cache 1回が14秒から2分超へ戻る。

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// useTempLocalRepCacheDir はローカルキャッシュの置き場をテスト用の一時ディレクトリへ向ける。
func useTempLocalRepCacheDir(t *testing.T) {
	t.Helper()
	orig := gkill_options.CacheDir
	gkill_options.CacheDir = t.TempDir()
	t.Cleanup(func() { gkill_options.CacheDir = orig })
}

// newLocalCachedTestDBFile は元DBファイルを1つ作って、そのパスを返す。
func newLocalCachedTestDBFile(t *testing.T, name string, create func(ctx context.Context, filename string) error) string {
	t.Helper()
	filename := filepath.Join(t.TempDir(), name)
	if err := create(context.Background(), filename); err != nil {
		t.Fatalf("failed to create source db %s: %v", name, err)
	}
	return filename
}

func TestLocalCachedReps_SkipRebuildWhenSourceUnchanged(t *testing.T) {
	type trackable interface {
		UpdateCache(ctx context.Context) error
		LastUpdateCacheChanged() bool
	}

	const userID = "localcacheuser"
	ctx := context.Background()

	cases := []struct {
		name string
		repo func(*testing.T) trackable
	}{
		{"kmemo", func(t *testing.T) trackable {
			filename := newLocalCachedTestDBFile(t, "Kmemo.db", func(ctx context.Context, filename string) error {
				rep, err := NewKmemoRepositorySQLite3Impl(ctx, filename, true)
				if err != nil {
					return err
				}
				return rep.Close(ctx)
			})
			rep, err := NewKmemoRepositorySQLite3ImplLocalCached(ctx, userID, filename, false)
			if err != nil {
				t.Fatalf("failed to create local cached kmemo rep: %v", err)
			}
			return rep
		}},
		{"tag", func(t *testing.T) trackable {
			filename := newLocalCachedTestDBFile(t, "Tag.db", func(ctx context.Context, filename string) error {
				rep, err := NewTagRepositorySQLite3Impl(ctx, filename, true)
				if err != nil {
					return err
				}
				return rep.Close(ctx)
			})
			rep, err := NewTagRepositorySQLite3ImplLocalCached(ctx, userID, filename, false)
			if err != nil {
				t.Fatalf("failed to create local cached tag rep: %v", err)
			}
			return rep
		}},
		{"text", func(t *testing.T) trackable {
			filename := newLocalCachedTestDBFile(t, "Text.db", func(ctx context.Context, filename string) error {
				rep, err := NewTextRepositorySQLite3Impl(ctx, filename, true)
				if err != nil {
					return err
				}
				return rep.Close(ctx)
			})
			rep, err := NewTextRepositorySQLite3ImplLocalCached(ctx, userID, filename, false)
			if err != nil {
				t.Fatalf("failed to create local cached text rep: %v", err)
			}
			return rep
		}},
		{"urlog", func(t *testing.T) trackable {
			filename := newLocalCachedTestDBFile(t, "URLog.db", func(ctx context.Context, filename string) error {
				rep, err := NewURLogRepositorySQLite3Impl(ctx, filename, true)
				if err != nil {
					return err
				}
				return rep.Close(ctx)
			})
			rep, err := NewURLogRepositorySQLite3ImplLocalCached(ctx, userID, filename, false)
			if err != nil {
				t.Fatalf("failed to create local cached urlog rep: %v", err)
			}
			return rep
		}},
		{"timeis", func(t *testing.T) trackable {
			filename := newLocalCachedTestDBFile(t, "TimeIs.db", func(ctx context.Context, filename string) error {
				rep, err := NewTimeIsRepositorySQLite3Impl(ctx, filename, true)
				if err != nil {
					return err
				}
				return rep.Close(ctx)
			})
			rep, err := NewTimeIsRepositorySQLite3ImplLocalCached(ctx, userID, filename, false)
			if err != nil {
				t.Fatalf("failed to create local cached timeis rep: %v", err)
			}
			return rep
		}},
		{"notification", func(t *testing.T) trackable {
			filename := newLocalCachedTestDBFile(t, "Notification.db", func(ctx context.Context, filename string) error {
				rep, err := NewNotificationRepositorySQLite3Impl(ctx, filename, true)
				if err != nil {
					return err
				}
				return rep.Close(ctx)
			})
			rep, err := NewNotificationRepositorySQLite3ImplLocalCached(ctx, userID, filename, false)
			if err != nil {
				t.Fatalf("failed to create local cached notification rep: %v", err)
			}
			return rep
		}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			useTempLocalRepCacheDir(t)
			repo := c.repo(t)

			// 起動直後は基準が無いので必ず「変更あり」＝キャッシュを構築させる
			if err := repo.UpdateCache(ctx); err != nil {
				t.Fatalf("UpdateCache failed: %v", err)
			}
			if !repo.LastUpdateCacheChanged() {
				t.Fatal("初回は「変更あり」であるべき（キャッシュが構築されなくなる）")
			}

			// 再構築成功を通知するまでは「変更あり」のまま（失敗した回を取りこぼさない）
			if err := repo.UpdateCache(ctx); err != nil {
				t.Fatalf("UpdateCache failed: %v", err)
			}
			if !repo.LastUpdateCacheChanged() {
				t.Fatal("commit前は「変更あり」のままであるべき（再構築に失敗しても古いキャッシュが残る）")
			}

			committer, ok := repo.(cacheRebuildCommitter)
			if !ok {
				t.Fatalf("%s のローカルキャッシュrepが cacheRebuildCommitter を実装していない（フルリビルドを飛ばせない）", c.name)
			}
			committer.CommitCacheRebuild()

			// 元DBが変わっていなければ「変更なし」＝コピーもフルリビルドも飛ばせる
			if err := repo.UpdateCache(ctx); err != nil {
				t.Fatalf("UpdateCache failed: %v", err)
			}
			if repo.LastUpdateCacheChanged() {
				t.Error("元DBが変わっていないのに「変更あり」（毎回コピー＋全rep分のフルリビルドに戻っている）")
			}
		})
	}
}

// TestLocalCachedRep_DetectsSourceChange は、元DBが変わったときはちゃんと拾い直すことを確認する。
// 「変更なし」を返せるようにした副作用で更新を取りこぼしていないか、が本題。
func TestLocalCachedRep_DetectsSourceChange(t *testing.T) {
	ctx := context.Background()
	useTempLocalRepCacheDir(t)

	filename := newLocalCachedTestDBFile(t, "Kmemo.db", func(ctx context.Context, filename string) error {
		rep, err := NewKmemoRepositorySQLite3Impl(ctx, filename, true)
		if err != nil {
			return err
		}
		return rep.Close(ctx)
	})

	repo, err := NewKmemoRepositorySQLite3ImplLocalCached(ctx, "localcacheuser", filename, false)
	if err != nil {
		t.Fatalf("failed to create local cached kmemo rep: %v", err)
	}

	if err := repo.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache failed: %v", err)
	}
	committer, ok := repo.(cacheRebuildCommitter)
	if !ok {
		t.Fatal("kmemo のローカルキャッシュrepが cacheRebuildCommitter を実装していない")
	}
	committer.CommitCacheRebuild()
	if err := repo.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache failed: %v", err)
	}
	if repo.LastUpdateCacheChanged() {
		t.Fatal("元DBが変わっていないのに「変更あり」")
	}

	// 元DBへ直接書き込む。
	// mtimeの分解能に依存しないよう、サイズが必ず変わるよう別接続から追記する。
	sourceRep, err := NewKmemoRepositorySQLite3Impl(ctx, filename, true)
	if err != nil {
		t.Fatalf("failed to reopen source kmemo rep: %v", err)
	}
	if err := sourceRep.AddKmemoInfo(ctx, makeKmemo("local-cached-change-001", "hello")); err != nil {
		t.Fatalf("AddKmemoInfo failed: %v", err)
	}
	if err := sourceRep.Close(ctx); err != nil {
		t.Fatalf("failed to close source kmemo rep: %v", err)
	}

	stat, err := os.Stat(filename)
	if err != nil {
		t.Fatalf("failed to stat source db: %v", err)
	}
	if stat.Size() == 0 {
		t.Fatal("元DBが空のまま（テストの前提が壊れている）")
	}

	if err := repo.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache failed: %v", err)
	}
	if !repo.LastUpdateCacheChanged() {
		t.Error("元DBが変わったのに「変更なし」（追記した記録が検索に出てこなくなる）")
	}

	kmemos, err := repo.FindKmemo(ctx, &find.FindQuery{})
	if err != nil {
		t.Fatalf("FindKmemo failed: %v", err)
	}
	found := false
	for _, kmemo := range kmemos {
		if kmemo.ID == "local-cached-change-001" {
			found = true
			break
		}
	}
	if !found {
		t.Error("元DBへ追記した記録がローカルキャッシュ経由で見えない（コピーが飛ばされている）")
	}
}

// TestReKyouLocalCachedReps_AlwaysReportChanged は、ReKyou/MiReKyou のローカルキャッシュrepが
// 変更検知を持ち込んでいないことを確認する。
//
// この2つのキャッシュ内容は自分のDBファイルだけでなく他repのターゲット解決結果にも依存し、
// GkillRepositories.UpdateCache がアドレス確定後にもう一度更新する。
// mtimeで判定するとその2回目が飛ばされ、ターゲット未解決の中身が残る。
func TestReKyouLocalCachedReps_AlwaysReportChanged(t *testing.T) {
	ctx := context.Background()
	useTempLocalRepCacheDir(t)

	repositories, err := NewGkillRepositories(sanitizeTestUserID(t.Name()))
	if err != nil {
		t.Fatalf("failed to create repositories: %v", err)
	}
	t.Cleanup(func() { _ = repositories.Close(context.Background()) })
	repositories.ReKyouReps.GkillRepositories = repositories

	reKyouFile := newLocalCachedTestDBFile(t, "ReKyou.db", func(ctx context.Context, filename string) error {
		rep, err := NewReKyouRepositorySQLite3Impl(ctx, filename, true, repositories)
		if err != nil {
			return err
		}
		return rep.Close(ctx)
	})
	miReKyouFile := newLocalCachedTestDBFile(t, "MiReKyou.db", func(ctx context.Context, filename string) error {
		rep, err := NewMiReKyouRepositorySQLite3Impl(ctx, filename, true, repositories)
		if err != nil {
			return err
		}
		return rep.Close(ctx)
	})

	reKyouRep, err := NewReKyouRepositorySQLite3ImplLocalCached(ctx, "localcacheuser", reKyouFile, false, repositories)
	if err != nil {
		t.Fatalf("failed to create local cached rekyou rep: %v", err)
	}
	miReKyouRep, err := NewMiReKyouRepositorySQLite3ImplLocalCached(ctx, "localcacheuser", miReKyouFile, false, repositories)
	if err != nil {
		t.Fatalf("failed to create local cached mirekyou rep: %v", err)
	}

	for _, c := range []struct {
		name string
		repo interface {
			UpdateCache(ctx context.Context) error
			LastUpdateCacheChanged() bool
		}
	}{
		{"rekyou", reKyouRep},
		{"mirekyou", miReKyouRep},
	} {
		t.Run(c.name, func(t *testing.T) {
			for i := range 3 {
				if err := c.repo.UpdateCache(ctx); err != nil {
					t.Fatalf("UpdateCache #%d failed: %v", i+1, err)
				}
				if !c.repo.LastUpdateCacheChanged() {
					t.Fatalf("%s は常に「変更あり」であるべき（アドレス確定後の2回目の再構築が飛び、ターゲット未解決のまま残る）", c.name)
				}
			}
			if _, ok := c.repo.(cacheRebuildCommitter); ok {
				t.Errorf("%s は CommitCacheRebuild を実装してはいけない（変更検知に載せてはいけないrep）", c.name)
			}
		})
	}
}
