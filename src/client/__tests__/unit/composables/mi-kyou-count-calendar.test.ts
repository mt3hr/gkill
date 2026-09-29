/**
 * mi版件数カレンダーの検証。
 * use-kyou-count-calendar.ts と対称実装なので、is_activeゲートと
 * 日付キーのネイティブ化互換を同様に固定する（kyou-count-calendar.test.ts のmi版）。
 */
import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { createApp, defineComponent, h, nextTick, reactive } from 'vue'
import '@/classes/api/gkill-api'
import { useMiKyouCountCalendar } from '@/classes/use-mi-kyou-count-calendar'
import { MiSortType } from '@/classes/api/find_query/mi-sort-type'
import type { MiKyouCountCalendarProps } from '@/pages/views/mi-kyou-count-calendar-props'
import type { MiKyouCountCalendarEmits } from '@/pages/views/mi-kyou-count-calendar-emits'

function mountCalendar(emits: MiKyouCountCalendarEmits) {
    const props = reactive({
        kyous: [] as unknown[],
        mi_sort_type: MiSortType.create_time,
        is_active: true,
    }) as unknown as MiKyouCountCalendarProps

    let api: ReturnType<typeof useMiKyouCountCalendar> | null = null
    const Host = defineComponent({
        setup() {
            api = useMiKyouCountCalendar({ props, emits })
            return () => h('div')
        },
    })
    const container = document.createElement('div')
    document.body.appendChild(container)
    const app = createApp(Host)
    app.mount(container)
    return { app, api: api!, props }
}

function makeMiKyou(id: string, related_time: Date): unknown {
    return { id, related_time, data_type: 'mi_create' }
}

describe('useMiKyouCountCalendar', () => {
    beforeEach(() => {
        document.body.innerHTML = ''
    })
    afterEach(() => {
        document.body.innerHTML = ''
    })

    it('is_active=false中はkyousが変わっても集計せず、trueになったら追いつく', async () => {
        const emits = (() => { }) as unknown as MiKyouCountCalendarEmits
        const { app, api, props } = mountCalendar(emits)
        await nextTick()

        ;(props as unknown as { is_active: boolean }).is_active = false
        await nextTick()

        props.kyous.push(makeMiKyou('a', new Date(2026, 7, 10)) as never)
        await nextTick()
        expect(api.events.value, '非表示中は集計しない').toHaveLength(0)

        ;(props as unknown as { is_active: boolean }).is_active = true
        await nextTick()
        expect(api.events.value.length, '表示されたら非表示中の変更へ追いつく').toBeGreaterThan(0)
        app.unmount()
    })

    it('is_active=false中のソート種別変更も表示時に追いつく', async () => {
        const emits = (() => { }) as unknown as MiKyouCountCalendarEmits
        const { app, api, props } = mountCalendar(emits)
        await nextTick()

        props.kyous.push(makeMiKyou('a', new Date(2026, 7, 10)) as never)
        props.kyous.push({ id: 'b', related_time: new Date(2026, 7, 11), data_type: 'mi_limit' } as never)
        await nextTick()
        expect(api.events.value, 'create射影の1件だけが集計される').toHaveLength(1)

        ;(props as unknown as { is_active: boolean }).is_active = false
        await nextTick()
        ;(props as unknown as { mi_sort_type: MiSortType }).mi_sort_type = MiSortType.limit_time
        await nextTick()
        expect(api.events.value, '非表示中は再集計しない').toHaveLength(1)

        ;(props as unknown as { is_active: boolean }).is_active = true
        await nextTick()
        const start = api.events.value[0].start as Date
        expect(api.events.value).toHaveLength(1)
        expect(start.getDate(), '表示時にlimit射影で集計し直されている').toBe(11)
        app.unmount()
    })

    it('日付キーの互換: 境界日でも同じ日に集計され、start/endが日の両端になる', async () => {
        const emits = (() => { }) as unknown as MiKyouCountCalendarEmits
        const { app, api, props } = mountCalendar(emits)
        await nextTick()

        const boundary_dates = [
            new Date(2026, 0, 1, 0, 0, 0),
            new Date(2025, 11, 31, 23, 59, 59),
            new Date(2024, 1, 29, 12, 0, 0),
        ]
        for (let i = 0; i < boundary_dates.length; i++) {
            props.kyous.push(makeMiKyou(`boundary-${i}`, boundary_dates[i]) as never)
        }
        await nextTick()

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
            const start = found!.start as Date
            const end = found!.end as Date
            expect(start.getHours()).toBe(0)
            expect(end.getTime()).toBe(new Date(start.getFullYear(), start.getMonth(), start.getDate() + 1).getTime() - 1)
        }
        app.unmount()
    })
})

// 年月表示をクリックして開く日付ピッカー（rykv 版 kyou-count-calendar.test.ts と対称）。
// mi 版はスライダーを持たないので requested_focus_time はその日の 00:00:00 で1回だけ出る
describe('useMiKyouCountCalendar の日付ピッカー', () => {
    beforeEach(() => {
        document.body.innerHTML = ''
    })
    afterEach(() => {
        document.body.innerHTML = ''
    })

    function collect_emits() {
        const emitted: Array<{ event: string, args: unknown[] }> = []
        const emits = ((event: string, ...args: unknown[]) => {
            emitted.push({ event, args })
        }) as unknown as MiKyouCountCalendarEmits
        return { emitted, emits }
    }

    it('日を選ぶとピッカーが閉じ、表示月がその日へ移る', async () => {
        const { emits } = collect_emits()
        const { app, api } = mountCalendar(emits)
        api.date.value = new Date(2026, 8, 1)
        api.is_show_date_picker.value = true
        await nextTick()

        expect(api.date_picker_model.value.getTime()).toBe(api.date.value.getTime())

        api.date_picker_model.value = new Date(2016, 4, 15, 13, 45, 30)
        await nextTick()

        expect(api.is_show_date_picker.value, '選んだあともピッカーが開いたまま').toBe(false)
        expect(api.calendar_year_month.value, '表示月が移っていない').toBe('2016/05')
        expect(api.date.value.getDate()).toBe(15)
        app.unmount()
    })

    it('requested_focus_time はその日の 00:00:00 で1回だけ出る', async () => {
        const { emitted, emits } = collect_emits()
        const { app, api } = mountCalendar(emits)
        api.date.value = new Date(2026, 8, 1)
        await nextTick()
        await nextTick()
        emitted.length = 0

        // 時刻付きで選んでも、出る時刻は 0 時（rykv 版のようにスライダーの時刻は乗らない）
        api.date_picker_model.value = new Date(2016, 4, 15, 13, 45, 30)
        await nextTick()
        await nextTick()

        expect(emitted.map((e) => e.event), '一覧を動かす requested_focus_time が1回ちょうどではない').toEqual(['requested_focus_time'])
        const time = emitted[0].args[0] as Date
        expect(time.getTime()).toBe(new Date(2016, 4, 15, 0, 0, 0).getTime())
        app.unmount()
    })
})
