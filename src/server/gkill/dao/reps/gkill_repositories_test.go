package reps

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
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

// ---------------------------------------------------------------------------
// 2026-08-24 の再監査: 記録を削除してもタグは語彙に残り続け、
// 検索候補に0件しか返さない項目が溜まっていた。
//
// フィルタは GkillRepositories（rep の集約）の層にあるので、
// キャッシュrepか素のrepかでは挙動が変わらない。判定に使う最新版アドレス表は
// UpdateCache の Phase2 が IS_DELETED を行から読み直して埋めるので、
// キャッシュONでも同じ値になる。
// ---------------------------------------------------------------------------

// registerLatestAddressWithDeleted は削除フラグつきで最新版アドレスを1件登録する。
func registerLatestAddressWithDeleted(t *testing.T, repositories *GkillRepositories, targetID string, isDeleted bool) {
	t.Helper()
	_, err := repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(context.Background(), gkill_cache.LatestDataRepositoryAddress{
		IsDeleted:                              isDeleted,
		TargetID:                               targetID,
		LatestDataRepositoryName:               "tag",
		DataUpdateTime:                         testTime(),
		LatestDataRepositoryAddressUpdatedTime: testTime(),
	})
	if err != nil {
		t.Fatalf("AddOrUpdateLatestDataRepositoryAddress failed: %v", err)
	}
}

func TestGkillRepositoriesGetAllTagNames_DropsTagsWhoseTargetIsDeleted(t *testing.T) {
	ctx := context.Background()
	repositories := newIDsSemanticsRepositories(t)

	liveTag := makeTag("tag-live-001", "target-live-001", "生きているタグ")
	deadTag := makeTag("tag-dead-001", "target-dead-001", "消した記録のタグ")
	for _, tag := range []Tag{liveTag, deadTag} {
		if err := repositories.TagReps[0].AddTagInfo(ctx, tag); err != nil {
			t.Fatalf("AddTagInfo failed: %v", err)
		}
	}
	registerLatestAddressWithDeleted(t, repositories, liveTag.TargetID, false)
	registerLatestAddressWithDeleted(t, repositories, deadTag.TargetID, true)

	names, err := repositories.GetAllTagNames(ctx)
	if err != nil {
		t.Fatalf("GetAllTagNames failed: %v", err)
	}
	if len(names) != 1 || names[0] != "生きているタグ" {
		t.Errorf("対象が削除済みのタグは語彙から落ちるはず: got %v", names)
	}
}

func TestGkillRepositoriesGetAllTagNames_KeepsTagsWhoseTargetIsNotInTheAddressTable(t *testing.T) {
	ctx := context.Background()
	repositories := newIDsSemanticsRepositories(t)

	// プラグインや git の記録は最新版アドレス表に載らない。
	// 載っていないことを「削除済み」と読むと語彙が黙って痩せる
	orphanTag := makeTag("tag-orphan-001", "target-not-in-address-table", "表に載らない対象のタグ")
	if err := repositories.TagReps[0].AddTagInfo(ctx, orphanTag); err != nil {
		t.Fatalf("AddTagInfo failed: %v", err)
	}
	// 表を空にしないため、無関係の生存レコードを1件だけ入れておく
	registerLatestAddressWithDeleted(t, repositories, "target-unrelated-001", false)

	names, err := repositories.GetAllTagNames(ctx)
	if err != nil {
		t.Fatalf("GetAllTagNames failed: %v", err)
	}
	if len(names) != 1 || names[0] != "表に載らない対象のタグ" {
		t.Errorf("アドレス表に載っていない対象のタグは残すはず: got %v", names)
	}
}

