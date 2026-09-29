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

// 年月表示をクリックして開く日付ピッカー（v-model は date_picker_model）。
// 選んだ日は日付セルをクリックしたのと同じ扱い ―― その月へ移り、requested_focus_time を
// スライダーの時刻付きで1回だけ出す。setter の中身を1行消しても型検査は通り、
// 「ピッカーが閉じない」「月が移らない」「一覧が動かない」がエラー無しで起きる
describe('useKyouCountCalendar の日付ピッカー', () => {
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
        }) as unknown as KyouCountCalendarEmits
        return { emitted, emits }
    }

    it('日を選ぶとピッカーが閉じ、表示月がその日へ移る', async () => {
        const { emits } = collect_emits()
        const root = makeCalendarDom()
        const { app, api } = mountCalendar(emits, root)
        api.date.value = new Date(2026, 8, 1)
        api.is_show_date_picker.value = true
        await nextTick()

        // ピッカーの表示値は現在の表示月（getter）
        expect(api.date_picker_model.value.getTime()).toBe(api.date.value.getTime())

        api.date_picker_model.value = new Date(2016, 4, 15)
        await nextTick()

        expect(api.is_show_date_picker.value, '選んだあともピッカーが開いたまま').toBe(false)
        expect(api.date.value.getFullYear(), '表示年が移っていない').toBe(2016)
        expect(api.date.value.getMonth(), '表示月が移っていない').toBe(4)
        expect(api.date.value.getDate()).toBe(15)
        app.unmount()
    })

    it('requested_focus_time はスライダーの時刻付きで1回だけ出る（既定は 23:59:59）', async () => {
        const { emitted, emits } = collect_emits()
        const root = makeCalendarDom()
        const { app, api } = mountCalendar(emits, root)
        api.date.value = new Date(2026, 8, 1)
        await nextTick()
        await nextTick()
        emitted.length = 0

        api.date_picker_model.value = new Date(2016, 4, 15)
        await nextTick()
        await nextTick()

        expect(emitted.map((e) => e.event), '一覧を動かす requested_focus_time が1回ちょうどではない').toEqual(['requested_focus_time'])
        // for_mi=false の既定スライダーは 86399 秒 = 23:59:59（一覧はその日の末尾へ寄る）
        const time = emitted[0].args[0] as Date
        expect(time.getTime()).toBe(new Date(2016, 4, 15, 23, 59, 59).getTime())
        app.unmount()
    })

    it('スライダーを動かしていれば、選んだ日のその時刻で出る', async () => {
        const { emitted, emits } = collect_emits()
        const root = makeCalendarDom()
        const { app, api } = mountCalendar(emits, root)
        api.date.value = new Date(2026, 8, 1)
        api.slider_model.value = 3661 // 01:01:01
        await nextTick()
        await nextTick()
        // スライダーの watch が出した分は数えない
        emitted.length = 0

        api.date_picker_model.value = new Date(2016, 4, 15)
        await nextTick()
        await nextTick()

        expect(emitted.map((e) => e.event)).toEqual(['requested_focus_time'])
        const time = emitted[0].args[0] as Date
        expect(time.getTime(), 'スライダーの時刻が乗っていない').toBe(new Date(2016, 4, 15, 1, 1, 1).getTime())
        app.unmount()
    })
})
