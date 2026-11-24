package reps

import (
	"context"
	"encoding/json"
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
// 索引の構築ぶんしか飛ばない。ここが壊れると、Dnoteで万件規模の一覧を舐めた瞬間に
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

// TestPluginKCAdapter_FindKCFiltersByWord は、KCの検索対象列が native と同じ TITLE + NUM_VALUE であることを確認する。
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

	// 数値（NUM_VALUE）でも当たる
	byValue, err := adapters.KC.FindKC(ctx, &find.FindQuery{Words: []string{"1002"}})
	if err != nil {
		t.Fatalf("FindKC(数値): %v", err)
	}
	if len(byValue) != 1 || byValue[0].NumValue.String() != "1002" {
		t.Errorf("FindKC(1002) = %+v, want 1件（NUM_VALUE=1002）", byValue)
	}
}

// findKyous（型別アダプタの Kyou 検索）もワードで絞る。
// 以前はここでワードを見ておらず、rep_types 指定や ForMi で型別アダプタが選ばれると
// 語に関係なくプラグインの記録が全件当たっていた。
func TestPluginTypedAdapter_FindKyousFiltersByWord(t *testing.T) {
	fake := newFakePluginRepositoryWithProvides(t, behaviorTyped, allTypedProvides)
	ctx := context.Background()
	adapters := NewPluginTypedRepositories(fake.rep)
	if err := fake.rep.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache: %v", err)
	}

	count := func(query *find.FindQuery) map[string]bool {
		t.Helper()
		got, err := adapters.KC.FindKyous(ctx, query)
		if err != nil {
			t.Fatalf("FindKyous: %v", err)
		}
		ids := map[string]bool{}
		for _, kyous := range got {
			for _, kyou := range kyous {
				ids[kyou.ID] = true
			}
		}
		return ids
	}

	if ids := count(&find.FindQuery{Words: []string{"歩数1"}}); len(ids) != 1 || !ids["typed-1"] {
		t.Errorf("FindKyous(歩数1) = %v, want typed-1 だけ", ids)
	}
	if ids := count(&find.FindQuery{Words: []string{}, NotWords: []string{"歩数1"}}); len(ids) != fakeTypedKyouCount-1 || ids["typed-1"] {
		t.Errorf("FindKyous(-歩数1) = %v, want typed-1 以外", ids)
	}
	// ID は前方一致
	if ids := count(&find.FindQuery{Words: []string{"typed-2"}}); len(ids) != 1 || !ids["typed-2"] {
		t.Errorf("FindKyous(typed-2) = %v, want ID 前方一致で typed-2 だけ", ids)
	}
	if ids := count(&find.FindQuery{Words: []string{"ped-2"}}); len(ids) != 0 {
		t.Errorf("FindKyous(ped-2) = %v, want ID の途中の部分一致では当たらない", ids)
	}
	// 語なし（非nil空）は素通し
	if ids := count(&find.FindQuery{Words: []string{}, NotWords: []string{}}); len(ids) != fakeTypedKyouCount {
		t.Errorf("FindKyous(語なし) = %v, want 全件", ids)
	}
}

// mixedProvides は TimeIs / URLog / Kmemo とタグを宣言した provides。
var mixedProvides = []gkill_plugin.PluginProvidedKind{
	gkill_plugin.PluginProvidesTimeIs,
	gkill_plugin.PluginProvidesURLog,
	gkill_plugin.PluginProvidesKmemo,
	gkill_plugin.PluginProvidesTag,
}

// adapterKyouIDs はアダプタの FindKyous が返した ID の集合を返す。
func adapterKyouIDs(t *testing.T, find func(context.Context, *find.FindQuery) (map[string][]Kyou, error), query *find.FindQuery) map[string]bool {
	t.Helper()
	got, err := find(context.Background(), query)
	if err != nil {
		t.Fatalf("FindKyous: %v", err)
	}
	ids := map[string]bool{}
	for _, kyous := range got {
		for _, kyou := range kyous {
			ids[kyou.ID] = true
		}
	}
	return ids
}

