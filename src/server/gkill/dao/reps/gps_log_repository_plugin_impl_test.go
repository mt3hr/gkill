package reps

import (
	"context"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/gkill_plugin"
)

// gpsLogProvides はGPSログだけを宣言したprovides。
var gpsLogProvides = []gkill_plugin.PluginProvidedKind{gkill_plugin.PluginProvidesGPSLog}

// TestNormalizeGPSLogPeriod は期間指定の正規化の契約を固定する。
// 3実装（GPXディレクトリ・プラグインアダプタ・プラグイン本体）が
// 同じ判定を書き写してドリフトしないよう、ここに1本だけ置く。
func TestNormalizeGPSLogPeriod(t *testing.T) {
	early := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	late := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	if start, end := NormalizeGPSLogPeriod(nil, nil); start != nil || end != nil {
		t.Errorf("両方nil = %v〜%v, want nil〜nil（全件の意味）", start, end)
	}
	if start, end := NormalizeGPSLogPeriod(&early, nil); start == nil || end == nil || !start.Equal(early) || !end.Equal(early) {
		t.Errorf("終了だけnil = %v〜%v, want %v〜%v", start, end, early, early)
	}
	if start, end := NormalizeGPSLogPeriod(nil, &late); start == nil || end == nil || !start.Equal(late) || !end.Equal(late) {
		t.Errorf("開始だけnil = %v〜%v, want %v〜%v", start, end, late, late)
	}
	// 逆順は入れ替える
	if start, end := NormalizeGPSLogPeriod(&late, &early); start == nil || end == nil || !start.Equal(early) || !end.Equal(late) {
		t.Errorf("逆順 = %v〜%v, want %v〜%v", start, end, early, late)
	}
	// 引数のポインタは書き換えない
	argStart, argEnd := late, early
	NormalizeGPSLogPeriod(&argStart, &argEnd)
	if !argStart.Equal(late) || !argEnd.Equal(early) {
		t.Error("引数の時刻が書き換わっている")
	}
}

// TestGetPluginGPSLogs_PagesThrough は、ページングで全点を繋げることを確認する。
// 親のstdioは1レスポンス32MBが上限なので、1回では返しきれない。
func TestGetPluginGPSLogs_PagesThrough(t *testing.T) {
	fake := newFakePluginRepositoryWithProvides(t, behaviorTyped, gpsLogProvides)
	ctx := context.Background()

	gpsLogs, err := fake.rep.GetPluginGPSLogs(ctx, nil, nil)
	if err != nil {
		t.Fatalf("GetPluginGPSLogs: %v", err)
	}
	if len(gpsLogs) != fakeGPSLogTotal {
		t.Fatalf("点数 = %d, want %d", len(gpsLogs), fakeGPSLogTotal)
	}
	// 昇順で、取り違えや重複が無いこと
	for i := 1; i < len(gpsLogs); i++ {
		if gpsLogs[i].RelatedTime.Before(gpsLogs[i-1].RelatedTime) {
			t.Fatalf("%d番目で昇順が崩れている", i)
		}
	}
}

// TestGetPluginGPSLogs_RequiresProvides は、providesにgpslogが無ければ
// エラーになることを確認する。
func TestGetPluginGPSLogs_RequiresProvides(t *testing.T) {
	fake := newFakePluginRepository(t, behaviorTyped)
	if _, err := fake.rep.GetPluginGPSLogs(context.Background(), nil, nil); err == nil {
		t.Error("providesにgpslogが無いのにエラーにならない")
	}
}

