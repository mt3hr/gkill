import { describe, test, expect, afterEach, vi } from 'vitest'
import { classify_device_kind, type DeviceKind, type DeviceKindEnv } from '@/classes/use-device-kind'

// 実機のUA文字列。回帰を止めるために書き換えないこと。
const ua = {
    windows_chrome: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36',
    mac_safari: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Safari/605.1.15',
    electron: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) gkill/1.0.0 Chrome/126.0.6478.183 Electron/31.3.0 Safari/537.36',
    ipad_os17: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15',
    ipad_ios12: 'Mozilla/5.0 (iPad; CPU OS 12_5_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/12.1.2 Mobile/15E148 Safari/604.1',
    iphone: 'Mozilla/5.0 (iPhone; CPU iPhone OS 17_5_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1',
    android_phone: 'Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Mobile Safari/537.36',
    android_tablet: 'Mozilla/5.0 (Linux; Android 13; SM-X710) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36',
    android_webview_phone: 'Mozilla/5.0 (Linux; Android 14; Pixel 8 Build/AP2A.240905.003; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/131.0.0.0 Mobile Safari/537.36',
    android_webview_tablet: 'Mozilla/5.0 (Linux; Android 13; SM-X710 Build/TP1A.220624.014; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/131.0.0.0 Safari/537.36',
    firefox_android_tablet: 'Mozilla/5.0 (Android 13; Tablet; rv:132.0) Gecko/132.0 Firefox/132.0',
} as const

function make_env(overrides: Partial<DeviceKindEnv>): DeviceKindEnv {
    return {
        has_media_query: true,
        any_pointer_fine: false,
        any_hover_hover: false,
        max_touch_points: 0,
        user_agent: '',
        ua_data_mobile: null,
        screen_short_side: 0,
        ...overrides,
    }
}

/** マウス/トラックパッドがある状態 */
const with_fine_pointer = { any_pointer_fine: true, any_hover_hover: true }
/** タッチだけの状態 */
const touch_only = { any_pointer_fine: false, any_hover_hover: false }

describe('classify_device_kind', () => {
    const cases: Array<[string, DeviceKindEnv, DeviceKind]> = [
        // Step 0: ブラウザ実行環境ではない
        ['matchMediaが無い(jsdom/SSR)', make_env({ has_media_query: false, screen_short_side: 768 }), 'pc'],

        // Step 2: 精密ポインタがあるPC
        ['非タッチのWindows Chrome', make_env({ ...with_fine_pointer, user_agent: ua.windows_chrome, max_touch_points: 0, screen_short_side: 1080 }), 'pc'],
        ['タッチ搭載Windowsノート', make_env({ ...with_fine_pointer, user_agent: ua.windows_chrome, max_touch_points: 10, screen_short_side: 1080 }), 'pc'],
        ['macOS Safari', make_env({ ...with_fine_pointer, user_agent: ua.mac_safari, max_touch_points: 0, screen_short_side: 1080 }), 'pc'],
        ['Electron(デスクトップ版gkill)', make_env({ ...with_fine_pointer, user_agent: ua.electron, max_touch_points: 0, screen_short_side: 1080 }), 'pc'],
        ['iPad + トラックパッド', make_env({ ...with_fine_pointer, user_agent: ua.ipad_os17, max_touch_points: 5, screen_short_side: 820 }), 'pc'],

        // Step 3: タブレット
        ['素のiPad(iPadOS 17のデスクトップUA)', make_env({ ...touch_only, user_agent: ua.ipad_os17, max_touch_points: 5, screen_short_side: 820 }), 'tablet'],
        ['iPad(iOS 12のモバイルUA)', make_env({ ...touch_only, user_agent: ua.ipad_ios12, max_touch_points: 5, screen_short_side: 768 }), 'tablet'],
        ['Androidタブレット', make_env({ ...touch_only, user_agent: ua.android_tablet, max_touch_points: 10, screen_short_side: 800 }), 'tablet'],
        ['Androidタブレット(gkillアプリのWebView)', make_env({ ...touch_only, user_agent: ua.android_webview_tablet, max_touch_points: 10, screen_short_side: 800 }), 'tablet'],
        ['Firefox Androidタブレット', make_env({ ...touch_only, user_agent: ua.firefox_android_tablet, max_touch_points: 10, screen_short_side: 800 }), 'tablet'],

        // Step 1: スマートフォン
        ['iPhone Safari', make_env({ ...touch_only, user_agent: ua.iphone, max_touch_points: 5, screen_short_side: 393 }), 'smart_phone'],
        ['Androidスマートフォン', make_env({ ...touch_only, user_agent: ua.android_phone, max_touch_points: 5, screen_short_side: 412 }), 'smart_phone'],
        ['Androidスマートフォン(gkillアプリのWebView)', make_env({ ...touch_only, user_agent: ua.android_webview_phone, max_touch_points: 5, screen_short_side: 412 }), 'smart_phone'],
        // Step 1 を Step 2 より前に置いていることの証明
        ['スタイラス対応Androidスマートフォン', make_env({ ...with_fine_pointer, user_agent: ua.android_phone, max_touch_points: 5, screen_short_side: 412 }), 'smart_phone'],
        // UA-CH が UA 文字列より優先されることの証明
        ['UA-CHがmobileを主張', make_env({ ...with_fine_pointer, user_agent: ua.windows_chrome, ua_data_mobile: true, screen_short_side: 412 }), 'smart_phone'],

        // Step 4: 未知のタッチ専用端末
        ['未知のタッチ専用端末(短辺800)', make_env({ ...touch_only, user_agent: 'UnknownBrowser/1.0', max_touch_points: 10, screen_short_side: 800 }), 'tablet'],
        ['未知のタッチ専用端末(短辺600)', make_env({ ...touch_only, user_agent: 'UnknownBrowser/1.0', max_touch_points: 10, screen_short_side: 600 }), 'tablet'],
        ['未知のタッチ専用端末(短辺599)', make_env({ ...touch_only, user_agent: 'UnknownBrowser/1.0', max_touch_points: 10, screen_short_side: 599 }), 'smart_phone'],
        ['未知のタッチ専用端末(短辺390)', make_env({ ...touch_only, user_agent: 'UnknownBrowser/1.0', max_touch_points: 10, screen_short_side: 390 }), 'smart_phone'],
    ]

    test.each(cases)('%s -> %s', (_name, env, expected) => {
        expect(classify_device_kind(env)).toBe(expected)
    })

    test('any-pointerとany-hoverの片方だけではpcにしない', () => {
        const base = { user_agent: 'UnknownBrowser/1.0', max_touch_points: 10, screen_short_side: 800 }
        expect(classify_device_kind(make_env({ ...base, any_pointer_fine: true, any_hover_hover: false }))).toBe('tablet')
        expect(classify_device_kind(make_env({ ...base, any_pointer_fine: false, any_hover_hover: true }))).toBe('tablet')
    })
})

