/**
 * 「HTML で出力」ダイアログの中身。
 *
 * ここで選んだものがそのまま出力 HTML の中身になる。既定値と、
 * 「メディアを埋め込まない」と「同名フォルダ」の組み合わせを止めるところを固定する。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
// データクラスと req_res は gkill-api.ts と循環 import になっている。
// 先に読んで評価順を決めないと `class extends value undefined` になる（ADR-0903）
import '@/classes/api/gkill-api'
import { nextTick } from 'vue'
import { useExportRykvHTMLView } from '@/classes/use-export-rykv-html-view'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { ExportRykvHTMLResponse } from '@/classes/api/req_res/export-rykv-html-response'
import type { GkillAPI } from '@/classes/api/gkill-api'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import type { ExportRykvHTMLViewProps } from '@/pages/views/export-rykv-html-view-props'

const saved_files: Array<{ file_name: string }> = []
vi.mock('@/classes/save-as', () => ({
    save_as: (_data: Blob | File | string, file_name: string) => {
        saved_files.push({ file_name })
    },
}))

function make_query(query_id: string): FindKyouQuery {
    const query = new FindKyouQuery()
    query.query_id = query_id
    return query
}

function make_response(customize?: (res: ExportRykvHTMLResponse) => void): ExportRykvHTMLResponse {
    const res = new ExportRykvHTMLResponse()
    res.output_format = 'single'
    res.file_name = '出力.html'
    res.html = '<!doctype html><html></html>'
    if (customize) {
        customize(res)
    }
    return res
}

function make_props(customize?: (props: ExportRykvHTMLViewProps) => void): {
    props: ExportRykvHTMLViewProps,
    export_rykv_html: ReturnType<typeof vi.fn>,
} {
    const export_rykv_html = vi.fn(async () => make_response())
    const props = {
        application_config: {} as ApplicationConfig,
        gkill_api: { export_rykv_html } as unknown as GkillAPI,
        columns: [make_query('q1'), make_query('q2'), make_query('q3')],
        focused_column_index: 1,
    } as ExportRykvHTMLViewProps
    if (customize) {
        customize(props)
    }
    return { props, export_rykv_html }
}

function make_emits() {
    const calls: Array<{ event: string }> = []
    const emits = ((event: string) => {
        calls.push({ event })
    }) as unknown as Parameters<typeof useExportRykvHTMLView>[0]['emits']
    return { emits, calls }
}

beforeEach(() => {
    saved_files.length = 0
})

describe('useExportRykvHTMLView の既定値', () => {
    it('計画どおりの既定で開く', () => {
        const { props } = make_props()
        const { emits } = make_emits()
        const view = useExportRykvHTMLView({ props, emits })

        expect(view.include_tags.value).toBe(true)
        expect(view.include_texts.value).toBe(true)
        expect(view.include_notifications.value).toBe(true)
        expect(view.include_timeiss.value).toBe(true)
        expect(view.include_gps_log.value).toBe(true)
        expect(view.include_plugin_content.value, 'プラグインの本文は既定で入れない').toBe(false)
        expect(view.include_dnote.value).toBe(true)
        expect(view.include_map.value).toBe(true)
        expect(view.include_sidebar.value, 'サイドバーは既定で出さない').toBe(false)
        expect(view.media_mode.value).toBe('thumbnail')
        expect(view.output_format.value).toBe('single')
    })
})

describe('メディアを埋め込まないと同名フォルダは選べない', () => {
    it('「埋め込まない」へ変えると、選択済みの同名フォルダが単一 HTML へ戻る', async () => {
        const { props } = make_props()
        const { emits } = make_emits()
        const view = useExportRykvHTMLView({ props, emits })

        view.output_format.value = 'sidecar'
        view.media_mode.value = 'none'
        await nextTick()

        // disabled にするだけでは足りない。選択済みの値が送信まで残り、
        // 中身の無いフォルダを作ろうとしてサーバに弾かれる
        expect(view.output_format.value, '同名フォルダが選ばれたまま残っている').toBe('single')
        expect(view.is_sidecar_selectable.value).toBe(false)
    })

    it('メディアを入れるなら同名フォルダを選べる', () => {
        const { props } = make_props()
        const { emits } = make_emits()
        const view = useExportRykvHTMLView({ props, emits })

        view.media_mode.value = 'full'
        expect(view.is_sidecar_selectable.value).toBe(true)
    })
})

describe('出力する列', () => {
    it('既定では表示中のすべての列', () => {
        const { props } = make_props()
        const { emits } = make_emits()
        const view = useExportRykvHTMLView({ props, emits })

        expect(view.build_request().columns.map(q => q.query_id)).toEqual(['q1', 'q2', 'q3'])
    })

    it('フォーカス列を選ぶと、その1本だけ', () => {
        const { props } = make_props()
        const { emits } = make_emits()
        const view = useExportRykvHTMLView({ props, emits })

        view.target.value = 'focused'
        expect(view.build_request().columns.map(q => q.query_id)).toEqual(['q2'])
    })

    it('フォーカスされている列が無ければ、全列のまま', () => {
        const { props } = make_props(p => { p.focused_column_index = -1 })
        const { emits } = make_emits()
        const view = useExportRykvHTMLView({ props, emits })

        view.target.value = 'focused'
        expect(view.has_focused_column.value).toBe(false)
        expect(view.build_request().columns.map(q => q.query_id)).toEqual(['q1', 'q2', 'q3'])
    })

    // 列の条件はサイドバーが生成したまま渡す。削ると出力先の列が空になる
    it('列の条件はそのまま渡す（作り直さない）', () => {
        const { props } = make_props()
        const { emits } = make_emits()
        const view = useExportRykvHTMLView({ props, emits })

        expect(view.build_request().columns[0]).toBe(props.columns[0])
    })
})

describe('出力の受け取り', () => {
    it('単一 HTML を保存して閉じる', async () => {
        const { props } = make_props()
        const { emits, calls } = make_emits()
        const view = useExportRykvHTMLView({ props, emits })

        await view.export_html()

        expect(saved_files.map(f => f.file_name)).toEqual(['出力.html'])
        expect(calls.map(c => c.event)).toContain('requested_close_dialog')
        expect(view.is_exporting.value, '出力中のままになっている').toBe(false)
    })

    it('同名フォルダは ZIP として保存する', async () => {
        const { props, export_rykv_html } = make_props()
        export_rykv_html.mockImplementation(async () => make_response(res => {
            res.output_format = 'sidecar'
            res.file_name = '出力.zip'
            res.html = ''
            res.zip_base64 = 'UEsDBA=='
        }))
        const { emits } = make_emits()
        const view = useExportRykvHTMLView({ props, emits })

        await view.export_html()
        expect(saved_files.map(f => f.file_name)).toEqual(['出力.zip'])
    })

    it('エラーのときは保存もせず閉じもしない', async () => {
        const { props, export_rykv_html } = make_props()
        export_rykv_html.mockImplementation(async () => make_response(res => {
            res.errors = [{ error_code: 'ERR000454', error_message: 'だめ' }] as ExportRykvHTMLResponse['errors']
            res.html = ''
        }))
        const { emits, calls } = make_emits()
        const view = useExportRykvHTMLView({ props, emits })

        await view.export_html()

        expect(saved_files, 'エラーなのにファイルを保存している').toEqual([])
        expect(calls.map(c => c.event)).toContain('received_errors')
        expect(calls.map(c => c.event), 'エラーなのに閉じている').not.toContain('requested_close_dialog')
        expect(view.is_exporting.value).toBe(false)
    })

    it('出力中は二重に投げない', async () => {
        const { props, export_rykv_html } = make_props()
        let resolve_response: (() => void) | null = null
        export_rykv_html.mockImplementation(() => new Promise(resolve => {
            resolve_response = () => resolve(make_response())
        }))
        const { emits } = make_emits()
        const view = useExportRykvHTMLView({ props, emits })

        const first = view.export_html()
        await view.export_html()
        expect(export_rykv_html, '出力中にもう1本投げている').toHaveBeenCalledTimes(1)

        resolve_response!()
        await first
    })
})
