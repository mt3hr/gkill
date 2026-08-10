package reps

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
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

// GkillRepositories.FindTags / FindTexts は「IDs未指定なら最新版のIDで補完する」。
// この補完条件は len(IDs)==0 も含んでいたが、FindQueryのUse*フラグ全廃で
// 「nil=未使用 / 非nil空=明示的に0件指定」の意味論になったため IDs==nil のみになった。
// 非nil空で補完してしまうと「0件指定」が「全件」に化けるので、
// nil と 非nil空 の両方を固定する。
//
// nilで補完が効いていることも一緒に見ないと、
// 「補完が丸ごと壊れていて常に0件」でも空スライス側のテストだけは通ってしまう。

// newIDsSemanticsRepositories は Tag rep / Text rep と最新版アドレスDAOを1本ずつ持つ
// 最小構成の GkillRepositories を返す。repNameはファイル名から決まるので tag / text になる。
func newIDsSemanticsRepositories(t *testing.T) *GkillRepositories {
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

	return &GkillRepositories{
		TagReps:                        TagRepositories{newTempTagRepo(t)},
		TextReps:                       TextRepositories{newTempTextRepo(t)},
		LatestDataRepositoryAddressDAO: addrDAO,
	}
}

// registerLatestAddressForIDsSemantics は最新版アドレスを1件登録する。
// これが無いと補完で詰められるIDが0件になり、nil指定でも何も返らなくなる。
func registerLatestAddressForIDsSemantics(t *testing.T, repositories *GkillRepositories, targetID string, repName string) {
	t.Helper()
	_, err := repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(context.Background(), gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              false,
		TargetID:                               targetID,
		LatestDataRepositoryName:               repName,
		DataUpdateTime:                         testTime(),
		LatestDataRepositoryAddressUpdatedTime: testTime(),
	})
	if err != nil {
		t.Fatalf("AddOrUpdateLatestDataRepositoryAddress failed: %v", err)
	}
}

func TestGkillRepositoriesFindTags_EmptyIDsIsExplicitZeroHit(t *testing.T) {
	ctx := context.Background()
	repositories := newIDsSemanticsRepositories(t)

	tag := makeTag("tag-ids-001", "target-ids-001", "タグ")
	if err := repositories.TagReps[0].AddTagInfo(ctx, tag); err != nil {
		t.Fatalf("AddTagInfo failed: %v", err)
	}
	registerLatestAddressForIDsSemantics(t, repositories, tag.ID, "tag")

	// IDs=nil（未使用）: 最新版のIDで補完されて見つかる
	nilIDsResults, err := repositories.FindTags(ctx, &find.FindQuery{OnlyLatestData: true})
	if err != nil {
		t.Fatalf("FindTags(IDs=nil) failed: %v", err)
	}
	if len(nilIDsResults) != 1 {
		t.Fatalf("IDs=nil は最新版のIDで補完されて1件返るはず: got %d件", len(nilIDsResults))
	}

	// IDs=[]（非nil空）: 明示的な0件指定なので補完しない
	emptyIDsResults, err := repositories.FindTags(ctx, &find.FindQuery{IDs: []string{}, OnlyLatestData: true})
	if err != nil {
		t.Fatalf("FindTags(IDs=[]) failed: %v", err)
	}
	if len(emptyIDsResults) != 0 {
		t.Errorf("IDs=[] は明示的な0件指定なので補完してはいけない: got %d件", len(emptyIDsResults))
	}
}

func TestGkillRepositoriesFindTexts_EmptyIDsIsExplicitZeroHit(t *testing.T) {
	ctx := context.Background()
	repositories := newIDsSemanticsRepositories(t)

	text := makeText("text-ids-001", "target-ids-001", "テキスト")
	if err := repositories.TextReps[0].AddTextInfo(ctx, text); err != nil {
		t.Fatalf("AddTextInfo failed: %v", err)
	}
	registerLatestAddressForIDsSemantics(t, repositories, text.ID, "text")

	// IDs=nil（未使用）: 最新版のIDで補完されて見つかる
	nilIDsResults, err := repositories.FindTexts(ctx, &find.FindQuery{OnlyLatestData: true})
	if err != nil {
		t.Fatalf("FindTexts(IDs=nil) failed: %v", err)
	}
	if len(nilIDsResults) != 1 {
		t.Fatalf("IDs=nil は最新版のIDで補完されて1件返るはず: got %d件", len(nilIDsResults))
	}

	// IDs=[]（非nil空）: 明示的な0件指定なので補完しない
	emptyIDsResults, err := repositories.FindTexts(ctx, &find.FindQuery{IDs: []string{}, OnlyLatestData: true})
	if err != nil {
		t.Fatalf("FindTexts(IDs=[]) failed: %v", err)
	}
	if len(emptyIDsResults) != 0 {
		t.Errorf("IDs=[] は明示的な0件指定なので補完してはいけない: got %d件", len(emptyIDsResults))
	}
}
