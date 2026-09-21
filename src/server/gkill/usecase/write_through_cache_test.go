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

	"github.com/mt3hr/gkill/src/server/gkill/api/message"
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

// newWriteThroughTargetKyou は「タグ/テキストを付ける対象のKyou」を1件だけ持つrepを作る。
//
// AddTag / AddText は txID が nil のとき対象の実在を検査する
// （存在しないIDへ付けると、どこにも付いていない宙吊りの記録ができるため）。
// このファイルのテストは write-through の検査が目的で対象の実在は関心事ではないが、
// 検査を通すために実体を1件用意する。
func newWriteThroughTargetKyou(t *testing.T, repositories *reps.GkillRepositories, targetID string) {
	t.Helper()
	kmemoRep, err := reps.NewKmemoRepositorySQLite3Impl(context.Background(), filepath.Join(t.TempDir(), "target_kmemo.db"), false)
	if err != nil {
		t.Fatalf("failed to create kmemo repo: %v", err)
	}
	t.Cleanup(func() { _ = kmemoRep.Close(context.Background()) })

	now := time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)
	if err := kmemoRep.AddKmemoInfo(context.Background(), reps.Kmemo{
		ID:           targetID,
		Content:      "タグを付ける対象",
		RelatedTime:  now,
		CreateTime:   now,
		CreateApp:    "test_app",
		CreateDevice: "test_device",
		CreateUser:   "testuser",
		UpdateTime:   now,
		UpdateApp:    "test_app",
		UpdateDevice: "test_device",
		UpdateUser:   "testuser",
	}); err != nil {
		t.Fatalf("failed to add target kmemo: %v", err)
	}
	repositories.KmemoReps = reps.KmemoRepositories{kmemoRep}
	repositories.Reps = reps.Repositories{kmemoRep}
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
	newWriteThroughTargetKyou(t, repositories, "target-001")

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
	newWriteThroughTargetKyou(t, repositories, "target-002")

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
	newWriteThroughTargetKyou(t, repositories, "target-003")

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

// ─── タグの対象IDの実在検査 ───────────────────────────────────────────────────

// newWriteThroughTagTempRep は確定前のタグを置く一時リポジトリを作る。
func newWriteThroughTagTempRep(t *testing.T) reps.TagTempRepository {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	rep, err := reps.NewTagTempRepositorySQLite3Impl(context.Background(), db, &sync.RWMutex{})
	if err != nil {
		t.Fatalf("NewTagTempRepositorySQLite3Impl: %v", err)
	}
	return rep
}

// TestAddTagRejectsMissingTarget は存在しない対象IDへのタグ追加が弾かれることを固定する。
//
// 以前は成功が返り、どこにも付いていない宙吊りのタグができていた。
// そのタグ名は get_all_tag_names の語彙に載るので、
// 0件しかヒットしない候補が検索の選択肢に混ざり続ける（指摘で実測）。
// IDの取り違えはAIが最も起こしやすい事故で、そのとき静かに失敗していた。
func TestAddTagRejectsMissingTarget(t *testing.T) {
	ctx := context.Background()
	uc := &UsecaseContext{}
	repositories, _ := newWriteThroughTagRepositories(t, true)
	newWriteThroughTargetKyou(t, repositories, "target-001")

	tag := newWriteThroughTag("tag-missing", "target-does-not-exist", "日記")
	addedTag, gkillErrors, err := uc.AddTag(ctx, repositories, "testuser", "test_device", "ja", tag, nil)
	if err != nil {
		t.Fatalf("AddTag returned a hard error instead of a gkill error: %v", err)
	}
	if len(gkillErrors) == 0 {
		t.Fatal("存在しない対象IDへのタグ追加が成功している。宙吊りのタグができる")
	}
	if gkillErrors[0].ErrorCode != message.NotFoundKyouInfoError {
		t.Errorf("ErrorCode = %s, want %s（「サーバ障害」ではなく「指定ミス」として返すこと）",
			gkillErrors[0].ErrorCode, message.NotFoundKyouInfoError)
	}
	if addedTag != nil {
		t.Error("弾いたのにタグを返している")
	}
}

// TestAddTagSkipsTargetCheckInTransaction は
// トランザクション中は対象の実在を検査しないことを固定する。
//
// **これを検査してしまうとメモ帳(KFTL)からのタグ付き投入が全滅する。**
// KFTL は1つのトランザクションで「対象のKyouを作る」→「それにタグを付ける」を
// 挿入順に流すので、タグを付ける時点で対象はまだ一時リポジトリの中にいて確定していない。
func TestAddTagSkipsTargetCheckInTransaction(t *testing.T) {
	ctx := context.Background()
	uc := &UsecaseContext{}
	repositories, _ := newWriteThroughTagRepositories(t, true)
	newWriteThroughTargetKyou(t, repositories, "target-001")
	repositories.TempReps = &reps.TempReps{TagTempRep: newWriteThroughTagTempRep(t)}

	txID := "tx-001"
	tag := newWriteThroughTag("tag-in-tx", "target-not-committed-yet", "日記")
	_, gkillErrors, err := uc.AddTag(ctx, repositories, "testuser", "test_device", "ja", tag, &txID)
	if err != nil {
		t.Fatalf("AddTag failed: %v", err)
	}
	if len(gkillErrors) != 0 {
		t.Fatalf("トランザクション中は対象の実在を検査しないこと。検査するとKFTLのタグ付き投入が全滅する: %v", gkillErrors)
	}
}
