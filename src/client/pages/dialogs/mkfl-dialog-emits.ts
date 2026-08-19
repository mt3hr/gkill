'use strict'

import type { MKFLViewEmits } from "@/pages/views/mkfl-view-emits"

/**
 * 打刻メモ帳ダイアログのイベント。
 *
 * 中身は MKFLView そのものなので、イベントも同じ。
 * 以前は MKFLViewEmits を丸ごと写していて、片方だけ直すと静かにずれる状態だった。
 */
export type MKFLDialogEmits = MKFLViewEmits
