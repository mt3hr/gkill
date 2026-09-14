/**
 * KFTL送信の結果を一覧へ伝える経路のテスト。
 *
 * 解釈と書き込みはサーバの1実装（/api/submit_kftl_text）だけが行い、応答の created[] には
 * id しか載らない（ADR-0507）。ビューは送信のあとに get_kyou で実体を引いてから
 * registered_kyou / updated_kyou を上げる。送信より前に引くと「まだ無い」応答を掴むので、
 * 順序はここで固定する。「おかしな行」「付くタグ」「板名」も /api/parse_kftl_text の応答で決まる。
 *
 * API のモックは、本文を「、」で区切った件数ぶんの id を返し、「。」で始まる行をタグとして
 * 返す小さな偽サーバ。判定の中身（何が不正か）はサーバの責務なので、ここでは応答を差し替えて
 * ビューの振る舞いだけを見る。
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

/** 偽サーバの解析: 「、」だけの行で記録を区切り、「。」で始まる行をタグとして数える */
function fake_parse(kftl_text: string) {
    const lines = kftl_text.split('\n')
    const tags = new Array<string>()
    let record_count = 0
    let has_body = false
    for (const line of lines) {
        if (line === '、') {
            if (has_body) {
                record_count++
            }
            has_body = false
            continue
        }
        if (line.startsWith('。')) {
            const tag = line.slice(1)
            if (tag !== '' && !tags.includes(tag)) {
                tags.push(tag)
            }
            continue
        }
        if (line !== '' && line !== '！') {
            has_body = true
        }
    }
    if (has_body) {
        record_count++
    }
    return { tags, record_count }
}

function make_api(log: CallLog, overrides: Record<string, unknown> = {}) {
    return {
        generate_uuid: vi.fn(() => `uuid-${log.calls.length}-${Math.random().toString(36).slice(2, 8)}`),
        parse_kftl_text: vi.fn(async (req: { kftl_text: string }) => {
            log.calls.push('parse_kftl_text')
            const parsed = fake_parse(req.kftl_text)
            return { messages: null, errors: null, invalid_lines: [], tags: parsed.tags, mi_board_names: [], record_count: parsed.record_count }
        }),
        submit_kftl_text: vi.fn(async (req: { kftl_text: string }) => {
            log.calls.push('submit_kftl_text')
            const parsed = fake_parse(req.kftl_text)
            const created = new Array<{ id: string, data_type: string, updated: boolean }>()
            for (let i = 0; i < parsed.record_count; i++) {
                created.push({ id: `created-${i}`, data_type: 'kmemo', updated: false })
            }
            return { messages: null, errors: null, created }
        }),
        get_kyou: vi.fn(async (req: { id: string }) => {
            log.calls.push(`get_kyou:${req.id}`)
            return { kyou_histories: [{ id: req.id }], messages: null, errors: null }
        }),
        ...overrides,
    }
}

