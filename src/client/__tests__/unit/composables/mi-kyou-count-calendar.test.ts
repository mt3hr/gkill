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
