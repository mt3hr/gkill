package reps

// `/api/get_kyou`（update_time 指定なし）の実体 GetKyouHistoriesByRepName の回帰テスト。
//
// 1. rep名の指定が無いときは UnWrap() せず、包まれたrep（キャッシュrep）の GetKyouHistories を回すこと。
//    UnWrap() するとキャッシュを丸ごとバイパスして生の leaf rep へ戻る。実データでは leaf が約850本あり、
//    引き直し1回ごとに SQLite を ~760本開いて閉じ、git 84本を全走査し、プラグイン6本へIPCしていた
//    （selectMatchRepsFromQuery について ADR-0101 が禁止したのと同じ壊れ方。ADR-0218）。
// 2. rep名の指定があるときは従来どおり UnWrap() して leaf 名で照合すること
//    （キャッシュrepの GetRepName() は集約名を返すので、剥がさないと一致しない。e568c8b8）。
// 3. GkillRepositories 側は、最新版アドレス表に載っているIDならプラグインrepへ聞かないこと。
//    プラグインKyouのIDは表に載らないので、載っている＝ネイティブの記録で、プラグインは必ず空で返す。
//    載っていないID（プラグインKyou・追加直後の記録）では従来どおり全repへ聞くこと。

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	gkill_cache "github.com/mt3hr/gkill/src/server/gkill/dao/reps/cache"
)

// historiesStubRep は Repository を埋め込んで必要なメソッドだけ差し替えたスタブ。
// キャッシュrep（unwrapped に leaf を持つ）と leaf（unwrapped が自分自身）の両方に使う。
type historiesStubRep struct {
	Repository
	repName        string
	unwrapCalls    *int
	historiesCalls *int
	unwrapped      []Repository
	histories      []Kyou
}

func (s *historiesStubRep) GetRepName(_ context.Context) (string, error) {
	return s.repName, nil
}

func (s *historiesStubRep) UnWrap() ([]Repository, error) {
	*s.unwrapCalls++
	if s.unwrapped == nil {
		return []Repository{s}, nil
	}
	return s.unwrapped, nil
}

func (s *historiesStubRep) GetKyouHistories(_ context.Context, id string) ([]Kyou, error) {
	*s.historiesCalls++
	matched := []Kyou{}
	for _, kyou := range s.histories {
		if kyou.ID == id {
			matched = append(matched, kyou)
		}
	}
	return matched, nil
}

// historiesStubPluginRep はプラグインrep。PluginRepository を埋め込むので型アサーションで区別される。
type historiesStubPluginRep struct {
	PluginRepository
	repName        string
	historiesCalls *int
}

func (s *historiesStubPluginRep) GetRepName(_ context.Context) (string, error) {
	return s.repName, nil
}

func (s *historiesStubPluginRep) UnWrap() ([]Repository, error) {
	return []Repository{s}, nil
}

func (s *historiesStubPluginRep) GetKyouHistories(_ context.Context, _ string) ([]Kyou, error) {
	*s.historiesCalls++
	return []Kyou{}, nil
}

func newHistoriesKyou(id string, repName string, updateTime time.Time) Kyou {
	return Kyou{
		ID:         id,
		RepName:    repName,
		DataType:   "kmemo",
		UpdateTime: updateTime,
	}
}

