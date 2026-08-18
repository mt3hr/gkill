package reps

import (
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
