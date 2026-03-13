import { onMounted, ref } from 'vue'

export function useSaihateStarsOverlay() {
    // ── Template refs ──
    const starField = ref<HTMLElement | null>(null)

    // ── Internal helpers ──
    function createStar(className: string, top: number, left: number, duration?: number) {
        const star = document.createElement('div')
        star.className = className
        star.style.top = `${top}px`
        star.style.left = `${left}px`
        if (duration) star.style.animationDuration = `${duration}s`
        starField.value?.appendChild(star)
    }

    function createShootingStar() {
        const star = document.createElement('div')
        star.className = 'shooting-star'
        const length = Math.random() * 100 + 100
        const startX = Math.random() * window.innerWidth
        const startY = Math.random() * window.innerHeight * 0.5
        const duration = (Math.random() * 0.5 + 0.5).toFixed(2)

        star.style.width = `${length}px`
        star.style.height = '2px'
        star.style.position = 'absolute'
        star.style.top = `${startY}px`
        star.style.left = `${startX}px`
        star.style.transform = 'rotate(135deg)'
        star.style.background = 'linear-gradient(135deg, rgba(255,255,255,0) 0%, rgba(255,255,255,0.6) 50%, white 100%)'
        star.style.animation = `shooting ${duration}s ease-out forwards`
        star.style.pointerEvents = 'none'
        star.style.opacity = '0'
        starField.value?.appendChild(star)

        setTimeout(() => star.remove(), +duration * 1000)
    }

    function loopShootingStars() {
        const count = Math.floor(Math.random() * 3) + 1
        for (let i = 0; i < count; i++) {
            setTimeout(createShootingStar, Math.random() * 300)
        }
        setTimeout(loopShootingStars, Math.random() * 1500 + 500)
    }

    // ── Lifecycle ──
    onMounted(() => {
        const h = window.innerHeight
        const w = window.innerWidth

        for (let i = 0; i < 100; i++) {
            createStar('background-star', Math.random() * h, Math.random() * w, Math.random() * 2 + 1)
        }
        for (let i = 0; i < 5; i++) {
            createStar('background-star red-star', Math.random() * h, Math.random() * w)
            createStar('background-star big-star', Math.random() * h, Math.random() * w)
            createStar('background-star blue-star', Math.random() * h, Math.random() * w)
        }

        loopShootingStars()
    })

    // ── Return ──
    return {
        // Template refs
        starField,
    }
}
