/**
 * プラグイン設定ダイアログ（`usePluginConfigDialog`）の保存経路。
 *
 * プラグインの設定フォームは iframe の中にあり、sandbox に `allow-same-origin` を
 * 付けていないので **iframe から gkill の API を直接叩けない**。
 * 保存だけは親がこのコンポーザブルで肩代わりする、という取り決めになっている:
 *
 *   iframe → 親 : `{ gkill_plugin_config: { <key>: <value>, ... } }`
 *   親 → iframe : `{ gkill_plugin_config_result: { ok: boolean, error?: string } }`
 *
 * 経路を間違えても例外は出ない。**プラグイン側は結果メッセージを待つだけなので、
 * 「保存ボタンを押しても何も起こらない」形で静かに壊れる**。
 * 一方で `e.source` の判定を緩めると、無関係なウィンドウからの postMessage で
 * プラグインの設定が書き換えられてしまう。どちらもここで固定する。
 */
import { describe, expect, test, vi, beforeEach, afterEach } from 'vitest'
import { createApp, defineComponent, h, markRaw, ref, nextTick, type Ref } from 'vue'

import { GkillAPI } from '@/classes/api/gkill-api'
import { usePluginConfigDialog } from '@/classes/use-plugin-config-dialog'

const get_plugin_config_html = vi.fn()
const post_plugin_config = vi.fn()

/** iframe.contentWindow の代わり。postMessage で受けた中身を貯める */
class FakeContentWindow {
    received = new Array<unknown>()
    postMessage(data: unknown): void {
        this.received.push(data)
    }
}

let mounted_apps = new Array<ReturnType<typeof createApp>>()

function mount_dialog(options: { rep_name?: string, use_dark_theme?: boolean } = {}) {
    const show: Ref<boolean> = ref(false)
    let dialog: ReturnType<typeof usePluginConfigDialog> | null = null
    const Host = defineComponent({
        setup() {
            dialog = usePluginConfigDialog({
                props: {
                    rep_name: options.rep_name ?? 'gkill_plugin_example',
                    application_config: { use_dark_theme: options.use_dark_theme ?? false },
                } as unknown as Parameters<typeof usePluginConfigDialog>[0]['props'],
                // defineModel はコンパイラマクロなので、.vue 側で作ったモデルを渡す形になっている
                show: show as unknown as Parameters<typeof usePluginConfigDialog>[0]['show'],
            })
            return () => h('div')
        },
    })
    const app = createApp(Host)
    app.mount(document.createElement('div'))
    mounted_apps.push(app)

    // **markRaw を外さないこと。** ref() に素のオブジェクトを入れると reactive Proxy に包まれ、
    // `iframe_ref.value.contentWindow` が別物になって `e.source` の一致判定が必ず外れる。
    // 本番は本物の HTMLIFrameElement で、Vue は DOM ノードを包まないのでこの差は出ない
    const content_window = new FakeContentWindow()
    dialog!.iframe_ref.value = markRaw({ contentWindow: content_window }) as unknown as HTMLIFrameElement
    return { show, dialog: dialog!, content_window }
}

/** iframe から親へのメッセージを模す */
async function post_from_iframe(content_window: FakeContentWindow, data: unknown): Promise<void> {
    const event = new MessageEvent('message', { data: data })
    Object.defineProperty(event, 'source', { value: content_window })
    window.dispatchEvent(event)
    await nextTick()
    await nextTick()
}

beforeEach(() => {
    mounted_apps = []
    get_plugin_config_html.mockReset().mockResolvedValue({ html: '<form></form>', errors: null })
    post_plugin_config.mockReset().mockResolvedValue({ errors: null })
    vi.spyOn(GkillAPI, 'get_gkill_api').mockReturnValue({
        get_session_id: () => 'session-1',
        get_plugin_config_html: get_plugin_config_html,
        post_plugin_config: post_plugin_config,
    } as unknown as GkillAPI)
})
afterEach(() => {
    for (const app of mounted_apps) {
        app.unmount()
    }
    vi.restoreAllMocks()
})

