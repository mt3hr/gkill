package reps

// プラグイン診断情報（stderrリング / 型別索引の統計）のテスト。
// 「is_alive=true なのに0件」がAPIから診断できなかった問題（外部監査 D1/D2）への回帰。

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/mt3hr/gkill/src/server/gkill/api/gkill_plugin"
)

// リングバッファ: 末尾保持・一周時の行境界・大きすぎる書き込み。
func TestPluginStderrRing(t *testing.T) {
	t.Run("書いた内容がTailで読める", func(t *testing.T) {
		ring := newPluginStderrRing()
		fmt.Fprintf(ring, "build error: conversations.json が見つかりません\n")
		if got := ring.Tail(); !strings.Contains(got, "conversations.json") {
			t.Errorf("Tail = %q, want contains conversations.json", got)
		}
	})

	t.Run("空なら空文字", func(t *testing.T) {
		ring := newPluginStderrRing()
		if got := ring.Tail(); got != "" {
			t.Errorf("Tail = %q, want empty", got)
		}
	})

	t.Run("一周しても末尾が残り、行境界から始まる", func(t *testing.T) {
		ring := newPluginStderrRing()
		for i := range 500 {
			fmt.Fprintf(ring, "line %04d: some plugin output that fills the ring buffer\n", i)
		}
		tail := ring.Tail()
		if !strings.Contains(tail, "line 0499") {
			t.Errorf("最後の行が残っていない: %q", tail[:min(80, len(tail))])
		}
		if strings.Contains(tail, "line 0000") {
			t.Error("古い行が残っている（リングが機能していない）")
		}
		// 行の途中から始まらない（先頭は "line " のはず）
		if !strings.HasPrefix(tail, "line ") {
			t.Errorf("Tailが行境界から始まっていない: %q", tail[:min(40, len(tail))])
		}
	})

	t.Run("リングより大きい1回の書き込みは末尾だけ残る", func(t *testing.T) {
		ring := newPluginStderrRing()
		big := strings.Repeat("x", pluginStderrRingSize*2) + "END"
		if _, err := ring.Write([]byte(big)); err != nil {
			t.Fatalf("Write: %v", err)
		}
		if got := ring.Tail(); !strings.HasSuffix(got, "END") {
			t.Errorf("末尾が残っていない: ...%q", got[max(0, len(got)-10):])
		}
	})
}

// statsTestIndexSource は Stats テスト用の最小 pluginIndexSource。
type statsTestIndexSource struct {
	kyous []gkill_plugin.PluginKyou
}

func (s *statsTestIndexSource) indexRepName() string               { return "StatsTestRep" }
func (s *statsTestIndexSource) indexIsDeclaredRepName(string) bool { return false }
func (s *statsTestIndexSource) indexPluginName() string            { return "stats_test_plugin" }
func (s *statsTestIndexSource) indexProvidedKinds() map[gkill_plugin.PluginProvidedKind]struct{} {
	return map[gkill_plugin.PluginProvidedKind]struct{}{}
}
func (s *statsTestIndexSource) indexFetchAll(ctx context.Context) ([]gkill_plugin.PluginKyou, error) {
	return s.kyous, nil
}

// 型別索引の統計: 未構築は OK=false、構築後は件数と時刻範囲が出る。
func TestPluginTypedIndexStats(t *testing.T) {
	oldest := time.Date(2025, 9, 1, 12, 0, 0, 0, time.Local)
	newest := time.Date(2026, 8, 20, 12, 0, 0, 0, time.Local)
	source := &statsTestIndexSource{
		kyous: []gkill_plugin.PluginKyou{
			{ID: "k1", RepName: "StatsTestRep", DataType: "stats_test", RelatedTime: oldest, UpdateTime: oldest},
			{ID: "k2", RepName: "StatsTestRep", DataType: "stats_test", RelatedTime: newest, UpdateTime: newest},
		},
	}
	index := newPluginTypedIndex(source)

	// 未構築: OK=false（未構築と実データ0件を呼び出し側が区別できる）
	if stats := index.Stats(); stats.OK {
		t.Errorf("未構築なのに OK=true: %+v", stats)
	}

	if err := index.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	stats := index.Stats()
	if !stats.OK {
		t.Fatalf("構築後なのに OK=false: %+v", stats)
	}
	if stats.RecordCount != 2 {
		t.Errorf("RecordCount = %d, want 2", stats.RecordCount)
	}
	if !stats.Oldest.Equal(oldest) || !stats.Newest.Equal(newest) {
		t.Errorf("時刻範囲 = %v..%v, want %v..%v", stats.Oldest, stats.Newest, oldest, newest)
	}
	if stats.Truncated {
		t.Error("切り捨てが起きていないのに Truncated=true")
	}
	if stats.BuiltAt.IsZero() {
		t.Error("BuiltAt がゼロ値")
	}
}
