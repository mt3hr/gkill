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
import { useExportKyouHTMLView } from '@/classes/use-export-kyou-html-view'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { ExportKyouHTMLResponse } from '@/classes/api/req_res/export-kyou-html-response'
import { GkillMessage } from '@/classes/api/gkill-message'
import {
    EXPORT_DEFAULT_MEDIA_BUDGET_BYTES,
    EXPORT_DEFAULT_MEDIA_ITEM_LIMIT_BYTES,
    type ExportKyouHTMLRequest,
} from '@/classes/api/req_res/export-kyou-html-request'
import type { GkillAPI } from '@/classes/api/gkill-api'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import type { ExportKyouHTMLViewProps } from '@/pages/views/export-kyou-html-view-props'

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

function make_response(customize?: (res: ExportKyouHTMLResponse) => void): ExportKyouHTMLResponse {
    const res = new ExportKyouHTMLResponse()
    res.output_format = 'single'
    res.file_name = '出力.html'
    res.html = '<!doctype html><html></html>'
    if (customize) {
        customize(res)
    }
    return res
}

function make_props(customize?: (props: ExportKyouHTMLViewProps) => void): {
    props: ExportKyouHTMLViewProps,
    export_kyou_html: ReturnType<typeof vi.fn>,
} {
    const export_kyou_html = vi.fn(async () => make_response())
    const props = {
        application_config: {} as ApplicationConfig,
        gkill_api: { export_kyou_html } as unknown as GkillAPI,
        columns: [make_query('q1'), make_query('q2'), make_query('q3')],
        focused_column_index: 1,
        // rykv の既定（集計とマップを選べる）。mi のケースは customize で上書きする
        screen: 'rykv',
    } as ExportKyouHTMLViewProps
    if (customize) {
        customize(props)
    }
    return { props, export_kyou_html }
}

function make_emits() {
    const calls: Array<{ event: string, args: unknown[] }> = []
    const emits = ((event: string, ...args: unknown[]) => {
        calls.push({ event, args })
    }) as unknown as Parameters<typeof useExportKyouHTMLView>[0]['emits']
    return { emits, calls }
}

beforeEach(() => {
    saved_files.length = 0
})

describe('useExportKyouHTMLView の既定値', () => {
    it('計画どおりの既定で開く', () => {
        const { props } = make_props()
        const { emits } = make_emits()
        const view = useExportKyouHTMLView({ props, emits })

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
        const view = useExportKyouHTMLView({ props, emits })

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
        const view = useExportKyouHTMLView({ props, emits })

        view.media_mode.value = 'full'
        expect(view.is_sidecar_selectable.value).toBe(true)
    })
})

describe('出力する列', () => {
    it('既定では表示中のすべての列', () => {
        const { props } = make_props()
        const { emits } = make_emits()
        const view = useExportKyouHTMLView({ props, emits })

        expect(view.build_request().columns.map(q => q.query_id)).toEqual(['q1', 'q2', 'q3'])
    })

    it('フォーカス列を選ぶと、その1本だけ', () => {
        const { props } = make_props()
        const { emits } = make_emits()
        const view = useExportKyouHTMLView({ props, emits })

        view.target.value = 'focused'
        expect(view.build_request().columns.map(q => q.query_id)).toEqual(['q2'])
    })

    it('フォーカスされている列が無ければ、全列のまま', () => {
        const { props } = make_props(p => { p.focused_column_index = -1 })
        const { emits } = make_emits()
        const view = useExportKyouHTMLView({ props, emits })

        view.target.value = 'focused'
        expect(view.has_focused_column.value).toBe(false)
        expect(view.build_request().columns.map(q => q.query_id)).toEqual(['q1', 'q2', 'q3'])
    })

    // 列の条件はサイドバーが生成したまま渡す。削ると出力先の列が空になる
    it('列の条件はそのまま渡す（作り直さない）', () => {
        const { props } = make_props()
        const { emits } = make_emits()
        const view = useExportKyouHTMLView({ props, emits })

        expect(view.build_request().columns[0]).toBe(props.columns[0])
    })
})

