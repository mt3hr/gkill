/**
 * useIDFKyouView composable tests.
 * ファイル種別の判定と、Markdown/テキストのインライン表示のロードを検証する。
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
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
