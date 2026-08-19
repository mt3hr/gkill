'use strict'

import type { KyouViewPropsBase } from "../views/kyou-view-props-base"

// Kyou 1件を包むだけのダイアログが共通で使う props。
// もとは `edit-lantana-dialog-props.ts` の `EditLantanaDialogProps` という名前で、
// 中身が Lantana と何の関係もないのに add-kc / add-time-is / add-ur-log からも
// import されていた（コピー元の名前がそのまま残っていた）。
//
// `kyou-dialog-props.ts` の `KyouDialogProps` とは別物。あちらは KyouDialog 専用で、
// `show_timeis_plaing_end_button` / `is_readonly_mi_check` を足したもの。
export type KyouViewDialogProps = KyouViewPropsBase
