package reps

import (
	"context"
	"testing"
	"time"
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