/** 送信が失敗する API。サーバは何も残さないので created は空 */
function make_failing_api(log: CallLog) {
    return make_api(log, {
        submit_kftl_text: vi.fn(async () => {
            log.calls.push('submit_kftl_text')
            return { messages: null, errors: [{ error_code: 'ERR', error_message: 'ng' }], created: [] }
        }),
    })
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
    // jsdom には ResizeObserver が無いので最小の実装を差し込む
    (globalThis as unknown as { ResizeObserver: unknown }).ResizeObserver = class {
        observe(): void { }
        unobserve(): void { }
        disconnect(): void { }
    }
    // 行ラベルの幅計算に使う分だけの2Dコンテキスト。jsdom は canvas を持たない
    HTMLCanvasElement.prototype.getContext = vi.fn(() => ({
        font: '',
        measureText: (text: string) => ({ width: text.length * 8 }),
    })) as unknown as typeof HTMLCanvasElement.prototype.getContext
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
            } as unknown as Parameters<typeof useKftlView>[0]['props']
            view = useKftlView({
                props,
                emits: emits as unknown as Parameters<typeof useKftlView>[0]['emits'],
            })
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

    test('get_kyou は submit_kftl_text より後に呼ぶ（送信前はまだ無い）', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))

        await submit_text(view, 'メモ')

        const submit_index = log.calls.indexOf('submit_kftl_text')
        const get_kyou_index = log.calls.findIndex(call => call.startsWith('get_kyou:'))
        expect(submit_index).toBeGreaterThanOrEqual(0)
        expect(get_kyou_index).toBeGreaterThan(submit_index)
    })

    test('送信の直前にサーバへ解析させ、解析→送信の順になる', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))

        await submit_text(view, 'メモ')

        const parse_index = log.calls.lastIndexOf('parse_kftl_text')
        const submit_index = log.calls.indexOf('submit_kftl_text')
        expect(parse_index).toBeGreaterThanOrEqual(0)
        expect(submit_index).toBeGreaterThan(parse_index)
    })

    test('送信に失敗したときは何も上げず、エラーだけ上げる', async () => {
        const log: CallLog = { calls: [] }
        const { view, emits } = mount_view(make_failing_api(log))

        await submit_text(view, 'メモ')

        expect(log.calls).toContain('submit_kftl_text')
        expect(emitted(emits, 'received_errors').length).toBe(1)
        expect(emitted(emits, 'registered_kyou').length).toBe(0)
        expect(emitted(emits, 'updated_kyou').length).toBe(0)
        expect(emitted(emits, 'requested_reload_list').length).toBe(0)
        expect(emitted(emits, 'saved_kyou_by_kftl').length).toBe(0)
    })

    test('打刻の終了（updated）は registered_kyou ではなく updated_kyou で上げる', async () => {
        const log: CallLog = { calls: [] }
        const api = make_api(log, {
            submit_kftl_text: vi.fn(async () => {
                log.calls.push('submit_kftl_text')
                return { messages: null, errors: null, created: [{ id: 'ended-timeis', data_type: 'timeis', updated: true }] }
            }),
        })
        const { view, emits } = mount_view(api)

        await submit_text(view, 'ーいえ\n作業')

        expect(emitted(emits, 'updated_kyou').length).toBe(1)
        expect(emitted(emits, 'registered_kyou').length).toBe(0)
    })

    // 「おかしな行」の判定はサーバ。2026-09-14 まで TS 側の判定が Go に追随しておらず、
    // `/mood` 単独が Web からだけ気分0で書かれていた（ADR-0503 の抜け）
    test('サーバが不正行を返したら送信せず、その行をピンクにしてエラーを上げる', async () => {
        const log: CallLog = { calls: [] }
        const api = make_api(log, {
            parse_kftl_text: vi.fn(async () => {
                log.calls.push('parse_kftl_text')
                return {
                    messages: null, errors: null, tags: [], mi_board_names: [], record_count: 0,
                    invalid_lines: [{ line_number: 2, line_text: '/mood', message: 'Invalid line found (line 2: "/mood"): needs a value' }],
                }
            }),
        })
        const { view, emits } = mount_view(api)

        await submit_text(view, 'メモ\n/mood')

        expect(log.calls).not.toContain('submit_kftl_text')
        expect(emitted(emits, 'received_errors').length).toBe(1)
        expect(view.invalid_line_numbers.value).toEqual([1])
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

    // 板・タグツリーの取り直しはこの合図で走る。引き直し（get_kyou）の完了を待ってから出すと、
    // 保存直後に別の画面へ移ったとき新しいタグがツリーに無いまま一覧が絞られ、記録が見えない
    test('saved_kyou_by_kftl は引き直し（get_kyou）より前に、応答の関連時刻で上がる', async () => {
        const log: CallLog = { calls: [] }
        let saved_before_get_kyou: boolean | null = null
        const emits_holder: { emits: ReturnType<typeof vi.fn> | null } = { emits: null }
        const api = make_api(log, {
            submit_kftl_text: vi.fn(async () => {
                log.calls.push('submit_kftl_text')
                return { messages: null, errors: null, created: [{ id: 'c1', data_type: 'kmemo', updated: false, related_time: '2099-01-02T03:04:05+09:00' }] }
            }),
            get_kyou: vi.fn(async (req: { id: string }) => {
                log.calls.push(`get_kyou:${req.id}`)
                saved_before_get_kyou = emitted(emits_holder.emits!, 'saved_kyou_by_kftl').length === 1
                return { kyou_histories: [{ id: req.id }], messages: null, errors: null }
            }),
        })
        const { view, emits } = mount_view(api)
        emits_holder.emits = emits

        await submit_text(view, 'メモ')

        expect(saved_before_get_kyou, '引き直しの前に saved_kyou_by_kftl が出ていない').toBe(true)
        const [time] = emitted(emits, 'saved_kyou_by_kftl')[0] as [Date]
        expect(time.getTime()).toBe(new Date('2099-01-02T03:04:05+09:00').getTime())
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

        expect(log.calls, 'マーカーで保存が走っていない').toContain('submit_kftl_text')
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

        expect(log.calls, '解析待ちの間の1文字で保存が消えている').toContain('submit_kftl_text')
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

        expect(log.calls, 'マーカー行が確定したのに保存が走っていない').toContain('submit_kftl_text')
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

        expect(log.calls, 'マーカーの後ろに空行があると保存が走らない').toContain('submit_kftl_text')
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
        expect(log.calls).not.toContain('submit_kftl_text')

        // 末尾の改行を1つ消す = マーカー行が本文の末尾になる
        view.onTextAreaBeforeInput()
        view.text_area_content.value = 'てすと\n！\n'
        view.onTextAreaInput()
        await nextTick()
        await flush_microtasks()
        await flush_microtasks()

        expect(log.calls, 'バックスペースで保存が走っている').not.toContain('submit_kftl_text')
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
        expect(log.calls, 'マーカー行が閉じる前に保存が走っている').not.toContain('submit_kftl_text')

        // 改行でマーカー行が確定する
        view.onTextAreaBeforeInput()
        view.text_area_content.value = 'てすと\n！\n'
        view.onTextAreaInput()
        await nextTick()
        await flush_microtasks()
        await flush_microtasks()

        expect(log.calls, '改行でマーカー行が確定したのに保存が走っていない').toContain('submit_kftl_text')
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
        expect(log.calls, '本文が変わっていないのに保存が走っている').not.toContain('submit_kftl_text')

        // 確定と改行がまとめて着地する(1回のフラッシュ窓に収まる場合)
        view.onTextAreaBeforeInput()
        view.text_area_content.value = 'てすと\n！\n'
        view.onTextAreaInput()
        await nextTick()
        await flush_microtasks()
        await flush_microtasks()

        expect(log.calls, 'IME確定で保存が走っていない').toContain('submit_kftl_text')
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

        expect(log.calls, '1行目のマーカーで保存が走っていない').toContain('submit_kftl_text')
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
        expect(log.calls).not.toContain('submit_kftl_text')

        // ここから利用者が打つ。マーカーは増えていない
        view.onTextAreaBeforeInput()
        view.text_area_content.value = 'メモ\n！\nつづき2'
        view.onTextAreaInput()
        await nextTick()
        await flush_microtasks()
        await flush_microtasks()

        expect(log.calls, 'マーカーが増えていないのに保存が走っている').not.toContain('submit_kftl_text')
    })

    test('利用者が打っていないのに本文が変わっただけでは保存しない', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))

        // タブ切替・localStorage からの復元はこの形（onTextAreaInput を通らない）
        view.text_area_content.value = 'メモ\n！\n'
        await nextTick()
        await flush_microtasks()
        await flush_microtasks()

        expect(log.calls, '打っていないのに保存が走っている').not.toContain('submit_kftl_text')
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

    test('送信に失敗したときはタブを閉じない', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_failing_api(log))
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

        expect(log.calls).not.toContain('submit_kftl_text')
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

        expect(log.calls).toContain('submit_kftl_text')
        expect(tabs.tabs.value.length, '保存できたタブが閉じていない').toBe(1)
    })

    test('保存マーカーが無いテンプレートでは保存しない', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))
        const tabs = useKftlTabs()

        await view.paste_template(make_template('ーみ\n買い物'))
        await flush_microtasks()

        expect(log.calls).not.toContain('submit_kftl_text')
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

        expect(log.calls.filter(call => call === 'submit_kftl_text').length).toBe(1)
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

        expect(log.calls.filter(call => call === 'submit_kftl_text').length).toBe(1)
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

        expect(log.calls.filter(call => call === 'submit_kftl_text').length).toBe(1)
    })
})

