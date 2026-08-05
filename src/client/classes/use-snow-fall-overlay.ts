'use strict'

import { onMounted, onUnmounted, ref } from 'vue'

export function useSnowFallOverlay() {
    const snow_field = ref<HTMLElement | null>(null)
    let timer_id: ReturnType<typeof setTimeout> | null = null

    function create_snowflake() {
        const flake = document.createElement('div')
        flake.className = 'snowflake'

        const size = Math.random() * 6 + 2
        const left = Math.random() * window.innerWidth
        const duration = Math.random() * 5 + 5

        flake.style.width = `${size}px`
        flake.style.height = `${size}px`
        flake.style.left = `${left}px`
        flake.style.animationDuration = `${duration}s`

        snow_field.value?.appendChild(flake)

        setTimeout(() => flake.remove(), duration * 1000)
    }

    function loop_snowfall() {
        create_snowflake()
        timer_id = setTimeout(loop_snowfall, 100)
    }

    onMounted(() => {
        loop_snowfall()
    })

    onUnmounted(() => {
        if (timer_id !== null) {
            clearTimeout(timer_id)
        }
    })

    return {
        snow_field,
    }
}
