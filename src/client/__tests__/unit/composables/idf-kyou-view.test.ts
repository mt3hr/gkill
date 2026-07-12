/**
 * useIDFKyouView composable tests.
 * ファイル種別の判定と、Markdown/テキストのインライン表示のロードを検証する。
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
// use-idf-kyou-viewはreq_res経由でGkillAPIRequestに依存する。
// GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の循環importがあるため、
// 本番同様に gkill-api を先に評価させないと class extends が undefined になる。
import '@/classes/api/gkill-api'
import { useIDFKyouView } from '@/classes/use-idf-kyou-view'
import type { IDFKyouProps } from '@/pages/views/idf-kyou-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

function createProps(file_name: string, file_url: string, is_list = false): IDFKyouProps {
    const idf_kyou = {
        file_name,
        file_url,
        is_image: false,
        is_video: false,
        is_audio: false,
        is_zip: false,
    }
    return {
        kyou: { id: 'kyou-1', typed_idf_kyou: idf_kyou },
        idf_kyou,
        height: 180,
        width: 400,
        is_image_request_to_thumb_size: is_list,
        enable_context_menu: true,
        enable_dialog: true,
    } as unknown as IDFKyouProps
}

const noop_emits = (() => { }) as unknown as KyouViewEmits

// fetchが返すバイト列を差し替える
function mockFetchBytes(bytes: Uint8Array): void {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
        ok: true,
        arrayBuffer: async () => bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength),
    }))
}

async function flush(): Promise<void> {
    // fetch → arrayBuffer → 動的import(marked/dompurify) → parse の各awaitを消化する
    for (let i = 0; i < 20; i++) {
        await Promise.resolve()
        await new Promise((resolve) => setTimeout(resolve, 0))
    }
}

describe('useIDFKyouView', () => {
    beforeEach(() => {
        vi.unstubAllGlobals()
    })

    afterEach(() => {
        vi.unstubAllGlobals()
    })

    it('.md をMarkdownと判定し、HTMLに変換して保持する', async () => {
        mockFetchBytes(new TextEncoder().encode('# 見出し\n\n- item\n'))

        const props = createProps('note.md', '/files/rep1/docs/note.md')
        const { is_markdown, is_text, markdown_html, text_content, text_loading } = useIDFKyouView({ props, emits: noop_emits })

        expect(is_markdown.value).toBe(true)
        expect(is_text.value).toBe(false)

        await flush()

        expect(text_loading.value).toBe(false)
        expect(markdown_html.value).toContain('<h1>見出し</h1>')
        expect(markdown_html.value).toContain('<li>item</li>')
        // Markdown表示のときはプレーンテキスト側は空のまま
        expect(text_content.value).toBeNull()
    })

    it('Markdown内の相対パス画像をfile_url基準で解決する', async () => {
        mockFetchBytes(new TextEncoder().encode('![alt](img/a.png)\n'))

        const props = createProps('note.md', '/files/rep1/docs/note.md')
        const { markdown_html } = useIDFKyouView({ props, emits: noop_emits })

        await flush()

        expect(markdown_html.value).toContain('/files/rep1/docs/img/a.png')
    })

    it('.txt は従来どおりプレーンテキストとして保持する', async () => {
        mockFetchBytes(new TextEncoder().encode('# これは見出しではない\n'))

        const props = createProps('note.txt', '/files/rep1/docs/note.txt')
        const { is_markdown, is_text, markdown_html, text_content } = useIDFKyouView({ props, emits: noop_emits })

        expect(is_text.value).toBe(true)
        expect(is_markdown.value).toBe(false)

        await flush()

        expect(text_content.value).toBe('# これは見出しではない\n')
        expect(markdown_html.value).toBe('')
    })

    it('Shift_JISのMarkdownを文字化けさせずにデコードする', async () => {
        // "# 日本語" を Shift_JIS でエンコードしたバイト列
        const shift_jis = new Uint8Array([
            0x23, 0x20, 0x93, 0xfa, 0x96, 0x7b, 0x8c, 0xea, 0x0a,
        ])
        mockFetchBytes(shift_jis)

        const props = createProps('note.md', '/files/rep1/docs/note.md')
        const { markdown_html } = useIDFKyouView({ props, emits: noop_emits })

        await flush()

        expect(markdown_html.value).toContain('<h1>日本語</h1>')
    })

    it('画像ファイルはインライン表示の対象外', () => {
        const props = createProps('photo.md', '/files/rep1/photo.md')
        // 拡張子はmdでもis_imageが立っていればMarkdown表示しない
        const idf = props.kyou.typed_idf_kyou
        if (idf) {
            idf.is_image = true
        }

        const { is_markdown } = useIDFKyouView({ props, emits: noop_emits })
        expect(is_markdown.value).toBe(false)
    })
})

describe('useIDFKyouView Markdown相対リンク → KyouDialog', () => {
    beforeEach(() => {
        vi.unstubAllGlobals()
    })

    afterEach(() => {
        vi.unstubAllGlobals()
    })

    // data-gkill-md-link 付きのanchor (markdown_to_safe_htmlが付与するものと同形)
    function createMdLinkAnchor(relative_path: string, href: string): HTMLAnchorElement {
        const anchor = document.createElement('a')
        anchor.setAttribute('href', href)
        anchor.setAttribute('data-gkill-md-link', relative_path)
        document.body.appendChild(anchor)
        return anchor
    }

    function createMouseEvent(target: Element, modifiers: Partial<MouseEvent> = {}): MouseEvent {
        return {
            target,
            preventDefault: vi.fn(),
            stopPropagation: vi.fn(),
            ctrlKey: false,
            metaKey: false,
            shiftKey: false,
            ...modifiers,
        } as unknown as MouseEvent
    }

    function createPropsWithAPI(api: unknown, enable_dialog = true): IDFKyouProps {
        const props = createProps('note.md', '/files/rep1/note.md')
        ;(props as unknown as { gkill_api: unknown }).gkill_api = api
        ;(props as unknown as { enable_dialog: boolean }).enable_dialog = enable_dialog
        return props
    }

    it('シングルクリックは遷移を無効化する (誤って新規タブが開かないように)', () => {
        const anchor = createMdLinkAnchor('index.md', '/files/rep1/index.md')
        const props = createPropsWithAPI({})
        const { on_markdown_content_click } = useIDFKyouView({ props, emits: noop_emits })

        const e = createMouseEvent(anchor)
        on_markdown_content_click(e)
        expect(e.preventDefault).toHaveBeenCalled()
    })

    it('修飾キー付きシングルクリックはブラウザ既定の動作に任せる', () => {
        const anchor = createMdLinkAnchor('index.md', '/files/rep1/index.md')
        const props = createPropsWithAPI({})
        const { on_markdown_content_click } = useIDFKyouView({ props, emits: noop_emits })

        const e = createMouseEvent(anchor, { ctrlKey: true })
        on_markdown_content_click(e)
        expect(e.preventDefault).not.toHaveBeenCalled()
    })

    it('マークされていない要素のクリックは何もしない', () => {
        const plain = document.createElement('span')
        document.body.appendChild(plain)
        const props = createPropsWithAPI({})
        const { on_markdown_content_click } = useIDFKyouView({ props, emits: noop_emits })

        const e = createMouseEvent(plain)
        on_markdown_content_click(e)
        expect(e.preventDefault).not.toHaveBeenCalled()
    })

    it('ダブルクリックで対象Kyouを解決し requested_open_rykv_dialog をemitする', async () => {
        const anchor = createMdLinkAnchor('index.md', '/files/rep1/index.md')
        const target_kyou = { id: 'kyou-target' }
        const api = {
            get_idf_kyou_by_relative_path: vi.fn().mockResolvedValue({ errors: [], kyou_id: 'kyou-target' }),
            get_kyou: vi.fn().mockResolvedValue({ errors: [], kyou_histories: [target_kyou] }),
        }
        const emits = vi.fn() as unknown as KyouViewEmits
        const props = createPropsWithAPI(api)
        const { on_markdown_content_dblclick } = useIDFKyouView({ props, emits })

        const e = createMouseEvent(anchor)
        await on_markdown_content_dblclick(e)

        expect(e.preventDefault).toHaveBeenCalled()
        expect(e.stopPropagation).toHaveBeenCalled()
        expect(api.get_idf_kyou_by_relative_path).toHaveBeenCalledWith(
            expect.objectContaining({ target_id: 'kyou-1', relative_path: 'index.md' }),
        )
        expect(emits).toHaveBeenCalledWith('requested_open_rykv_dialog', 'kyou', target_kyou)
    })

    it('相対パスのフラグメントを除いてサーバへ渡す', async () => {
        const anchor = createMdLinkAnchor('index.md#section', '/files/rep1/index.md#section')
        const api = {
            get_idf_kyou_by_relative_path: vi.fn().mockResolvedValue({ errors: [], kyou_id: '' }),
        }
        vi.stubGlobal('open', vi.fn())
        const props = createPropsWithAPI(api)
        const { on_markdown_content_dblclick } = useIDFKyouView({ props, emits: noop_emits })

        await on_markdown_content_dblclick(createMouseEvent(anchor))

        expect(api.get_idf_kyou_by_relative_path).toHaveBeenCalledWith(
            expect.objectContaining({ relative_path: 'index.md' }),
        )
    })

    it('対象がKyouとして見つからない場合は従来どおり新規タブで開く', async () => {
        const anchor = createMdLinkAnchor('missing.md', '/files/rep1/missing.md')
        const api = {
            get_idf_kyou_by_relative_path: vi.fn().mockResolvedValue({ errors: [], kyou_id: '' }),
        }
        const open_mock = vi.fn()
        vi.stubGlobal('open', open_mock)
        const emits = vi.fn() as unknown as KyouViewEmits
        const props = createPropsWithAPI(api)
        const { on_markdown_content_dblclick } = useIDFKyouView({ props, emits })

        await on_markdown_content_dblclick(createMouseEvent(anchor))

        expect(open_mock).toHaveBeenCalledWith('/files/rep1/missing.md', '_blank')
        expect(emits).not.toHaveBeenCalled()
    })

    it('API失敗時も従来どおり新規タブで開く', async () => {
        const anchor = createMdLinkAnchor('index.md', '/files/rep1/index.md')
        const api = {
            get_idf_kyou_by_relative_path: vi.fn().mockRejectedValue(new Error('network error')),
        }
        const open_mock = vi.fn()
        vi.stubGlobal('open', open_mock)
        const props = createPropsWithAPI(api)
        const { on_markdown_content_dblclick } = useIDFKyouView({ props, emits: noop_emits })

        await on_markdown_content_dblclick(createMouseEvent(anchor))

        expect(open_mock).toHaveBeenCalledWith('/files/rep1/index.md', '_blank')
    })

    it('enable_dialog=false のときはAPIを呼ばず新規タブで開く', async () => {
        const anchor = createMdLinkAnchor('index.md', '/files/rep1/index.md')
        const api = {
            get_idf_kyou_by_relative_path: vi.fn(),
        }
        const open_mock = vi.fn()
        vi.stubGlobal('open', open_mock)
        const props = createPropsWithAPI(api, false)
        const { on_markdown_content_dblclick } = useIDFKyouView({ props, emits: noop_emits })

        await on_markdown_content_dblclick(createMouseEvent(anchor))

        expect(api.get_idf_kyou_by_relative_path).not.toHaveBeenCalled()
        expect(open_mock).toHaveBeenCalledWith('/files/rep1/index.md', '_blank')
    })

    it('マークされていない要素のダブルクリックは何もしない (親のdblclickに任せる)', async () => {
        const plain = document.createElement('span')
        document.body.appendChild(plain)
        const api = {
            get_idf_kyou_by_relative_path: vi.fn(),
        }
        const props = createPropsWithAPI(api)
        const { on_markdown_content_dblclick } = useIDFKyouView({ props, emits: noop_emits })

        const e = createMouseEvent(plain)
        await on_markdown_content_dblclick(e)

        expect(e.preventDefault).not.toHaveBeenCalled()
        expect(e.stopPropagation).not.toHaveBeenCalled()
        expect(api.get_idf_kyou_by_relative_path).not.toHaveBeenCalled()
    })
})
