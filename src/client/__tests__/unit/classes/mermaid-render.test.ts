/**
 * render_mermaid_diagrams のテスト。
 * 実物のmermaidはjsdomで動かすには重いのでモックする。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
// mermaid-render → gkill-api の import 連鎖で class extends が undefined にならないよう先に評価する
import '@/classes/api/gkill-api'
import { render_mermaid_diagrams } from '@/classes/mermaid-render'
import { MERMAID_DATA_ATTRIBUTE } from '@/classes/markdown-to-html'

const mermaid_mock = {
    initialize: vi.fn(),
    parse: vi.fn(),
    render: vi.fn(),
}

vi.mock('mermaid', () => ({ default: mermaid_mock }))

const MERMAID_SOURCE = 'graph TD\n  A --> B\n'

// 実物のmermaidの戻り値に合わせたSVG。
// - ルート要素には render() に渡した id が付く
// - mermaidが注入するCSSは #id スコープなので、idは保持されている必要がある
function fake_rendered_svg(id: string, inner: string = '<g></g>'): string {
    return `<svg id="${id}" width="100%"><style>#${id} .node rect{fill:#eee}</style>${inner}</svg>`
}

// ノードのラベルはforeignObject内のHTML (mermaidのhtmlLabelsは既定でtrue)
const FOREIGN_OBJECT_LABEL =
    '<g><foreignObject width="100" height="20">'
    + '<div xmlns="http://www.w3.org/1999/xhtml" class="nodeLabel">ラベル</div>'
    + '</foreignObject></g>'

function createContainer(...children: Array<string>): HTMLElement {
    const container = document.createElement('div')
    for (const child of children) {
        const pre = document.createElement('pre')
        pre.setAttribute(MERMAID_DATA_ATTRIBUTE, '')
        pre.textContent = child
        container.appendChild(pre)
    }
    document.body.appendChild(container)
    return container
}

describe('render_mermaid_diagrams', () => {
    beforeEach(() => {
        document.body.innerHTML = ''
        mermaid_mock.initialize.mockReset()
        mermaid_mock.parse.mockReset().mockResolvedValue({ diagramType: 'flowchart' })
        mermaid_mock.render.mockReset().mockImplementation((id: string) => Promise.resolve({ svg: fake_rendered_svg(id) }))
    })

    it('プレースホルダが無いときはmermaidをロードしない', async () => {
        const container = document.createElement('div')
        container.innerHTML = '<p>ただのMarkdown</p>'

        await render_mermaid_diagrams(container)

        expect(mermaid_mock.initialize).not.toHaveBeenCalled()
        expect(mermaid_mock.render).not.toHaveBeenCalled()
    })

    it('プレースホルダをSVGに置き換える', async () => {
        const container = createContainer(MERMAID_SOURCE)

        await render_mermaid_diagrams(container)

        expect(mermaid_mock.render).toHaveBeenCalledTimes(1)
        expect(mermaid_mock.render.mock.calls[0][1]).toBe(MERMAID_SOURCE)
        expect(container.querySelector('.gkill_mermaid svg')).not.toBeNull()
        expect(container.querySelector(`pre[${MERMAID_DATA_ATTRIBUTE}]`)).toBeNull()
    })

    it('描画済みのものは再描画しない', async () => {
        const container = createContainer(MERMAID_SOURCE)

        await render_mermaid_diagrams(container)
        await render_mermaid_diagrams(container)

        expect(mermaid_mock.render).toHaveBeenCalledTimes(1)
    })

    it('構文エラーのときはソースのコードブロックのまま残す', async () => {
        mermaid_mock.parse.mockResolvedValue(false)
        const container = createContainer('graph TD\n  A --> \n')

        await render_mermaid_diagrams(container)

        expect(mermaid_mock.render).not.toHaveBeenCalled()
        const placeholder = container.querySelector(`pre[${MERMAID_DATA_ATTRIBUTE}]`)
        expect(placeholder).not.toBeNull()
        expect(placeholder?.textContent).toContain('graph TD')
    })

    it('描画が例外を投げたときもソースのコードブロックのまま残す', async () => {
        mermaid_mock.render.mockRejectedValue(new Error('render failed'))
        const container = createContainer(MERMAID_SOURCE)

        await render_mermaid_diagrams(container)

        const placeholder = container.querySelector(`pre[${MERMAID_DATA_ATTRIBUTE}]`)
        expect(placeholder).not.toBeNull()
        expect(placeholder?.textContent).toContain('graph TD')
    })

    it('SVGはサニタイズされる', async () => {
        mermaid_mock.render.mockImplementation((id: string) =>
            Promise.resolve({ svg: fake_rendered_svg(id, '<script>alert(1)</script>') }))
        const container = createContainer(MERMAID_SOURCE)

        await render_mermaid_diagrams(container)

        const wrapper = container.querySelector('.gkill_mermaid')
        expect(wrapper).not.toBeNull()
        expect(wrapper?.innerHTML).not.toContain('<script')
    })

    // mermaidはrender()に渡したidを戻り値のSVGルートに付ける。
    // 一時要素の掃除をidで行うと、DOMに挿入した描画済みSVGのほうを消してしまう。
    it('描画したSVGをdocumentから消してしまわない', async () => {
        const container = createContainer(MERMAID_SOURCE)

        await render_mermaid_diagrams(container)

        const wrapper = container.querySelector('.gkill_mermaid')
        expect(wrapper?.innerHTML).not.toBe('')
        const svg = container.querySelector('.gkill_mermaid svg')
        expect(svg).not.toBeNull()
        // mermaidが注入するCSSは #id スコープなので、idも保持されていること
        expect(svg?.getAttribute('id')).toBe(mermaid_mock.render.mock.calls[0][0])
    })

    it('foreignObject内のラベルをサニタイズで消さない', async () => {
        mermaid_mock.render.mockImplementation((id: string) =>
            Promise.resolve({ svg: fake_rendered_svg(id, FOREIGN_OBJECT_LABEL) }))
        const container = createContainer(MERMAID_SOURCE)

        await render_mermaid_diagrams(container)

        const wrapper = container.querySelector('.gkill_mermaid')
        expect(wrapper?.querySelector('foreignObject')).not.toBeNull()
        expect(wrapper?.textContent).toContain('ラベル')
    })

    it('mermaidがbodyに残した一時要素を掃除する', async () => {
        mermaid_mock.render.mockImplementation((id: string) => {
            // mermaidが描画のためにbodyへ差し込む一時要素 (通常はmermaid自身が消す)
            const temp = document.createElement('div')
            temp.id = 'd' + id
            document.body.appendChild(temp)
            return Promise.resolve({ svg: fake_rendered_svg(id) })
        })
        const container = createContainer(MERMAID_SOURCE)

        await render_mermaid_diagrams(container)

        const id = mermaid_mock.render.mock.calls[0][0]
        expect(document.getElementById('d' + id)).toBeNull()
        expect(container.querySelector('.gkill_mermaid svg')).not.toBeNull()
    })
})
