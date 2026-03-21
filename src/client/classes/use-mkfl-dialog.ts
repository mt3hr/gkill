'use strict'

import { computed, onBeforeUnmount, ref, watch, type Ref } from 'vue'
import type { MKFLDialogProps } from '@/pages/dialogs/mkfl-dialog-props'
import type { MKFLDialogEmits } from '@/pages/dialogs/mkfl-dialog-emits'
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'

export function useMKFLDialog(options: {
    props: MKFLDialogProps
    emits: MKFLDialogEmits
}) {
    const { props } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("mkfl-dialog", {
        centerMode: "always",
    })

    const dialog_body_ref = ref<HTMLElement | null>(null)
    const observed_body_width = ref(0)
    const observed_body_height = ref(0)

    const default_view_width = computed(() => Math.min(props.app_content_width.valueOf() * 0.85, 600))
    const default_view_height = computed(() => props.app_content_height.valueOf() * 0.85)

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
        // MKFLView内の action_height 減算で毎サイクル縮小する循環が発生するため、
        // default_view_height を使用する。
        if (ui.userSize.value && observed_body_height.value > 0) {
            return observed_body_height.value
        }
        return default_view_height.value
    })

    let body_ro: ResizeObserver | null = null
    watch(dialog_body_ref, (el, oldEl) => {
        if (body_ro && oldEl) { try { body_ro.unobserve(oldEl) } catch { /* noop */ } }
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

    async function show(): Promise<void> {
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        is_show_dialog.value = false
    }

    return {
        is_show_dialog,
        ui,
        dialog_body_ref,
        view_width,
        view_height,
        show,
        hide,
    }
}
