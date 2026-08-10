'use strict'

import { computed, shallowRef, type ComputedRef, type ShallowRef } from 'vue'

/**
 * 端末種別。
 *
 * enum ではなく文字列リテラル union にしているのは、ランタイム実体が不要で
 * テーブル駆動テストが `expect(...).toBe('tablet')` で書けるため。
 */
export type DeviceKind = 'pc' | 'tablet' | 'smart_phone'

/**
 * 端末種別の判定に必要な入力だけを抜き出したもの。
 *
 * `classify_device_kind` を純粋関数に保つための境界。実環境の読み取りは
 * `read_env()` に閉じ込めてあるので、テストはグローバルを差し替えずに済む
 * （`foldable-struct-move.ts` と同じ流儀）。
 */
export interface DeviceKindEnv {
    /** matchMedia が使えるか。false なら jsdom / SSR とみなす */
    has_media_query: boolean
    /** `(any-pointer: fine)` — マウス / トラックパッド / アクティブペンのいずれかがある */
    any_pointer_fine: boolean
    /** `(any-hover: hover)` — ホバーできる入力デバイスがある */
    any_hover_hover: boolean
    max_touch_points: number
    user_agent: string
    /** `navigator.userAgentData.mobile`。未対応ブラウザでは null */
    ua_data_mobile: boolean | null
    /** 物理スクリーンの短辺(px) */
    screen_short_side: number
}

/** タブレットとスマートフォンを分ける短辺の閾値。Android の sw600dp に合わせている。 */
const tablet_short_side_threshold = 600

/**
 * 端末種別を判定する。
 *
 * 判定順序そのものが仕様なので、並べ替えないこと。
 *
 * 0. matchMedia が無い（jsdom / SSR）→ pc
 * 1. UA / UA-CH でスマートフォンを先に確定する
 * 2. 精密ポインタ（fine かつ hover）があれば pc
 * 3. UA でタブレットを確定する
 * 4. 未知のタッチ専用端末は画面短辺で分ける
 *
 * Step 1 を Step 2 より前に置いているのは、スタイラス対応スマートフォン
 * （Galaxy の S-Pen 等）が `any-pointer: fine` / `any-hover: hover` を報告して
 * pc に誤分類されるのを防ぐため。
 *
 * 逆に Step 2 を Step 3 より前に置いているのは、
 * 「iPad にトラックパッド / キーボードを繋いだらマウス操作ができるので pc 扱いにする」
 * という方針を採ったため。素の iPad は `any-pointer: coarse` / `any-hover: none`
 * なので Step 3 に落ちて tablet になる。Android タブレット + マウスも同じ規則で
 * pc になるが、同一ルールの帰結として許容している。
 */
export function classify_device_kind(env: DeviceKindEnv): DeviceKind {
    // Step 0: ブラウザ実行環境ではない（jsdom / SSR）。
    // jsdom は ontouchstart を生やす一方で screen が 0x0 なので、ここで打ち切らないと
    // 「タッチできる短辺0pxの端末」＝スマートフォン判定になってしまう。
    // 実機の情報が何も無いときはデスクトップとみなすのが既定として妥当。
    // 特定の端末種別を前提にするテストは matchMedia をスタブすること。
    if (!env.has_media_query) {
        return 'pc'
    }

    const ua = env.user_agent

    // Step 1: スマートフォン。
    // User-Agent Client Hints は UA 文字列より信頼できるので最優先。
    if (env.ua_data_mobile === true) {
        return 'smart_phone'
    }
    if (/iPhone|iPod/.test(ua)) {
        return 'smart_phone'
    }
    // Chrome はスマートフォンのとき UA に Mobile を含め、タブレットでは落とす。
    // これが Android のスマホ/タブレット判別の唯一実用的な信号。
    // gkill 公式 Android アプリの WebView は UA に "; wv" が挿入されるだけで
    // このルールは変わらないため、WebView の特別扱いは要らない。
    if (/Android/.test(ua) && /Mobile/.test(ua)) {
        return 'smart_phone'
    }

    // Step 2: ネイティブHTML5 D&Dを実行できる入力デバイスがあるか。
    // primary の pointer / hover ではなく any-pointer / any-hover を見ているのは、
    // タッチパネル搭載Windowsノートは OS 設定次第で primary が coarse に振れる一方、
    // any-pointer: fine はマウス / トラックパッド / アクティブペンのいずれかがあれば
    // 必ず true になるため。fine と hover の AND を取るのは保守側に倒すため。
    if (env.any_pointer_fine && env.any_hover_hover) {
        return 'pc'
    }

    // Step 3: タブレット。
    if (/iPad/.test(ua)) {
        return 'tablet'
    }
    // iPadOS 13以降は UA が Macintosh を名乗る。タッチスクリーン搭載Macは存在せず
    // 実機Macの maxTouchPoints は 0 なので、これで判別できる
    // （将来Appleがタッチ対応Macを出すと崩れる）。
    if (/Macintosh/.test(ua) && env.max_touch_points > 1) {
        return 'tablet'
    }
    if (/Android/.test(ua)) {
        return 'tablet'
    }
    if (/Tablet|Silk|PlayBook|Kindle/.test(ua)) {
        return 'tablet'
    }

    // Step 4: 未知のタッチ専用端末。
    // 閾値は端末クラスの境界であって、use-rykv-view.ts / use-mi-view.ts の 760px
    // （ナビゲーションドロワー開閉のレイアウト分岐点）とは別物。流用しないこと。
    return env.screen_short_side >= tablet_short_side_threshold ? 'tablet' : 'smart_phone'
}