// rep名の指定が無い取得はキャッシュrepをそのまま回し、leaf へ降りない。
func TestGetKyouHistoriesByRepName_NoRepNameKeepsCachedRep(t *testing.T) {
	ctx := context.Background()
	base := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

	cachedUnwrapCalls, cachedHistoriesCalls := 0, 0
	leafUnwrapCalls, leafHistoriesCalls := 0, 0
	leaf := &historiesStubRep{
		repName:        "LeafKmemoRep",
		unwrapCalls:    &leafUnwrapCalls,
		historiesCalls: &leafHistoriesCalls,
		histories:      []Kyou{newHistoriesKyou("kyou-1", "LeafKmemoRep", base)},
	}
	cached := &historiesStubRep{
		repName:        "CachedKmemoReps",
		unwrapCalls:    &cachedUnwrapCalls,
		historiesCalls: &cachedHistoriesCalls,
		unwrapped:      []Repository{leaf},
		histories: []Kyou{
			newHistoriesKyou("kyou-1", "LeafKmemoRep", base),
			newHistoriesKyou("kyou-1", "LeafKmemoRep", base.Add(time.Minute)),
			newHistoriesKyou("kyou-other", "LeafKmemoRep", base),
		},
	}

	got, err := Repositories{cached}.GetKyouHistoriesByRepName(ctx, "kyou-1", nil)
	if err != nil {
		t.Fatalf("GetKyouHistoriesByRepName failed: %v", err)
	}

	if cachedUnwrapCalls != 0 {
		t.Errorf("UnWrap() が %d 回呼ばれた。rep名の指定が無い取得でキャッシュをバイパスしてはいけない", cachedUnwrapCalls)
	}
	if leafHistoriesCalls != 0 {
		t.Errorf("leaf の GetKyouHistories が %d 回呼ばれた。キャッシュrepで答えるべき", leafHistoriesCalls)
	}
	if cachedHistoriesCalls != 1 {
		t.Errorf("キャッシュrepの GetKyouHistories が %d 回呼ばれた, want 1", cachedHistoriesCalls)
	}
	if len(got) != 2 {
		t.Fatalf("histories len = %d, want 2 (同じIDの2版)", len(got))
	}
	if !got[0].UpdateTime.After(got[1].UpdateTime) {
		t.Errorf("UpdateTime 降順で返すべき: got[0]=%v got[1]=%v", got[0].UpdateTime, got[1].UpdateTime)
	}
}

// rep名の指定があるときは従来どおり leaf まで降りて、名前が一致する leaf だけを回る。
func TestGetKyouHistoriesByRepName_WithRepNameUnwrapsToMatchingLeaf(t *testing.T) {
	ctx := context.Background()
	base := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

	cachedUnwrapCalls, cachedHistoriesCalls := 0, 0
	pcUnwrapCalls, pcHistoriesCalls := 0, 0
	phoneUnwrapCalls, phoneHistoriesCalls := 0, 0
	pcLeaf := &historiesStubRep{
		repName:        "PCKmemoRep",
		unwrapCalls:    &pcUnwrapCalls,
		historiesCalls: &pcHistoriesCalls,
		histories:      []Kyou{newHistoriesKyou("kyou-1", "PCKmemoRep", base)},
	}
	phoneLeaf := &historiesStubRep{
		repName:        "PhoneKmemoRep",
		unwrapCalls:    &phoneUnwrapCalls,
		historiesCalls: &phoneHistoriesCalls,
		histories:      []Kyou{newHistoriesKyou("kyou-1", "PhoneKmemoRep", base.Add(time.Minute))},
	}
	cached := &historiesStubRep{
		repName:        "CachedKmemoReps",
		unwrapCalls:    &cachedUnwrapCalls,
		historiesCalls: &cachedHistoriesCalls,
		unwrapped:      []Repository{pcLeaf, phoneLeaf},
	}

	repName := "PhoneKmemoRep"
	got, err := Repositories{cached}.GetKyouHistoriesByRepName(ctx, "kyou-1", &repName)
	if err != nil {
		t.Fatalf("GetKyouHistoriesByRepName failed: %v", err)
	}

	if cachedUnwrapCalls != 1 {
		t.Errorf("UnWrap() が %d 回呼ばれた, want 1（rep名照合には leaf 名が要る）", cachedUnwrapCalls)
	}
	if cachedHistoriesCalls != 0 {
		t.Errorf("キャッシュrepの GetKyouHistories が %d 回呼ばれた。集約名では一致しないので呼ばれないはず", cachedHistoriesCalls)
	}
	if pcHistoriesCalls != 0 {
		t.Errorf("名前が一致しない leaf の GetKyouHistories が %d 回呼ばれた", pcHistoriesCalls)
	}
	if phoneHistoriesCalls != 1 {
		t.Errorf("名前が一致する leaf の GetKyouHistories が %d 回呼ばれた, want 1", phoneHistoriesCalls)
	}
	if len(got) != 1 || got[0].RepName != "PhoneKmemoRep" {
		t.Fatalf("histories = %+v, want PhoneKmemoRep の1件", got)
	}
}

func newHistoriesAddressDAO(t *testing.T) gkill_cache.LatestDataRepositoryAddressDAO {
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
	return addrDAO
}

