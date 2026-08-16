/**
 * KFTL送信の結果を一覧へ伝える経路のテスト。
 *
 * KFTLは送信全体をトランザクションで包むが、tx中の add_* は added_kyou を返せない
 * （一時リポジトリにしか無い）。そのためリクエストクラスはidだけ積み、
 * commit_tx のあとに get_kyou で実体を引いてから registered_kyou / updated_kyou を上げる。
 * commitより前に引くと「まだ無い」応答を掴むので、順序はここで固定する。
 */
import { afterEach, beforeEach, describe, test, expect, vi } from 'vitest'
import { createApp, defineComponent, h, nextTick } from 'vue'
import { i18n } from '../../helpers/setup-i18n'

vi.mock('@/i18n', () => ({ i18n, default: i18n }))
vi.mock('@/classes/delete-gkill-cache', () => ({
    default: vi.fn().mockResolvedValue(undefined),
    delete_gkill_config_cache: vi.fn().mockResolvedValue(undefined),
    delete_gkill_all_tag_names_cache: vi.fn().mockResolvedValue(undefined),
    delete_gkill_attached_datas_cache: vi.fn().mockResolvedValue(undefined),
}))

import { useKftlView } from '@/classes/use-kftl-view'
import { reset_kftl_tabs_for_test, useKftlTabs } from '@/classes/use-kftl-tabs'
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
    // タブのストアはモジュールシングルトン（/mkfl で KFTLView が2つ同時に生きるため）。
    // 落とさないとテスト間でタブと本文が漏れる
    localStorage.clear()
    reset_kftl_tabs_for_test();
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

/** 行の解析は行ごとに await するので、マイクロタスクを全部流し切る */
function flush_microtasks(): Promise<void> {
    return new Promise<void>((resolve) => setTimeout(resolve, 0))
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

describe('KFTLのタブ', () => {
    test('保存したタブは閉じる。最後の1枚なら空のタブが1枚残る', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))
        const tabs = useKftlTabs()

        await submit_text(view, 'メモ')

        expect(tabs.tabs.value.length, 'タブが0枚になっている').toBe(1)
        expect(view.text_area_content.value).toBe('')
    })

    test('2枚あるとき、保存したタブだけが消えてもう1枚の内容は残る', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))
        const tabs = useKftlTabs()

        const first_tab_id = view.active_tab_id.value
        tabs.set_tab_content(first_tab_id, '残るほう')
        view.add_tab()
        const second_tab_id = view.active_tab_id.value

        await submit_text(view, '送るほう')

        expect(tabs.tabs.value.map(tab => tab.id)).toEqual([first_tab_id])
        expect(tabs.get_tab_content(first_tab_id)).toBe('残るほう')
        expect(view.active_tab_id.value).toBe(first_tab_id)
        expect(second_tab_id).not.toBe(first_tab_id)
    })

    test('エラーで破棄したときはタブを閉じない', async () => {
        const log: CallLog = { calls: [] }
        const api = make_api(log, {
            add_kmemo: vi.fn(async () => {
                log.calls.push('add_kmemo')
                return { messages: null, errors: [{ error_code: 'ERR', error_message: 'ng' }] }
            }),
        })
        const { view } = mount_view(api)
        const tabs = useKftlTabs()
        const tab_id = view.active_tab_id.value

        await submit_text(view, 'メモ')

        expect(tabs.tabs.value.map(tab => tab.id)).toEqual([tab_id])
        expect(tabs.get_tab_content(tab_id)).toBe('メモ')
    })

    // 確認ダイアログは非モーダル（App.vue の .gkill-float-scrim が pointer-events: none）なので、
    // 確認中でも背後のタブバーは押せてしまう。送信対象は最初に捕まえたタブに固定する
    test('確認の往復中にタブを切り替えても、送信対象はずれない', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))
        const tabs = useKftlTabs()

        const target_tab_id = view.active_tab_id.value
        tabs.set_tab_content(target_tab_id, '。未知のタグ\n送るほう')
        const other_tab_id = tabs.add_tab('別のタブ')
        view.activate_tab(target_tab_id)

        // タグ確認で中断する
        await view.submit()
        expect(view.show_confirm_unknown_tag_dialog.value).toBe(true)

        // 確認中にタブを切り替えようとしてもロックされている
        view.activate_tab(other_tab_id)
        expect(view.active_tab_id.value).toBe(target_tab_id)

        // ストアを直に叩いて切り替えても、送信対象は最初のタブのまま
        view.activate_tab(other_tab_id)
        await view.confirm_submit()

        expect(tabs.tabs.value.map(tab => tab.id)).toEqual([other_tab_id])
        expect(tabs.get_tab_content(other_tab_id)).toBe('別のタブ')
    })

    test('保存マーカーで終わるタブへ切り替えただけでは送信しない', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))
        const tabs = useKftlTabs()

        const first_tab_id = view.active_tab_id.value
        const marker_tab_id = tabs.add_tab('メモ\n！\n')
        view.activate_tab(first_tab_id)

        view.activate_tab(marker_tab_id)
        await nextTick()
        await flush_microtasks()

        expect(log.calls).not.toContain('add_kmemo')
        expect(tabs.tabs.value.length).toBe(2)
    })

    test('テンプレートは上書きせず新しいタブで開く', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))
        const tabs = useKftlTabs()

        const first_tab_id = view.active_tab_id.value
        tabs.set_tab_content(first_tab_id, '書きかけ')

        view.paste_template({
            name: 'template_name',
            id: 'template_id',
            title: '買い物',
            template: 'ーみ\n買い物',
            children: null,
            key: '',
            is_checked: false,
            indeterminate: false,
            is_dir: false,
        })

        expect(tabs.tabs.value.length).toBe(2)
        expect(tabs.get_tab_content(first_tab_id), '書きかけが上書きされている').toBe('書きかけ')
        expect(view.text_area_content.value).toBe('ーみ\n買い物')
        expect(view.tab_label(tabs.tabs.value[1], 1)).toBe('買い物')
    })
})

