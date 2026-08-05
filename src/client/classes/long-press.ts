// src/directives/long-press.ts
import type { Directive } from 'vue'

interface LongPressBindingObject {
    handler: (e: PointerEvent) => void
    pressMs?: number
    suppressClick?: boolean
}

type LongPressBindingValue = ((e: PointerEvent) => void) | LongPressBindingObject

type LongPressElement = HTMLElement & {
    __gkillLongPressCleanup__?: () => void
}

export const v_long_press: Directive = {
    mounted(el, binding) {
        const target = el as LongPressElement
        const value = binding.value as LongPressBindingValue
        const is_object_value = typeof value === 'object' && value !== null
        const handler = typeof value === 'function' ? value : is_object_value ? value.handler : undefined
        const press_ms = is_object_value && typeof value.pressMs === 'number' ? value.pressMs : 600
        const suppress_click = is_object_value && typeof value.suppressClick === 'boolean' ? value.suppressClick : true

        let timer: number | undefined
        let long_press_triggered = false

        const down = (e: PointerEvent) => {
            if (e.button !== 0 || timer) return
            long_press_triggered = false
            timer = window.setTimeout(() => {
                handler?.(e) // 長押し確定時にユーザー関数を呼ぶ
                long_press_triggered = true
                timer = undefined
            }, press_ms)
        }

        const up = () => {
            if (timer !== undefined) {
                clearTimeout(timer)
                timer = undefined
            }
        }

        const click = (e: Event) => {
            if (!suppress_click || !long_press_triggered) return
            e.preventDefault()
            e.stopImmediatePropagation()
            long_press_triggered = false
        }

        target.addEventListener('pointerdown', down)
        target.addEventListener('pointerup', up)
        target.addEventListener('pointerleave', up)
        target.addEventListener('pointercancel', up)
        target.addEventListener('click', click, true)

        target.__gkillLongPressCleanup__ = () => {
            if (timer !== undefined) {
                clearTimeout(timer)
                timer = undefined
            }
            long_press_triggered = false
            target.removeEventListener('pointerdown', down)
            target.removeEventListener('pointerup', up)
            target.removeEventListener('pointerleave', up)
            target.removeEventListener('pointercancel', up)
            target.removeEventListener('click', click, true)
            delete target.__gkillLongPressCleanup__
        }
    },
    unmounted(el) {
        const target = el as LongPressElement
        target.__gkillLongPressCleanup__?.()
    },
}
