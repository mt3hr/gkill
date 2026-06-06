import { onMounted, onBeforeUnmount, type Ref } from 'vue'

// use-scoped-enter-for-kftl.ts と同じヘルパー関数を使う

function isTextInput(el: Element | null): boolean {
    if (!el) return false
    const he = el as HTMLElement
    const tag = he.tagName?.toLowerCase()
    if (he.isContentEditable) return true
    if (tag === 'input' || tag === 'textarea' || tag === 'select') return true
    if (he.getAttribute?.('role') === 'textbox') return true
    return false
}

function isInsideDialog(el: Element | null): boolean {
    if (!el) return false
    return !!el.closest('.gkill-floating-dialog, [role="dialog"][aria-modal="true"]')
}

function isButtonLike(el: Element | null): boolean {
    if (!el) return false
    return !!el.closest('button, [type="button"], [type="submit"], [role="button"], .v-btn')
}

function isAnyBlockingModalOpen(): boolean {
    const overlays = Array.from(document.querySelectorAll('[role="dialog"][aria-modal="true"]:not(.kyou_dialog)'))
    return overlays.some((ov) => {
        const el = ov as HTMLElement
        const cls = el.className + ' ' + (el.getAttribute('aria-label') || '')
        return !/v-tooltip|v-menu|menu|tooltip|snackbar/i.test(cls)
    })
}

/**
 * Ctrl+V が押されたとき、テキスト入力中やダイアログ内ボタンにフォーカスがある場合は
 * 発火せず、それ以外の場合に openClipboardDialog を呼ぶ。
 * useScopedEnterForKFTL と同じ条件判定ロジックを踏襲する。
 */
export function useScopedCtrlVForClipboard(
    _rootRef: Ref<HTMLElement | null>,
    openClipboardDialog: () => void,
    enabledRef?: Ref<boolean>,
) {
    let listener: (e: KeyboardEvent) => void

    onMounted(() => {
        listener = (e: KeyboardEvent) => {
            if (enabledRef && !enabledRef.value) return
            if (e.key !== 'v' && e.key !== 'V') return
            if (!e.ctrlKey && !e.metaKey) return
            if (e.altKey || e.shiftKey) return
            if (e.isComposing) return
            if (e.repeat) return

            const target = e.target as Element | null
            // テキスト入力中は発火しない
            if (isTextInput(target) || isTextInput(document.activeElement)) return
            // ダイアログ内のボタンがフォーカスされている場合は発火しない
            if (
                (isInsideDialog(target) && isButtonLike(target)) ||
                (isInsideDialog(document.activeElement) && isButtonLike(document.activeElement))
            ) return
            // ダイアログが開いている場合は発火しない（ダイアログ内の paste ハンドラに任せる）
            if (isAnyBlockingModalOpen()) return
            // フォーカスがダイアログ内のどこかにある場合も発火しない
            if (isInsideDialog(document.activeElement)) return

            openClipboardDialog()
            e.preventDefault()
            e.stopPropagation()
        }

        window.addEventListener('keydown', listener, { capture: true, passive: false })
    })

    onBeforeUnmount(() => {
        window.removeEventListener('keydown', listener, { capture: true } as EventListenerOptions)
    })
}