// 1本のプラグインが TimeIs / URLog / Kmemo を混ぜて返しても、
// 種類ごとのアダプタが自分の種類の記録だけを返す（ADR-0317）。
func TestPluginTypedAdapters_MixedKindsAreRoutedPerKyou(t *testing.T) {
	fake := newFakePluginRepositoryWithProvides(t, behaviorTypedMixed, mixedProvides)
	ctx := context.Background()
	adapters := NewPluginTypedRepositories(fake.rep)
	if adapters.TimeIs == nil || adapters.URLog == nil || adapters.Kmemo == nil || adapters.Tag == nil {
		t.Fatalf("providesに書いた種別のアダプタが作られていない: %+v", adapters)
	}
	if err := fake.rep.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache: %v", err)
	}

	if ids := adapterKyouIDs(t, adapters.TimeIs.FindKyous, nil); len(ids) != 2 || !ids["mixed-timeis-closed"] || !ids["mixed-timeis-open"] {
		t.Errorf("TimeIs.FindKyous = %v, want TimeIs の2件だけ", ids)
	}
	if ids := adapterKyouIDs(t, adapters.URLog.FindKyous, nil); len(ids) != 1 || !ids["mixed-urlog"] {
		t.Errorf("URLog.FindKyous = %v, want mixed-urlog だけ", ids)
	}
	if ids := adapterKyouIDs(t, adapters.Kmemo.FindKyous, nil); len(ids) != 1 || !ids["mixed-kmemo"] {
		t.Errorf("Kmemo.FindKyous = %v, want mixed-kmemo だけ", ids)
	}
	kmemo, err := adapters.Kmemo.GetKyou(ctx, "mixed-kmemo", nil)
	if err != nil || kmemo == nil || kmemo.DataType != "kmemo" {
		t.Errorf("Kmemo.GetKyou = %+v, %v, want data_type kmemo", kmemo, err)
	}
	if got, err := adapters.TimeIs.GetKyou(ctx, "mixed-kmemo", nil); err != nil || got != nil {
		t.Errorf("TimeIs.GetKyou(Kmemo の ID) = %+v, %v, want nil", got, err)
	}
}

// プラグインの TimeIs も playing_time（その時刻に実行中か）で絞る。
// 以前は絞っておらず、プラグインの TimeIs が全件「実行中」として返っていた（ADR-0317）。
func TestPluginTimeIsAdapter_FindKyousFiltersByPlayingTime(t *testing.T) {
	fake := newFakePluginRepositoryWithProvides(t, behaviorTypedMixed, mixedProvides)
	ctx := context.Background()
	adapters := NewPluginTypedRepositories(fake.rep)
	if err := fake.rep.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache: %v", err)
	}
	base := fakeMixedBaseTime
	cases := []struct {
		name string
		at   time.Time
		want []string
	}{
		{"開始より前", base.Add(-time.Minute), nil},
		{"開始ちょうど", base, []string{"mixed-timeis-closed"}},
		{"区間の中", base.Add(30 * time.Minute), []string{"mixed-timeis-closed"}},
		{"終了ちょうど", base.Add(time.Hour), []string{"mixed-timeis-closed"}},
		{"区間のあと・終了なしの開始前", base.Add(90 * time.Minute), nil},
		{"終了なしは開始以降ずっと", base.Add(5 * time.Hour), []string{"mixed-timeis-open"}},
	}
	for _, c := range cases {
		at := c.at
		ids := adapterKyouIDs(t, adapters.TimeIs.FindKyous, &find.FindQuery{PlayingTime: &at})
		if len(ids) != len(c.want) {
			t.Errorf("%s: FindKyous(playing %v) = %v, want %v", c.name, at, ids, c.want)
			continue
		}
		for _, id := range c.want {
			if !ids[id] {
				t.Errorf("%s: FindKyous(playing %v) = %v, want %v", c.name, at, ids, c.want)
			}
		}
		// FindTimeIs（メモ帳の打刻終了・共有・HTML 出力が直に呼ぶ経路）も同じ条件で絞る。
		// 絞らないと終了済みのプラグインの打刻が終了の対象に選ばれ、プラグインの ID へ native の版が書かれる。
		timeiss, err := adapters.TimeIs.FindTimeIs(context.Background(), &find.FindQuery{PlayingTime: &at})
		if err != nil {
			t.Fatalf("%s: FindTimeIs: %v", c.name, err)
		}
		timeisIDs := map[string]bool{}
		for _, timeis := range timeiss {
			timeisIDs[timeis.ID] = true
		}
		if len(timeisIDs) != len(c.want) {
			t.Errorf("%s: FindTimeIs(playing %v) = %v, want %v", c.name, at, timeisIDs, c.want)
			continue
		}
		for _, id := range c.want {
			if !timeisIDs[id] {
				t.Errorf("%s: FindTimeIs(playing %v) = %v, want %v", c.name, at, timeisIDs, c.want)
			}
		}
	}
}

