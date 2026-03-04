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

export const vLongPress: Directive = {
    mounted(el, binding) {
        const target = el as LongPressElement
        const value = binding.value as LongPressBindingValue
        const isObjectValue = typeof value === 'object' && value !== null
        const handler = typeof value === 'function' ? value : isObjectValue ? value.handler : undefined
        const pressMs = isObjectValue && typeof value.pressMs === 'number' ? value.pressMs : 600
        const suppressClick = isObjectValue && typeof value.suppressClick === 'boolean' ? value.suppressClick : true

        let timer: number | undefined
        let longPressTriggered = false

        const down = (e: PointerEvent) => {
            if (e.button !== 0 || timer) return
            longPressTriggered = false
            timer = window.setTimeout(() => {
                handler?.(e) // 長押し確定時にユーザー関数を呼ぶ
                longPressTriggered = true
                timer = undefined
            }, pressMs)
        }

        const up = () => {
            if (timer !== undefined) {
                clearTimeout(timer)
                timer = undefined
            }
        }

        const click = (e: Event) => {
            if (!suppressClick || !longPressTriggered) return
            e.preventDefault()
            e.stopImmediatePropagation()
            longPressTriggered = false
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
            longPressTriggered = false
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
