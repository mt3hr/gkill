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
        add_mirekyou: record('add_mirekyou'),
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

/** テンプレートの葉。paste_template に渡す */
function make_template(template: string, title: string = '買い物') {
    return {
        name: 'template_name',
        id: 'template_id',
        title: title,
        template: template,
        children: null,
        key: '',
        is_checked: false,
        indeterminate: false,
        is_dir: false,
    }
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

/**
 * 保存マーカー（行に「！」だけ）で保存が走る経路。
 *
 * 判定は「打った瞬間に確定した本文」で行う。行ラベルと不正行の再計算は
 * `get_invalid_line_indexs` が行ごとに await するので行数に比例して伸び、
 * その待ちのあとに本文を読み直すと、待っている間に打たれた1文字で末尾が
 * マーカーでなくなり **エラーも出ないまま保存が起きない**。
 */
describe('KFTLの保存マーカー', () => {
    test('マーカー付きで打つと保存が走る', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))

        view.onTextAreaBeforeInput()
        view.text_area_content.value = 'メモ\n！\n'
        view.onTextAreaInput()
        await nextTick()
        await flush_microtasks()
        await flush_microtasks()

        expect(log.calls, 'マーカーで保存が走っていない').toContain('add_kmemo')
    })

    // 「たまに保存されない」の正体
    test('解析待ちの間に打ち足しても保存を取りこぼさない', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))

        view.onTextAreaBeforeInput()
        view.text_area_content.value = 'メモ\n！\n'
        view.onTextAreaInput()
        // watch が解析(await)に入った直後に、続きが1文字打たれた状況
        await nextTick()
        view.onTextAreaBeforeInput()
        view.text_area_content.value = 'メモ\n！\nつ'
        view.onTextAreaInput()
        await flush_microtasks()
        await flush_microtasks()

        expect(log.calls, '解析待ちの間の1文字で保存が消えている').toContain('add_kmemo')
    })

    // 「素早く入力すると \n！\n が反応しない」の正体。
    //
    // watch は flush:'post' なので、1回のフラッシュ窓の中で本文が2回変わると
    // **1回しか呼ばれず、中間の値(マーカーで終わっている本文)は一度も観測されない**。
    // 行数の多いタブでは解析(get_invalid_line_indexs は行ごとに await)がメインスレッドを
    // 掴むので、その間に打たれたキーがまとめて着地して現実に起きる。
    // endsWith で判定している限り、この窓では末尾が既にマーカーではない。
    test('1回のフラッシュ窓でマーカーの後ろまで打たれても保存を取りこぼさない', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))

        view.onTextAreaBeforeInput()
        view.text_area_content.value = 'メモ\n！\n'
        view.onTextAreaInput()
        // nextTick を挟まない = watch はまだ動いていない。ここで続きが着地する
        view.onTextAreaBeforeInput()
        view.text_area_content.value = 'メモ\n！\nつ'
        view.onTextAreaInput()
        await nextTick()
        await flush_microtasks()
        await flush_microtasks()

        expect(log.calls, 'マーカー行が確定したのに保存が走っていない').toContain('add_kmemo')
    })

    // 実機で報告された形。IMEの確定Enterと改行Enterで、マーカー行の後ろに
    // 空行がもう1本入ることがある。
    //
    //     てすと
    //     ！
    //     (空行)
    //
    // 「マーカー行が本文の末尾か」で見ていると、この本文の末尾は空行なので
    // **打った時点では発火せず、バックスペースで最後の改行を消した瞬間に発火する**。
    // 「IMEから順当に入力すると効かないのに、バックスペースを押すと効く」の正体。
    test('マーカー行の後ろに空行が続いても保存が走る', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))

        view.onTextAreaBeforeInput()
        view.text_area_content.value = 'てすと\n！\n\n'
        view.onTextAreaInput()
        await nextTick()
        await flush_microtasks()
        await flush_microtasks()

        expect(log.calls, 'マーカーの後ろに空行があると保存が走らない').toContain('add_kmemo')
    })

    // バックスペースはマーカー行を増やさないので、保存の起点にはならない。
    // (旧実装はここで発火していた。同じ本文が二重に登録される原因でもある)
    test('マーカーの後ろをバックスペースで消しても再送信しない', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))

        // 既にマーカー入りの本文がある状態(打っていないので保存は走らない)
        view.text_area_content.value = 'てすと\n！\n\n'
        await nextTick()
        await flush_microtasks()
        expect(log.calls).not.toContain('add_kmemo')

        // 末尾の改行を1つ消す = マーカー行が本文の末尾になる
        view.onTextAreaBeforeInput()
        view.text_area_content.value = 'てすと\n！\n'
        view.onTextAreaInput()
        await nextTick()
        await flush_microtasks()
        await flush_microtasks()

        expect(log.calls, 'バックスペースで保存が走っている').not.toContain('add_kmemo')
    })

    // IMEでは「変換の確定」と「改行」が別々の入力として着地する。
    // 確定した時点(マーカー行がまだ改行で閉じていない)では走らず、
    // 改行が入って行が確定した時点で走る
    test('IMEの確定と改行が別々に着地しても保存が走る', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))

        // 変換確定。マーカーはまだ行として閉じていない
        view.onTextAreaBeforeInput()
        view.text_area_content.value = 'てすと\n！'
        view.onTextAreaInput()
        await nextTick()
        await flush_microtasks()
        expect(log.calls, 'マーカー行が閉じる前に保存が走っている').not.toContain('add_kmemo')

        // 改行でマーカー行が確定する
        view.onTextAreaBeforeInput()
        view.text_area_content.value = 'てすと\n！\n'
        view.onTextAreaInput()
        await nextTick()
        await flush_microtasks()
        await flush_microtasks()

        expect(log.calls, '改行でマーカー行が確定したのに保存が走っていない').toContain('add_kmemo')
    })

    // IME変換中は v-model がモデルを更新しない(Vueが composing の間 input を捨てる)。
    // @input だけが何度も飛ぶので、印が立ったまま本文が変わらない状態が続く。
    // ここで発火してはいけないし、確定したときに取りこぼしてもいけない
    test('IME変換中(本文が変わらない)は発火せず、確定したら発火する', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))

        view.text_area_content.value = 'てすと\n'
        await nextTick()
        await flush_microtasks()

        // 変換中のキー入力。@input は飛ぶが本文は変わらない
        view.onTextAreaBeforeInput()
        view.onTextAreaInput()
        view.onTextAreaBeforeInput()
        view.onTextAreaInput()
        view.onTextAreaBeforeInput()
        view.onTextAreaInput()
        await nextTick()
        await flush_microtasks()
        expect(log.calls, '本文が変わっていないのに保存が走っている').not.toContain('add_kmemo')

        // 確定と改行がまとめて着地する(1回のフラッシュ窓に収まる場合)
        view.onTextAreaBeforeInput()
        view.text_area_content.value = 'てすと\n！\n'
        view.onTextAreaInput()
        await nextTick()
        await flush_microtasks()
        await flush_microtasks()

        expect(log.calls, 'IME確定で保存が走っていない').toContain('add_kmemo')
    })

    // マーカーが1行目にある場合。前後の改行を要求する endsWith では拾えない
    test('マーカーが1行目でも保存が走る', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))

        view.onTextAreaBeforeInput()
        view.text_area_content.value = '！\nメモ\n'
        view.onTextAreaInput()
        await nextTick()
        await flush_microtasks()
        await flush_microtasks()

        expect(log.calls, '1行目のマーカーで保存が走っていない').toContain('add_kmemo')
    })

    // マーカーが増えていないなら「保存して」という新しい指示ではない。
    // これが効かないと、マーカーの残った本文を1文字打つたびに保存が走る
    test('既にあるマーカーの後ろを編集しても再送信しない', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))

        // 復元などでマーカー入りの本文が入っている状態(打っていないので保存は走らない)
        view.text_area_content.value = 'メモ\n！\nつづき'
        await nextTick()
        await flush_microtasks()
        expect(log.calls).not.toContain('add_kmemo')

        // ここから利用者が打つ。マーカーは増えていない
        view.onTextAreaBeforeInput()
        view.text_area_content.value = 'メモ\n！\nつづき2'
        view.onTextAreaInput()
        await nextTick()
        await flush_microtasks()
        await flush_microtasks()

        expect(log.calls, 'マーカーが増えていないのに保存が走っている').not.toContain('add_kmemo')
    })

    test('利用者が打っていないのに本文が変わっただけでは保存しない', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))

        // タブ切替・localStorage からの復元はこの形（onTextAreaInput を通らない）
        view.text_area_content.value = 'メモ\n！\n'
        await nextTick()
        await flush_microtasks()
        await flush_microtasks()

        expect(log.calls, '打っていないのに保存が走っている').not.toContain('add_kmemo')
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
        expect(view.is_confirm_unknown_tag_open.value).toBe(true)

        // 確認中にタブを切り替えようとしてもロックされている
        view.activate_tab(other_tab_id)
        expect(view.active_tab_id.value).toBe(target_tab_id)

        // ストアを直に叩いて切り替えても、送信対象は最初のタブのまま
        view.activate_tab(other_tab_id)
        await view.confirm_submit()

        expect(tabs.tabs.value.map(tab => tab.id)).toEqual([other_tab_id])
        expect(tabs.get_tab_content(other_tab_id)).toBe('別のタブ')
    })

    // 確認ダイアログは共有部品(ConfirmUnknownTagDialog)なので、ブラウザバックで閉じられると
    // cancel_submit を通らない。`closed` でロックを倒さないとタブが二度と切り替えられなくなる
    test('確認をブラウザバックで閉じてもタブ操作のロックが外れる', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))
        const tabs = useKftlTabs()

        const target_tab_id = view.active_tab_id.value
        tabs.set_tab_content(target_tab_id, '。未知のタグ\n送るほう')
        const other_tab_id = tabs.add_tab('別のタブ')
        view.activate_tab(target_tab_id)

        await view.submit()
        expect(view.is_tab_locked.value).toBe(true)

        // ダイアログが「閉じた」と言ってきただけ（cancel_submit は通らない）
        view.onConfirmUnknownTagClosed()

        expect(view.is_tab_locked.value).toBe(false)
        view.activate_tab(other_tab_id)
        expect(view.active_tab_id.value).toBe(other_tab_id)
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

        view.paste_template(make_template('ーみ\n買い物'))

        expect(tabs.tabs.value.length).toBe(2)
        expect(tabs.get_tab_content(first_tab_id), '書きかけが上書きされている').toBe('書きかけ')
        expect(view.text_area_content.value).toBe('ーみ\n買い物')
        expect(view.tab_label(tabs.tabs.value[1], 1)).toBe('買い物')
    })

    // テンプレートは textarea の @input を起こさないので、保存マーカーの自動送信を
    // watch の印（user_input_tab_id）だけに任せると発火しない
    test('保存マーカーで終わるテンプレートを選ぶと保存が走る', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))
        const tabs = useKftlTabs()

        await view.paste_template(make_template('メモ\n！\n'))
        await flush_microtasks()

        expect(log.calls).toContain('add_kmemo')
        expect(log.calls).toContain('commit_tx')
        expect(tabs.tabs.value.length, '保存できたタブが閉じていない').toBe(1)
    })

    test('保存マーカーが無いテンプレートでは保存しない', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))
        const tabs = useKftlTabs()

        await view.paste_template(make_template('ーみ\n買い物'))
        await flush_microtasks()

        expect(log.calls).not.toContain('add_kmemo')
        expect(tabs.tabs.value.length).toBe(2)
    })

    // 判定を watch 経由に戻すと、watch の `new_value === old_value` 早期returnで
    // ここだけが黙って落ちる。差し戻しによる静かな再発を止めるための見張り
    test('貼る前のタブの本文がテンプレートと同じでも保存が走る', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))
        const tabs = useKftlTabs()
        tabs.set_tab_content(view.active_tab_id.value, 'メモ\n！\n')

        await view.paste_template(make_template('メモ\n！\n'))
        await flush_microtasks()

        expect(log.calls.filter(call => call === 'add_kmemo').length).toBe(1)
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

    // テンプレートは毎回一意な新しいタブを作り、それをアクティブにするのは貼ったウィンドウだけ。
    // 1回の選択で開いている枚数ぶん保存されることは構造的に起きない
    test('テンプレートを貼っても、もう1枚のウィンドウは送信しない', async () => {
        const log: CallLog = { calls: [] }
        const api = make_api(log)
        const first_window = mount_view(api)
        const second_window = mount_view(api)

        const second_tab_id = second_window.view.active_tab_id.value

        await first_window.view.paste_template(make_template('メモ\n！\n'))
        await flush_microtasks()

        expect(log.calls.filter(call => call === 'add_kmemo').length).toBe(1)
        expect(second_window.view.active_tab_id.value, '別のウィンドウまで貼り先へ移った').toBe(second_tab_id)
    })

    // is_requested_submit はビューごとなので、これだけではウィンドウをまたいだ二重送信を防げない。
    // KFTLはtxで束ねて送るので、二重送信するとKyouが丸ごと重複登録される
    test('同じタブを2枚のウィンドウが同時に保存しても登録は1回', async () => {
        const log: CallLog = { calls: [] }
        const api = make_api(log)
        const first_window = mount_view(api)
        const second_window = mount_view(api)

        expect(second_window.view.active_tab_id.value, '前提: 2枚が同じタブを映している')
            .toBe(first_window.view.active_tab_id.value)
        first_window.view.text_area_content.value = 'メモ'

        await Promise.all([
            first_window.view.submit(),
            second_window.view.submit(),
        ])
        await flush_microtasks()

        expect(log.calls.filter(call => call === 'add_kmemo').length).toBe(1)
        expect(log.calls.filter(call => call === 'commit_tx').length).toBe(1)
    })
})

