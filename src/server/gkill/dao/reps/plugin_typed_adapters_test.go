package reps

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/gkill_plugin"
)

// allTypedProvides は型別・付随データを全部宣言したprovides。
var allTypedProvides = []gkill_plugin.PluginProvidedKind{
	gkill_plugin.PluginProvidesKC,
	gkill_plugin.PluginProvidesTag,
	gkill_plugin.PluginProvidesText,
	gkill_plugin.PluginProvidesNotification,
}

// countPluginCommands は偽プラグインが受け取ったコマンドの回数を返す。
func countPluginCommands(t *testing.T, statePath string, command string) int {
	t.Helper()
	b, err := os.ReadFile(statePath + ".commands")
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		t.Fatalf("read commands log: %v", err)
	}
	count := 0
	for _, line := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(line) == command {
			count++
		}
	}
	return count
}

// TestPluginTypedIndex_BuildsFromSingleFindKyous は本機能の中核不変条件を守る。
//
// 全アダプタの読み取りメソッドを何度叩いても、プラグインへの find_kyous は
// 索引の構築ぶんしか飛ばない。ここが壊れると、Dnoteで15,000件を舐めた瞬間に
// 直列stdio呼び出しが件数ぶん発生し、プラグインプロセスが殺され続ける。
func TestPluginTypedIndex_BuildsFromSingleFindKyous(t *testing.T) {
	fake := newFakePluginRepositoryWithProvides(t, behaviorTyped, allTypedProvides)
	ctx := context.Background()

	adapters := NewPluginTypedRepositories(fake.rep)
	if adapters.KC == nil || adapters.Tag == nil || adapters.Text == nil || adapters.Notification == nil {
		t.Fatalf("providesに書いた種別のアダプタが作られていない: %+v", adapters)
	}

	// 索引を1回だけ構築させる
	if err := fake.rep.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache: %v", err)
	}
	buildCount := countPluginCommands(t, fake.statePath, "find_kyous")
	if buildCount != 1 {
		t.Fatalf("索引構築の find_kyous = %d回, want 1", buildCount)
	}

	// 読み取りを大量に叩く
	for i := range 200 {
		id := fmt.Sprintf("typed-%d", i%fakeTypedKyouCount)
		if _, err := adapters.KC.GetKC(ctx, id, nil); err != nil {
			t.Fatalf("GetKC: %v", err)
		}
		if _, err := adapters.KC.GetKCHistories(ctx, id); err != nil {
			t.Fatalf("GetKCHistories: %v", err)
		}
		if _, err := adapters.Tag.GetTagsByTargetID(ctx, id); err != nil {
			t.Fatalf("GetTagsByTargetID: %v", err)
		}
		if _, err := adapters.Text.GetTextsByTargetID(ctx, id); err != nil {
			t.Fatalf("GetTextsByTargetID: %v", err)
		}
		if _, err := adapters.Notification.GetNotificationsByTargetID(ctx, id); err != nil {
			t.Fatalf("GetNotificationsByTargetID: %v", err)
		}
		if _, err := adapters.KC.FindKC(ctx, nil); err != nil {
			t.Fatalf("FindKC: %v", err)
		}
		if _, err := adapters.Tag.GetAllTagNames(ctx); err != nil {
			t.Fatalf("GetAllTagNames: %v", err)
		}
	}

	afterCount := countPluginCommands(t, fake.statePath, "find_kyous")
	if afterCount != buildCount {
		t.Errorf("読み取り1400回で find_kyous が %d回 → %d回 に増えた。アダプタがプラグインへ往復している", buildCount, afterCount)
	}
}