describe('設定HTMLの読み込み', () => {
    test('開いたときに取りに行き、本文を持つ', async () => {
        const { show, dialog } = mount_dialog()
        expect(get_plugin_config_html).not.toHaveBeenCalled()

        show.value = true
        await nextTick()
        await nextTick()

        expect(get_plugin_config_html).toHaveBeenCalledTimes(1)
        expect(get_plugin_config_html.mock.calls[0][0].rep_name).toBe('gkill_plugin_example')
        expect(dialog.html.value).toBe('<form></form>')
        expect(dialog.error_message.value).toBe('')
    })

    test('閉じるときには取りに行かない', async () => {
        const { show } = mount_dialog()
        show.value = true
        await nextTick()
        await nextTick()
        get_plugin_config_html.mockClear()

        show.value = false
        await nextTick()
        await nextTick()

        expect(get_plugin_config_html).not.toHaveBeenCalled()
    })

    test('エラーは本文ではなくエラー表示に出す', async () => {
        get_plugin_config_html.mockResolvedValue({
            html: '', errors: [{ error_message: '設定を読めません' }],
        })
        const { show, dialog } = mount_dialog()
        show.value = true
        await nextTick()
        await nextTick()

        expect(dialog.error_message.value).toBe('設定を読めません')
        expect(dialog.html.value).toBe('')
        expect(dialog.is_loading.value).toBe(false)
    })
})

describe('iframe からの保存依頼', () => {
    test('受け取ったフォームを post_plugin_config へ渡し、結果を iframe へ返す', async () => {
        const { content_window } = mount_dialog()

        await post_from_iframe(content_window, {
            gkill_plugin_config: { source_dirs: '~/Downloads' },
        })

        expect(post_plugin_config).toHaveBeenCalledTimes(1)
        const req = post_plugin_config.mock.calls[0][0]
        expect(req.rep_name).toBe('gkill_plugin_example')
        expect(req.form_data).toEqual({ source_dirs: '~/Downloads' })
        expect(content_window.received).toContainEqual({
            gkill_plugin_config_result: { ok: true, error: undefined },
        })
    })

    // 保存後は「読み込み件数」などプラグインが出す状態が変わるので取り直す。
    // ここを飛ばすと、保存できているのに画面が古いままで「効いていない」ように見える
    test('保存に成功したら設定HTMLを取り直す', async () => {
        const { content_window } = mount_dialog()
        get_plugin_config_html.mockClear()

        await post_from_iframe(content_window, { gkill_plugin_config: { a: 'b' } })

        expect(get_plugin_config_html).toHaveBeenCalledTimes(1)
    })

    test('保存に失敗したら iframe へ失敗を返し、取り直さない', async () => {
        post_plugin_config.mockResolvedValue({ errors: [{ error_message: '書き込めません' }] })
        const { dialog, content_window } = mount_dialog()
        get_plugin_config_html.mockClear()

        await post_from_iframe(content_window, { gkill_plugin_config: { a: 'b' } })

        expect(content_window.received).toContainEqual({
            gkill_plugin_config_result: { ok: false, error: '書き込めません' },
        })
        expect(dialog.error_message.value).toBe('書き込めません')
        expect(get_plugin_config_html).not.toHaveBeenCalled()
    })

    // プラグインは第三者が書いた HTML なので、何を送ってくるか分からない。
    // 文字列以外をそのまま通すと API のワイヤ型（Record<string, string>）が壊れる
    test('文字列でない値は文字列にしてから渡す', async () => {
        const { content_window } = mount_dialog()

        await post_from_iframe(content_window, {
            gkill_plugin_config: { count: 3, enabled: true, nothing: null },
        })

        expect(post_plugin_config.mock.calls[0][0].form_data).toEqual({
            count: '3', enabled: 'true', nothing: 'null',
        })
    })

    test.each([
        ['別のウィンドウから届いたメッセージ', 'other-source'],
        ['gkill_plugin_config を含まないメッセージ', 'no-key'],
        ['gkill_plugin_config がオブジェクトでないメッセージ', 'not-object'],
    ])('%s では保存しない', async (_label, kind) => {
        const { content_window } = mount_dialog()

        if (kind === 'other-source') {
            // **e.source の判定を緩めない。** 緩めると無関係なウィンドウから
            // プラグインの設定を書き換えられる
            await post_from_iframe(new FakeContentWindow(), {
                gkill_plugin_config: { a: 'b' },
            })
        } else if (kind === 'no-key') {
            await post_from_iframe(content_window, { something_else: 1 })
        } else {
            await post_from_iframe(content_window, { gkill_plugin_config: 'not an object' })
        }

        expect(post_plugin_config).not.toHaveBeenCalled()
    })
})

describe('テーマの通知', () => {
    test.each([
        [true, 'dark'],
        [false, 'light'],
    ])('use_dark_theme=%s なら %s を送る', (use_dark_theme, want) => {
        const { dialog, content_window } = mount_dialog({ use_dark_theme: use_dark_theme })
        dialog.send_theme_to_iframe()
        expect(content_window.received).toEqual([{ gkill_theme: want }])
    })
})
