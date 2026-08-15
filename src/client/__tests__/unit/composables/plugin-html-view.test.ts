/**
 * PluginHtmlView が iframe へ本文を渡せているかの検証。
 *
 * プラグイン本文は iframe の中にあるので、親と iframe の間は postMessage しか通らない。
 * ここが片道でも切れると「枠だけ出て中身が真っ白」「ダブルクリックしても何も起きない」になり、
 * どちらもタイミング次第で出たり出なかったりするので手動では捕まえにくい。
 *
 * 1. 本文の注入は「ローダーが名乗ってから」。iframe.contentWindow はローダーが
 *    読み込まれる前(about:blank の時点)から真なので、それを見て先に送ると
 *    リスナー未登録の iframe にメッセージが届いて黙って消える。
 *    親は同じ HTML を送り直さないため、そうなると本文が二度と入らない。
 * 2. iframe 内のダブルクリックを親の DOM イベントとして撃ち直す。
 *    これが無いと Ryuu / 詳細ペイン / KyouDialog のどこでも
 *    プラグイン本文のダブルクリックだけ KyouDialog が開かない。
 * 3. 一覧(height が数値)は従来どおり srcdoc 直書きで、注入経路を使わない。
 */
import { describe, expect, test, vi, beforeEach, afterEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'

vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

// 末端の子は Vuetify を引き込む（vitest は node_modules の .css を解決できない）。
// スタブは描画時にしか効かず import は走ってしまうので、モジュールごと差し替える
vi.mock('@/pages/views/plugin-html-context-menu.vue', () => ({
    default: { name: 'plugin-html-context-menu', template: '<div />' },
}))
vi.mock('@/pages/dialogs/plugin-config-dialog.vue', () => ({
    default: { name: 'plugin-config-dialog', template: '<div />' },
}))

import PluginHtmlView from '@/pages/views/plugin-html-view.vue'
import { GkillAPI } from '@/classes/api/gkill-api'
import type { Kyou } from '@/classes/datas/kyou'
import type { PluginHtmlViewProps } from '@/pages/views/plugin-html-view-props'

const PLUGIN_HTML = '<html><body>プラグイン本文</body></html>'

const get_plugin_content_html = vi.fn()

function make_kyou(id = 'plugin-kyou-1'): Kyou {
    return {
        id: id,
        rep_name: 'ClaudeCode',
        typed_plugin: { rep_name: 'ClaudeCode' },
    } as unknown as Kyou
}

function make_props(overrides: Partial<PluginHtmlViewProps> = {}): PluginHtmlViewProps {
    return {
        gkill_api: {},
        application_config: { use_dark_theme: false },
        kyou: make_kyou(),
        highlight_targets: [],
        enable_context_menu: true,
        enable_dialog: true,
        // Ryuu / rykv詳細ペイン / KyouDialog はどれも文字列を渡す = ローダー + 注入経路
        height: 'auto',
        width: 'auto',
        ...overrides,
    } as unknown as PluginHtmlViewProps
}

/** load_html() の await を回しきる */
async function flush(wrapper: VueWrapper): Promise<void> {
    for (let i = 0; i < 4; i++) {
        await Promise.resolve()
    }
    await wrapper.vm.$nextTick()
}

function iframe_of(wrapper: VueWrapper): HTMLIFrameElement {
    const iframe = wrapper.find('iframe')
    expect(iframe.exists(), 'iframe が描画されていない').toBe(true)
    return iframe.element as HTMLIFrameElement
}

/**
 * iframe から親へのメッセージを模す。
 * onWindowMessage は source で自分の iframe か判定するので、そこまで含めて再現する。
 */
function post_from_iframe(iframe: HTMLIFrameElement, data: Record<string, unknown>): void {
    window.dispatchEvent(new MessageEvent('message', { data: data, source: iframe.contentWindow }))
}

/** iframe へ送られた本文注入メッセージの回数 */
function injected_count(post_message: ReturnType<typeof vi.fn>): number {
    return post_message.mock.calls.filter(
        (call) => typeof (call[0] as Record<string, unknown>)?.gkill_plugin_html === 'string',
    ).length
}

/** iframe へ送られたテーマ通知の回数 */
function theme_count(post_message: ReturnType<typeof vi.fn>): number {
    return post_message.mock.calls.filter(
        (call) => typeof (call[0] as Record<string, unknown>)?.gkill_theme === 'string',
    ).length
}

/** contentWindow.postMessage を差し替える。attachTo しないと contentWindow が生えない */
function spy_post_message(iframe: HTMLIFrameElement): ReturnType<typeof vi.fn> {
    const content_window = iframe.contentWindow
    expect(content_window, 'iframe に browsing context が無い').toBeTruthy()
    const post_message = vi.fn()
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        ; (content_window as any).postMessage = post_message
    return post_message
}

async function mount_view(props: Partial<PluginHtmlViewProps> = {}) {
    const wrapper = mount(PluginHtmlView, {
        props: make_props(props) as never,
        attachTo: document.body,
    })
    const iframe = iframe_of(wrapper)
    const post_message = spy_post_message(iframe)
    await flush(wrapper)
    return { wrapper: wrapper, iframe: iframe, post_message: post_message }
}

beforeEach(() => {
    get_plugin_content_html.mockReset()
    get_plugin_content_html.mockResolvedValue({ html: PLUGIN_HTML, errors: null, messages: null })
    GkillAPI.set_gkill_api({
        get_session_id: () => 'session',
        get_plugin_content_html: get_plugin_content_html,
    } as never)
})

afterEach(() => {
    document.body.innerHTML = ''
})

describe('本文の注入はローダーの合図を待つ', () => {
    // 修正前は html が届いた時点で無条件に送り、送信済み印を立てていた。
    // ローダーがまだリスナーを張っていないとメッセージは消え、
    // その後の @load でも送信済み印に阻まれて送り直さないため空箱のままだった
    test('ready を受けてから本文が送られる', async () => {
        const { wrapper, iframe, post_message } = await mount_view()

        const before = injected_count(post_message)
        post_from_iframe(iframe, { gkill_plugin_loader_ready: true })
        await wrapper.vm.$nextTick()

        expect(
            injected_count(post_message),
            'ローダーが名乗ったのに本文を送っていない（空箱のまま）',
        ).toBeGreaterThan(before)
        const last = post_message.mock.calls[post_message.mock.calls.length - 1][0] as Record<string, string>
        expect(last.gkill_plugin_html).toContain('プラグイン本文')
    })

    // ローダーが読み込み直されたら、前に送ったぶんは document ごと失われている。
    // 送信済み印を落として送り直せないと、ここで永久に空になる
    test('ready が来るたびに送り直す', async () => {
        const { wrapper, iframe, post_message } = await mount_view()

        post_from_iframe(iframe, { gkill_plugin_loader_ready: true })
        await wrapper.vm.$nextTick()
        const after_first = injected_count(post_message)

        post_from_iframe(iframe, { gkill_plugin_loader_ready: true })
        await wrapper.vm.$nextTick()

        expect(
            injected_count(post_message),
            '2回目の ready で送り直していない（読み込み直したローダーに本文が入らない）',
        ).toBeGreaterThan(after_first)
    })

    test('本文より先に ready が来ても取りこぼさない', async () => {
        // 本文の取得を保留しておき、ready のほうを先に届かせる
        let resolve_html: (value: unknown) => void = () => { }
        get_plugin_content_html.mockReturnValue(new Promise((resolve) => { resolve_html = resolve }))

        const wrapper = mount(PluginHtmlView, { props: make_props() as never, attachTo: document.body })
        const iframe = iframe_of(wrapper)
        const post_message = spy_post_message(iframe)

        post_from_iframe(iframe, { gkill_plugin_loader_ready: true })
        await wrapper.vm.$nextTick()
        expect(injected_count(post_message), '本文がまだ無いのに送っている').toBe(0)

        resolve_html({ html: PLUGIN_HTML, errors: null, messages: null })
        await flush(wrapper)

        expect(injected_count(post_message), '先に来ていた ready を忘れている').toBe(1)
    })

    test('同じ本文を ready 抜きで二重送信しない', async () => {
        const { wrapper, iframe, post_message } = await mount_view()

        post_from_iframe(iframe, { gkill_plugin_loader_ready: true })
        await wrapper.vm.$nextTick()
        const after_ready = injected_count(post_message)

        // 本文が書き込まれたあとの load 相当。ここで送り直すと注入ループになる
        await iframe.dispatchEvent(new Event('load'))
        await wrapper.vm.$nextTick()

        expect(injected_count(post_message), 'load のたびに送り直すと注入ループになる').toBe(after_ready)
    })
})

describe('テーマ通知', () => {
    // 本文が描かれた合図（最初のサイズ通知）でテーマを送る。
    // iframe の load だけに頼ると、document.close() が2度目の load を焚かないブラウザで
    // 本文がライトテーマのまま残る
    test('最初のサイズ通知でテーマを送る', async () => {
        const { wrapper, iframe, post_message } = await mount_view()

        const before = theme_count(post_message)
        post_from_iframe(iframe, { gkill_iframe_size: { width: 300, height: 200 } })
        await wrapper.vm.$nextTick()

        expect(theme_count(post_message), '本文にテーマが届かない（ライトテーマのまま残る）')
            .toBe(before + 1)
    })

    // 本文側はテーマを受け取るとサイズを測り直して送ってくる。
    // サイズのたびに送り返すと10ms周期のピンポンが止まらなくなる
    test('2回目以降のサイズ通知では送り返さない', async () => {
        const { wrapper, iframe, post_message } = await mount_view()

        post_from_iframe(iframe, { gkill_iframe_size: { width: 300, height: 200 } })
        await wrapper.vm.$nextTick()
        const after_first = theme_count(post_message)

        post_from_iframe(iframe, { gkill_iframe_size: { width: 300, height: 240 } })
        post_from_iframe(iframe, { gkill_iframe_size: { width: 300, height: 260 } })
        await wrapper.vm.$nextTick()

        expect(theme_count(post_message), 'テーマとサイズのピンポンが止まらなくなる').toBe(after_first)
    })
})

describe('iframe 内のダブルクリックを親へ流す', () => {
    test('gkill_iframe_dblclick で本物の dblclick が親へ伝播する', async () => {
        const { wrapper, iframe } = await mount_view()

        const seen = new Array<Event>()
        wrapper.element.addEventListener('dblclick', (e) => seen.push(e))

        post_from_iframe(iframe, { gkill_iframe_dblclick: true })
        await wrapper.vm.$nextTick()

        expect(seen, 'iframe 内のダブルクリックが親へ届いていない（KyouDialog が開かない）').toHaveLength(1)
        expect(seen[0].bubbles, 'bubbles でないと KyouView / RyuuItemView の @dblclick に届かない').toBe(true)
    })

    test('取得した本文には転送スクリプトが付く', async () => {
        const { wrapper, iframe, post_message } = await mount_view()

        post_from_iframe(iframe, { gkill_plugin_loader_ready: true })
        await wrapper.vm.$nextTick()

        const last = post_message.mock.calls[post_message.mock.calls.length - 1][0] as Record<string, string>
        expect(last.gkill_plugin_html, 'プラグイン側に手を入れずに転送するための注入が無い')
            .toContain('gkill_iframe_dblclick')
    })

    test('本文が空なら転送スクリプトだけを描かない', async () => {
        get_plugin_content_html.mockResolvedValue({ html: '', errors: null, messages: null })
        const { wrapper, iframe, post_message } = await mount_view()

        post_from_iframe(iframe, { gkill_plugin_loader_ready: true })
        await wrapper.vm.$nextTick()

        expect(injected_count(post_message), '中身が無いのに空の iframe を出している').toBe(0)
    })
})

describe('一覧（height が数値）は srcdoc 直書きのまま', () => {
    test('注入経路を使わない', async () => {
        const { wrapper, iframe, post_message } = await mount_view({ height: 120 })

        post_from_iframe(iframe, { gkill_plugin_loader_ready: true })
        await wrapper.vm.$nextTick()

        expect(injected_count(post_message), '一覧で postMessage 注入に切り替わっている').toBe(0)
        expect(iframe.getAttribute('srcdoc'), 'srcdoc に本文が入っていない').toContain('プラグイン本文')
    })
})
