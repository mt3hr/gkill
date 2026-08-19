'use strict'

import { computed, nextTick, onBeforeUnmount, onMounted, type Ref, ref, watch } from 'vue'
import type { KFTLDialogEmits } from '@/pages/dialogs/kftl-dialog-emits'
import type { KFTLDialogProps } from '@/pages/dialogs/kftl-dialog-props'
import KFTLView from '@/pages/views/kftl-view.vue'
import HelpDialog from '@/pages/dialogs/help-dialog.vue'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

// 2枚目以降を中央から少しずつずらす量(px)。ずらさないとピクセル単位で完全に重なる
const KFTL_DIALOG_CASCADE_STEP = 28

export function useKFTLDialog(options: {
    props: KFTLDialogProps
    emits: KFTLDialogEmits
}) {
    const { props, emits } = options

    const kftl_view = ref<InstanceType<typeof KFTLView> | null>(null);
    const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)
    const dialog_body_ref = ref<HTMLElement | null>(null)
    const default_view_width = computed(() => Math.min(props.app_content_width.valueOf() * 0.85, 600))
    const default_view_height = computed(() => props.app_content_height.valueOf() * 0.75)
    const observed_body_width = ref(0)
    const observed_body_height = ref(0)
    const view_width = computed(() => {
        if (observed_body_width.value > 0) {
            return observed_body_width.value
        }
        return default_view_width.value
    })
    const view_height = computed(() => {
        // userSize がある場合（ユーザーがリサイズ済み）はコンテナ高さが固定されているため、
        // observed_body_height をそのまま使っても循環しない。
        // userSize が null の場合（Cookie消去後等）はコンテナ高さがコンテンツ依存になり、
        // KFTLView内の action_height(10px) 減算で毎サイクル縮小する循環が発生するため、
        // default_view_height を使用する。
        if (ui.userSize.value && observed_body_height.value > 0) {
            return observed_body_height.value
        }
        return default_view_height.value
    })
    let body_ro: ResizeObserver | null = null
    watch(dialog_body_ref, (el, old_el) => {
        if (body_ro && old_el) { try { body_ro.unobserve(old_el) } catch { /* noop */ } }
        if (el) {
            if (!body_ro) {
                body_ro = new ResizeObserver((entries) => {
                    for (const entry of entries) {
                        observed_body_width.value = entry.contentRect.width
                        observed_body_height.value = entry.contentRect.height
                    }
                })
            }
            body_ro.observe(el)
        }
    }, { flush: 'post' })
    onBeforeUnmount(() => { body_ro?.disconnect(); body_ro = null })
    const is_show_dialog: Ref<boolean> = ref(false)
    // 閉じ方（×・Escape・ブラウザバック）を問わずちょうど1回。ホストがここで一覧から外す
    useDialogHistoryStack(is_show_dialog, { onClosed: () => emits('closed') })
    // 1枚目は従来のキーのまま（保存済みのサイズを引き継ぐ）。
    // 2枚目以降はスロットごとに分ける ―― 同じキーだと位置とサイズを奪い合う
    const floating_dialog_key = props.slot_index === 0 ? "kftl-dialog" : `kftl-dialog-${props.slot_index + 1}`
    const ui = useFloatingDialog(floating_dialog_key, {
        centerMode: "always",
        centerOffset: {
            x: props.slot_index * KFTL_DIALOG_CASCADE_STEP,
            y: props.slot_index * KFTL_DIALOG_CASCADE_STEP,
        },
        onEscape: () => hide(),
    })
    watch(ui.isTransparent, () => {
        ui.resetSize()
        nextTick(() => ui.resetToCenter())
    })
    async function show(): Promise<void> {
        is_show_dialog.value = true
        nextTick(() => kftl_view.value?.focus_kftl_text_area())
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }
    // ホストは配列へ1件足すだけ。開くのは自分の役目
    // （rykv-dialog-host-item と同じ形）
    onMounted(() => show())

    return {
        kftl_view,
        help_dialog,
        dialog_body_ref,
        view_width,
        view_height,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