// ── useDeviceKind のシングルトン性 / リアクティブ性 ──
// 状態がモジュールレベルなので、ケースごとに vi.resetModules() + 動的importが必須。
// 静的importだと最初のケースの状態が全ケースに漏れる。

interface FakeMediaQueryList {
    matches: boolean
    media: string
    listeners: Array<() => void>
    addEventListener: (type: string, listener: () => void) => void
}

function stub_match_media(initial: Record<string, boolean>) {
    const lists = new Map<string, FakeMediaQueryList>()
    const match_media = vi.fn((query: string): FakeMediaQueryList => {
        const existing = lists.get(query)
        if (existing) {
            return existing
        }
        const created: FakeMediaQueryList = {
            matches: initial[query] ?? false,
            media: query,
            listeners: [],
            addEventListener: (_type, listener) => {
                created.listeners.push(listener)
            },
        }
        lists.set(query, created)
        return created
    })
    vi.stubGlobal('matchMedia', match_media)

    return {
        match_media: match_media,
        set_matches: (query: string, matches: boolean) => {
            const list = lists.get(query)
            if (!list) {
                throw new Error(`matchMedia(${query}) がまだ呼ばれていない`)
            }
            list.matches = matches
            for (const listener of list.listeners) {
                listener()
            }
        },
    }
}

async function import_fresh() {
    vi.resetModules()
    return await import('@/classes/use-device-kind')
}

describe('useDeviceKind', () => {
    afterEach(() => {
        vi.unstubAllGlobals()
    })

    test('何度呼んでも同一のオブジェクト参照を返し、matchMediaの購読が増えない', async () => {
        const { match_media } = stub_match_media({ '(any-pointer: fine)': true, '(any-hover: hover)': true })
        const { useDeviceKind } = await import_fresh()

        const first = useDeviceKind()
        const call_count_after_first = match_media.mock.calls.length
        for (let i = 0; i < 50; i++) {
            expect(useDeviceKind()).toBe(first)
        }
        expect(match_media.mock.calls.length).toBe(call_count_after_first)
    })

    test('メディアクエリの変化で、先に取得済みのrefも更新される', async () => {
        const { set_matches } = stub_match_media({ '(any-pointer: fine)': false, '(any-hover: hover)': false })
        const { useDeviceKind } = await import_fresh()

        const { device_kind, is_pc, is_tablet } = useDeviceKind()
        // UAはjsdomの既定(Mozilla/5.0 ... jsdom/xx)なのでStep 4に落ちる
        expect(is_pc.value).toBe(false)

        set_matches('(any-pointer: fine)', true)
        set_matches('(any-hover: hover)', true)

        expect(device_kind.value).toBe('pc')
        expect(is_pc.value).toBe(true)
        expect(is_tablet.value).toBe(false)
    })

    test('matchMediaが無い環境でも例外にならずpcになる', async () => {
        vi.stubGlobal('matchMedia', undefined)
        const { useDeviceKind } = await import_fresh()

        const { device_kind, is_pc, is_tablet, is_smart_phone, has_touch } = useDeviceKind()
        expect(device_kind.value).toBe('pc')
        expect(is_pc.value).toBe(true)
        expect(is_tablet.value).toBe(false)
        expect(is_smart_phone.value).toBe(false)
        // has_touch は device_kind とは独立した信号なので、matchMedia が無くても
        // ontouchstart を見て答える（jsdom は ontouchstart を生やすので true）
        expect(has_touch.value).toBe('ontouchstart' in window)
    })
})
