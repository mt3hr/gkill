'use strict'

import { computed, nextTick, type Ref, ref, watch } from 'vue'
import moment from 'moment'
import type { Kyou } from '@/classes/datas/kyou'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { SharedRykvViewProps } from '@/pages/views/shared-rykv-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { GetKyousRequest } from '@/classes/api/req_res/get-kyous-request'
import type { GkillError } from '@/classes/api/gkill-error'
import type { OpenedRykvDialog, RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import { new_reload_batch, refresh_kyou, refresh_kyou_in_list } from '@/classes/kyou-reload'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'
import { remove_kyou_from_list_by_id } from '@/classes/kyou-local-insert'
import type { ComponentRef } from '@/classes/component-ref'
import { log_unless_aborted } from '@/classes/abort-error'

/**
 * 共有された rykv 画面。**通常の rykv（use-rykv-view）とは別実装。**
 *
 * 共有は「渡された検索結果を1列で見せるだけ」なので、列の機構（query_id での同一性・
 * 列状態の localStorage・サイドバーとの同期・再検索）を一切使わない。
 * 以前は use-rykv-view に相乗りしていたが、その機構は共有では
 * **一度も動かない死にコード**だった（`init()` を早期 return で止め、専用の初期化を別に持ち、
 * 列は既定の1本のまま query_id が空文字で居座っていた）。
 * 読む人は「共有のときはここを通らない」を常に考えなければならなかった。
 *
 * 先例は shared-mi-view（共有された mi 画面）。作法もそちらに倣う ——
 * **モードのフラグを持たず、読み取り専用であることは子へ渡すリテラルで表す。**
 */
export function useSharedRykvView(options: {
    props: SharedRykvViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const kyou_list_view = ref<ComponentRef | null>(null)

    // ── State refs ──
    const match_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
    const focused_kyou: Ref<Kyou | null> = ref(null)
    const focused_time: Ref<Date> = ref(moment().toDate())
    const is_loading: Ref<boolean> = ref(true)
    const is_show_kyou_detail_view: Ref<boolean> = ref(false)
    const is_show_kyou_count_calendar: Ref<boolean> = ref(false)
    const is_show_gps_log_map: Ref<boolean> = ref(false)
    const gps_log_map_start_time: Ref<Date> = ref(moment().toDate())
    const gps_log_map_end_time: Ref<Date> = ref(moment().toDate())
    const gps_log_map_marker_time: Ref<Date> = ref(moment().toDate())
    const opened_dialogs: Ref<Array<OpenedRykvDialog>> = ref([])

    /**
     * 関連する記録（Ryuu）が「どの条件を土台にするか」。
     *
     * 共有では検索条件を持たないので、既定の条件のまま渡す。
     * 絞り込みは共有 API 側（共有リンクに紐づく検索結果）が済ませている。
     */
    const default_query: Ref<FindKyouQuery> = ref(new FindKyouQuery())

    // ── Computed ──
    const kyou_list_view_height = computed(() => props.app_content_height)

    // ── Business logic ──
    /**
     * 共有された記録を1回だけ取る。**検索条件は送らない。**
     * 何を共有するかはサーバ側（共有リンクに紐づく検索条件）が決めていて、
     * ここで条件を足すと共有した本人の意図より狭くなる。
     *
     * スピナーは **finally で必ず倒す**。例外のときに立てっぱなしにすると
     * 「読み込み中のまま固まる」画面になり、原因が画面からは何も分からない。
     */
    async function load_content(): Promise<void> {
        try {
            await props.gkill_api.delete_updated_gkill_caches()
            const kyous = (await props.gkill_api.get_kyous(new GetKyousRequest())).kyous
            const wait_promises = new Array<Promise<Array<GkillError>>>()
            for (let i = 0; i < kyous.length; i++) {
                wait_promises.push(kyous[i].load_all())
            }
            await Promise.all(wait_promises)
            match_kyous.value = kyous
        } finally {
            is_loading.value = false
        }
    }

    async function reload_kyou(kyou: Kyou): Promise<void> {
        // 一覧・詳細・開いているダイアログは同じ更新から派生しているので、
        // 同じ値を渡して1往復に合流させる
        const requested_at = new_reload_batch()
        await refresh_kyou_in_list(match_kyous.value, kyou, {
            requested_at: requested_at,
            replace: (next_list) => { match_kyous.value = next_list },
        })
        if (focused_kyou.value && focused_kyou.value.id === kyou.id) {
            const refreshed = await refresh_kyou(kyou, undefined, requested_at)
            if (refreshed) {
                focused_kyou.value = refreshed
            }
        }
        for (let i = 0; i < opened_dialogs.value.length; i++) {
            if (opened_dialogs.value[i].kyou.id === kyou.id) {
                const refreshed = await refresh_kyou(kyou, undefined, requested_at)
                if (refreshed) {
                    opened_dialogs.value[i] = { ...opened_dialogs.value[i], kyou: refreshed }
                }
            }
        }
    }

    function onDeletedKyou(deleted_kyou: Kyou): void {
        remove_kyou_from_list_by_id(match_kyous.value, deleted_kyou.id)
        if (focused_kyou.value?.id === deleted_kyou.id) {
            focused_kyou.value = null
        }
        emits('deleted_kyou', deleted_kyou)
    }

    /**
     * 地図の時間窓を「その時刻の属する日」にし、マーカーをその時刻へ立てる。
     *
     * **開始と終了を同じ時刻にしてはいけない。** GPSLog の取得は両端を含む
     * 厳密な範囲比較なので、幅0の窓では軌跡が1本も出ない（use-rykv-view と同じ規則）。
     */
    function focus_gps_log_map_on(time: Date): void {
        gps_log_map_start_time.value = moment(time).startOf('day').toDate()
        gps_log_map_end_time.value = moment(time).endOf('day').toDate()
        gps_log_map_marker_time.value = time
    }

    function onClickedKyou(kyou: Kyou): void {
        focused_kyou.value = kyou
        focus_gps_log_map_on(kyou.related_time)
    }

    /** 件数カレンダーで日を選んだ。一覧も地図もその日へ送る */
    function onRequestedFocusTime(time: Date): void {
        focused_time.value = time
        focus_gps_log_map_on(time)
    }

    /** 地図の軌跡をクリックした。地図の窓は動かさず、一覧だけその時刻へ送る */
    function onGpsLogMapRequestedFocusTime(time: Date): void {
        focused_time.value = time
    }

    function open_rykv_dialog(kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload): void {
        const dialog_id = props.gkill_api.generate_uuid()
        opened_dialogs.value.push({
            id: dialog_id,
            kind,
            kyou: kyou.clone(),
            payload: payload ?? null,
            opened_at: Date.now(),
        })
    }

    function close_rykv_dialog(dialog_id: string): void {
        for (let i = 0; i < opened_dialogs.value.length; i++) {
            if (opened_dialogs.value[i].id === dialog_id) {
                opened_dialogs.value.splice(i, 1)
                break
            }
        }
    }

    // ── Watchers ──
    watch(() => focused_time.value, () => {
        kyou_list_view.value?.scroll_to_time(focused_time.value)
    })

    // ── Event relay objects ──
    // 束は必ず build_kyou_view_relay で作る。手書きのオブジェクトリテラルにすると
    // relay-bundle-source-scan.test.ts が束として認識できず、
    // 「v-on と @ で同じイベントを二重に配線する」のを検出できない
    const crudRelayHandlers = build_kyou_view_relay(emits, {
        deleted_kyou: (kyou: Kyou) => onDeletedKyou(kyou),
        requested_reload_kyou: (kyou: Kyou) => reload_kyou(kyou),
        requested_open_rykv_dialog: (kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => open_rykv_dialog(kind, kyou, payload),
        // 共有ページは検索条件を持たないので一覧の引き直しはしない
        requested_reload_list: () => { },
    })

    const subViewFocusHandlers = {
        focused_kyou: (kyou: Kyou) => onClickedKyou(kyou),
        clicked_kyou: (kyou: Kyou) => onClickedKyou(kyou),
    }

    const rykvDialogHandlers = {
        ...crudRelayHandlers,
        ...subViewFocusHandlers,
        closed: (dialog_id: string) => close_rykv_dialog(dialog_id),
    }

    // ── Init ──
    // 例外は握りつぶさず console へ出す。スピナーは load_content の finally で倒れるので、
    // 画面は「空の一覧」になり、原因は console に残る
    nextTick(() => { void load_content().catch(log_unless_aborted) })

    // ── Return ──
    return {
        // Template refs
        kyou_list_view,

        // State
        match_kyous,
        focused_kyou,
        focused_time,
        is_loading,
        is_show_kyou_detail_view,
        is_show_kyou_count_calendar,
        is_show_gps_log_map,
        gps_log_map_start_time,
        gps_log_map_end_time,
        gps_log_map_marker_time,
        opened_dialogs,
        default_query,

        // Computed
        kyou_list_view_height,

        // Business logic
        load_content,
        reload_kyou,
        onDeletedKyou,
        onClickedKyou,
        onRequestedFocusTime,
        onGpsLogMapRequestedFocusTime,
        focus_gps_log_map_on,
        open_rykv_dialog,
        close_rykv_dialog,

        // Event relay objects
        crudRelayHandlers,
        subViewFocusHandlers,
        rykvDialogHandlers,
    }
}
