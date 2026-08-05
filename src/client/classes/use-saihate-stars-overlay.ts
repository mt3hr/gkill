import { onMounted, onUnmounted, ref } from 'vue'

export function useSaihateStarsOverlay() {
    // ── Template refs ──
    const star_field = ref<HTMLElement | null>(null)

    // 流星ループの再スケジュール用タイマー。
    // App.vue はテーマで v-if しているので、テーマを切り替えるたびに
    // このcomposableはアンマウントされる。解除しないとループが永久に残り、
    // ダークテーマに戻すたびにループが1本ずつ増えて流星の頻度が上がっていく。
    // 雪側(use-snow-fall-overlay.ts)と同じ形にしてある。
    let shooting_star_timer_id: ReturnType<typeof setTimeout> | null = null
    let stopped = false

    // ── Internal helpers ──
    function create_star(class_name: string, top: number, left: number, duration?: number) {
        const star = document.createElement('div')
        star.className = class_name
        star.style.top = `${top}px`
        star.style.left = `${left}px`
        if (duration) star.style.animationDuration = `${duration}s`
        star_field.value?.appendChild(star)
    }

    function create_shooting_star() {
        const star = document.createElement('div')
        star.className = 'shooting-star'
        const length = Math.random() * 100 + 100
        const start_x = Math.random() * window.innerWidth
        const start_y = Math.random() * window.innerHeight * 0.5
        const duration = (Math.random() * 0.5 + 0.5).toFixed(2)

        star.style.width = `${length}px`
        star.style.height = '2px'
        star.style.position = 'absolute'
        star.style.top = `${start_y}px`
        star.style.left = `${start_x}px`
        star.style.transform = 'rotate(135deg)'
        star.style.background = 'linear-gradient(135deg, rgba(255,255,255,0) 0%, rgba(255,255,255,0.6) 50%, white 100%)'
        star.style.animation = `shooting ${duration}s ease-out forwards`
        star.style.pointerEvents = 'none'
        star.style.opacity = '0'
        star_field.value?.appendChild(star)

        setTimeout(() => star.remove(), +duration * 1000)
    }

    function loop_shooting_stars() {
        if (stopped) {
            return
        }
        const count = Math.floor(Math.random() * 3) + 1
        for (let i = 0; i < count; i++) {
            setTimeout(create_shooting_star, Math.random() * 300)
        }
        shooting_star_timer_id = setTimeout(loop_shooting_stars, Math.random() * 1500 + 500)
    }

    // ── Lifecycle ──
    onMounted(() => {
        const h = window.innerHeight
        const w = window.innerWidth

        for (let i = 0; i < 100; i++) {
            create_star('background-star', Math.random() * h, Math.random() * w, Math.random() * 2 + 1)
        }
        for (let i = 0; i < 5; i++) {
            create_star('background-star red-star', Math.random() * h, Math.random() * w)
            create_star('background-star big-star', Math.random() * h, Math.random() * w)
            create_star('background-star blue-star', Math.random() * h, Math.random() * w)
        }

        loop_shooting_stars()
    })

    onUnmounted(() => {
        stopped = true
        if (shooting_star_timer_id !== null) {
            clearTimeout(shooting_star_timer_id)
            shooting_star_timer_id = null
        }
    })

    // ── Return ──
    return {
        // Template refs
        star_field,
    }
}