/**
 * リポストタスク(「～～」で開いて「～～」で閉じるブロック)の送信。
 *
 * MiReKyou は対象の Kyou とは別の Kyou なので、1回の送信で2件登録される。
 * ブロックの中に書いたタグは対象ではなく MiReKyou 自身に付く。
 */
describe('KFTLのリポストタスク', () => {
    interface AddMiReKyouCall { mirekyou: Record<string, unknown> }
    interface AddKmemoCall { kmemo: Record<string, unknown> }
    interface AddTagCall { tag: Record<string, unknown> }

    function make_capturing_api(log: CallLog) {
        const ok = { messages: null, errors: null }
        const mirekyou_calls = new Array<AddMiReKyouCall>()
        const kmemo_calls = new Array<AddKmemoCall>()
        const tag_calls = new Array<AddTagCall>()
        const api = make_api(log, {
            add_mirekyou: vi.fn(async (req: AddMiReKyouCall) => {
                log.calls.push('add_mirekyou')
                mirekyou_calls.push(req)
                return ok
            }),
            add_kmemo: vi.fn(async (req: AddKmemoCall) => {
                log.calls.push('add_kmemo')
                kmemo_calls.push(req)
                return ok
            }),
            add_tag: vi.fn(async (req: AddTagCall) => {
                log.calls.push('add_tag')
                tag_calls.push(req)
                return ok
            }),
        })
        return { api, mirekyou_calls, kmemo_calls, tag_calls }
    }

    test('メモとリポストタスクの両方を登録し、registered_kyou を2件上げる', async () => {
        const log: CallLog = { calls: [] }
        const { view, emits } = mount_view(make_api(log))

        await submit_text(view, '牛乳を買う\n～～\n仕事\n～～')

        expect(log.calls).toContain('add_kmemo')
        expect(log.calls).toContain('add_mirekyou')
        expect(emitted(emits, 'registered_kyou').length).toBe(2)
    })

    test('target_id が同じレコードで書いたメモの id を指す', async () => {
        const log: CallLog = { calls: [] }
        const { api, mirekyou_calls, kmemo_calls } = make_capturing_api(log)
        const { view } = mount_view(api)

        await submit_text(view, '牛乳を買う\n～～\n仕事\n～～')

        expect(mirekyou_calls.length).toBe(1)
        expect(kmemo_calls.length).toBe(1)
        expect(mirekyou_calls[0].mirekyou.target_id).toBe(kmemo_calls[0].kmemo.id)
        expect(mirekyou_calls[0].mirekyou.id).not.toBe(kmemo_calls[0].kmemo.id)
        expect(mirekyou_calls[0].mirekyou.is_checked).toBe(false)
        expect(mirekyou_calls[0].mirekyou.board_name).toBe('仕事')
    })

    // Mi の KFTL と同じく、日時の前の「？」は要らない
    test('日時は「？」なしのベタ書きでも解釈される', async () => {
        const log: CallLog = { calls: [] }
        const { api, mirekyou_calls } = make_capturing_api(log)
        const { view } = mount_view(api)

        await submit_text(view, '牛乳を買う\n～～\n仕事\n2025-03-20\n\n2025-03-22\n～～')

        const mirekyou = mirekyou_calls[0].mirekyou
        expect((mirekyou.estimate_start_time as Date).getFullYear()).toBe(2025)
        expect((mirekyou.estimate_start_time as Date).getDate()).toBe(20)
        expect(mirekyou.estimate_end_time).toBeNull()
        expect((mirekyou.limit_time as Date).getDate()).toBe(22)
    })

    test('日時に「？」を付けても同じ結果になる', async () => {
        const log: CallLog = { calls: [] }
        const { api, mirekyou_calls } = make_capturing_api(log)
        const { view } = mount_view(api)

        await submit_text(view, '牛乳を買う\n～～\n仕事\n？2025-03-20\n\n？2025-03-22\n～～')

        const mirekyou = mirekyou_calls[0].mirekyou
        expect((mirekyou.estimate_start_time as Date).getDate()).toBe(20)
        expect((mirekyou.limit_time as Date).getDate()).toBe(22)
    })

    test('ブロックの中のタグはリポストタスクに、閉じたあとのタグはメモに付く', async () => {
        const log: CallLog = { calls: [] }
        const { api, mirekyou_calls, kmemo_calls, tag_calls } = make_capturing_api(log)
        const { view } = mount_view(api)

        await submit_text(view, '牛乳を買う\n～～\n。今日中\n仕事\n～～\n。買い物')

        const mi_re_kyou_id = mirekyou_calls[0].mirekyou.id
        const kmemo_id = kmemo_calls[0].kmemo.id
        const tag_of = (name: string) => tag_calls.find(call => call.tag.tag === name)
        expect(tag_of('今日中')?.tag.target_id).toBe(mi_re_kyou_id)
        expect(tag_of('買い物')?.tag.target_id).toBe(kmemo_id)
    })

    // 対象の無いMiReKyouは検索でターゲット解決に失敗して結果から落ちるので、
    // 画面に出ないのに消せない行が残る。書く前にエラーにしてトランザクションごと捨てる
    test('レコードに対象のメモが無ければ何も保存せず破棄する', async () => {
        const log: CallLog = { calls: [] }
        const { view, emits } = mount_view(make_api(log))

        await submit_text(view, '～～\n仕事\n～～')

        expect(log.calls).not.toContain('add_mirekyou')
        expect(log.calls).toContain('discard_tx')
        expect(log.calls).not.toContain('commit_tx')
        expect(emitted(emits, 'registered_kyou').length).toBe(0)
    })

    // 板名行は自由入力なので、打ち間違いがそのまま新しい板になる。
    // Mi と同じくリポストタスクでも送信前に確認を出す
    test('まだ無い板名なら送信前に確認を出して保存しない', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))

        view.text_area_content.value = '牛乳を買う\n～～\n未知の板\n～～'
        await view.submit()

        expect(view.unknown_mi_boards.value).toEqual(['未知の板'])
        expect(log.calls).not.toContain('add_mirekyou')
    })

    test('ブロックの中の知らないタグでも送信前に確認を出す', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))

        view.text_area_content.value = '牛乳を買う\n～～\n。知らないタグ\n仕事\n～～'
        await view.submit()

        expect(view.unknown_tags.value).toContain('知らないタグ')
        expect(log.calls).not.toContain('add_mirekyou')
    })
})

