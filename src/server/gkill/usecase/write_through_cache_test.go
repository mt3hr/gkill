package usecase

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/dao/reps"
	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
	_ "modernc.org/sqlite"
)

// 書き込み後のキャッシュ反映が「XxxRepsの長さが1か」で判定されていたころ、
// providesを持つプラグイン(fitbitのprovides:["kc","tag"]等)を1つ入れるだけで
// XxxRepsが2個になり、判定が常にfalseになって反映が丸ごと飛んでいた。
// 読み取りはキャッシュrepしか見ない(下層repへフォールバックしない)ので、
// 追加したタグが次のUpdateCacheまで最大1分間見えなくなる。
// さらにその間にPWAが古い応答をキャッシュし直すと、
// 更新IDの再通知が来ないぶん恒久的に古いまま焼き付く。

func newWriteThroughTagRep(t *testing.T, name string) reps.TagRepository {
	t.Helper()
	rep, err := reps.NewTagRepositorySQLite3Impl(context.Background(), filepath.Join(t.TempDir(), name+".db"), true)
	if err != nil {
		t.Fatalf("failed to create tag repo %s: %v", name, err)
	}
	t.Cleanup(func() { _ = rep.Close(context.Background()) })
	return rep
}

// newWriteThroughTagRepositories はプラグインの型別アダプタが登録済みの状態、
// つまりTagRepsが「キャッシュrep + アダプタ」の2個になった状態を再現する。
func newWriteThroughTagRepositories(t *testing.T, withCache bool) (*reps.GkillRepositories, reps.TagRepository) {
	t.Helper()

	addrDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	t.Cleanup(func() { _ = addrDB.Close() })
	addrDAO, err := gkill_cache.NewLatestDataRepositoryAddressSQLite3Impl("testuser", addrDB, &sync.RWMutex{})
	if err != nil {
		t.Fatalf("NewLatestDataRepositoryAddressSQLite3Impl: %v", err)
	}

	writeRep := newWriteThroughTagRep(t, "write_tag")
	cacheRep := newWriteThroughTagRep(t, "cache_tag")
	pluginRep := newWriteThroughTagRep(t, "plugin_tag") // プラグインの型別アダプタ相当

	repositories := &reps.GkillRepositories{
		WriteTagRep:                    writeRep,
		LatestDataRepositoryAddressDAO: addrDAO,
	}
	if withCache {
		repositories.TagReps = reps.TagRepositories{cacheRep, pluginRep}
		repositories.CachedReps.Tag = cacheRep
		return repositories, cacheRep
	}
	// キャッシュ無効時はXxxRepsが実体repのまま。CachedRepsはnil
	repositories.TagReps = reps.TagRepositories{writeRep, pluginRep}
	return repositories, writeRep
}

func newWriteThroughTag(id, targetID, tag string) reps.Tag {
	now := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	return reps.Tag{
		IsDeleted:    false,
		ID:           id,
		TargetID:     targetID,
		Tag:          tag,
		RelatedTime:  now,
		CreateTime:   now,
		CreateApp:    "test_app",
		CreateDevice: "test_device",
		CreateUser:   "testuser",
		UpdateTime:   now,
		UpdateApp:    "test_app",
		UpdateDevice: "test_device",
		UpdateUser:   "testuser",
	}
}

