import { onMounted, onBeforeUnmount, type Ref } from 'vue';

function is_text_input(el: Element | null): boolean {
    if (!el) return false;
    const he = el as HTMLElement;
    const tag = he.tagName?.toLowerCase();
    if (he.isContentEditable) return true;
    if (tag === 'input' || tag === 'textarea' || tag === 'select') return true;
    if (he.getAttribute?.('role') === 'textbox') return true;
    return false;
}

function is_inside_dialog(el: Element | null): boolean {
    if (!el) return false;
    return !!el.closest('.gkill-floating-dialog, [role="dialog"][aria-modal="true"]');
}

function is_button_like(el: Element | null): boolean {
    if (!el) return false;
    return !!el.closest('button, [type="button"], [type="submit"], [role="button"], .v-btn');
}

function is_any_blocking_modal_open(): boolean {
    // ツールチップ/メニュー類は無視して、実モーダルっぽいものだけブロック
    const overlays = Array.from(document.querySelectorAll('[role="dialog"][aria-modal="true"]:not(.kyou_dialog)'));
    return overlays.some((ov) => {
        const el = ov as HTMLElement;
        const cls = el.className + ' ' + (el.getAttribute('aria-label') || '');
        return !/v-tooltip|v-menu|menu|tooltip|snackbar/i.test(cls);
    });
}

export function useScopedEnterForKFTL(
    root_ref: Ref<HTMLElement | null>,
    open_kftl: () => void,
    enabled_ref?: Ref<boolean>,
    opts: { debug?: boolean } = {}
) {
    const { debug = false } = opts;
    let listener: (e: KeyboardEvent) => void;

    onMounted(() => {
        listener = (e: KeyboardEvent) => {
            if (enabled_ref && !enabled_ref.value) { if (debug) console.debug('[KFTL] disabled'); return; }
            if (e.key !== 'Enter') return;
            if (e.isComposing) { if (debug) console.debug('[KFTL] composing'); return; }
            if (e.repeat) { if (debug) console.debug('[KFTL] repeat'); return; }
            if (e.metaKey || e.ctrlKey || e.altKey || e.shiftKey) { if (debug) console.debug('[KFTL] with modifier'); return; }

            const target = e.target as Element | null;
            if (is_text_input(target) || is_text_input(document.activeElement)) {
                if (debug) console.debug('[KFTL] text input focused');
                return;
            }
            if (
                (is_inside_dialog(target) && is_button_like(target)) ||
                (is_inside_dialog(document.activeElement) && is_button_like(document.activeElement))
            ) {
                if (debug) console.debug('[KFTL] dialog button focused');
                return;
            }

            if (is_any_blocking_modal_open()) { if (debug) console.debug('[KFTL] modal open'); return; }

            open_kftl();
            e.preventDefault();
            e.stopPropagation();
        };

        window.addEventListener('keydown', listener, { capture: true, passive: false });
    });

    onBeforeUnmount(() => {
        window.removeEventListener('keydown', listener, { capture: true } as EventListenerOptions);
    });
}