// 最新版アドレス表に載っているIDでは、プラグインrepへ聞かない。載っていないIDでは聞く。
func TestGkillRepositoriesGetKyouHistoriesByRepName_SkipsPluginsForAddressedID(t *testing.T) {
	ctx := context.Background()
	base := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

	cachedUnwrapCalls, cachedHistoriesCalls := 0, 0
	pluginHistoriesCalls := 0
	cached := &historiesStubRep{
		repName:        "CachedKmemoReps",
		unwrapCalls:    &cachedUnwrapCalls,
		historiesCalls: &cachedHistoriesCalls,
		histories:      []Kyou{newHistoriesKyou("native-1", "LeafKmemoRep", base)},
	}
	plugin := &historiesStubPluginRep{repName: "gkill_plugin_stub", historiesCalls: &pluginHistoriesCalls}

	repositories := &GkillRepositories{
		Reps:                           Repositories{cached, plugin},
		LatestDataRepositoryAddressDAO: newHistoriesAddressDAO(t),
	}
	if _, err := repositories.LatestDataRepositoryAddressDAO.AddOrUpdateLatestDataRepositoryAddress(ctx, gkill_cache.LatestDataRepositoryAddress{
		TargetID:                 "native-1",
		LatestDataRepositoryName: "LeafKmemoRep",
		DataUpdateTime:           base,
	}); err != nil {
		t.Fatalf("AddOrUpdateLatestDataRepositoryAddress: %v", err)
	}

	// 表に載っているID: プラグインへは聞かない
	got, err := repositories.GetKyouHistoriesByRepName(ctx, "native-1", nil)
	if err != nil {
		t.Fatalf("GetKyouHistoriesByRepName(native-1) failed: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("histories len = %d, want 1", len(got))
	}
	if pluginHistoriesCalls != 0 {
		t.Errorf("アドレス表に載っているIDでプラグインの GetKyouHistories が %d 回呼ばれた。プラグインは必ず空で返すので聞かなくてよい", pluginHistoriesCalls)
	}
	if cachedHistoriesCalls != 1 {
		t.Errorf("キャッシュrepの GetKyouHistories が %d 回呼ばれた, want 1", cachedHistoriesCalls)
	}
	if cachedUnwrapCalls != 0 {
		t.Errorf("UnWrap() が %d 回呼ばれた", cachedUnwrapCalls)
	}

	// 表に無いID（プラグインKyou・追加直後の記録）: 従来どおり全repへ聞く
	if _, err := repositories.GetKyouHistoriesByRepName(ctx, "plugin-1", nil); err != nil {
		t.Fatalf("GetKyouHistoriesByRepName(plugin-1) failed: %v", err)
	}
	if pluginHistoriesCalls != 1 {
		t.Errorf("アドレス表に無いIDでプラグインの GetKyouHistories が %d 回呼ばれた, want 1", pluginHistoriesCalls)
	}
	if cachedHistoriesCalls != 2 {
		t.Errorf("キャッシュrepの GetKyouHistories が %d 回呼ばれた, want 2", cachedHistoriesCalls)
	}
}

// アドレス表を持たない組み立て（素の構造体）でも落ちず、全repへ聞く。
func TestGkillRepositoriesGetKyouHistoriesByRepName_WithoutAddressDAO(t *testing.T) {
	ctx := context.Background()

	cachedUnwrapCalls, cachedHistoriesCalls := 0, 0
	pluginHistoriesCalls := 0
	cached := &historiesStubRep{repName: "CachedKmemoReps", unwrapCalls: &cachedUnwrapCalls, historiesCalls: &cachedHistoriesCalls}
	plugin := &historiesStubPluginRep{repName: "gkill_plugin_stub", historiesCalls: &pluginHistoriesCalls}

	repositories := &GkillRepositories{Reps: Repositories{cached, plugin}}
	if _, err := repositories.GetKyouHistoriesByRepName(ctx, "any", nil); err != nil {
		t.Fatalf("GetKyouHistoriesByRepName failed: %v", err)
	}
	if cachedHistoriesCalls != 1 || pluginHistoriesCalls != 1 {
		t.Errorf("cached=%d plugin=%d, want 1/1（絞り込む根拠が無いので全repへ聞く）", cachedHistoriesCalls, pluginHistoriesCalls)
	}
}
