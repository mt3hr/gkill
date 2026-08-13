'use strict'

/**
 * 相関グラフのヒートマップのトラック幅。
 *
 * `use-dnote-correlation-graph-view.ts` の `matrix_style` が `grid-template-columns` を
 * 組み立て、`dnote-correlation-graph-view.vue` の
 * `.correlation_matrix { gap: 1px }` と `.matrix_cell { padding: 2px; border: 1px }` が
 * 同じ計算に乗っている。**片方だけ変えると幅の見積もりが崩れるので必ず同時に直すこと。**
 *
 * セルは `minmax(CELL_MIN_WIDTH, 1fr)` で、折れ線・棒グラフと同じく列の幅いっぱいまで伸びる。
 * 上限を付けると広い画面で行列だけが小さく残って見にくいため。
 *
 * **`1fr` を使うには `.correlation_matrix_scroll { contain: inline-size }` が要る。**
 * `1fr` は柔軟トラックなので、そのままだと max-content（＝各列が最長の指標名の幅）が
 * 祖先の intrinsic sizing へ伝わる。rykv の Dnote 列は `width: fit-content` なので、
 * それを受けて列ごと広がってしまう（指標7個で約1400pxになったのが元の不具合）。
 * `contain: inline-size` は「中身は横幅の計算に関与しない」を意味するので、
 * 行列は親から与えられた幅を埋めるだけになり、伝播が止まる。片方だけ消さないこと。
 *
 * 最小必要幅 = ROW_HEADER_WIDTH + N * CELL_MIN_WIDTH + (N + 1) * gap(1px)
 *   76 + 41N → N=7 で 363px、N=10 で 486px
 * これを下回る幅しか無いときは `overflow-x: auto` が行列だけを横スクロールさせる。
 *
 * セルの内寸は CELL_MIN_WIDTH - padding 2px - border 2px = 36px。
 * 係数 "-0.62" が 0.8rem で約28px、"n=1234" が 0.62rem で約33px なので、
 * 一番狭まったときでも両方入る。フォントを大きくするなら CELL_MIN_WIDTH も上げること。
 */
export const CORRELATION_MATRIX_ROW_HEADER_WIDTH = 76
export const CORRELATION_MATRIX_CELL_MIN_WIDTH = 40

/** `.correlation_matrix { gap }` と同じ値。上の計算式が前提にしている */
export const CORRELATION_MATRIX_GAP = 1

/**
 * build_correlation_matrix_columns は指標数から `grid-template-columns` を組み立てる。
 *
 * 指標が0件のときに `repeat(0, ...)` を出すと不正なCSSになり宣言ごと落ちるので、
 * 最低1列は引く。
 */
export function build_correlation_matrix_columns(metric_count: number): string {
    const columns = Math.max(1, metric_count)
    return `${CORRELATION_MATRIX_ROW_HEADER_WIDTH}px repeat(${columns}, minmax(${CORRELATION_MATRIX_CELL_MIN_WIDTH}px, 1fr))`
}

/** correlation_matrix_min_width は指標数に対する行列の最小必要幅（px）を返す */
export function correlation_matrix_min_width(metric_count: number): number {
    const columns = Math.max(1, metric_count)
    return CORRELATION_MATRIX_ROW_HEADER_WIDTH
        + columns * CORRELATION_MATRIX_CELL_MIN_WIDTH
        + (columns + 1) * CORRELATION_MATRIX_GAP
}