/**
 * リポストタスク(「～～」で開いて「～～」で閉じるブロック)の送信。
 *
 * MiReKyou は対象の Kyou とは別の Kyou なので、1回の送信で2件登録される。
 * ブロックの中に書いたタグは対象ではなく MiReKyou 自身に付く。
 */

// リポストタスク（`～～`）や支出（`ーん`）で何が書かれるか（target_id・タグの付け先・支払いごとの Nlog）は
// サーバの Go 実装だけが持つ（ADR-0507）。対のテストは kftl_mirekyou_test.go / kftl_nlog_test.go /
// handle_submit_kftl_text_test.go。ここではサーバの解析結果（tags / mi_board_names）から確認が出ることだけを見る。
describe('サーバの解析結果からの確認', () => {
    test('まだ無い板名なら送信前に確認を出して保存しない', async () => {
        const log: CallLog = { calls: [] }
        const api = make_api(log, {
            parse_kftl_text: vi.fn(async () => {
                log.calls.push('parse_kftl_text')
                return { messages: null, errors: null, invalid_lines: [], tags: [], mi_board_names: ['まだ無い板'], record_count: 1 }
            }),
        })
        const { view } = mount_view(api)

        view.text_area_content.value = 'ーみ\nタスク\nまだ無い板'
        await view.submit()

        expect(view.unknown_mi_boards.value).toEqual(['まだ無い板'])
        expect(log.calls).not.toContain('submit_kftl_text')
    })

    test('板名の確認を通すと送信される', async () => {
        const log: CallLog = { calls: [] }
        const api = make_api(log, {
            parse_kftl_text: vi.fn(async () => {
                log.calls.push('parse_kftl_text')
                return { messages: null, errors: null, invalid_lines: [], tags: [], mi_board_names: ['まだ無い板'], record_count: 1 }
            }),
        })
        const { view, emits } = mount_view(api)

        view.text_area_content.value = 'ーみ\nタスク\nまだ無い板'
        await view.submit()
        await view.confirm_mi_board_submit()

        expect(log.calls).toContain('submit_kftl_text')
        expect(emitted(emits, 'registered_kyou').length).toBe(1)
    })

    test('知らないタグはサーバの tags から拾って確認を出す', async () => {
        const log: CallLog = { calls: [] }
        const { view } = mount_view(make_api(log))

        view.text_area_content.value = 'メモ\n。知らないタグ'
        await view.submit()

        expect(view.is_confirm_unknown_tag_open.value).toBe(true)
        expect(log.calls).not.toContain('submit_kftl_text')
    })
})

