'use strict'

import { onMounted, onUnmounted, ref } from 'vue'

export function useSnowFallOverlay() {
    const snowField = ref<HTMLElement | null>(null)
    let timerId: ReturnType<typeof setTimeout> | null = null

    function createSnowflake() {
        const flake = document.createElement('div')
        flake.className = 'snowflake'

        const size = Math.random() * 6 + 2
        const left = Math.random() * window.innerWidth
        const duration = Math.random() * 5 + 5

        flake.style.width = `${size}px`
        flake.style.height = `${size}px`
        flake.style.left = `${left}px`
        flake.style.animationDuration = `${duration}s`

        snowField.value?.appendChild(flake)

        setTimeout(() => flake.remove(), duration * 1000)
    }

    function loopSnowfall() {
        createSnowflake()
        timerId = setTimeout(loopSnowfall, 100)
    }

    onMounted(() => {
        loopSnowfall()
    })

    onUnmounted(() => {
        if (timerId !== null) {
            clearTimeout(timerId)
        }
    })

    return {
        snowField,
    }
}
