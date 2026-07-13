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
            // mermaidのラベルはforeignObject内のHTMLを使うため html プロファイルも許可する。
            // foreignObjectはSVGプロファイルに含まれないので、mermaid自身のサニタイズ設定と揃えて明示的に許可する。
            // (許可しないとノードのラベルがまるごと消え、ラベルのない図になる)
            wrapper.innerHTML = DOMPurify.sanitize(svg, {
                USE_PROFILES: { svg: true, svgFilters: true, html: true },
                ADD_TAGS: ['foreignobject'],
                ADD_ATTR: ['dominant-baseline'],
                HTML_INTEGRATION_POINTS: { foreignobject: true },
            })
            target.replaceWith(wrapper)
        } catch {
            // 描画失敗時はソースのコードブロックのまま残す
        } finally {
            // mermaidが描画のためにbodyへ差し込む一時要素を掃除する。
            // 一時要素は #d{id} で、描画されたSVGはその子。通常はmermaid自身が消すが、
            // 例外時に残ることがあるため念のため消す。
            // idそのもの (#{id}) で消してはならない。mermaidはrender()に渡したidを
            // 戻り値のSVGルートにも付けるため、いま挿入したSVGのほうを消してしまう。
            document.getElementById('d' + id)?.remove()
        }
    }
}
