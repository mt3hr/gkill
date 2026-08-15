/**
 * KFTL送信の結果を一覧へ伝える経路のテスト。
 *
 * KFTLは送信全体をトランザクションで包むが、tx中の add_* は added_kyou を返せない
 * （一時リポジトリにしか無い）。そのためリクエストクラスはidだけ積み、
 * commit_tx のあとに get_kyou で実体を引いてから registered_kyou / updated_kyou を上げる。
 * commitより前に引くと「まだ無い」応答を掴むので、順序はここで固定する。
 */
import { afterEach, beforeEach, describe, test, expect, vi } from 'vitest'
import { createApp, defineComponent, h } from 'vue'
import { i18n } from '../../helpers/setup-i18n'

vi.mock('@/i18n', () => ({ i18n, default: i18n }))
vi.mock('@/classes/delete-gkill-cache', () => ({
    default: vi.fn().mockResolvedValue(undefined),
    delete_gkill_config_cache: vi.fn().mockResolvedValue(undefined),
    delete_gkill_all_tag_names_cache: vi.fn().mockResolvedValue(undefined),
    delete_gkill_attached_datas_cache: vi.fn().mockResolvedValue(undefined),
}))

import { useKftlView } from '@/classes/use-kftl-view'
import { ApplicationConfig } from '@/classes/datas/config/application-config'

interface CallLog {
    calls: Array<string>
}

function make_api(log: CallLog, overrides: Record<string, unknown> = {}) {
    const ok = { messages: null, errors: null }
    const record = (name: string, result: unknown = ok) => vi.fn(async () => {
        log.calls.push(name)
        return result
    })
    return {
        generate_uuid: vi.fn(() => `uuid-${log.calls.length}-${Math.random().toString(36).slice(2, 8)}`),
        add_kmemo: record('add_kmemo'),
        add_kc: record('add_kc'),
        add_lantana: record('add_lantana'),
        add_mi: record('add_mi'),
        add_nlog: record('add_nlog'),
        add_urlog: record('add_urlog'),
        add_timeis: record('add_timeis'),
        add_tag: record('add_tag'),
        add_text: record('add_text'),
        update_timeis: record('update_timeis'),
        get_kyous: vi.fn(async () => ({ kyous: [], messages: null, errors: null })),
        commit_tx: record('commit_tx'),
        discard_tx: record('discard_tx'),
        get_kyou: vi.fn(async (req: { id: string }) => {
            log.calls.push(`get_kyou:${req.id}`)
            return { kyou_histories: [{ id: req.id }], messages: null, errors: null }
        }),
        ...overrides,
    }
}

// KFTLViewは行ラベルの計算で本物のtextareaを id 引きするので、DOMに置いておく。
// jsdomの clientWidth は常に0なので、幅も持たせないと行数計算が NaN になる
let text_area_element: HTMLTextAreaElement | null = null

// 行ラベルの幅計算が canvas の measureText を、フローティングダイアログが
// ResizeObserver を使う。jsdom にはどちらも無い
beforeEach(() => {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (globalThis as any).ResizeObserver = class {
        observe(): void { }
        unobserve(): void { }
        disconnect(): void { }
    }
    HTMLCanvasElement.prototype.getContext = vi.fn(() => ({
        font: '',
        measureText: (text: string) => ({ width: text.length * 8 }),
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
    })) as any
    text_area_element = document.createElement('textarea')
    text_area_element.id = 'kftl_text_area'
    Object.defineProperty(text_area_element, 'clientWidth', { value: 600, configurable: true })
    Object.defineProperty(text_area_element, 'clientHeight', { value: 400, configurable: true })
    document.body.appendChild(text_area_element)
})

afterEach(() => {
    text_area_element?.remove()
    text_area_element = null
})

function mount_view(api: unknown) {
    const emits = vi.fn()
    let view: ReturnType<typeof useKftlView> | null = null
    const Host = defineComponent({
        setup() {
            const application_config = new ApplicationConfig()
            application_config.device = 'test-device'
            application_config.user_id = 'testuser'
            // 設定が読み込まれるまで送信ボタンは無効。立てないと do_submit が素通りする
            application_config.is_loaded = true
            const props = {
                gkill_api: api,
                application_config: application_config,
                app_content_height: 600,
                app_content_width: 800,
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
            } as any
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            view = useKftlView({ props, emits: emits as any })
            return () => h('div')
        },
    })
    const app = createApp(Host)
    app.mount(document.createElement('div'))
    return { app, view: view!, emits }
}

