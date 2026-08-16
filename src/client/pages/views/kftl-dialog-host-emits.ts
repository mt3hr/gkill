'use strict'

import type { KFTLViewEmits } from "./kftl-view-emits"

/**
 * ホストが開いているメモ帳ウィンドウのイベントをそのまま上へ流す。
 *
 * 中身は `KFTLDialogEmits` から `closed` を除いたもの（＝ `KFTLViewEmits` と同じ）。
 * `closed` はホストが自分で受けて一覧から外すので、外へは出さない
 */
export type KFTLDialogHostEmits = KFTLViewEmits