const any_pointer_fine_query = '(any-pointer: fine)'
const any_hover_hover_query = '(any-hover: hover)'

function read_env(): DeviceKindEnv {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
        return {
            has_media_query: false,
            any_pointer_fine: false,
            any_hover_hover: false,
            max_touch_points: 0,
            user_agent: '',
            ua_data_mobile: null,
            screen_short_side: 0,
        }
    }

    // 短辺は screen 基準で取る。innerWidth と違ってウィンドウリサイズでも
    // 画面回転でも min() が変わらないので、resize を購読しなくてよくなる。
    // screen が 0 を返す環境（jsdom など）ではウィンドウサイズに退避する。
    const screen_short_side = window.screen && window.screen.width > 0 && window.screen.height > 0
        ? Math.min(window.screen.width, window.screen.height)
        : Math.min(window.innerWidth, window.innerHeight)

    const ua_data_mobile = navigator.userAgentData?.mobile
    return {
        has_media_query: true,
        any_pointer_fine: window.matchMedia(any_pointer_fine_query).matches,
        any_hover_hover: window.matchMedia(any_hover_hover_query).matches,
        // jsdom では undefined になるので数値であることを確認する
        max_touch_points: typeof navigator.maxTouchPoints === 'number' ? navigator.maxTouchPoints : 0,
        user_agent: navigator.userAgent,
        ua_data_mobile: typeof ua_data_mobile === 'boolean' ? ua_data_mobile : null,
        screen_short_side: screen_short_side,
    }
}

function read_has_touch(): boolean {
    if (typeof window === 'undefined') {
        return false
    }
    return 'ontouchstart' in window || navigator.maxTouchPoints > 0
}

export interface DeviceKindState {
    device_kind: ShallowRef<DeviceKind>
    is_pc: ComputedRef<boolean>
    is_tablet: ComputedRef<boolean>
    is_smart_phone: ComputedRef<boolean>
    has_touch: ShallowRef<boolean>
}

let device_kind_state: DeviceKindState | null = null

function create_device_kind_state(): DeviceKindState {
    const device_kind = shallowRef<DeviceKind>(classify_device_kind(read_env()))
    const has_touch = shallowRef<boolean>(read_has_touch())

    if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
        const update = () => {
            device_kind.value = classify_device_kind(read_env())
            has_touch.value = read_has_touch()
        }
        for (const query of [any_pointer_fine_query, any_hover_hover_query]) {
            const media_query_list = window.matchMedia(query)
            // 旧Safariの addListener フォールバックは対象ブラウザ的に不要。
            // 使えない環境では初期値のまま固定される。
            if (typeof media_query_list.addEventListener === 'function') {
                // ここで登録したリスナは意図的に解除しない。この状態はページ寿命と
                // 同じシングルトンで、リスナも常にこの2本だけ。コンポーネントの破棄に
                // 合わせて onScopeDispose で外すと、生き残っている他のコンポーネントの
                // 判定が更新されなくなる。
                media_query_list.addEventListener('change', update)
            }
        }
    }

    return {
        device_kind: device_kind,
        is_pc: computed(() => device_kind.value === 'pc'),
        is_tablet: computed(() => device_kind.value === 'tablet'),
        is_smart_phone: computed(() => device_kind.value === 'smart_phone'),
        has_touch: has_touch,
    }
}

/**
 * 端末種別（PC / タブレット / スマートフォン）とタッチの有無。
 *
 * 状態はモジュールレベルのシングルトンで、初回呼び出し時に一度だけ生成する
 * （モジュール評価時に window を触ると SSR / vitest の module graph で落ちるため遅延）。
 * `foldable-struct.vue` はツリーのノード数ぶん再帰インスタンス化されるので、
 * 呼び出しごとに ref や matchMedia リスナを作ってはいけない。この関数は
 * 毎回同一のオブジェクト参照を返すだけで、追加のアロケーションをしない。
 *
 * `is_pc` と `has_touch` は別の概念なので使い分けること。
 * ドラッグ&ドロップの可否は `is_pc`、タッチ端末向けの代替導線（ロングプレスでの
 * コンテキストメニュー補完など）は `has_touch` で判断する。
 */
export function useDeviceKind(): DeviceKindState {
    if (!device_kind_state) {
        device_kind_state = create_device_kind_state()
    }
    return device_kind_state
}