func TestGkillRepositoriesGetAllTagNamesIncludingDeletedTargets_KeepsEverything(t *testing.T) {
	ctx := context.Background()
	repositories := newIDsSemanticsRepositories(t)

	// 「そのタグ名は実在するか」の検証はこちらを使う。
	// GetAllTagNames で検証すると、include_deleted_data で削除済みを開いた
	// タグ検索に「未知のタグ」という誤った警告が出る
	liveTag := makeTag("tag-live-002", "target-live-002", "生きているタグ")
	deadTag := makeTag("tag-dead-002", "target-dead-002", "消した記録のタグ")
	for _, tag := range []Tag{liveTag, deadTag} {
		if err := repositories.TagReps[0].AddTagInfo(ctx, tag); err != nil {
			t.Fatalf("AddTagInfo failed: %v", err)
		}
	}
	registerLatestAddressWithDeleted(t, repositories, liveTag.TargetID, false)
	registerLatestAddressWithDeleted(t, repositories, deadTag.TargetID, true)

	names, err := repositories.GetAllTagNamesIncludingDeletedTargets(ctx)
	if err != nil {
		t.Fatalf("GetAllTagNamesIncludingDeletedTargets failed: %v", err)
	}
	if len(names) != 2 {
		t.Errorf("検証用の一覧は対象の生死を問わず全部返すはず: got %v", names)
	}
}

// 2026-08-24 の再監査 事象10: 1つの応答の中で種別名が食い違っていた
// （updated_mi は mi_check、updated_kyou は mi_create）。
// Mi の5射影は同じ1行から SQL が合成するラベルで、MI テーブルに DATA_TYPE 列は無い。
// 5つとも UPDATE_TIME が同着なので、素の MaxFunc は SQLite の UNION 出力順に従っていた。
func TestCompareMiProjectionPreference(t *testing.T) {
	// 正準（作成時刻の射影）が勝つ。検索の既定 mi_sort_type=create_time と揃える
	if compareMiProjectionPreference("mi_create", "mi_check") <= 0 {
		t.Error("mi_create が mi_check に勝たない")
	}
	if compareMiProjectionPreference("mi_check", "mi_create") >= 0 {
		t.Error("mi_check が mi_create に勝ってしまう")
	}
	// 同じなら差は無い
	if compareMiProjectionPreference("mi_check", "mi_check") != 0 {
		t.Error("同じ射影に差が出ている")
	}
	// どちらも正準でないなら順序を作らない（UpdateTime 側の判断に委ねる）
	if compareMiProjectionPreference("mi_check", "mi_limit") != 0 {
		t.Error("正準でない射影同士に順序を作っている")
	}
}

// repNamesStubRep は名前だけを持つリーフ。GetAllRepNames の列挙の確認用。
type repNamesStubRep struct {
	Repository
	repName  string
	repNames []string // nil なら RepNamesProvider を名乗らない
}

func (s *repNamesStubRep) GetRepName(_ context.Context) (string, error) { return s.repName, nil }
func (s *repNamesStubRep) UnWrap() ([]Repository, error)                { return []Repository{s}, nil }

// repNamesProviderStubRep は複数の rep 名を名乗るリーフ。
type repNamesProviderStubRep struct{ repNamesStubRep }

func (s *repNamesProviderStubRep) GetRepNames(_ context.Context) ([]string, error) {
	return s.repNames, nil
}

// UnWrap は自分自身（外側の型）を返す。埋め込み側の UnWrap に任せると内側の型が返り、
// RepNamesProvider を名乗らないリーフとして数えられてしまう。
func (s *repNamesProviderStubRep) UnWrap() ([]Repository, error) { return []Repository{s}, nil }

// GetAllRepNames が RepNamesProvider の申告名を全部並べ、名乗らないリーフは GetRepName の1つになること。
// プラグインが名乗った名前がここに載らないと、サイドバーの rep 一覧に出ず利用者が選べない。
// 1つのリーフが複数の名前を返しても集約が詰まらない（チャネル容量）ことも兼ねる。
func TestGkillRepositoriesGetAllRepNames_IncludesDeclaredRepNames(t *testing.T) {
	plugin := &repNamesProviderStubRep{repNamesStubRep{repName: "ArchivedGit", repNames: []string{"racoonboard", "ocha", "urlog"}}}
	native := &repNamesStubRep{repName: "ocha"}
	repositories := &GkillRepositories{Reps: Repositories{plugin, native}}

	got, err := repositories.GetAllRepNames(context.Background())
	if err != nil {
		t.Fatalf("GetAllRepNames failed: %v", err)
	}
	want := []string{"ocha", "racoonboard", "urlog"}
	if !slices.Equal(got, want) {
		t.Errorf("GetAllRepNames = %v, want %v（申告名を全部・manifest名は入れない・同名は1つ）", got, want)
	}
}
