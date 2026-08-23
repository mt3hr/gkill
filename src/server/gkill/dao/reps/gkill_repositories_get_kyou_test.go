package reps

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/main/common/gkill_options"
)

// M-01: GkillRepositories.GetKyou は、最新版アドレス表に載っていないID
// （プラグインKyou、および追加直後～次回UpdateCACHEまでのネイティブ記録）を
// updateTime 付きで取得しても panic せず、正しく解決する。
// 修正前は GetLatestDataRepositoryAddress の (nil,nil) と rep.GetKyou の (nil,nil) を
// 素で参照して nil deref panic → recoverMiddleware で 500 になっていた。
func TestGkillRepositoriesGetKyou_NilAddressDoesNotPanic(t *testing.T) {
	ctx := context.Background()
	base := time.Date(2026, 8, 1, 10, 0, 0, 0, time.Local)

	t.Run("アドレス表に無いIDでも版指定で解決できる", func(t *testing.T) {
		repositories := newIDsSemanticsRepositories(t)
		kmemoRepo := newTempKmemoRepo(t)
		repositories.Reps = Repositories{kmemoRepo}

		kmemo := makeKmemo("m01-present", "body")
		kmemo.CreateTime, kmemo.RelatedTime, kmemo.UpdateTime = base, base, base
		if err := kmemoRepo.AddKmemoInfo(ctx, kmemo); err != nil {
			t.Fatal(err)
		}

		// アドレス未登録（= 表に無い）でも、全repに問い合わせて版が見つかる。
		got, err := repositories.GetKyou(ctx, "m01-present", &base)
		if err != nil {
			t.Fatalf("GetKyou returned error: %v", err)
		}
		if got == nil {
			t.Fatal("GetKyou returned nil for an existing version")
		}
		if got.ID != "m01-present" {
			t.Errorf("GetKyou returned wrong id: %s", got.ID)
		}
	})

	t.Run("指定した版が無いときは panic せず nil を返す", func(t *testing.T) {
		repositories := newIDsSemanticsRepositories(t)
		kmemoRepo := newTempKmemoRepo(t)
		repositories.Reps = Repositories{kmemoRepo}

		kmemo := makeKmemo("m01-absent", "body")
		kmemo.CreateTime, kmemo.RelatedTime, kmemo.UpdateTime = base, base, base
		if err := kmemoRepo.AddKmemoInfo(ctx, kmemo); err != nil {
			t.Fatal(err)
		}

		other := base.Add(48 * time.Hour)
		got, err := repositories.GetKyou(ctx, "m01-absent", &other)
		if err != nil {
			t.Fatalf("GetKyou returned error: %v", err)
		}
		if got != nil {
			t.Errorf("GetKyou should return nil for an absent version, got %#v", got)
		}
	})
}

// GkillRepositories.GetKyou は、キャッシュrepを挟んだ構成でも実在のKyouを見つけられる。
//
// GetKyou は最新版アドレス表の rep 名で問い合わせ先を1repに絞るが、比較の相手は
// Reps.UnWrap() が返す leaf rep の実名である。アドレス表側に "KmemoReps" のような
// 集約名を焼くと比較が永遠に外れ、全repが continue されて (nil, nil) が返る。
// 呼び出し元の usecase/tag.go・usecase/text.go は nil を「対象が存在しない」と読むので、
// 実在する記録へのタグ/テキスト追加が ERR000092 で全滅する。
//
// 上の TestGkillRepositoriesGetKyou_NilAddressDoesNotPanic は Reps に leaf rep を
// 直接入れていてアドレス表に1行も載らないため、この穴を踏まない。
func TestGkillRepositoriesGetKyou_FindsKyouThroughCachedRep(t *testing.T) {
	ctx := context.Background()
	enable := true
	old := gkill_options.CacheKmemoReps
	gkill_options.CacheKmemoReps = &enable
	t.Cleanup(func() { gkill_options.CacheKmemoReps = old })

	// t.Cleanup は LIFO なので、TempDir を先に確保してから Close を登録する。
	// 逆にすると TempDir の削除が Close より先に走り、DBを掴んだままの削除で Windows が転ける。
	dir := t.TempDir()

	repositories, err := NewGkillRepositories(sanitizeTestUserID(t.Name()))
	if err != nil {
		t.Fatalf("failed to create repositories: %v", err)
	}
	t.Cleanup(func() { _ = repositories.Close(context.Background()) })

	kmemoRep, err := NewKmemoRepositorySQLite3Impl(ctx, filepath.Join(dir, "Kmemo_TestDevice_20260824.db"), true)
	if err != nil {
		t.Fatalf("failed to create kmemo repo: %v", err)
	}
	leafRepName, err := kmemoRep.GetRepName(ctx)
	if err != nil {
		t.Fatalf("failed to get leaf rep name: %v", err)
	}

	const targetID = "cached-rep-target"
	if err := kmemoRep.AddKmemoInfo(ctx, makeKmemo(targetID, "タグを付ける対象")); err != nil {
		t.Fatalf("failed to add kmemo: %v", err)
	}

	cachedKmemoRep, err := NewKmemoRepositoryCachedSQLite3Impl(ctx, KmemoRepositories{kmemoRep}, repositories.CacheMemoryDB, repositories.CacheMemoryDBMutex, sanitizeTestUserID(t.Name())+"_KMEMO_CACHE")
	if err != nil {
		t.Fatalf("failed to create cached kmemo repo: %v", err)
	}
	repositories.KmemoReps = KmemoRepositories{cachedKmemoRep}
	repositories.WriteKmemoRep = kmemoRep
	repositories.Reps = Repositories{cachedKmemoRep}

	if err := repositories.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache() error: %v", err)
	}

	addr, err := repositories.LatestDataRepositoryAddressDAO.GetLatestDataRepositoryAddress(ctx, targetID)
	if err != nil {
		t.Fatalf("GetLatestDataRepositoryAddress() error: %v", err)
	}
	if addr == nil {
		t.Fatal("最新版アドレス表に行が無い。この検査は「表に載っている状態でGetKyouを引く」ためのもので、載っていないと絞り込み自体が働かず素通りする")
	}
	if addr.LatestDataRepositoryName != leafRepName {
		t.Errorf("LatestDataRepositoryName = %q, want %q（集約名を焼くと GetKyou の絞り込みが leaf 名と永遠に一致しない）", addr.LatestDataRepositoryName, leafRepName)
	}

	got, err := repositories.GetKyou(ctx, targetID, nil)
	if err != nil {
		t.Fatalf("GetKyou() error: %v", err)
	}
	if got == nil {
		t.Fatal("GetKyou() がキャッシュrep構成で nil を返した。usecase/tag.go・usecase/text.go の実在検査がこれを「対象なし」と読み、実在する記録へのタグ/テキスト追加が ERR000092 になる")
	}
	if got.ID != targetID {
		t.Errorf("GetKyou().ID = %q, want %q", got.ID, targetID)
	}
	if got.RepName != leafRepName {
		t.Errorf("GetKyou().RepName = %q, want %q", got.RepName, leafRepName)
	}
}