// TestPluginTypedIndex_MissDoesNotCallPlugin は、索引に無いIDを大量に引いても
// プラグインへの往復が増えないことを確認する。
// 1件ずつ聞きに行く実装にすると、プラグインを入れ替えた直後に
// 画面の行数ぶんの直列stdio呼び出しが発生する。
func TestPluginTypedIndex_MissDoesNotCallPlugin(t *testing.T) {
	fake := newFakePluginRepositoryWithProvides(t, behaviorTyped, allTypedProvides)
	ctx := context.Background()
	adapters := NewPluginTypedRepositories(fake.rep)

	if err := fake.rep.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache: %v", err)
	}
	before := countPluginCommands(t, fake.statePath, "find_kyous")

	for i := range 500 {
		kc, err := adapters.KC.GetKC(ctx, fmt.Sprintf("not-in-index-%d", i), nil)
		if err != nil {
			t.Fatalf("GetKC: %v", err)
		}
		if kc != nil {
			t.Fatalf("索引に無いIDでKCが返った: %+v", kc)
		}
	}

	// 非同期の温め直しが走りうるので、最短間隔(30秒)に守られて増えないことを見る
	time.Sleep(50 * time.Millisecond)
	after := countPluginCommands(t, fake.statePath, "find_kyous")
	if after != before {
		t.Errorf("索引ミス500回で find_kyous が %d回 → %d回 に増えた", before, after)
	}
}

// TestPluginTypedIndex_RefreshRateLimited は、UpdateCacheが
// Reps/Tag/Text/Notification相当で4連発してもプラグインを1回しか叩かないことを確認する。
func TestPluginTypedIndex_RefreshRateLimited(t *testing.T) {
	fake := newFakePluginRepositoryWithProvides(t, behaviorTyped, allTypedProvides)
	ctx := context.Background()
	adapters := NewPluginTypedRepositories(fake.rep)

	for range 4 {
		if err := fake.rep.UpdateCache(ctx); err != nil {
			t.Fatalf("UpdateCache: %v", err)
		}
	}
	if err := adapters.Tag.UpdateCache(ctx); err != nil {
		t.Fatalf("Tag.UpdateCache: %v", err)
	}
	if err := adapters.Text.UpdateCache(ctx); err != nil {
		t.Fatalf("Text.UpdateCache: %v", err)
	}

	count := countPluginCommands(t, fake.statePath, "find_kyous")
	if count != 1 {
		t.Errorf("UpdateCache 6回で find_kyous = %d回, want 1（最短間隔が効いていない）", count)
	}
}

// TestPluginTypedAdapters_TypedUpdateTimeMatchesKyou は、
// 型別データとKyouの更新時刻が完全に一致することを確認する。
//
// クライアントは「KyouのUpdateTimeと秒精度で一致する版」を選んで表示するので、
// ここがずれると typed_kc が null のままになり、Dnoteが黙って0になる。
func TestPluginTypedAdapters_TypedUpdateTimeMatchesKyou(t *testing.T) {
	fake := newFakePluginRepositoryWithProvides(t, behaviorTyped, allTypedProvides)
	ctx := context.Background()
	adapters := NewPluginTypedRepositories(fake.rep)
	if err := fake.rep.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache: %v", err)
	}

	kyou, err := adapters.KC.GetKyou(ctx, "typed-0", nil)
	if err != nil {
		t.Fatalf("GetKyou: %v", err)
	}
	if kyou == nil {
		t.Fatal("GetKyou が nil を返した")
	}
	kcs, err := adapters.KC.GetKCHistories(ctx, "typed-0")
	if err != nil {
		t.Fatalf("GetKCHistories: %v", err)
	}
	if len(kcs) != 1 {
		t.Fatalf("GetKCHistories = %d件, want 1", len(kcs))
	}
	if !kcs[0].UpdateTime.Equal(kyou.UpdateTime) {
		t.Errorf("KC.UpdateTime = %v, Kyou.UpdateTime = %v（一致していないとクライアントが型別データを拾えない）", kcs[0].UpdateTime, kyou.UpdateTime)
	}
	if kcs[0].Title != "歩数0" || kcs[0].NumValue.String() != "1000" {
		t.Errorf("KC = %+v, want 歩数0/1000", kcs[0])
	}
}