// 型別アダプタ（索引）の経路でも、TimeIs の終了の Kyou は IncludeEndTimeIs のときだけ返す。
// 索引は終了の行も持っている（indexFetchAll が IncludeEndTimeIs を立てて取る）。ADR-0318。
func TestPluginTimeIsAdapter_FindKyousIncludesEndOnlyWhenAsked(t *testing.T) {
	fake := newFakePluginRepositoryWithProvides(t, behaviorTypedMixed, mixedProvides)
	ctx := context.Background()
	adapters := NewPluginTypedRepositories(fake.rep)
	if err := fake.rep.UpdateCache(ctx); err != nil {
		t.Fatalf("UpdateCache: %v", err)
	}
	countDataTypes := func(query *find.FindQuery) map[string]int {
		t.Helper()
		got, err := adapters.TimeIs.FindKyous(ctx, query)
		if err != nil {
			t.Fatalf("FindKyous: %v", err)
		}
		counts := map[string]int{}
		for _, kyous := range got {
			for _, kyou := range kyous {
				counts[kyou.DataType]++
			}
		}
		return counts
	}
	if counts := countDataTypes(&find.FindQuery{}); counts["timeis_start"] != 2 || counts["timeis_end"] != 0 {
		t.Errorf("IncludeEndTimeIs=false: %v, want timeis_start 2件だけ", counts)
	}
	if counts := countDataTypes(&find.FindQuery{IncludeEndTimeIs: true}); counts["timeis_start"] != 2 || counts["timeis_end"] != 1 {
		t.Errorf("IncludeEndTimeIs=true: %v, want timeis_start 2件と timeis_end 1件（終了なしは終了の Kyou を持たない）", counts)
	}
}

// Lantana は native と同じく気分値（MOOD）を文字列として照合する。
// 以前は「ワード条件が有効なら0件」で、除外語だけの検索でも消えていた（native は残る）。
func TestPluginTypedFindWordText_Lantana(t *testing.T) {
	record := &pluginTypedRecord{ID: "lantana-1", Lantana: &Lantana{ID: "lantana-1", Mood: 7}}
	if got := pluginTypedFindWordText(record, gkill_plugin.PluginProvidesLantana); got != "7" {
		t.Fatalf("Lantana の検索対象テキストは気分値のはず: got %q", got)
	}
	if !pluginMatchWords(pluginTypedFindWordText(record, gkill_plugin.PluginProvidesLantana), record.ID, &find.FindQuery{Words: []string{"7"}}) {
		t.Errorf("気分7は `7` で当たるべき")
	}
	if pluginMatchWords(pluginTypedFindWordText(record, gkill_plugin.PluginProvidesLantana), record.ID, &find.FindQuery{Words: []string{"8"}}) {
		t.Errorf("気分7は `8` で当たってはいけない")
	}
	if pluginMatchWords(pluginTypedFindWordText(record, gkill_plugin.PluginProvidesLantana), record.ID, &find.FindQuery{Words: []string{}, NotWords: []string{"7"}}) {
		t.Errorf("気分7は `-7` で消えるべき")
	}
	if !pluginMatchWords(pluginTypedFindWordText(record, gkill_plugin.PluginProvidesLantana), record.ID, &find.FindQuery{Words: []string{}, NotWords: []string{"8"}}) {
		t.Errorf("気分7は `-8` で残るべき")
	}
	// 他の型の列も native と同じ
	mi := &pluginTypedRecord{ID: "mi-1", Mi: &Mi{Title: "買い物", BoardName: "家事"}}
	if !pluginMatchWords(pluginTypedFindWordText(mi, gkill_plugin.PluginProvidesMi), mi.ID, &find.FindQuery{Words: []string{"家事"}}) {
		t.Errorf("Mi は BOARD_NAME でも当たるべき")
	}
	nlog := &pluginTypedRecord{ID: "nlog-1", Nlog: &Nlog{Title: "昼食", Shop: "食堂", Amount: json.Number("1500")}}
	if !pluginMatchWords(pluginTypedFindWordText(nlog, gkill_plugin.PluginProvidesNlog), nlog.ID, &find.FindQuery{Words: []string{"1500"}}) {
		t.Errorf("Nlog は AMOUNT でも当たるべき")
	}
	// 実データを持たない種別は空文字
	if got := pluginTypedFindWordText(record, gkill_plugin.PluginProvidesKmemo); got != "" {
		t.Errorf("持たない種別は空文字のはず: got %q", got)
	}
}