// TestGPSLogPluginAdapter_Contract は GPSLogRepository の契約を確認する。
// 期間は両端を含み、nilは解決され、逆順は入れ替わり、0件でもエラーにならない。
func TestGPSLogPluginAdapter_Contract(t *testing.T) {
	fake := newFakePluginRepositoryWithProvides(t, behaviorTyped, gpsLogProvides)
	ctx := context.Background()

	adapter, ok := NewGPSLogPluginRepIfProvided(fake.rep)
	if !ok {
		t.Fatal("providesにgpslogがあるのにアダプタが作られない")
	}

	all, err := adapter.GetAllGPSLogs(ctx)
	if err != nil {
		t.Fatalf("GetAllGPSLogs: %v", err)
	}
	if len(all) != fakeGPSLogTotal {
		t.Fatalf("全件 = %d, want %d", len(all), fakeGPSLogTotal)
	}

	// 両端を含む: 同じ時刻を2つ渡すとちょうどその時刻の点だけが返る
	target := all[10].RelatedTime
	exact, err := adapter.GetGPSLogs(ctx, &target, &target)
	if err != nil {
		t.Fatalf("GetGPSLogs(同一時刻): %v", err)
	}
	if len(exact) != 1 || !exact[0].RelatedTime.Equal(target) {
		t.Errorf("同一時刻の切り出し = %d件, want 1件", len(exact))
	}

	// 逆順で渡しても同じ結果になる
	start, end := all[5].RelatedTime, all[15].RelatedTime
	forward, err := adapter.GetGPSLogs(ctx, &start, &end)
	if err != nil {
		t.Fatalf("GetGPSLogs: %v", err)
	}
	backward, err := adapter.GetGPSLogs(ctx, &end, &start)
	if err != nil {
		t.Fatalf("GetGPSLogs(逆順): %v", err)
	}
	if len(forward) != 11 || len(backward) != len(forward) {
		t.Errorf("期間の切り出し = %d件 / 逆順 %d件, want 11件ずつ", len(forward), len(backward))
	}

	// 範囲外は0件でエラーにしない
	far := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	empty, err := adapter.GetGPSLogs(ctx, &far, &far)
	if err != nil {
		t.Fatalf("範囲外でエラーになった: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("範囲外 = %d件, want 0", len(empty))
	}
}

// TestGPSLogPluginAdapter_CachesSnapshot は、何度引いてもプラグインへの
// get_gps_logs が増えないことを確認する。
// 地図の描画1回ごと・地図フィルタの評価1回ごとに引かれるので、
// 素通しすると列の数だけ行列が伸びて ErrPluginBusy になる。
func TestGPSLogPluginAdapter_CachesSnapshot(t *testing.T) {
	fake := newFakePluginRepositoryWithProvides(t, behaviorTyped, gpsLogProvides)
	ctx := context.Background()
	adapter, _ := NewGPSLogPluginRepIfProvided(fake.rep)

	if _, err := adapter.GetAllGPSLogs(ctx); err != nil {
		t.Fatalf("GetAllGPSLogs: %v", err)
	}
	before := countPluginCommands(t, fake.statePath, "get_gps_logs")
	if before == 0 {
		t.Fatal("get_gps_logs が1回も呼ばれていない")
	}

	for range 100 {
		if _, err := adapter.GetAllGPSLogs(ctx); err != nil {
			t.Fatalf("GetAllGPSLogs: %v", err)
		}
		if _, err := adapter.GetGPSLogs(ctx, nil, nil); err != nil {
			t.Fatalf("GetGPSLogs: %v", err)
		}
	}
	after := countPluginCommands(t, fake.statePath, "get_gps_logs")
	if after != before {
		t.Errorf("200回引いて get_gps_logs が %d回 → %d回 に増えた", before, after)
	}
}

// TestGPSLogPluginAdapter_ReturnedSliceIsAppendSafe は、返したスライスに
// appendしてもスナップショットが壊れないことを確認する。
// 共有スナップショットを返しているので、容量を切っていないと
// 呼び出し側のappendが他の呼び出し側の結果を書き換える。
func TestGPSLogPluginAdapter_ReturnedSliceIsAppendSafe(t *testing.T) {
	fake := newFakePluginRepositoryWithProvides(t, behaviorTyped, gpsLogProvides)
	ctx := context.Background()
	adapter, _ := NewGPSLogPluginRepIfProvided(fake.rep)

	all, err := adapter.GetAllGPSLogs(ctx)
	if err != nil {
		t.Fatalf("GetAllGPSLogs: %v", err)
	}
	start, end := all[0].RelatedTime, all[10].RelatedTime
	part, err := adapter.GetGPSLogs(ctx, &start, &end)
	if err != nil {
		t.Fatalf("GetGPSLogs: %v", err)
	}
	wantNext := all[11]

	// 切り出した結果にappendする（容量が切ってあれば別配列になる）
	part = append(part, GPSLog{Latitude: -1, Longitude: -1})
	_ = part

	again, err := adapter.GetAllGPSLogs(ctx)
	if err != nil {
		t.Fatalf("GetAllGPSLogs(2回目): %v", err)
	}
	if again[11] != wantNext {
		t.Errorf("appendでスナップショットが壊れた: [11] = %+v, want %+v", again[11], wantNext)
	}
}

// TestGPSLogPluginAdapter_IsReadOnly は、アップロード先の候補から外れる目印が
// 付いていることを確認する。
// 付いていないと、認証済みリクエストにrep名を指定するだけで
// プラグインフォルダにGPXを書き込めてしまう。
func TestGPSLogPluginAdapter_IsReadOnly(t *testing.T) {
	fake := newFakePluginRepositoryWithProvides(t, behaviorTyped, gpsLogProvides)
	adapter, _ := NewGPSLogPluginRepIfProvided(fake.rep)

	if _, readOnly := adapter.(ReadOnlyGPSLogRepository); !readOnly {
		t.Error("ReadOnlyGPSLogRepository を実装していない。アップロード先に選べてしまう")
	}
}

// TestGPSLogPluginAdapter_ErrorReturnsEmpty は、プラグインが失敗しても
// エラーではなく空を返すことを確認する。
//
// GPSLogRepositories も find_filter の collectFromRepos も
// 「1つでもrepがエラーを返したら全体を失敗にする」作りなので、
// ここでエラーを返すとプラグインが混んでいるだけで地図も検索も丸ごと落ちる。
func TestGPSLogPluginAdapter_ErrorReturnsEmpty(t *testing.T) {
	// providesにgpslogを書いていないプラグインをアダプタに包む。
	// GetPluginGPSLogs はエラーを返すので、失敗経路をそのまま通せる。
	fake := newFakePluginRepositoryWithProvides(t, behaviorTyped, gpsLogProvides)
	adapter := &gpsLogRepositoryPluginImpl{plugin: &erroringGPSLogPlugin{PluginRepository: fake.rep}}

	ctx := WithFindWarnings(context.Background())
	gpsLogs, err := adapter.GetAllGPSLogs(ctx)
	if err != nil {
		t.Fatalf("プラグイン失敗でエラーを返している（地図も検索も丸ごと落ちる）: %v", err)
	}
	if len(gpsLogs) != 0 {
		t.Errorf("失敗時 = %d件, want 0", len(gpsLogs))
	}
	if warnings := PluginFindWarnings(ctx); len(warnings) == 0 {
		t.Error("失敗が警告として記録されていない（静かな欠落になる）")
	}
}

// erroringGPSLogPlugin は GetPluginGPSLogs だけが必ず失敗するプラグイン。
type erroringGPSLogPlugin struct {
	PluginRepository
}

func (e *erroringGPSLogPlugin) GetPluginGPSLogs(_ context.Context, _ *time.Time, _ *time.Time) ([]GPSLog, error) {
	return nil, context.DeadlineExceeded
}
