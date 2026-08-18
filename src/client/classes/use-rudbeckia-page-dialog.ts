'use strict'

import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch, type Ref } from 'vue'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'
import type { RudbeckiaPageDialogProps } from '@/pages/dialogs/rudbeckia-page-dialog-props'
import type { RudbeckiaPageDialogEmits } from '@/pages/dialogs/rudbeckia-page-dialog-emits'
import type { RudbeckiaPageKind } from '@/pages/views/rudbeckia-page-kind'
import type { KyouChangeChannel } from '@/classes/kyou-change-bus'

/** ダイアログ自身のヘッダの高さ。中のビューへ渡す高さから差し引く */
const DIALOG_HEADER_HEIGHT = 40
/** ホストしたビューが自前で出すアプリバーの高さ。単独ページと同じ値 */
const HOSTED_APP_BAR_HEIGHT = 50
/** 2枚目以降を中央からずらす量。メモ帳ウィンドウと同じ */
const RUDBECKIA_PAGE_DIALOG_CASCADE_STEP = 28

/**
 * ポート（開発コード rudbeckia）の画面ウィンドウ1枚。
 *
 * 枠の作り方は use-mkfl-dialog.ts と同じ。違うのは寸法の決め方で、
 * 未リサイズ時の幅と高さを **非スコープCSSで固定してある**（rudbeckia-page-dialog.vue の
 * `<style>`）ので、ResizeObserver の実測値をそのまま子へ渡してよい。
 * kftl-dialog.vue の「userSize が無いときは既定値」ガードは、高さがコンテンツ依存で
 * 子の高さ減算によって毎フレーム縮む場合の回避策であって、ここでは要らない。
 * **CSSで固定するのとガードを残すのは、どちらか片方だけ**にすること
 * （両方やると固定した高さが無視される）。
 */
export function useRudbeckiaPageDialog(options: {
    props: RudbeckiaPageDialogProps
    emits: RudbeckiaPageDialogEmits
}) {
    const { props, emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    // ×・Escape・ブラウザバックのどれで閉じてもここが1回だけ呼ばれる
    useDialogHistoryStack(is_show_dialog, { onClosed: () => emits('closed') })

    // slot 0 は素の名前、以降は枝番。同じキーで2枚出すと位置とサイズを奪い合う
    const floating_dialog_key = computed(() => props.slot_index === 0
        ? `rudbeckia-${props.kind}-dialog`
        : `rudbeckia-${props.kind}-dialog-${props.slot_index + 1}`)
    const ui = useFloatingDialog(floating_dialog_key.value, {
        centerMode: 'always',
        // ずらすのは cascade_index。slot_index は種類ごとの採番なので、
        // それで決めると4種類とも 0 になり、全部が同じ位置に重なる
        centerOffset: {
            x: props.cascade_index * RUDBECKIA_PAGE_DIALOG_CASCADE_STEP,
            y: props.cascade_index * RUDBECKIA_PAGE_DIALOG_CASCADE_STEP,
        },
        onEscape: () => hide(),
    })

    /**
     * 列の検索条件とスクロール位置の保存キーの枝番。
     * slot 0 は空文字＝従来キーそのままで、単独ページで作った列をそのまま引き継ぐ。
     * uuid にしてはいけない（毎回別のキーになり、列が復元できずキーが増え続ける）
     */
    const column_state_instance_key = computed(() => props.slot_index === 0 ? '' : String(props.slot_index + 1))

    /**
     * このウィンドウがバスへ繋がる口。computed なので依存が変わらない限り同じオブジェクトが返る。
     * 毎回作り直すと、それを見ている購読側のウォッチャが無意味に発火する
     */
    const kyou_change_channel = computed<KyouChangeChannel | null>(() => props.kyou_change_bus
        ? { bus: props.kyou_change_bus, origin_id: props.origin_id }
        : null)

    // ── 寸法 ──
    const dialog_body_ref = ref<HTMLElement | null>(null)
    const observed_body_width = ref(0)
    const observed_body_height = ref(0)

    // 実測が入るまでの繋ぎ。CSS の既定サイズ（min(1200px, 92vw) × 88vh）と揃えておく
    const default_view_width = computed(() => Math.min(props.app_content_width.valueOf() * 0.92, 1200))
    const default_view_height = computed(() => (props.app_content_height.valueOf() + HOSTED_APP_BAR_HEIGHT) * 0.88)

    /** `<v-layout>` に渡す箱の大きさ。ホストしたビューのアプリバーもこの中に入る */
    const layout_width = computed(() => observed_body_width.value > 0 ? observed_body_width.value : default_view_width.value)
    const layout_height = computed(() => observed_body_height.value > 0 ? observed_body_height.value : default_view_height.value)

    /** ビューへ渡す内容領域。ビュー自身のアプリバーぶんを差し引く */
    const view_width = computed(() => layout_width.value)
    const view_height = computed(() => Math.max(0, layout_height.value - HOSTED_APP_BAR_HEIGHT))

    let body_resize_observer: ResizeObserver | null = null
    watch(dialog_body_ref, (element, old_element) => {
        if (body_resize_observer && old_element) {
            try { body_resize_observer.unobserve(old_element) } catch { /* noop */ }
        }
        if (element) {
            if (!body_resize_observer) {
                body_resize_observer = new ResizeObserver((entries) => {
                    for (const entry of entries) {
                        observed_body_width.value = entry.contentRect.width
                        observed_body_height.value = entry.contentRect.height
                    }
                })
            }
            body_resize_observer.observe(element)
        }
    }, { flush: 'post' })
    onBeforeUnmount(() => {
        body_resize_observer?.disconnect()
        body_resize_observer = null
    })

    watch(ui.isTransparent, () => {
        ui.resetSize()
        nextTick(() => ui.resetToCenter())
    })

    // ダイアログのヘッダにタイトルは出さない（gkill のフローティングダイアログ共通）。
    // ヘルプだけ画面ごとに違うページを開く
    const help_screen_names: Record<RudbeckiaPageKind, string> = {
        rykv: 'rykv',
        mi: 'mi',
        plaing: 'plaing',
        dashboard: 'dashboard',
    }
    const help_screen_name = computed(() => help_screen_names[props.kind])

    // ── Business logic ──
    async function show(): Promise<void> {
        is_show_dialog.value = true
    }

    async function hide(): Promise<void> {
        await close_dialog_via_history(is_show_dialog)
    }

    // ホストは配列へ積むだけ。開くのはダイアログ自身（メモ帳ウィンドウと同じ）
    onMounted(() => { show() })

    return {
        // State
        is_show_dialog,
        ui,
        dialog_body_ref,

        // Computed
        column_state_instance_key,
        kyou_change_channel,
        layout_width,
        layout_height,
        view_width,
        view_height,
        help_screen_name,

        // Business logic
        show,
        hide,
    }
}

export { DIALOG_HEADER_HEIGHT, HOSTED_APP_BAR_HEIGHT, RUDBECKIA_PAGE_DIALOG_CASCADE_STEP }
