'use strict'

// 位置を手計算しない理由（実測しない限り正しい定数は書けない）:
// documents/adr/0039-context-menu-position-by-vuetify.md

import { computed, ref, type Ref } from 'vue'

/**
 * コンテキストメニュー（`*-context-menu.vue`）共通の開閉状態と位置。
 *
 * 位置は自前で計算しない。`menu_target` を `<v-menu :target="menu_target">` に渡すと、
 * Vuetify の connected location strategy が **メニューの実寸を測ったうえで**
 * ビューポートからはみ出す方向へ flip / shift してくれる。
 *
 * 以前は各composableが
 * `left: min(innerWidth - 130, x); top: min(max(50, innerHeight - (8 + 48 * N)), y)`
 * という見積り式を25箇所にコピペしていた。幅130pxは実際のリスト幅と無関係で、
 * 高さの `N` はテンプレートの項目数と手で同期する不文律だったため、
 * 項目を足すたびに静かにずれていた（struct系は実項目5個に対し `N=2` のままだった）。
 */
export function useContextMenuPosition() {
    const is_show: Ref<boolean> = ref(false)
    const position_x: Ref<number> = ref(0)
    const position_y: Ref<number> = ref(0)

    // Vuetifyは配列を 0×0 の Box として扱う（vuetify/lib/util/box.js の getTargetBox）
    const menu_target = computed<[number, number]>(() => [position_x.value, position_y.value])

    /**
     * 右クリック / 長押しの座標でメニューを開く。
     *
     * 座標はビューポート相対（`clientX` / `clientY`）でよい。Vuetify 側も
     * `getBoundingClientRect` ベースのビューポート座標で突き合わせている。
     */
    function open_at(e: MouseEvent): void {
        position_x.value = e.clientX
        position_y.value = e.clientY
        is_show.value = true
    }

    return {
        is_show,
        position_x,
        position_y,
        menu_target,
        open_at,
    }
}