function emitted(emits: ReturnType<typeof vi.fn>, name: string): Array<unknown[]> {
    return emits.mock.calls.filter(call => call[0] === name).map(call => call.slice(1))
}

/** タグ確認・板名確認を素通りさせて送信する */
async function submit_text(view: ReturnType<typeof useKftlView>, text: string): Promise<void> {
    view.text_area_content.value = text
    await view.confirm_mi_board_submit()
}

describe('KFTL送信後のイベント', () => {
    test('作ったKyouの件数だけ registered_kyou を上げる', async () => {
        const log: CallLog = { calls: [] }
        const { view, emits } = mount_view(make_api(log))

        // 「、」は記録の区切り。1回の送信で2件できる
        await submit_text(view, '一件目\n、\n二件目')

        expect(emitted(emits, 'registered_kyou').length).toBe(2)
        expect(emitted(emits, 'requested_reload_list').length).toBe(0)
    })

    test('get_kyou は commit_tx より後に呼ぶ（commit前はまだ検索に出ない）', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))

        await submit_text(view, 'メモ')

        const commit_index = log.calls.indexOf('commit_tx')
        const get_kyou_index = log.calls.findIndex(call => call.startsWith('get_kyou:'))
        expect(commit_index).toBeGreaterThanOrEqual(0)
        expect(get_kyou_index).toBeGreaterThan(commit_index)
    })

    test('エラーで破棄したときは何も上げない', async () => {
        const log: CallLog = { calls: [] }
        const api = make_api(log, {
            add_kmemo: vi.fn(async () => {
                log.calls.push('add_kmemo')
                return { messages: null, errors: [{ error_code: 'ERR', error_message: 'ng' }] }
            }),
        })
        const { view, emits } = mount_view(api)

        await submit_text(view, 'メモ')

        expect(log.calls).toContain('discard_tx')
        expect(emitted(emits, 'registered_kyou').length).toBe(0)
        expect(emitted(emits, 'updated_kyou').length).toBe(0)
        expect(emitted(emits, 'requested_reload_list').length).toBe(0)
    })

    test('引き直せなかったときだけ requested_reload_list へ1回落とす', async () => {
        const log: CallLog = { calls: [] }
        const api = make_api(log, {
            get_kyou: vi.fn(async () => ({ kyou_histories: [], messages: null, errors: null })),
        })
        const { view, emits } = mount_view(api)

        // 「、」は記録の区切り。1回の送信で2件できる
        await submit_text(view, '一件目\n、\n二件目')

        expect(emitted(emits, 'registered_kyou').length).toBe(0)
        expect(emitted(emits, 'requested_reload_list').length).toBe(1)
    })

    // 送信後の引き直しはKyouの件数ぶん往復するので、その間ずっと入力欄を
    // readonly のままにすると体感で数秒固まる。保存はcommitで終わっているので待たせない
    test('引き直しの完了を待たずに入力欄のreadonlyを解除する', async () => {
        const log: CallLog = { calls: [] }
        let release_get_kyou: (() => void) | null = null
        let notify_get_kyou_entered: (() => void) | null = null
        const get_kyou_gate = new Promise<void>((resolve) => { release_get_kyou = resolve })
        // 引き直しが「始まった」時点を捉える。ここから完了までの間が観測したい区間
        const get_kyou_entered = new Promise<void>((resolve) => { notify_get_kyou_entered = resolve })
        const api = make_api(log, {
            get_kyou: vi.fn(async (req: { id: string }) => {
                notify_get_kyou_entered?.()
                await get_kyou_gate
                return { kyou_histories: [{ id: req.id }], messages: null, errors: null }
            }),
        })
        const { view } = mount_view(api)

        const submitting = submit_text(view, 'メモ')
        await get_kyou_entered
        // 引き直しが飛行中のうちに、もう入力できるようになっていること
        expect(view.is_requested_submit.value, '引き直しの完了まで入力欄がreadonlyのままになっている').toBe(false)

        release_get_kyou?.()
        await submitting
        expect(view.is_requested_submit.value).toBe(false)
    })

    test('saved_kyou_by_kftl は従来どおり上がる（板ツリー/タグツリーの取り直し）', async () => {
        const log: CallLog = { calls: [] }
        const { view, emits } = mount_view(make_api(log))

        await submit_text(view, 'メモ')

        expect(emitted(emits, 'saved_kyou_by_kftl').length).toBe(1)
    })
})
