/**
 * 相関グラフのヒートマップが、狭い列に収まる寸法から外れていないかを検査する。
 *
 * 以前は `.correlation_matrix { min-width: max-content }` があったせいで、
 * グリッドが max-content 制約で測られ、`minmax(84px, 1fr)` の全列が
 * 「行列中で最も長い見出し」の幅に揃えられていた。指標7個で約1400px になり、
 * rykv では auto table layout の td ごとその幅になって他の列を画面外へ押し出していた。
 *
 * jsdom はレイアウトを行わないので実寸は測れない。
 * 生成される grid-template-columns の文字列と、CSS のソースを機械検査する。
 */
import { existsSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { describe, expect, it } from 'vitest'
import { ref } from 'vue'

import {
    CORRELATION_MATRIX_CELL_MIN_WIDTH,
    CORRELATION_MATRIX_GAP,
    CORRELATION_MATRIX_ROW_HEADER_WIDTH,
    build_correlation_matrix_columns,
    correlation_matrix_min_width,
} from '@/classes/dnote-correlation-matrix-layout'
import { useDnoteCorrelationGraphView } from '@/classes/use-dnote-correlation-graph-view'

function find_repo_root(): string {
    let dir = process.cwd()
    for (let i = 0; i < 10; i++) {
        if (existsSync(join(dir, 'package.json')) && existsSync(join(dir, 'src', 'client'))) {
            return dir
        }
        const parent = dirname(dir)
        if (parent === dir) {
            break
        }
        dir = parent
    }
    throw new Error(`リポジトリルートが見つからない: cwd=${process.cwd()}`)
}

const repo_root = find_repo_root()

/**
 * rykv の Dnote 列の実効幅。
 * 列は400pxで、セクションの padding 4px とスクロールバー最大15pxを引いた最悪値。
 */
const RYKV_USABLE_WIDTH = 400 - 4 - 15

function make_query(metric_count: number) {
    return {
        id: 'graph-1',
        title: 'テスト',
        granularity: 'day',
        method: 'pearson',
        lag: 0,
        metrics: Array.from({ length: metric_count }, (_, i) => ({ id: `m${i}`, title: `指標${i}` })),
    }
}

function matrix_columns_of(metric_count: number): string {
    const { matrix_style } = useDnoteCorrelationGraphView({
        props: { gkill_api: {}, application_config: {}, editable: false },
        emits: (() => { }),
        model_value: ref(make_query(metric_count)),
    } as never)
    return (matrix_style.value as { gridTemplateColumns: string }).gridTemplateColumns
}

describe('相関グラフの行列の寸法', () => {
    it('指標数ぶんのトラックを組み立てる', () => {
        for (const n of [2, 5, 7, 10]) {
            expect(matrix_columns_of(n), `指標${n}個のトラックが期待と違う`).toBe(
                `${CORRELATION_MATRIX_ROW_HEADER_WIDTH}px repeat(${n}, minmax(${CORRELATION_MATRIX_CELL_MIN_WIDTH}px, 1fr))`,
            )
        }
    })

    // 本命。定数を緩めた瞬間に落ちる
    it('指標7個が rykv の400px列に横スクロールなしで収まる', () => {
        expect(
            correlation_matrix_min_width(7),
            `指標7個が rykv の400px列に収まらない（実効幅${RYKV_USABLE_WIDTH}px）`,
        ).toBeLessThanOrEqual(RYKV_USABLE_WIDTH)
    })

    // 1fr は列幅いっぱいまで伸ばすために必要だが、contain が無いと祖先の幅を引きずる
    it('セルは 1fr で伸びる', () => {
        expect(matrix_columns_of(7)).toContain('1fr')
    })

    it('指標0個でも repeat(0, ...) を出さない（不正CSSで宣言ごと落ちる）', () => {
        expect(build_correlation_matrix_columns(0)).toContain('repeat(1,')
        expect(matrix_columns_of(0)).toContain('repeat(1,')
    })

    it('最小必要幅の計算がトラックと gap に一致する', () => {
        expect(correlation_matrix_min_width(7)).toBe(
            CORRELATION_MATRIX_ROW_HEADER_WIDTH + 7 * CORRELATION_MATRIX_CELL_MIN_WIDTH + 8 * CORRELATION_MATRIX_GAP,
        )
    })
})

/**
 * 相関グラフのCSSが、定数の前提と食い違っていないか。
 *
 * 「max-content を付けてはいけない」のような注意書き自体を拾わないよう、
 * コメントを外してから宣言だけを見る。
 */
function find_css_violations(raw_source: string): Array<string> {
    const source = raw_source.replace(/\/\*[\s\S]*?\*\//g, '').replace(/<!--[\s\S]*?-->/g, '')
    const violations: Array<string> = []
    if (source.includes('min-width: max-content')) {
        violations.push('min-width: max-content は全列を最長見出しの幅に揃えるので戻さないこと')
    }
    if (!source.includes('gap: 1px')) {
        violations.push('gap が 1px でない（dnote-correlation-matrix-layout.ts の計算が前提にしている）')
    }
    if (!source.includes('padding: 2px')) {
        violations.push('セルの padding が 2px でない（同上）')
    }
    if (!/\.correlation_matrix_scroll\s*\{[^}]*contain:\s*inline-size/.test(source)) {
        violations.push('contain: inline-size が無い。1fr の max-content が祖先へ伝わり Dnote 列が広がる')
    }
    if (!source.includes('border: 1px solid transparent')) {
        violations.push('セルの border が 1px でない（同上）')
    }
    if (!source.includes('overflow-x: auto')) {
        violations.push('.correlation_matrix_scroll の横スクロール（指標8個以上の保険）が消えている')
    }
    if (/writing-mode|rotate\(/.test(source.replace(/<svg[\s\S]*?<\/svg>/g, ''))) {
        violations.push('見出しは横書きのまま折り返す方針。writing-mode / rotate は使わない')
    }
    if (/\.column_header\s*\{[^}]*\btop\s*:/.test(source)) {
        violations.push('.column_header の top は効かない（縦スクローラは祖先の .dnote-scroll-wrap）')
    }
    return violations
}

describe('相関グラフのCSS', () => {
    it('定数の前提から外れていない', () => {
        const source = readFileSync(join(repo_root, 'src/client/pages/views/dnote-correlation-graph-view.vue'), 'utf8')
        expect(find_css_violations(source)).toEqual([])
    })

    it('検出ロジックが違反を見つけられる（自己検査）', () => {
        const fixture = [
            '.correlation_matrix { min-width: max-content; gap: 2px; }',
            '.matrix_cell { padding: 6px; border: 2px solid transparent; }',
            '.column_header { position: sticky; top: 0; }',
        ].join('\n')
        const violations = find_css_violations(fixture)
        expect(violations.length).toBeGreaterThanOrEqual(3)
        expect(violations.join('\n')).toContain('max-content')
        expect(violations.join('\n')).toContain('.column_header の top')
    })

})