// TestAddTagWritesThroughToCacheWithPluginTypedAdapter は
// 型別アダプタでTagRepsが2個になっていても、追加したタグが
// UpdateCacheを待たずに読めることを確認する。
func TestAddTagWritesThroughToCacheWithPluginTypedAdapter(t *testing.T) {
	ctx := context.Background()
	uc := &UsecaseContext{}
	repositories, cacheRep := newWriteThroughTagRepositories(t, true)

	if len(repositories.TagReps) == 1 {
		t.Fatal("前提が崩れている: TagRepsはアダプタ込みで2個であること")
	}

	tag := newWriteThroughTag("tag-001", "target-001", "日記")
	addedTag, gkillErrors, err := uc.AddTag(ctx, repositories, "testuser", "test_device", "ja", tag, nil)
	if err != nil {
		t.Fatalf("AddTag failed: %v", err)
	}
	if len(gkillErrors) != 0 {
		t.Fatalf("AddTag returned errors: %v", gkillErrors)
	}
	if addedTag == nil {
		t.Fatal("AddTagが追加したタグを返していない。キャッシュrepに反映されていないと応答のadded_tagがnullになる")
	}

	cached, err := cacheRep.GetTagsByTargetID(ctx, "target-001")
	if err != nil {
		t.Fatalf("GetTagsByTargetID(cache rep) failed: %v", err)
	}
	if len(cached) != 1 {
		t.Fatalf("キャッシュrepに反映されていない。len = %d, want 1", len(cached))
	}

	// 実際の読み取り経路(GkillRepositories)からも即座に見えること
	got, err := repositories.GetTagsByTargetID(ctx, "target-001")
	if err != nil {
		t.Fatalf("GetTagsByTargetID(repositories) failed: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("読み取り経路から見えていない。len = %d, want 1", len(got))
	}
	if got[0].Tag != "日記" {
		t.Errorf("Tag = %q, want %q", got[0].Tag, "日記")
	}
}

// TestUpdateTagWritesThroughToCacheWithPluginTypedAdapter は更新側も同じであることを確認する。
func TestUpdateTagWritesThroughToCacheWithPluginTypedAdapter(t *testing.T) {
	ctx := context.Background()
	uc := &UsecaseContext{}
	repositories, _ := newWriteThroughTagRepositories(t, true)

	tag := newWriteThroughTag("tag-002", "target-002", "日記")
	if _, gkillErrors, err := uc.AddTag(ctx, repositories, "testuser", "test_device", "ja", tag, nil); err != nil || len(gkillErrors) != 0 {
		t.Fatalf("AddTag failed: err = %v, gkillErrors = %v", err, gkillErrors)
	}

	renamed := tag
	renamed.Tag = "日誌"
	renamed.UpdateTime = tag.UpdateTime.Add(time.Minute)
	updatedTag, gkillErrors, err := uc.UpdateTag(ctx, repositories, "testuser", "test_device", "ja", renamed, nil)
	if err != nil {
		t.Fatalf("UpdateTag failed: %v", err)
	}
	if len(gkillErrors) != 0 {
		t.Fatalf("UpdateTag returned errors: %v", gkillErrors)
	}
	if updatedTag == nil {
		t.Fatal("UpdateTagが更新後のタグを返していない")
	}
	if updatedTag.Tag != "日誌" {
		t.Errorf("Tag = %q, want %q。キャッシュrepに反映されていないと改名前の値が返る", updatedTag.Tag, "日誌")
	}
}

// TestAddTagWithoutCachedRep はキャッシュ無効時(CachedRepsがnil)に
// 書き込み自体が壊れないことを確認する。
func TestAddTagWithoutCachedRep(t *testing.T) {
	ctx := context.Background()
	uc := &UsecaseContext{}
	repositories, writeRep := newWriteThroughTagRepositories(t, false)

	if repositories.CachedReps.Tag != nil {
		t.Fatal("前提が崩れている: キャッシュ無効時はCachedReps.Tagがnilであること")
	}

	tag := newWriteThroughTag("tag-003", "target-003", "日記")
	if _, gkillErrors, err := uc.AddTag(ctx, repositories, "testuser", "test_device", "ja", tag, nil); err != nil || len(gkillErrors) != 0 {
		t.Fatalf("AddTag failed: err = %v, gkillErrors = %v", err, gkillErrors)
	}

	got, err := writeRep.GetTagsByTargetID(ctx, "target-003")
	if err != nil {
		t.Fatalf("GetTagsByTargetID failed: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("実体repに書かれていない。len = %d, want 1", len(got))
	}
}

// repsCountGuardPattern は復活させてはいけない個数判定。
// ReKyou / MiReKyou は `repositories.ReKyouReps.ReKyouRepositories` と入れ子なので
// 末尾は Reps / Repositories の両方を見る。
var repsCountGuardPattern = regexp.MustCompile(`len\([\w.]*Rep(?:s|ositories)\)\s*==\s*1\s*&&\s*\*gkill_options\.Cache\w+Reps`)

// TestNoRepsCountCacheGuard は「XxxRepsの長さが1ならキャッシュrep」という
// 個数判定がソースへ再び現れたら落とす。
// GkillDAOManagerはキャッシュrepでXxxRepsを差し替えたあとにプラグインの型別アダプタを
// appendするので、この判定はプラグイン1つで崩れる。
// 書き込み後の反映はrepositories.WriteThroughXxxCacheを使うこと。
func TestNoRepsCountCacheGuard(t *testing.T) {
	root := ".."
	violations := []string{}
	scanned := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".go") {
			return nil
		}
		if strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		scanned++
		for i, line := range strings.Split(string(content), "\n") {
			if repsCountGuardPattern.MatchString(line) {
				violations = append(violations, fmt.Sprintf("%s:%d: %s", filepath.ToSlash(path), i+1, strings.TrimSpace(line)))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk %s: %v", root, err)
	}
	// 走査対象が0件でも「違反なし」で通ってしまうので、実際に読めたことを確かめる
	if scanned < 100 {
		t.Fatalf("走査できたGoファイルが %d 件しかない。rootの指定(%s)が間違っている可能性がある", scanned, root)
	}
	if len(violations) != 0 {
		t.Fatalf("個数によるキャッシュrep判定が復活している。repositories.WriteThroughXxxCacheを使うこと:\n%s", strings.Join(violations, "\n"))
	}
}
