'use strict'

import { MERMAID_DATA_ATTRIBUTE } from '@/classes/markdown-to-html'
import { GkillAPI } from '@/classes/api/gkill-api'

// 描画済みのプレースホルダに付ける値。v-virtual-scrollの再描画で二重に走らせないために見る。
const RENDERED = 'rendered'

// mermaidが生成するsvg要素のid。ページ内で一意である必要がある。
let mermaid_id_counter = 0

// initializeを何度も呼ばないための状態 (テーマが変わったときだけ呼び直す)
let initialized_theme: string | null = null

type MermaidModule = Awaited<typeof import('mermaid')>['default']

async function setup_mermaid(): Promise<MermaidModule> {
    const { default: mermaid } = await import('mermaid')

    const theme = GkillAPI.get_gkill_api().get_use_dark_theme() ? 'dark' : 'default'
    if (initialized_theme !== theme) {
        // securityLevel: 'strict' でmermaid内部のサニタイズも有効にする
        mermaid.initialize({ startOnLoad: false, securityLevel: 'strict', theme })
        initialized_theme = theme
    }
    return mermaid
}

// container内のMermaidプレースホルダをSVGに置き換える。
// 描画に失敗したものはプレースホルダ (=Mermaidソースのコードブロック) のまま残す。
export async function render_mermaid_diagrams(container: HTMLElement): Promise<void> {
    const targets = Array.from(
        container.querySelectorAll(`[${MERMAID_DATA_ATTRIBUTE}]:not([${MERMAID_DATA_ATTRIBUTE}="${RENDERED}"])`)
    )
    if (targets.length === 0) {
        // Mermaidを含まないMarkdownではmermaidをロードしない (初期表示を重くしないため)
        return
    }

    const [mermaid, { default: DOMPurify }] = await Promise.all([
        setup_mermaid(),
        import('dompurify'),
    ])

    for (const target of targets) {
        const code = target.textContent ?? ''
        if (code.trim() === '') {
            continue
        }

        const id = `gkill_mermaid_${mermaid_id_counter++}`
        try {
            // 構文エラーのときにmermaidが赤いエラー図を描かないよう、先に検証する
            const parse_result = await mermaid.parse(code, { suppressErrors: true })
            if (!parse_result) {
                continue
            }

            const { svg } = await mermaid.render(id, code)

            const wrapper = document.createElement('div')
            wrapper.className = 'gkill_mermaid'
            wrapper.setAttribute(MERMAID_DATA_ATTRIBUTE, RENDERED)
            // mermaidのラベルはforeignObject内のHTMLを使うため html プロファイルも許可する
            wrapper.innerHTML = DOMPurify.sanitize(svg, {
                USE_PROFILES: { svg: true, svgFilters: true, html: true },
            })
            target.replaceWith(wrapper)
        } catch {
            // 描画失敗時はソースのコードブロックのまま残す
        } finally {
            // mermaidが描画のためにbodyへ差し込む一時要素を掃除する
            document.getElementById(id)?.remove()
            document.getElementById('d' + id)?.remove()
        }
    }
}
