import { i18n } from '@/i18n'
import { computed, nextTick, onUnmounted, ref, type Ref, toRaw, watch } from 'vue'
import type { KyouCountCalendarProps } from '@/pages/views/kyou-count-calendar-props'
import type { KyouCountCalendarEmits } from '@/pages/views/kyou-count-calendar-emits'
import moment from 'moment'
import type { ComponentRef } from '@/classes/component-ref'

export function useKyouCountCalendar(options: {
    props: KyouCountCalendarProps,
    emits: KyouCountCalendarEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const kyou_counter_calendar = ref<ComponentRef | null>(null)

    // 日付セルのクリックハンドラをまとめて外すためのもの。
    // 張り直すたびに前回ぶんを abort する。
    let calendar_listener_abort: AbortController | null = null

    // ── State refs ──
    const date = ref(new Date(Date.now()))
    const slider_model: Ref<number> = ref(props.for_mi ? 0 : 86399)
    const events: Ref<Array<Record<string, unknown>>> = ref(new Array<Record<string, unknown>>())
    // 非表示中にkyousが変わったら立てて、表示時にupdate_eventsで追いつくためのフラグ
    let is_events_stale = false

    // ── Computed ──
    const time = computed(() => {
        return ('00' + Math.floor(slider_model.value / 3600).toString()).slice(-2) + ":" +
            ('00' + (Math.floor(slider_model.value / 60) % 60).toString()).slice(-2) + ":" +
            ('00' + Math.floor(slider_model.value % 60).toString()).slice(-2)
    })

    // ── Watchers ──
    watch(() => date.value, () => {
        nextTick(() => {
            set_handler_on_calendar_date_texts()
        })
    })

    // 参照の入れ替えだけでなく件数の増減でも更新する。
    // 削除は use-rykv-view.ts の remove_kyou_from_list_by_id が splice で行うため、
    // 参照だけ見ていると Kyou を消してもカレンダーの件数バッジが更新されなかった。
    // 非表示中(is_active=false)は集計しない。親はv-showで隠すだけなのでwatcherは生きており、
    // 数十万件の検索のたびに見えないカレンダーへ全件集計していた
    watch([() => props.kyous, () => props.kyous.length], () => {
        if (!props.is_active) {
            is_events_stale = true
            return
        }
        update_events()
    })

    // 表示されたとき、非表示中に溜まった変更へ追いつく
    watch(() => props.is_active, () => {
        if (props.is_active && is_events_stale) {
            is_events_stale = false
            update_events()
        }
    })

    watch(() => slider_model.value, () => {
        clicked_date(date.value)
    })

    // ── Business logic ──
    const pad2 = (n: number): string => ('00' + n.toString()).slice(-2)

    function update_events(): void {
        events.value.splice(0)
        if (!props.kyous) {
            return
        }
        // 走査は生の配列に対して行う。deepなref配下のリアクティブProxy越しに読むと
        // 1要素ごとに track と toReactive が走り、要素ぶんのProxyを確保する(30万件では効く)。
        // これは watch のコールバック内で、読んだ値に依存を張る必要が無いので意味論も変わらない。
        const raw_kyous = toRaw(props.kyous)
        const date_event_map: Map<string, number> = new Map<string, number>()
        for (let i = 0; i < raw_kyous.length; i++) {
            const kyou = raw_kyous[i]
            // momentは1件あたりの生成+formatが重く、数十万件で秒単位になるためネイティブで組む
            const d = kyou.related_time
            const date_str = d.getFullYear().toString() + "-" + pad2(d.getMonth() + 1) + "-" + pad2(d.getDate())
            const count = date_event_map.get(date_str)?.valueOf()
            if (count) {
                date_event_map.set(date_str, count + 1)
            } else {
                date_event_map.set(date_str, 1)
            }
        }

        date_event_map.forEach((count: number, date_str: string): void => {
            const [year, month, day] = date_str.split("-").map(Number)
            const start = new Date(year, month - 1, day)
            events.value.push({
                title: count.toString(),
                start: start,
                end: new Date(new Date(year, month - 1, day + 1).getTime() - 1),
            })
        })
    }

    function onWheel(e: WheelEvent) {
        if (0 < e.deltaY) {
            date.value = add_months(date.value, 1)
        } else {
            date.value = add_months(date.value, -1)
        }
    }

    function clicked_date(date: Date): void {
        emits('requested_focus_time', moment(moment(date).format("yyyy-MM-DD") + " " + time.value).toDate())
    }

    // カレンダーの日付セルにクリックハンドラを張り直す。
    //
    // 毎回新しい無名クロージャを渡すので addEventListener の重複排除が効かず、
    // 張り直すたびにリスナーが積み上がっていた。Vuetify の .v-calendar-weekly__day は
    // 6週×7日の固定グリッドで月移動してもDOMノードが再利用されるため確実に累積する。
    // しかも date が変わるのは月移動だけでなく、日付クリックでも変わる
    // (kyou-count-calendar.vue の @update:model-value) ので、
    // クリックのたびに1本増えて requested_focus_time が多重発火していた。
    //
    // AbortController でまとめて外してから張り直す。
    function set_handler_on_calendar_date_texts(): void {
        calendar_listener_abort?.abort()
        calendar_listener_abort = new AbortController()
        const signal = calendar_listener_abort.signal

        // document 全体だと、同じページにある別のカレンダーにもハンドラが付いてしまう
        // (rykv-view と shared-mi-view がどちらも KyouCountCalendar を使う)。
        // 自分の配下だけに限定する。
        const root: ParentNode = kyou_counter_calendar.value?.$el ?? document
        const calendar_date_text_selector = ".v-calendar-weekly__day"
        root.querySelectorAll(calendar_date_text_selector).forEach((element) => {
            element.addEventListener('click', (() => {
                if (!element.textContent || element.textContent.trim() === "") {
                    return
                }
                const year = date.value.getFullYear().toString()
                const month = (date.value.getMonth() + 1).toString()
                const day = (element as HTMLElement).innerText.toString().split("\n")[0].split(" ").slice(-1)[0].replaceAll(i18n.global.t("DAY_TITLE"), "")
                clicked_date(moment(year + "-" + month + "-" + day).toDate())
            }), { signal })
        })
    }

    function add_months(date: Date, diff: number) {
        const added_date = new Date(date)
        added_date.setMonth(added_date.getMonth() + diff)
        return added_date
    }

    // ── Lifecycle ──
    onUnmounted(() => {
        calendar_listener_abort?.abort()
        calendar_listener_abort = null
    })

    // ── Init calls ──
    if (props.is_active) {
        update_events()
    } else {
        is_events_stale = true
    }
    nextTick(() => {
        set_handler_on_calendar_date_texts()
    })

    // ── Return ──
    return {
        // Template refs
        kyou_counter_calendar,

        // State
        date,
        slider_model,
        events,
        time,

        // Business logic
        add_months,
        onWheel,
    }
}