describe('出力の受け取り', () => {
    it('単一 HTML を保存して閉じる', async () => {
        const { props } = make_props()
        const { emits, calls } = make_emits()
        const view = useExportKyouHTMLView({ props, emits })

        await view.export_html()

        expect(saved_files.map(f => f.file_name)).toEqual(['出力.html'])
        expect(calls.map(c => c.event)).toContain('requested_close_dialog')
        expect(view.is_exporting.value, '出力中のままになっている').toBe(false)
    })

    it('出力中を倒してから閉じるよう頼む', async () => {
        // ダイアログは「出力中は閉じない」ガードを持っている（× も Escape も戻るも止める）。
        // 倒す前に閉じるよう頼むと自分のガードに弾かれ、保存は済んでいるのに
        // ダイアログだけ開いたまま残る。順序は型でもレンダリングでも落ちないので、ここで固定する
        const { props } = make_props()
        const { emits, calls } = make_emits()
        const view = useExportKyouHTMLView({ props, emits })

        await view.export_html()

        const exporting_false_at = calls.findIndex(c => c.event === 'changed_is_exporting' && c.args[0] === false)
        const close_at = calls.findIndex(c => c.event === 'requested_close_dialog')
        expect(exporting_false_at, '出力が終わったことをダイアログへ伝えていない').toBeGreaterThanOrEqual(0)
        expect(close_at, '閉じるよう頼んでいない').toBeGreaterThanOrEqual(0)
        expect(
            exporting_false_at,
            '出力中を倒す前に閉じるよう頼んでいる（ダイアログ自身のガードに弾かれて開いたまま残る）',
        ).toBeLessThan(close_at)
    })

    it('同名フォルダは ZIP として保存する', async () => {
        const { props, export_kyou_html } = make_props()
        export_kyou_html.mockImplementation(async () => make_response(res => {
            res.output_format = 'sidecar'
            res.file_name = '出力.zip'
            res.html = ''
            res.zip_base64 = 'UEsDBA=='
        }))
        const { emits } = make_emits()
        const view = useExportKyouHTMLView({ props, emits })

        await view.export_html()
        expect(saved_files.map(f => f.file_name)).toEqual(['出力.zip'])
    })

    it('エラーのときは保存もせず閉じもしない', async () => {
        const { props, export_kyou_html } = make_props()
        export_kyou_html.mockImplementation(async () => make_response(res => {
            res.errors = [{ error_code: 'ERR000454', error_message: 'だめ' }] as ExportKyouHTMLResponse['errors']
            res.html = ''
        }))
        const { emits, calls } = make_emits()
        const view = useExportKyouHTMLView({ props, emits })

        await view.export_html()

        expect(saved_files, 'エラーなのにファイルを保存している').toEqual([])
        expect(calls.map(c => c.event)).toContain('received_errors')
        expect(calls.map(c => c.event), 'エラーなのに閉じている').not.toContain('requested_close_dialog')
        expect(view.is_exporting.value).toBe(false)
    })

    it('出力中は二重に投げない', async () => {
        const { props, export_kyou_html } = make_props()
        let resolve_response: (() => void) | null = null
        export_kyou_html.mockImplementation(() => new Promise(resolve => {
            resolve_response = () => resolve(make_response())
        }))
        const { emits } = make_emits()
        const view = useExportKyouHTMLView({ props, emits })

        const first = view.export_html()
        await view.export_html()
        expect(export_kyou_html, '出力中にもう1本投げている').toHaveBeenCalledTimes(1)

        resolve_response!()
        await first
    })
})

