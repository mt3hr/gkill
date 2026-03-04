import { onMounted, onBeforeUnmount, type Ref } from 'vue';

function isTextInput(el: Element | null): boolean {
    if (!el) return false;
    const he = el as HTMLElement;
    const tag = he.tagName?.toLowerCase();
    if (he.isContentEditable) return true;
    if (tag === 'input' || tag === 'textarea' || tag === 'select') return true;
    if (he.getAttribute?.('role') === 'textbox') return true;
    return false;
}

function isInsideDialog(el: Element | null): boolean {
    if (!el) return false;
    return !!el.closest('.gkill-floating-dialog, [role="dialog"][aria-modal="true"]');
}

function isButtonLike(el: Element | null): boolean {
    if (!el) return false;
    return !!el.closest('button, [type="button"], [type="submit"], [role="button"], .v-btn');
}

function isAnyBlockingModalOpen(): boolean {
    // ツールチップ/メニュー類は無視して、実モーダルっぽいものだけブロック
    const overlays = Array.from(document.querySelectorAll('[role="dialog"][aria-modal="true"]:not(.kyou_dialog)'));
    return overlays.some((ov) => {
        const el = ov as HTMLElement;
        const cls = el.className + ' ' + (el.getAttribute('aria-label') || '');
        return !/v-tooltip|v-menu|menu|tooltip|snackbar/i.test(cls);
    });
}

export function useScopedEnterForKFTL(
    rootRef: Ref<HTMLElement | null>,
    openKFTL: () => void,
    enabledRef?: Ref<boolean>,
    opts: { debug?: boolean } = {}
) {
    const { debug = false } = opts;
    let listener: (e: KeyboardEvent) => void;

    onMounted(() => {
        listener = (e: KeyboardEvent) => {
            if (enabledRef && !enabledRef.value) { if (debug) console.debug('[KFTL] disabled'); return; }
            if (e.key !== 'Enter') return;
            if (e.isComposing) { if (debug) console.debug('[KFTL] composing'); return; }
            if (e.repeat) { if (debug) console.debug('[KFTL] repeat'); return; }
            if (e.metaKey || e.ctrlKey || e.altKey || e.shiftKey) { if (debug) console.debug('[KFTL] with modifier'); return; }

            const target = e.target as Element | null;
            if (isTextInput(target) || isTextInput(document.activeElement)) {
                if (debug) console.debug('[KFTL] text input focused');
                return;
            }
            if (
                (isInsideDialog(target) && isButtonLike(target)) ||
                (isInsideDialog(document.activeElement) && isButtonLike(document.activeElement))
            ) {
                if (debug) console.debug('[KFTL] dialog button focused');
                return;
            }

            if (isAnyBlockingModalOpen()) { if (debug) console.debug('[KFTL] modal open'); return; }

            openKFTL();
            e.preventDefault();
            e.stopPropagation();
        };

        window.addEventListener('keydown', listener, { capture: true, passive: false });
    });

    onBeforeUnmount(() => {
        window.removeEventListener('keydown', listener, { capture: true } as any);
    });
}
