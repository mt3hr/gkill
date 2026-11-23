'use strict'

import { computed, onBeforeUnmount, type Ref, ref, watch } from 'vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'

/**
 * Kyou 詳細ペイン（と、その下にぶら下がる Ryuu ペイン）の寸法。
 *
 * **ビューポート単位（100vh / 100vw）とメディアクエリを使わない。**
 * 詳細ペインを持つビューはポート（rudbeckia）のフローティングダイアログの中でも描かれ、
 * そこでは基準が画面ではなくダイアログの箱になる。ダイアログの幅はメディアクエリからは
 * 見えないので、すべて渡された寸法から作る（`gkill-client-rudbeckia` の不変条件）。
 *
 * 通常の rykv（rykv-view）と共有された rykv（shared-rykv-view）の2箇所で同じ計算を使う。
 * 片方だけ直すと「ポートの中でだけ詳細ペインの高さがずれる」が静かに起きるので、
 * 計算はここ1箇所に置く。
 */
export function useRykvDetailPaneMetrics(options: {
    application_config: () => ApplicationConfig,
    app_content_height: () => number,
    app_content_width: () => number,
}) {
    const { application_config, app_content_height, app_content_width } = options

    /** 詳細ペインの実寸の幅。Ryuu ペインの幅をこれへ揃える */
    const kyou_detail_view_width: Ref<number> = ref(400) // 詳細ペインの初期幅とあわせる
    /** 詳細ペインの要素。CSS の resize でユーザが幅を変えるので実寸を追う */
    const kyou_detail_view_element = ref<HTMLElement | null>(null)

    /**
     * Ryuu（関連する記録）の定義が空か。
     * 空のときは Ryuu ペインの高さを0にして、詳細ペインへ全部渡す。
     */
    const is_ryuu_empty = computed(() => {
        const data = application_config().ryuu_json_data
        if (!data || !Array.isArray(data) || data.length === 0) return true
        return data.every((item: unknown) => {
            if (Array.isArray(item)) return item.length === 0
            if (item && typeof item === 'object' && 'queries' in item) {
                return !item.queries || !Array.isArray(item.queries) || item.queries.length === 0
            }
            return false
        })
    })

    const ryuu_pane_height_px = computed(() => is_ryuu_empty.value ? '0px' : `${app_content_height() * 0.2}px`)
    const kyou_detail_view_height_px = computed(() => {
        const ryuu_height = is_ryuu_empty.value ? 0 : app_content_height() * 0.2
        return `${app_content_height() - ryuu_height}px`
    })
    const app_content_width_px = computed(() => `${app_content_width()}px`)
    // 旧 `@media (max-width: 600px) { width: 100vw }` の置き換え。min-width は旧実装どおり触らない
    const kyou_detail_view_base_width_px = computed(() => app_content_width() <= 600 ? app_content_width_px.value : '400px')
    const ryuu_pane_width_px = computed(() => `${kyou_detail_view_width.value}px`)

    // 表示切り替えで element ごと作り直されるので、ref を watch して付け替える
    let kyou_detail_view_resize_observer: ResizeObserver | null = null
    watch(kyou_detail_view_element, (element, old_element) => {
        if (kyou_detail_view_resize_observer && old_element) {
            try { kyou_detail_view_resize_observer.unobserve(old_element) } catch { /* noop */ }
        }
        if (element) {
            if (!kyou_detail_view_resize_observer) {
                kyou_detail_view_resize_observer = new ResizeObserver((entries) => {
                    for (const entry of entries) {
                        kyou_detail_view_width.value = entry.borderBoxSize?.[0]?.inlineSize ?? entry.contentRect.width
                    }
                })
            }
            kyou_detail_view_resize_observer.observe(element)
        }
    }, { flush: 'post' })
    onBeforeUnmount(() => {
        kyou_detail_view_resize_observer?.disconnect()
        kyou_detail_view_resize_observer = null
    })

    return {
        // Template refs
        kyou_detail_view_element,

        // State
        kyou_detail_view_width,

        // Computed
        is_ryuu_empty,
        ryuu_pane_height_px,
        ryuu_pane_width_px,
        kyou_detail_view_height_px,
        kyou_detail_view_base_width_px,
        app_content_width_px,
    }
}