// 画面ごとに選べるものが違う。**ビューの中で画面を判定せず、
// ホストするラッパがリテラルで渡す**（ADR-0420）ので、ここでは props を直接組んで検す。
// 選べない区画を「欄を disabled にする」だけで済ませると、選択済みの値が
// 送信まで残って「焼き込みの印と中身が食い違う」になるので、要求の値まで見る。
describe('出力する画面', () => {
    it('rykv は screen を rykv として送り、集計とマップを選べる', async () => {
        const { props, export_kyou_html } = make_props()
        const view = useExportKyouHTMLView({ props, emits: make_emits().emits })
        expect(view.include_dnote.value, 'rykv は集計を既定で入れる').toBe(true)
        expect(view.include_map.value, 'rykv はマップを既定で入れる').toBe(true)

        await view.export_html()
        const req = export_kyou_html.mock.calls[0][0] as ExportKyouHTMLRequest
        expect(req.screen).toBe('rykv')
        expect(req.include_dnote).toBe(true)
        expect(req.include_map).toBe(true)
    })

    it('mi は screen を mi として送り、集計とマップを選べない（選べるかは screen だけで決まる）', async () => {
        const { props, export_kyou_html } = make_props(p => {
            p.screen = 'mi'
        })
        const view = useExportKyouHTMLView({ props, emits: make_emits().emits })
        expect(view.include_dnote.value, 'mi に集計は無いので最初から false').toBe(false)
        expect(view.include_map.value, 'mi にマップは無いので最初から false').toBe(false)

        await view.export_html()
        const req = export_kyou_html.mock.calls[0][0] as ExportKyouHTMLRequest
        expect(req.screen).toBe('mi')
        expect(req.include_dnote).toBe(false)
        expect(req.include_map).toBe(false)
    })

    it('選べない画面を後から true にしても要求には載らない', async () => {
        const { props, export_kyou_html } = make_props(p => {
            p.screen = 'mi'
        })
        const view = useExportKyouHTMLView({ props, emits: make_emits().emits })
        // 欄が出ていないのに ref を触れても、送る値は false のまま
        view.include_dnote.value = true
        view.include_map.value = true

        await view.export_html()
        const req = export_kyou_html.mock.calls[0][0] as ExportKyouHTMLRequest
        expect(req.include_dnote, '選べない集計が要求に漏れている').toBe(false)
        expect(req.include_map, '選べないマップが要求に漏れている').toBe(false)
    })

    it('「対象」の文言は画面で変わる（rykv は列、mi は板）', () => {
        const rykv = useExportKyouHTMLView({ props: make_props().props, emits: make_emits().emits })
        expect(rykv.target_all_label_key.value).toBe('EXPORT_TARGET_ALL_COLUMNS')
        expect(rykv.target_focused_label_key.value).toBe('EXPORT_TARGET_FOCUSED_COLUMN')

        const mi = useExportKyouHTMLView({
            props: make_props(p => { p.screen = 'mi' }).props,
            emits: make_emits().emits,
        })
        expect(mi.target_all_label_key.value).toBe('EXPORT_TARGET_ALL_BOARDS')
        expect(mi.target_focused_label_key.value).toBe('EXPORT_TARGET_FOCUSED_BOARD')
    })

    it('サーバが返した warning のメッセージ（MSG000096）は詰め替えずにそのまま通知する', async () => {
        const { props, export_kyou_html } = make_props()
        export_kyou_html.mockResolvedValueOnce(make_response(r => {
            const warning = new GkillMessage()
            warning.message_code = 'MSG000096'
            warning.message = '予算に収まらないため 1 件のメディア・添付ファイルを含めませんでした'
            warning.level = 'warning'
            r.partial = true
            r.messages = [warning]
        }))
        const { emits, calls } = make_emits()
        const view = useExportKyouHTMLView({ props, emits })

        await view.export_html()
        const messages = calls
            .filter(c => c.event === 'received_messages')
            .flatMap(c => c.args[0] as Array<{ level?: string, message: string }>)
        const warnings = messages.filter(m => m.level === 'warning')
        expect(warnings).toHaveLength(1)
        expect(warnings[0].message).toContain('予算に収まらないため')
        expect((warnings[0] as { message_code?: string }).message_code).toBe('MSG000096')
    })

    it('待機中の cancel() はダイアログを閉じる要求を出し、出力中の cancel() は中断してダイアログを閉じない', async () => {
        let resolve_response: (() => void) | null = null
        const { props, export_kyou_html } = make_props()
        export_kyou_html.mockImplementationOnce(() => new Promise(resolve => {
            resolve_response = () => resolve(make_response())
        }))
        const { emits, calls } = make_emits()
        const view = useExportKyouHTMLView({ props, emits })

        // 待機中: 閉じる要求が出る
        view.cancel()
        expect(calls.filter(c => c.event === 'requested_close_dialog')).toHaveLength(1)

        // 出力中: 中断のみ行い、まだ閉じない
        const running = view.export_html()
        expect(view.is_exporting.value).toBe(true)
        view.cancel()
        expect(view.is_exporting.value).toBe(false)
        expect(calls.filter(c => c.event === 'requested_close_dialog')).toHaveLength(1)

        resolve_response!()
        await running
    })

    it('予算に 0 や NaN が入ったときはクライアント既定値へフォールバックして送る', async () => {
        const { props, export_kyou_html } = make_props()
        const view = useExportKyouHTMLView({ props, emits: make_emits().emits })
        view.media_budget_mb.value = Number.NaN
        view.media_item_limit_mb.value = 0

        await view.export_html()
        const req = export_kyou_html.mock.calls[0][0] as ExportKyouHTMLRequest
        expect(req.media_budget_bytes).toBe(EXPORT_DEFAULT_MEDIA_BUDGET_BYTES)
        expect(req.media_item_limit_bytes).toBe(EXPORT_DEFAULT_MEDIA_ITEM_LIMIT_BYTES)
    })
})
