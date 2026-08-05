/**
 * オーバーレイのタイマー解除と、URLog画像のデータURI生成の回帰テスト。
 *
 * どちらも「動いているように見えるが裏で無駄が積み上がる」類なので、
 * 通常の表示テストでは気づけない。
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { createApp, defineComponent, h, nextTick } from 'vue'
import '@/classes/api/gkill-api'
import { useSaihateStarsOverlay } from '@/classes/use-saihate-stars-overlay'
import { useURLogView } from '@/classes/use-ur-log-view'
import type { URLogViewProps } from '@/pages/views/ur-log-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

describe('useSaihateStarsOverlay', () => {
    beforeEach(() => {
        vi.useFakeTimers()
    })
    afterEach(() => {
        vi.useRealTimers()
    })

    // App.vue はテーマで v-if しているので、テーマ切替のたびにアンマウントされる。
    // 解除しないと流星ループが永久に残り、切り替えるたびに1本ずつ増える。
    it('アンマウント後は流星ループのタイマーが残らない', async () => {
        const Host = defineComponent({
            setup() {
                const { starField } = useSaihateStarsOverlay()
                return () => h('div', { ref: starField })
            },
        })

        const container = document.createElement('div')
        document.body.appendChild(container)
        const app = createApp(Host)
        app.mount(container)

        // ループを何周かまわしてタイマーを確実に張らせる
        await vi.advanceTimersByTimeAsync(5000)
        expect(vi.getTimerCount()).toBeGreaterThan(0)

        app.unmount()
        container.remove()
        // アンマウント直後に残っている一発物(流星の描画・削除)を消化させる
        await vi.advanceTimersByTimeAsync(20000)

        expect(vi.getTimerCount()).toBe(0)
    })
})

describe('useURLogView の画像データURI', () => {
    function createOptions(favicon: string, thumbnail: string) {
        const props = {
            kyou: {
                id: 'kyou-1',
                typed_urlog: {
                    url: 'https://example.com',
                    title: 't',
                    description: 'd',
                    favicon_image: favicon,
                    thumbnail_image: thumbnail,
                },
            },
            enable_context_menu: true,
        } as unknown as URLogViewProps
        const emits = (() => { }) as unknown as KyouViewEmits
        return { props, emits }
    }

    it('base64の種別に応じたデータURIを返す', () => {
        const { favicon_src, thumbnail_src } = useURLogView(createOptions('iVBORw0KGgo=', '/9j/4AAQSkZJRg=='))
        expect(favicon_src.value).toBe('data:image/png;base64,iVBORw0KGgo=')
        expect(thumbnail_src.value).toBe('data:image/jpeg;base64,/9j/4AAQSkZJRg==')
    })

    it('空文字なら noimage を返す', () => {
        const { favicon_src, thumbnail_src } = useURLogView(createOptions('', ''))
        expect(favicon_src.value).not.toContain('base64,')
        expect(thumbnail_src.value).not.toContain('base64,')
    })

    // computed でなければ参照が毎回変わる。
    // テンプレートで直接呼んでいたころは再レンダーのたびに
    // 数百KB〜10MBの文字列連結が走っていた。
    it('依存が変わらなければ再評価せず同じ参照を返す', async () => {
        const { thumbnail_src } = useURLogView(createOptions('', 'iVBORw0KGgo='))
        const first = thumbnail_src.value
        await nextTick()
        expect(thumbnail_src.value).toBe(first)
    })
})
