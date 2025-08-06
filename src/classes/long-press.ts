// src/directives/long-press.ts
import type { Directive } from 'vue'

export const vLongPress: Directive = {
    mounted(el, binding) {
        const PRESS_MS = 600
        let timer: number | undefined

        const down = (e: PointerEvent) => {
            if (e.button !== 0 || timer) return
            timer = window.setTimeout(() => {
                binding.value?.(e)        // 長押し確定時にユーザー関数を呼ぶ
                timer = undefined
            }, PRESS_MS)
        }

        const up = () => {
            if (timer !== undefined) {
                clearTimeout(timer)
                timer = undefined
            }
        }

        el.addEventListener('pointerdown', down)
        el.addEventListener('pointerup', up)
        el.addEventListener('pointerleave', up)
        el.addEventListener('pointercancel', up)
    },
}