// TestPluginAttachedAdapters_DerivedIDsAreStable は、
// 索引を作り直しても付随データのIDが変わらないことを確認する。
// IDが揺れると、ユーザがgkill側で消したタグが復活する。
func TestPluginAttachedAdapters_DerivedIDsAreStable(t *testing.T) {
	fake := newFakePluginRepositoryWithProvides(t, behaviorTyped, allTypedProvides)
	ctx := context.Background()
	adapters := NewPluginTypedRepositories(fake.rep)

	if err := fake.rep.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache: %v", err)
	}
	first, err := adapters.Tag.GetTagsByTargetID(ctx, "typed-0")
	if err != nil {
		t.Fatalf("GetTagsByTargetID: %v", err)
	}
	if len(first) != 2 {
		t.Fatalf("タグ = %d件, want 2", len(first))
	}

	// 最短間隔を無視して作り直す
	fake.rep.TypedIndex().Invalidate()
	if err := fake.rep.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache(2回目): %v", err)
	}
	second, err := adapters.Tag.GetTagsByTargetID(ctx, "typed-0")
	if err != nil {
		t.Fatalf("GetTagsByTargetID(2回目): %v", err)
	}

	firstIDs := map[string]string{}
	for _, tag := range first {
		firstIDs[tag.Tag] = tag.ID
	}
	for _, tag := range second {
		if firstIDs[tag.Tag] != tag.ID {
			t.Errorf("タグ %q のIDが %q → %q に変わった", tag.Tag, firstIDs[tag.Tag], tag.ID)
		}
	}
}

// TestPluginAttachedAdapters_TagNamesAreListed は、プラグインのタグが
// タグ一覧に載ることを確認する。
// これが載らないと、rykvの既定の絞り込み「タグ無し」から漏れて何も表示されなくなる。
func TestPluginAttachedAdapters_TagNamesAreListed(t *testing.T) {
	fake := newFakePluginRepositoryWithProvides(t, behaviorTyped, allTypedProvides)
	ctx := context.Background()
	adapters := NewPluginTypedRepositories(fake.rep)
	if err := fake.rep.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache: %v", err)
	}

	names, err := adapters.Tag.GetAllTagNames(ctx)
	if err != nil {
		t.Fatalf("GetAllTagNames: %v", err)
	}
	found := false
	for _, name := range names {
		if name == "fitbit" {
			found = true
		}
	}
	if !found {
		t.Errorf("タグ一覧 = %v, want fitbit を含む", names)
	}
}

// TestPluginTypedAdapters_LatestDataAddress は、
// 付随データはアドレス表を返し、型別データは返さないことを確認する。
//
// 型別が返すと replaceLatestKyouInfos の対象になり、
// UpdateTimeが揺れた瞬間にレコードごと検索結果から消える。
// 付随が返さないと、--cache_in_memory=false でタグ・テキストが全部落ちる。
func TestPluginTypedAdapters_LatestDataAddress(t *testing.T) {
	fake := newFakePluginRepositoryWithProvides(t, behaviorTyped, allTypedProvides)
	ctx := context.Background()
	adapters := NewPluginTypedRepositories(fake.rep)
	if err := fake.rep.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache: %v", err)
	}

	kcAddresses, err := adapters.KC.GetLatestDataRepositoryAddress(ctx, false)
	if err != nil {
		t.Fatalf("KC.GetLatestDataRepositoryAddress: %v", err)
	}
	if len(kcAddresses) != 0 {
		t.Errorf("型別アダプタのアドレス = %d件, want 0", len(kcAddresses))
	}

	tagAddresses, err := adapters.Tag.GetLatestDataRepositoryAddress(ctx, false)
	if err != nil {
		t.Fatalf("Tag.GetLatestDataRepositoryAddress: %v", err)
	}
	if len(tagAddresses) == 0 {
		t.Error("付随アダプタのアドレスが0件。--cache_in_memory=false でタグが全部落ちる")
	}
	for _, address := range tagAddresses {
		if address.TargetIDInData == nil || *address.TargetIDInData == "" {
			t.Errorf("TargetIDInData が空: %+v", address)
		}
	}
}