/**
 * 支出は1つの「ーん」ブロックから支払いの数だけ Kyou が出る唯一の記法。
 * タグとテキストは支払いごとに付くので、add_tag / add_text の target_id が
 * その支払いの add_nlog の id と一致していなければならない。
 * 一致していないと、エラーも警告も出ないままタグだけが宙に浮く。
 */
describe('KFTLの支出', () => {
    interface NlogCallLog {
        nlog_ids: Array<string>
        tags: Array<{ tag: string, target_id: string }>
        texts: Array<{ id: string, target_id: string, text: string }>
    }

    function make_nlog_api(log: CallLog, nlog_log: NlogCallLog) {
        return make_api(log, {
            add_nlog: vi.fn(async (req: { nlog: { id: string } }) => {
                log.calls.push('add_nlog')
                nlog_log.nlog_ids.push(req.nlog.id)
                return { messages: null, errors: null }
            }),
            add_tag: vi.fn(async (req: { tag: { tag: string, target_id: string } }) => {
                log.calls.push('add_tag')
                nlog_log.tags.push({ tag: req.tag.tag, target_id: req.tag.target_id })
                return { messages: null, errors: null }
            }),
            add_text: vi.fn(async (req: { text: { id: string, target_id: string, text: string } }) => {
                log.calls.push('add_text')
                nlog_log.texts.push({ id: req.text.id, target_id: req.text.target_id, text: req.text.text })
                return { messages: null, errors: null }
            }),
        })
    }

    test('支払いの数だけ add_nlog と registered_kyou が出る', async () => {
        const log: CallLog = { calls: [] }
        const nlog_log: NlogCallLog = { nlog_ids: [], tags: [], texts: [] }
        const { view, emits } = mount_view(make_nlog_api(log, nlog_log))

        await submit_text(view, 'ーん\nコンビニ\nおにぎり\n150\nお茶\n120')

        expect(nlog_log.nlog_ids.length).toBe(2)
        expect(new Set(nlog_log.nlog_ids).size).toBe(2)
        expect(emitted(emits, 'registered_kyou').length).toBe(2)
    })

    // 以前は Nlog だけ id を採番し直していたので、タグがどの Nlog にも紐づいていなかった
    test('タグの target_id がその支払いの Nlog の id と一致する', async () => {
        const log: CallLog = { calls: [] }
        const nlog_log: NlogCallLog = { nlog_ids: [], tags: [], texts: [] }
        const { view } = mount_view(make_nlog_api(log, nlog_log))

        await submit_text(view, 'ーん\nコンビニ\nおにぎり\n150\n。食費\nお茶\n120\n。飲み物')

        expect(nlog_log.nlog_ids.length).toBe(2)
        expect(nlog_log.tags.length).toBe(2)
        const food = nlog_log.tags.find(tag => tag.tag === '食費')!
        const drink = nlog_log.tags.find(tag => tag.tag === '飲み物')!
        expect(nlog_log.nlog_ids).toContain(food.target_id)
        expect(nlog_log.nlog_ids).toContain(drink.target_id)
        expect(food.target_id).not.toBe(drink.target_id)
    })

    test('テキストの target_id もその支払いの Nlog の id と一致する', async () => {
        const log: CallLog = { calls: [] }
        const nlog_log: NlogCallLog = { nlog_ids: [], tags: [], texts: [] }
        const { view } = mount_view(make_nlog_api(log, nlog_log))

        await submit_text(view, 'ーん\nコンビニ\nおにぎり\n150\nーー\n朝ごはん用\nーー\nお茶\n120')

        expect(nlog_log.texts.length).toBe(1)
        expect(nlog_log.texts[0].text).toBe('朝ごはん用')
        expect(nlog_log.nlog_ids).toContain(nlog_log.texts[0].target_id)
    })

    test('ブロックの中の知らないタグでも送信前に確認を出す', async () => {
        const log: CallLog = { calls: [] }
        const nlog_log: NlogCallLog = { nlog_ids: [], tags: [], texts: [] }
        const { view } = mount_view(make_nlog_api(log, nlog_log))

        view.text_area_content.value = 'ーん\nコンビニ\nおにぎり\n150\n。知らないタグ'
        await view.submit()

        expect(view.unknown_tags.value).toContain('知らないタグ')
        expect(log.calls).not.toContain('add_nlog')
    })
})