// メモ帳ダイアログは複数枚開ける。タブの一覧と中身は共有だが、
// 「いま映しているタブ」はウィンドウごとに独立していないと並べて見られない
describe('KFTLを複数のウィンドウで開く', () => {
    test('アクティブタブはウィンドウごとに独立している', () => {
        const log: CallLog = { calls: [] }
        const first_window = mount_view(make_api(log))
        const second_window = mount_view(make_api(log))
        const tabs = useKftlTabs()

        const shared_tab_id = first_window.view.active_tab_id.value
        first_window.view.add_tab()
        const other_tab_id = first_window.view.active_tab_id.value

        expect(first_window.view.active_tab_id.value).toBe(other_tab_id)
        expect(second_window.view.active_tab_id.value, '別のウィンドウまで切り替わった').toBe(shared_tab_id)
        expect(tabs.tabs.value.length).toBe(2)
    })

    test('同じタブを映していれば打った内容が両方に出る', () => {
        const log: CallLog = { calls: [] }
        const first_window = mount_view(make_api(log))
        const second_window = mount_view(make_api(log))

        first_window.view.text_area_content.value = '片方で打った'

        expect(second_window.view.text_area_content.value).toBe('片方で打った')
    })

    test('片方が閉じたタブを映していたウィンドウは隣のタブへ移る', async () => {
        const log: CallLog = { calls: [] }
        const first_window = mount_view(make_api(log))
        const second_window = mount_view(make_api(log))
        const tabs = useKftlTabs()

        const first_tab_id = first_window.view.active_tab_id.value
        first_window.view.add_tab()
        const second_tab_id = first_window.view.active_tab_id.value
        second_window.view.activate_tab(second_tab_id)
        expect(second_window.view.active_tab_id.value).toBe(second_tab_id)

        // 1枚目のウィンドウが、2枚目のウィンドウが映しているタブを閉じる
        first_window.view.request_close_tab(second_tab_id)
        // 追随は watch なので、描画前に1tick待つ（利用者に空欄が見えることはない）
        await nextTick()

        expect(tabs.tabs.value.map(tab => tab.id)).toEqual([first_tab_id])
        expect(second_window.view.active_tab_id.value, 'タブが宙に浮いた').toBe(first_tab_id)
        expect(second_window.view.text_area_content.value).toBe('')
    })
})
