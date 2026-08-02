package reps

import (
	"fmt"
	"sync"
	"testing"
	"time"

	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// GkillRepositoriesはユーザ+デバイス単位で共有され、
// 検索(FindKyous)と追加/更新(usecase, handle_commit_tx)の両方から同時に触られる。
// 最新版アドレスのmapを素のまま公開していたころは
// 「rykvの自動更新中にKFTLで投稿する」「タブを2枚開く」だけで
// concurrent map read and map write に当たり、
// recoverできないfatal errorでサーバプロセスごと落ちていた。
//
// 以下のテストは -race で検出させることを狙っている。
// 通常実行でも運が悪ければfatal errorで落ちるので、素通りしたことが即OKではない。

// TestGkillRepositories_LatestDataAddressesConcurrentReadWrite は
// 1件ずつの読み書きが並行しても壊れないことを確認する。
func TestGkillRepositories_LatestDataAddressesConcurrentReadWrite(t *testing.T) {
	repositories := &GkillRepositories{}

	const parallelism = 50
	wg := &sync.WaitGroup{}
	for i := range parallelism {
		id := fmt.Sprintf("target-%d", i)

		wg.Add(1)
		go func() {
			defer wg.Done()
			repositories.SetLatestDataRepositoryAddress(id, gkill_cache.LatestDataRepositoryAddress{
				TargetID:                 id,
				LatestDataRepositoryName: "rep",
				DataUpdateTime:           time.Unix(int64(i), 0),
			})
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			// 書き込みと競合する読み取り。存在有無は問わない
			repositories.GetLatestDataRepositoryAddress(id)
		}()
	}
	wg.Wait()

	// 書いた分がすべて読めること
	for i := range parallelism {
		id := fmt.Sprintf("target-%d", i)
		addr, exist := repositories.GetLatestDataRepositoryAddress(id)
		if !exist {
			t.Fatalf("%s が書き込まれていない", id)
		}
		if addr.TargetID != id {
			t.Errorf("TargetID = %q, want %q", addr.TargetID, id)
		}
	}
}

// TestGkillRepositories_LatestDataAddressesConcurrentReplace は
// map全体の差し替え(検索開始時のキャッシュ再取得)と個別の読み書きが
// 並行しても壊れないことを確認する。
// 差し替えはmapのヘッダごと入れ替わるので、読み手が旧mapを見ている最中に
// 書き手が新mapへ切り替えると特に壊れやすい。
func TestGkillRepositories_LatestDataAddressesConcurrentReplace(t *testing.T) {
	repositories := &GkillRepositories{}
	repositories.SetLatestDataRepositoryAddresses(map[string]gkill_cache.LatestDataRepositoryAddress{
		"target-0": {TargetID: "target-0"},
	})

	wg := &sync.WaitGroup{}
	for i := range 30 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			repositories.SetLatestDataRepositoryAddresses(map[string]gkill_cache.LatestDataRepositoryAddress{
				"target-0": {TargetID: "target-0", DataUpdateTime: time.Unix(int64(i), 0)},
			})
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			repositories.GetLatestDataRepositoryAddress("target-0")
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			repositories.SetLatestDataRepositoryAddress(fmt.Sprintf("added-%d", i), gkill_cache.LatestDataRepositoryAddress{})
		}()
	}
	wg.Wait()

	if _, exist := repositories.GetLatestDataRepositoryAddress("target-0"); !exist {
		t.Error("差し替え後もtarget-0は存在するはず")
	}
}

// TestGkillRepositories_SetLatestDataAddressInitializesNilMap は、
// 検索を一度も通していない状態(mapがnil)で追加操作が来ても
// nil mapへの代入でpanicしないことを確認する。
// 実際に「初回検索前にKFTLで投稿する」経路でこれを踏む。
func TestGkillRepositories_SetLatestDataAddressInitializesNilMap(t *testing.T) {
	repositories := &GkillRepositories{}

	repositories.SetLatestDataRepositoryAddress("target-0", gkill_cache.LatestDataRepositoryAddress{TargetID: "target-0"})

	addr, exist := repositories.GetLatestDataRepositoryAddress("target-0")
	if !exist {
		t.Fatal("nil mapからの初期化ができていない")
	}
	if addr.TargetID != "target-0" {
		t.Errorf("TargetID = %q, want %q", addr.TargetID, "target-0")
	}
}
