package gkill_server_api

// リポスト（ReKyou / MiReKyou）が指している記録そのものを焼き込む。
//
// **参照先が焼かれていないと、出力先ではリポストの行が空で出る。**
// 画面はリポストを描くときに `get_kyou(target_id)` を引くが、出力 HTML の API は
// 焼き込んだ配列しか見ないので、参照先が入っていなければ何も返らない
// （例外もエラーも出ないので、見た目には「リポストが消えている」としか分からない）。
//
// **共有（/api/get_shared_kyous）ではこれをしない。** 共有は「検索結果だけを渡す」のが
// 約束で、参照先まで広げると共有していない記録の本文が相手に渡る。
// 出力は本人が自分のために作るファイルなので、参照先まで含める。

import (
	"context"

	"github.com/mt3hr/gkill/src/server/gkill/api/find"
	"github.com/mt3hr/gkill/src/server/gkill/api/message"
)

// exportReKyouTargetMaxRounds はリポストを辿る回数の上限。
// リポストのリポストもあるので1回では足りないが、無制限にすると
// 参照が輪になっているときに止まらない。
const exportReKyouTargetMaxRounds = 3

// collectExportReKyouTargets はリポストの参照先を焼き込みへ足す。
func (g *GkillServerAPI) collectExportReKyouTargets(
	ctx context.Context,
	opts *kyouBundleOptions,
	merged *exportMerger,
	payload *exportedKyouPayload,
) *message.GkillError {
	known := map[string]bool{}
	for _, kyou := range payload.Kyous {
		known[kyou.ID] = true
	}

	for round := 0; round < exportReKyouTargetMaxRounds; round++ {
		wanted := map[string]bool{}
		for _, reKyou := range payload.ReKyous {
			if reKyou.TargetID != "" && !known[reKyou.TargetID] {
				wanted[reKyou.TargetID] = true
			}
		}
		for _, miReKyou := range payload.MiReKyous {
			if miReKyou.TargetID != "" && !known[miReKyou.TargetID] {
				wanted[miReKyou.TargetID] = true
			}
		}
		if len(wanted) == 0 {
			return nil
		}

		ids := make([]string, 0, len(wanted))
		for id := range wanted {
			ids = append(ids, id)
			known[id] = true
		}

		// ID だけで引く。ほかの欄は nil のまま＝「絞らない」
		// （非 nil の空配列にすると 0 件指定になって1件も返らない）
		targetOpts := *opts
		targetOpts.FindQuery = &find.FindQuery{IDs: ids}

		bundle, gkillErr := g.collectKyouBundle(ctx, &targetOpts)
		if gkillErr != nil {
			return gkillErr
		}
		merged.add(bundle)
		merged.writeTo(payload)

		for _, kyou := range payload.Kyous {
			known[kyou.ID] = true
		}
	}
	return nil
}
