/**
 * カレンダーの日付セルに張るクリックハンドラの回帰テスト。
 *
 * 以前は張り直すたびに新しい無名クロージャを addEventListener していたため、
 * リスナーが積み上がって1クリックで requested_focus_time が多重発火していた。
 * 発火回数は表示に出ないので、数えないと気づけない。
 */
import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { createApp, defineComponent, h, nextTick, reactive } from 'vue'
import '@/classes/api/gkill-api'
import { useKyouCountCalendar } from '@/classes/use-kyou-count-calendar'
import type { KyouCountCalendarProps } from '@/pages/views/kyou-count-calendar-props'
import type { KyouCountCalendarEmits } from '@/pages/views/kyou-count-calendar-emits'

// Vuetifyのカレンダーが描く日付セルを模したDOMを用意する。
// jsdom は innerText を実装していないので、ハンドラが読む分を自前で生やす。
function makeCalendarDom(): HTMLElement {
    const root = document.createElement('div')
    for (let i = 1; i <= 3; i++) {
        const cell = document.createElement('div')
        cell.className = 'v-calendar-weekly__day'
        cell.textContent = String(i)
        Object.defineProperty(cell, 'innerText', { value: String(i), configurable: true })
        root.appendChild(cell)
    }
    document.body.appendChild(root)
    return root
}

function mountCalendar(emits: KyouCountCalendarEmits, root: HTMLElement) {
    // 本番の props は Vue のリアクティブオブジェクトなので、
    // watch が追跡できるよう reactive にしておく
    const props = reactive({
        kyous: [] as unknown[],
        for_mi: false,
    }) as unknown as KyouCountCalendarProps

    let api: ReturnType<typeof useKyouCountCalendar> | null = null
    const Host = defineComponent({
        setup() {
            api = useKyouCountCalendar({ props, emits })
            // テンプレートrefの代わりに、日付セルを持つDOMを直接与える
            api.kyou_counter_calendar.value = { $el: root }
            return () => h('div')
        },
    })
    const container = document.createElement('div')
    document.body.appendChild(container)
    const app = createApp(Host)
    app.mount(container)
    return { app, container, api: api!, props }
}

describe('useKyouCountCalendar の日付セルハンドラ', () => {
    beforeEach(() => {
        document.body.innerHTML = ''
    })
    afterEach(() => {
        document.body.innerHTML = ''
    })

    it('何度張り直しても1クリックにつき1回しか発火しない', async () => {
        const emitted: unknown[] = []
        const emits = ((event: string) => { emitted.push(event) }) as unknown as KyouCountCalendarEmits
        const root = makeCalendarDom()
        const { app, api } = mountCalendar(emits, root)

        // 月移動や日付クリックのたびに張り直される状況を模す
        for (let i = 0; i < 5; i++) {
            api.date.value = new Date(2026, i, 1)
            await nextTick()
            await nextTick()
        }

        const cell = root.querySelector('.v-calendar-weekly__day') as HTMLElement
        emitted.length = 0
        cell.click()

        expect(emitted).toHaveLength(1)
        app.unmount()
    })

    it('アンマウント後はハンドラが残らない', async () => {
        const emitted: unknown[] = []
        const emits = ((event: string) => { emitted.push(event) }) as unknown as KyouCountCalendarEmits
        const root = makeCalendarDom()
        const { app } = mountCalendar(emits, root)
        await nextTick()
        await nextTick()

        app.unmount()

        const cell = root.querySelector('.v-calendar-weekly__day') as HTMLElement
        emitted.length = 0
        cell.click()

        expect(emitted).toHaveLength(0)
    })

    // 削除は splice で行われるので、参照だけ見ていると件数が更新されない
    it('kyousの件数が減ったら再集計する', async () => {
        const emits = (() => { }) as unknown as KyouCountCalendarEmits
        const root = makeCalendarDom()
        const { app, api, props } = mountCalendar(emits, root)
        await nextTick()

        props.kyous.push({ related_time: new Date(), id: 'a' } as never)
        await nextTick()
        expect(api.events.value.length).toBeGreaterThan(0)

        props.kyous.splice(0, 1)
        await nextTick()

        // splice でも再集計されること（参照だけ見ていると 0 にならない）
        expect(api.events.value).toHaveLength(0)
        app.unmount()
    })
})
