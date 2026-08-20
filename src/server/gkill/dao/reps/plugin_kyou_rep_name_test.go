package reps

import (
	"context"
	"log/slog"
	"testing"

	"github.com/mt3hr/gkill/src/server/gkill/api/gkill_plugin"
)

// プラグインが rep_name を返さなくても Kyou.RepName が manifest の rep_name になること。
//
// rep名での絞り込みは Kyou.RepName で行う(find_filter.go の findKyous)。
// ここが空のままだと、reps を必ず送るGUIの通常検索から
// **そのプラグインの記録が丸ごと消える**。エラーも警告も出ないので気付けない。
//
// 空でない不一致は上書きしない。Kyou.rep_name はAPI応答にも出ていて、
// クライアントのコンテキストメニューや get_kyou_histories_by_rep_name が乗っているため。
func TestConvertPluginKyouToKyou_RepName(t *testing.T) {
	for _, c := range []struct {
		name        string
		pluginValue string
		want        string
	}{
		{name: "空ならmanifestの名前で埋める", pluginValue: "", want: "gkill_plugin_example"},
		{name: "申告があればそのまま", pluginValue: "gkill_plugin_example", want: "gkill_plugin_example"},
		{name: "食い違っていても上書きしない", pluginValue: "别の名前", want: "别の名前"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := convertPluginKyouToKyou(gkill_plugin.PluginKyou{
				ID:      "kyou-1",
				RepName: c.pluginValue,
			}, "gkill_plugin_example")
			if got.RepName != c.want {
				t.Errorf("RepName = %q, want %q", got.RepName, c.want)
			}
		})
	}
}

// countingLogHandler は Warn 以上のレコード数を数えるだけの slog.Handler。
type countingLogHandler struct {
	count *int
	msg   string
}

func (h *countingLogHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (h *countingLogHandler) Handle(_ context.Context, r slog.Record) error {
	if r.Message == h.msg {
		*h.count++
	}
	return nil
}
func (h *countingLogHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *countingLogHandler) WithGroup(_ string) slog.Handler      { return h }

// rep_name の不一致警告は組み合わせごとに**1回だけ**であること。
//
// 不一致は「そのプラグインが返す全レコード」で起きるので、素直に出すと
// 実データで数十万行のログになる（それ自体が検索を遅くする）。
// 一度きりにしているのが warnedPluginRepNameMismatches の存在理由なので、
// sync.Map を外しても他のテストは全部緑のまま通ってしまう。ここで固定する。
func TestWarnPluginRepNameMismatchOnce_WarnsOncePerPair(t *testing.T) {
	const manifestRepName = "gkill_plugin_warn_once_test"
	const actualRepName = "declared_by_plugin"
	const otherActualRepName = "declared_by_plugin_2"

	// パッケージ変数なので、他のテストの実行順に左右されないよう先に消しておく
	warnedPluginRepNameMismatches.Delete(manifestRepName + " -> " + actualRepName)
	warnedPluginRepNameMismatches.Delete(manifestRepName + " -> " + otherActualRepName)
	t.Cleanup(func() {
		warnedPluginRepNameMismatches.Delete(manifestRepName + " -> " + actualRepName)
		warnedPluginRepNameMismatches.Delete(manifestRepName + " -> " + otherActualRepName)
	})

	count := 0
	original := slog.Default()
	slog.SetDefault(slog.New(&countingLogHandler{count: &count, msg: "plugin rep_name mismatch"}))
	t.Cleanup(func() { slog.SetDefault(original) })

	for range 3 {
		convertPluginKyouToKyou(gkill_plugin.PluginKyou{
			ID:      "kyou-1",
			RepName: actualRepName,
		}, manifestRepName)
	}
	if count != 1 {
		t.Fatalf("同じ組み合わせで %d 回警告した, want 1。"+
			"レコードごとに出すと実データで数十万行になる", count)
	}

	// 別の組み合わせは別枠で1回出る（不一致の相手が変わったら知りたい）
	convertPluginKyouToKyou(gkill_plugin.PluginKyou{
		ID:      "kyou-2",
		RepName: otherActualRepName,
	}, manifestRepName)
	if count != 2 {
		t.Errorf("組み合わせが違えば別枠で警告するはず: count = %d, want 2", count)
	}
}
