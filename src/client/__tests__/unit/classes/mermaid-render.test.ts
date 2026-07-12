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
        mermaid_mock.render.mockReset().mockResolvedValue({ svg: '<svg><g></g></svg>' })
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
        mermaid_mock.render.mockResolvedValue({ svg: '<svg><script>alert(1)</script></svg>' })
        const container = createContainer(MERMAID_SOURCE)

        await render_mermaid_diagrams(container)

        const wrapper = container.querySelector('.gkill_mermaid')
        expect(wrapper).not.toBeNull()
        expect(wrapper?.innerHTML).not.toContain('<script')
    })
})