describe('おかしな行の表示', () => {
    // 判定はサーバなので、打鍵のたびに投げず、止まってから1回だけ投げる
    test('打鍵が止まってから1回だけ解析を投げ、行番号を添字に直してピンクにする', async () => {
        vi.useFakeTimers()
        try {
            const log: CallLog = { calls: [] }
            const api = make_api(log, {
                parse_kftl_text: vi.fn(async (req: { kftl_text: string }) => {
                    log.calls.push('parse_kftl_text')
                    const invalid_lines = req.kftl_text.includes('/mood')
                        ? [{ line_number: 3, line_text: '/mood', message: 'needs a value' }]
                        : []
                    return { messages: null, errors: null, invalid_lines, tags: [], mi_board_names: [], record_count: 1 }
                }),
            })
            const { view } = mount_view(api)
            await vi.runAllTimersAsync()
            const before = log.calls.filter(call => call === 'parse_kftl_text').length

            view.text_area_content.value = 'メ'
            await nextTick()
            view.text_area_content.value = 'メモ'
            await nextTick()
            view.text_area_content.value = 'メモ\n、\n/mood'
            await nextTick()
            // デバウンスの途中では投げない
            await vi.advanceTimersByTimeAsync(100)
            expect(log.calls.filter(call => call === 'parse_kftl_text').length).toBe(before)

            await vi.advanceTimersByTimeAsync(400)
            expect(log.calls.filter(call => call === 'parse_kftl_text').length, '3回の打鍵で1回だけ投げる').toBe(before + 1)
            expect(view.invalid_line_numbers.value).toEqual([2])
        } finally {
            vi.useRealTimers()
        }
    })

    test('解析に失敗（オフライン）したら前回のピンクを残す', async () => {
        vi.useFakeTimers()
        try {
            const log: CallLog = { calls: [] }
            let fail = false
            const api = make_api(log, {
                parse_kftl_text: vi.fn(async () => {
                    log.calls.push('parse_kftl_text')
                    if (fail) {
                        throw new Error('offline')
                    }
                    return { messages: null, errors: null, invalid_lines: [{ line_number: 1, line_text: '/mood', message: 'ng' }], tags: [], mi_board_names: [], record_count: 0 }
                }),
            })
            const { view } = mount_view(api)
            view.text_area_content.value = '/mood'
            await nextTick()
            await vi.advanceTimersByTimeAsync(500)
            expect(view.invalid_line_numbers.value).toEqual([0])

            fail = true
            view.text_area_content.value = '/mood\n'
            await nextTick()
            await vi.advanceTimersByTimeAsync(500)
            expect(view.invalid_line_numbers.value, '通信失敗で前回の表示が消えている').toEqual([0])
        } finally {
            vi.useRealTimers()
        }
    })
})