// ---------------------------------------------------------------------------
// 2巡目の指摘: 索引が ok:false を返すだけで、
// 「一度も構築していない」のか「構築に失敗した」のかが区別できず、
// 失敗の理由もどこにも出てこなかった。
// ---------------------------------------------------------------------------

// stubIndexSource は索引の単体テスト用。fetchErr が非nilなら構築が失敗する。
type stubIndexSource struct {
	fetchErr error
	kyous    []gkill_plugin.PluginKyou
}

func (s *stubIndexSource) indexRepName() string               { return "StubRep" }
func (s *stubIndexSource) indexIsDeclaredRepName(string) bool { return false }
func (s *stubIndexSource) indexPluginName() string            { return "stub_plugin" }
func (s *stubIndexSource) indexProvidedKinds() map[gkill_plugin.PluginProvidedKind]struct{} {
	return map[gkill_plugin.PluginProvidedKind]struct{}{}
}
func (s *stubIndexSource) indexFetchAll(_ context.Context) ([]gkill_plugin.PluginKyou, error) {
	if s.fetchErr != nil {
		return nil, s.fetchErr
	}
	return s.kyous, nil
}

func TestPluginTypedIndexStats_NeverBuiltIsDistinctFromFailed(t *testing.T) {
	index := newPluginTypedIndex(&stubIndexSource{})

	stats := index.Stats()
	if stats.OK {
		t.Errorf("一度も構築していないのに OK=true")
	}
	if stats.State != PluginTypedIndexStateNeverBuilt {
		t.Errorf("State = %q, want %q", stats.State, PluginTypedIndexStateNeverBuilt)
	}
	if stats.LastBuildError != "" {
		t.Errorf("まだ試していないのに失敗理由がある: %q", stats.LastBuildError)
	}
	if !stats.LastAttemptAt.IsZero() {
		t.Errorf("まだ試していないのに試行時刻がある: %v", stats.LastAttemptAt)
	}
}

func TestPluginTypedIndexStats_FailedCarriesTheReason(t *testing.T) {
	// 以前は kickRebuild が slog へ出して捨てるだけで、APIからは
	// 「ok=false・件数0」しか見えなかった
	source := &stubIndexSource{fetchErr: fmt.Errorf("plugin is busy")}
	index := newPluginTypedIndex(source)

	if err := index.build(context.Background()); err == nil {
		t.Fatal("構築が失敗するはずなのにエラーにならなかった")
	}

	stats := index.Stats()
	if stats.OK {
		t.Errorf("構築に失敗したのに OK=true")
	}
	if stats.State != PluginTypedIndexStateFailed {
		t.Errorf("State = %q, want %q", stats.State, PluginTypedIndexStateFailed)
	}
	if !strings.Contains(stats.LastBuildError, "plugin is busy") {
		t.Errorf("失敗理由が伝わっていない: %q", stats.LastBuildError)
	}
	// バックオフ中は再試行でエラーすら出ないので、試行時刻が無いと
	// 「なぜ何も起きていないのか」が分からない
	if stats.LastAttemptAt.IsZero() {
		t.Errorf("試行時刻が記録されていない")
	}
}

func TestPluginTypedIndexStats_SuccessClearsTheReason(t *testing.T) {
	source := &stubIndexSource{fetchErr: fmt.Errorf("transient failure")}
	index := newPluginTypedIndex(source)
	if err := index.build(context.Background()); err == nil {
		t.Fatal("1回目は失敗するはず")
	}

	// 一時的に混んでいただけなら、次の構築で理由は消える
	source.fetchErr = nil
	if err := index.build(context.Background()); err != nil {
		t.Fatalf("2回目は成功するはず: %v", err)
	}

	stats := index.Stats()
	if !stats.OK {
		t.Errorf("構築に成功したのに OK=false")
	}
	if stats.State != PluginTypedIndexStateOK {
		t.Errorf("State = %q, want %q", stats.State, PluginTypedIndexStateOK)
	}
	if stats.LastBuildError != "" {
		t.Errorf("成功したのに古い失敗理由が残っている: %q", stats.LastBuildError)
	}
}
