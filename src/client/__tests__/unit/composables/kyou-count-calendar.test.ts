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
        is_active: true,
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

    // 親はv-showで隠すだけなのでwatcherは生きている。非表示中に数十万件の集計を
    // 走らせない(is_active=false中はスキップし、表示時に追いつく)ことの回帰テスト
    it('is_active=false中はkyousが変わっても集計せず、trueになったら追いつく', async () => {
        const emits = (() => { }) as unknown as KyouCountCalendarEmits
        const root = makeCalendarDom()
        const { app, api, props } = mountCalendar(emits, root)
        await nextTick()

        ;(props as unknown as { is_active: boolean }).is_active = false
        await nextTick()

        props.kyous.push({ related_time: new Date(2026, 7, 10), id: 'a' } as never)
        await nextTick()
        expect(api.events.value, '非表示中は集計しない').toHaveLength(0)

        ;(props as unknown as { is_active: boolean }).is_active = true
        await nextTick()
        expect(api.events.value.length, '表示されたら非表示中の変更へ追いつく').toBeGreaterThan(0)
        app.unmount()
    })

    // moment(related_time).format("yyyy-MM-DD") をネイティブ実装へ置き換えた際の
    // 日付キー互換の検証。月初・月末・年跨ぎ・1桁月日の境界で同じ日付に集計されること
    it('日付キーの互換: 境界日でも従来(moment)と同じ日に集計される', async () => {
        const emits = (() => { }) as unknown as KyouCountCalendarEmits
        const root = makeCalendarDom()
        const { app, api, props } = mountCalendar(emits, root)
        await nextTick()

        const boundary_dates = [
            new Date(2026, 0, 1, 0, 0, 0),    // 年始・1桁月日
            new Date(2025, 11, 31, 23, 59, 59), // 年末・大晦日の終端
            new Date(2026, 1, 28, 12, 0, 0),  // 月末(平年2月)
            new Date(2024, 1, 29, 12, 0, 0),  // 閏日
            new Date(2026, 8, 9, 0, 0, 0),    // 1桁月・1桁日
        ]
        for (let i = 0; i < boundary_dates.length; i++) {
            props.kyous.push({ related_time: boundary_dates[i], id: `boundary-${i}` } as never)
        }
        await nextTick()

        // どの境界日も1日1件として独立に集計される
        expect(api.events.value).toHaveLength(boundary_dates.length)
        for (let i = 0; i < boundary_dates.length; i++) {
            const source = boundary_dates[i]
            const found = api.events.value.find((event) => {
                const start = event.start as Date
                return start.getFullYear() === source.getFullYear()
                    && start.getMonth() === source.getMonth()
                    && start.getDate() === source.getDate()
            })
            expect(found, `${source.toISOString()} が同じ日付キーへ集計されていない`).toBeTruthy()
            // startはその日の0時、endはその日の終端(翌日0時の1ms前)
            const start = found!.start as Date
            const end = found!.end as Date
            expect(start.getHours()).toBe(0)
            expect(end.getTime()).toBe(new Date(start.getFullYear(), start.getMonth(), start.getDate() + 1).getTime() - 1)
        }
        app.unmount()
    })
})