// TestPluginTypedAdapters_WritesReturnError は、書き込みが必ずエラーになることを確認する。
func TestPluginTypedAdapters_WritesReturnError(t *testing.T) {
	fake := newFakePluginRepositoryWithProvides(t, behaviorTyped, allTypedProvides)
	ctx := context.Background()
	adapters := NewPluginTypedRepositories(fake.rep)

	if err := adapters.KC.AddKCInfo(ctx, KC{}); err == nil {
		t.Error("AddKCInfo がエラーを返さない")
	}
	if err := adapters.Tag.AddTagInfo(ctx, Tag{}); err == nil {
		t.Error("AddTagInfo がエラーを返さない")
	}
	if err := adapters.Text.AddTextInfo(ctx, Text{}); err == nil {
		t.Error("AddTextInfo がエラーを返さない")
	}
	if err := adapters.Notification.AddNotificationInfo(ctx, Notification{}); err == nil {
		t.Error("AddNotificationInfo がエラーを返さない")
	}
}

// TestPluginTypedAdapters_CloseIsNoop は、アダプタのCloseがプロセスを閉じないことを確認する。
// 閉じてしまうと GkillRepositories.Close が種別ぶん close を送り、
// そのたびに実行スロットを最大30秒待つことになる。
func TestPluginTypedAdapters_CloseIsNoop(t *testing.T) {
	fake := newFakePluginRepositoryWithProvides(t, behaviorTyped, allTypedProvides)
	ctx := context.Background()
	adapters := NewPluginTypedRepositories(fake.rep)
	if err := fake.rep.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache: %v", err)
	}

	if err := adapters.KC.Close(ctx); err != nil {
		t.Fatalf("KC.Close: %v", err)
	}
	if err := adapters.Tag.Close(ctx); err != nil {
		t.Fatalf("Tag.Close: %v", err)
	}
	if !fake.rep.IsAlive(ctx) {
		t.Error("アダプタのCloseでプラグインプロセスが閉じられている")
	}
}

// TestPluginTypedRepositories_NoProvidesRegistersNothing は、
// providesを書いていないプラグインではアダプタが1つも作られず、
// UpdateCacheがプラグインを叩かないことを確認する（既存プラグインの後方互換）。
func TestPluginTypedRepositories_NoProvidesRegistersNothing(t *testing.T) {
	fake := newFakePluginRepository(t, behaviorTyped)
	ctx := context.Background()

	adapters := NewPluginTypedRepositories(fake.rep)
	if adapters.KC != nil || adapters.Tag != nil || adapters.Text != nil ||
		adapters.Notification != nil || adapters.Kmemo != nil || adapters.Mi != nil {
		t.Fatalf("providesが空なのにアダプタが作られている: %+v", adapters)
	}
	if fake.rep.TypedIndex() != nil {
		t.Error("providesが空なのに索引が作られている")
	}
	if err := fake.rep.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache: %v", err)
	}
	if count := countPluginCommands(t, fake.statePath, "find_kyous"); count != 0 {
		t.Errorf("providesが空なのに UpdateCache が find_kyous を %d回 投げた", count)
	}
}

// TestPluginKCAdapter_FindKCFiltersByWord は、KCの検索対象列がTITLEであることを確認する。
func TestPluginKCAdapter_FindKCFiltersByWord(t *testing.T) {
	fake := newFakePluginRepositoryWithProvides(t, behaviorTyped, allTypedProvides)
	ctx := context.Background()
	adapters := NewPluginTypedRepositories(fake.rep)
	if err := fake.rep.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache: %v", err)
	}

	all, err := adapters.KC.FindKC(ctx, nil)
	if err != nil {
		t.Fatalf("FindKC: %v", err)
	}
	if len(all) != fakeTypedKyouCount {
		t.Fatalf("FindKC(条件なし) = %d件, want %d", len(all), fakeTypedKyouCount)
	}

	query := &find.FindQuery{Words: []string{"歩数1"}}
	filtered, err := adapters.KC.FindKC(ctx, query)
	if err != nil {
		t.Fatalf("FindKC(ワード): %v", err)
	}
	if len(filtered) != 1 || filtered[0].Title != "歩数1" {
		t.Errorf("FindKC(歩数1) = %+v, want 1件（歩数1）", filtered)
	}
}
