package gkill_server_api

import (
	"os"
	"strings"
	"testing"
)

// HandleAddURLog の外向き取得の抑止（SkipFetchMetadata / SkipFetchFavicon）は、
// FillURLogFieldSkipping の第3・第4引数として渡る1行の配線だけで機能している。
// 両方とも bool なので第3・第4引数を入れ替えてもコンパイルは通り、既存テストも
// 全部緑のまま —— reps 層のテストはハンドラを通らないため、配線が外れても
// 目の前ではエラーにならない。実HTTP取得で確かめるテストは safefetch の
// SSRF 対策（private宛の接続拒否）と干渉するため置けない。
//
// そこでソースを走査して、呼び出しの引数列を機械的に固定する。

// urlogSkipWiringArgs は handle_add_urlog.go の呼び出し行に無ければならない抑止フラグの並び。
// 第3引数=メタデータ（Title・Description・Thumbnail）取得の抑止、第4引数=favicon取得の抑止。
const urlogSkipWiringArgs = "request.SkipFetchMetadata, request.SkipFetchFavicon"

// TestHandleAddURLogSkipWiring は、handle_add_urlog.go の FillURLogFieldSkipping 呼び出しが
// 抑止フラグ2つをこの順で渡していることを確認する。
//
// **落ちたら、MCP の fetch_metadata / fetch_favicon 抑止の配線が変わっている。**
// 呼び出しを FillURLogFieldSkipping(serverConfig, applicationConfig,
// request.SkipFetchMetadata, request.SkipFetchFavicon) の形へ戻すこと。
// 補完の呼び出し方を意図して変えたなら、reps 側のシグネチャと合わせて
// このテストも追随させること。
func TestHandleAddURLogSkipWiring(t *testing.T) {
	body, err := os.ReadFile("handle_add_urlog.go")
	if err != nil {
		t.Fatalf("read handle_add_urlog.go: %v", err)
	}

	lines := strings.Split(string(body), "\n")
	found := false
	for i, line := range lines {
		if !strings.Contains(line, "FillURLogFieldSkipping(") {
			continue
		}
		found = true
		if !strings.Contains(line, urlogSkipWiringArgs) {
			t.Errorf("handle_add_urlog.go:%d の FillURLogFieldSkipping 呼び出しに %q がこの順で無い。"+
				"第3・第4引数は両方 bool なので、入れ替わってもコンパイルも既存テストも通ってしまう",
				i+1, urlogSkipWiringArgs)
		}
	}
	if !found {
		t.Error("handle_add_urlog.go に FillURLogFieldSkipping( の呼び出しが無い。" +
			"外向き取得の抑止（SkipFetchMetadata / SkipFetchFavicon）が丸ごと外れている可能性がある。" +
			"補完の呼び出し方を変えたなら、このテストも追随させること")
	}
}
