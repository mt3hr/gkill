import { i18n } from '@/i18n'
import router from '@/router'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { computed, nextTick, type Ref, ref, watch } from 'vue'
import { Kyou } from '@/classes/datas/kyou'
import type { MiViewEmits } from '@/pages/views/mi-view-emits'
import type { MiViewProps } from '@/pages/views/mi-view-props'
import { GetKyousRequest } from '@/classes/api/req_res/get-kyous-request'
import { GetKyousResponse } from '@/classes/api/req_res/get-kyous-response'
import moment from 'moment'
import { deep_equals } from '@/classes/deep-equals'
import { useScopedEnterForKFTL } from '@/classes/use-scoped-enter-for-kftl'
import { useScopedCtrlVForClipboard } from '@/classes/use-scoped-ctrl-v-for-clipboard'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { Tag } from '@/classes/datas/tag'
import { Mi } from '@/classes/datas/mi'
import { MiReKyou } from '@/classes/datas/mi-re-kyou'
import { UpdateMiRequest } from '@/classes/api/req_res/update-mi-request'
import { UpdateMiReKyouRequest } from '@/classes/api/req_res/update-mi-re-kyou-request'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import { reset_dialog_history } from '@/classes/use-dialog-history-stack'
import { new_reload_batch, refresh_kyou, refresh_kyou_in_list } from '@/classes/kyou-reload'
import { useRegisteredKyouLocalInsert } from '@/classes/use-registered-kyou-local-insert'
import { apply_mi_projection, insert_kyou_sorted } from '@/classes/kyou-local-insert'
import type { OpenedRykvDialog, RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import type { ComponentRef } from '@/classes/component-ref'
import { MI_ALL_BOARD_KEY } from '@/classes/mi-board-names'

// ドラッグ&ドロップで受け取ったJSONからMiに写してよいフィールド。
// 内容は Mi.clone() が複製しているフィールドと同じ
const MI_DROP_ALLOWED_KEYS = new Set([
    "is_deleted", "id", "rep_name", "related_time", "data_type",
    "create_time", "create_app", "create_device", "create_user",
    "update_time", "update_app", "update_user", "update_device",
    "title", "is_checked", "board_name",
    "limit_time", "estimate_start_time", "estimate_end_time",
])

// MiReKyou版。MiReKyouはtitleを持たず、代わりにtarget_idを持つ
const MI_REKYOU_DROP_ALLOWED_KEYS = new Set([
    "is_deleted", "id", "rep_name", "related_time", "data_type",
    "create_time", "create_app", "create_device", "create_user",
    "update_time", "update_app", "update_user", "update_device",
    "target_id", "is_checked", "board_name",
    "limit_time", "estimate_start_time", "estimate_end_time",
])

// DataTransferのJSONは外部由来なので、許可したフィールド以外は捨てる
function parse_dropped_task<T extends object>(json: unknown, instance: T, allowed_keys: Set<string>): T {
    for (const key in json as object) {
        if (!allowed_keys.has(key)) {
            continue
        }
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        (instance as any)[key] = (json as any)[key]

        // 時刻はDate型に変換
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        if (key.endsWith("time") && (instance as any)[key]) {
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            (instance as any)[key] = new Date((instance as any)[key])
        }
    }
    return instance
}

export function useMiView(options: {
    props: MiViewProps,
    emits: MiViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const mi_root = ref<HTMLElement | null>(null)
    const query_editor_sidebar = ref<ComponentRef | null>(null)
    const add_mi_dialog = ref<ComponentRef | null>(null)
    const add_nlog_dialog = ref<ComponentRef | null>(null)
    const add_lantana_dialog = ref<ComponentRef | null>(null)
    const add_timeis_dialog = ref<ComponentRef | null>(null)
    const add_urlog_dialog = ref<ComponentRef | null>(null)
    const kftl_dialog = ref<ComponentRef | null>(null)
    const add_kc_dialog = ref<ComponentRef | null>(null)
    const mkfl_dialog = ref<ComponentRef | null>(null)
    const upload_file_dialog = ref<ComponentRef | null>(null)
    const save_clipboard_to_file_dialog = ref<ComponentRef | null>(null)
    const kyou_list_views = ref()

    // ── State refs ──
    const enable_context_menu = ref(true)
    const enable_dialog = ref(true)
    const opened_dialogs: Ref<Array<OpenedRykvDialog>> = ref([])
    // 板のドラッグ&ドロップ移動が進行中か
    const is_moving_board_task = ref(false)

    const querys: Ref<Array<FindKyouQuery>> = ref([new FindKyouQuery()])
    const querys_backup: Ref<Array<FindKyouQuery>> = ref(new Array<FindKyouQuery>()) // 更新検知用バックアップ
    const match_kyous_list: Ref<Array<Array<Kyou>>> = ref(new Array<Array<Kyou>>())
    const match_kyous_list_top_list: Ref<Array<number>> = ref(new Array<number>())
    const focused_query: Ref<FindKyouQuery> = ref(new FindKyouQuery())
    const focused_column_index: Ref<number> = ref(0)
    const focused_kyous_list: Ref<Array<Kyou>> = ref(new Array<Kyou>())
    const focused_kyou: Ref<Kyou | null> = ref(null)
    const focused_time: Ref<Date> = ref(moment().toDate())
    const is_show_kyou_detail_view: Ref<boolean> = ref(false)
    const is_show_kyou_count_calendar: Ref<boolean> = ref(false)
    const drawer: Ref<boolean | null> = ref(false)
    const drawer_mode_is_mobile: Ref<boolean | null> = ref(false)
    const is_loading: Ref<boolean> = ref(true)
    const inited = ref(false)
    const received_init_request = ref(false)
    const skip_search_this_tick = ref(false)
    const abort_controllers = new Map<string, AbortController>() // 列ごとの検索中断用。キーは列のquery_id
    const search_seqs = new Map<string, number>() // 列ごとの検索の世代番号。キーは列のquery_id。最後の検索だけが結果を書き戻せるようにする

    // ── Computed ──
    const kyou_list_view_height = computed(() => props.app_content_height)

    const page_list = computed(() => [
        { app_name: i18n.global.t('RYKV_APP_NAME'), page_name: 'rykv' },
        { app_name: i18n.global.t('MI_APP_NAME'), page_name: 'mi' },
        { app_name: i18n.global.t('KFTL_APP_NAME'), page_name: 'kftl' },
        { app_name: i18n.global.t('PLAING_TIMEIS_APP_NAME'), page_name: 'plaing' },
        { app_name: i18n.global.t('MKFL_APP_NAME'), page_name: 'mkfl' },
        { app_name: i18n.global.t('DASHBOARD_APP_NAME'), page_name: 'dashboard' },
        { app_name: i18n.global.t('SAIHATE_APP_NAME'), page_name: 'saihate' },
    ])

    // ── Watchers ──
    watch(() => is_show_kyou_count_calendar.value, () => {
        if (is_show_kyou_count_calendar.value) {
            update_focused_kyous_list(focused_column_index.value)
        }
    })

    watch(() => focused_time.value, () => {
        // 初期化前は初期クエリのquery_idが空文字のままなので、
        // 「列が無い」判定はquery_idの真偽値ではなく列の存在で行う
        const focused_column_query = querys.value[focused_column_index.value]
        if (!focused_column_query) {
            return
        }
        const kyou_list_view = get_kyou_list_view(focused_column_query.query_id)
        if (!kyou_list_view) {
            return
        }
        let target_kyou: Kyou | null = null
        for (let i = 0; i < focused_kyous_list.value.length; i++) {
            const kyou = focused_kyous_list.value[i]
            if (kyou.related_time.getTime() >= focused_time.value.getTime()) {
                target_kyou = kyou
                break
            }
        }
        if (inited.value) {
            kyou_list_view.scroll_to_kyou(target_kyou)
        }
    })

    // ── Internal helpers ──
    function update_focused_kyous_list(column_index: number): void {
        if (!match_kyous_list.value || match_kyous_list.value.length === 0) {
            return
        }
        focused_kyous_list.value = match_kyous_list.value[column_index]
    }

    // 列コンポーネントをquery_idで引く。v-forのテンプレートref配列はマウント順で
    // 並び順が保証されないため、indexではなくquery_idで解決する
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    function get_kyou_list_view(query_id: string): any {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        return kyou_list_views.value?.filter((kyou_list_view: any) => kyou_list_view.get_query_id() === query_id)[0]
    }

    // フォーカス切り替え等でfocused_queryを差し替えると、サイドバーが機械的に
    // updated_queryを発火しうるので、差し替えを行うfnをこれで包んで1回分だけ検索を抑止する。
    // 抑止解除のnextTickは「fnのリアクティブ書き込みでflushが予約された後」に登録すること。
    // 書き込みより先に登録すると、VueのnextTickはresolvedPromiseへ直結し、
    // 解除がウォッチャflushより先に走って抑止が一度も効かない(マイクロタスクはFIFO)。
    // コールバック式はこの登録順を構造的に保証する。
    // 立てっぱなしにするとユーザの次の編集を黙殺するため、必ずnextTickで倒す
    function run_with_sidebar_search_suppressed(fn: () => void): void {
        skip_search_this_tick.value = true
        fn()
        nextTick(() => skip_search_this_tick.value = false)
    }

    function remove_kyou_from_list_by_id(list: Array<Kyou>, deleted_id: string): void {
        for (let i = list.length - 1; i >= 0; i--) {
            if (list[i].id === deleted_id) {
                list.splice(i, 1)
            }
        }
    }

    function remove_kyou_from_multi_column_lists(lists: Array<Array<Kyou>>, deleted_id: string): void {
        for (let i = 0; i < lists.length; i++) {
            remove_kyou_from_list_by_id(lists[i], deleted_id)
        }
    }

    function is_target_mi_kyou(kyou: Kyou, mi_id: string): boolean {
        return kyou.typed_mi?.id === mi_id || kyou.typed_mirekyou?.id === mi_id || kyou.id === mi_id
    }

    function find_kyou_instances_by_mi_id(mi_id: string): Array<{ column_index: number, row_index: number, kyou: Kyou }> {
        const instances: Array<{ column_index: number, row_index: number, kyou: Kyou }> = []
        for (let column_index = 0; column_index < match_kyous_list.value.length; column_index++) {
            const column = match_kyous_list.value[column_index]
            for (let row_index = 0; row_index < column.length; row_index++) {
                const kyou = column[row_index]
                if (is_target_mi_kyou(kyou, mi_id)) {
                    instances.push({ column_index, row_index, kyou })
                }
            }
        }
        return instances
    }

    function remove_kyou_from_column_by_id(column_index: number, kyou_id: string): void {
        const column = match_kyous_list.value[column_index]
        if (!column) {
            return
        }
        for (let i = column.length - 1; i >= 0; i--) {
            if (column[i].id === kyou_id) {
                column.splice(i, 1)
            }
        }
    }

    // 移動先の列へ、その列の並び替え規則で差し込む。
    // 移動元の行の related_time / data_type は「移動元の列の mi_sort_type」で計算されており、
    // 移動先と一致するとは限らないので、射影を計算し直してから入れる
    function insert_kyou_into_column_if_absent(column_index: number, kyou: Kyou): void {
        const column = match_kyous_list.value[column_index]
        const column_query = querys.value[column_index]
        if (!column || !column_query) {
            return
        }
        const inserting_kyou = kyou.clone()
        apply_mi_projection(inserting_kyou, column_query.mi_sort_type)
        insert_kyou_sorted(column, inserting_kyou, column_query)
    }

    function patch_kyou_mi_board_name(kyou: Kyou, updated_mi: Mi | MiReKyou): void {
        // MiReKyouのKyouにはtyped_mirekyouが載っているのでそちらを更新する
        const target = (updated_mi instanceof MiReKyou || kyou.typed_mirekyou)
            ? (kyou.typed_mirekyou ??= new MiReKyou())
            : (kyou.typed_mi ??= new Mi())
        target.id = updated_mi.id
        target.board_name = updated_mi.board_name
        target.update_app = updated_mi.update_app
        target.update_device = updated_mi.update_device
        target.update_user = updated_mi.update_user
        target.update_time = updated_mi.update_time
    }

    function apply_board_move_locally(mi_id: string, before_board: string, after_board: string, updated_mi: Mi | MiReKyou): void {
        const instances = find_kyou_instances_by_mi_id(mi_id)
        if (instances.length === 0) {
            return
        }

        // 既存インスタンスにボード更新を反映
        for (let i = 0; i < instances.length; i++) {
            patch_kyou_mi_board_name(instances[i].kyou, updated_mi)
        }
        const target_kyou = instances[0].kyou

        for (let i = 0; i < querys.value.length; i++) {
            const query = querys.value[i]
            if (query.mi_board_name !== null && query.mi_board_name === before_board) {
                remove_kyou_from_column_by_id(i, target_kyou.id)
            }
            if (query.mi_board_name !== null && query.mi_board_name === after_board) {
                insert_kyou_into_column_if_absent(i, target_kyou)
            }
        }

        if (focused_kyou.value && is_target_mi_kyou(focused_kyou.value, mi_id)) {
            patch_kyou_mi_board_name(focused_kyou.value, updated_mi)
        }
        if (is_show_kyou_count_calendar.value) {
            update_focused_kyous_list(focused_column_index.value)
        }
    }

    // ── Business logic ──
    function onDeletedKyou(deleted_kyou: Kyou): void {
        remove_kyou_from_multi_column_lists(match_kyous_list.value, deleted_kyou.id)
        remove_kyou_from_list_by_id(focused_kyous_list.value, deleted_kyou.id)
        if (focused_kyou.value?.id === deleted_kyou.id) {
            focused_kyou.value = null
        }
        emits('deleted_kyou', deleted_kyou)
    }

    // 対象のKyouが載っている列のクエリを探す。focused_kyou と opened_dialogs は
    // 自分がどの列由来か知らないので、列を総当たりして最初に見つかったものを使う
    function find_column_query_for(kyou_id: string): FindKyouQuery | undefined {
        for (let i = 0; i < match_kyous_list.value.length; i++) {
            if (match_kyous_list.value[i].some(k => k.id === kyou_id)) {
                return querys.value[i]
            }
        }
        return undefined
    }

    async function reload_kyou(kyou: Kyou): Promise<void> {
        // 列・focused・開いているダイアログは同じ更新を受けて独立に引き直す。
        // 同じ値を渡して1往復に合流させる(渡さないと系統ごとに往復が増える)
        const requested_at = new_reload_batch();
        (async (): Promise<void> => {
            for (let i = 0; i < match_kyous_list.value.length; i++) {
                const column_query = querys.value[i]
                const target_list = match_kyous_list.value[i]
                if (!column_query || !target_list) {
                    continue
                }
                await refresh_kyou_in_list(target_list, kyou, {
                    requested_at: requested_at,
                    query: column_query,
                    replace: (next_list) => {
                        // await中に列の削除・再検索・別Kyouのreloadでリストが差し替わりうる。
                        // 列はquery_idで引き直し、リストごと巻き戻さず現在のリストの該当行だけ
                        // 差し替える(リストごと戻すと新しい検索結果や他のreload結果を潰す)
                        const current_index = querys.value.findIndex(q => q.query_id === column_query.query_id)
                        if (current_index === -1) {
                            return
                        }
                        const refreshed = next_list.find(next_kyou => next_kyou.id === kyou.id)
                        const current_list = match_kyous_list.value[current_index]
                        if (!refreshed || !current_list || !current_list.some(current_kyou => current_kyou.id === kyou.id)) {
                            return
                        }
                        let used_refreshed = false
                        match_kyous_list.value[current_index] = current_list.map(current_kyou => {
                            if (current_kyou.id !== kyou.id) {
                                return current_kyou
                            }
                            // 同一インスタンスを複数行に置くと後段のload_typed_datas等で副作用が出る
                            const next_kyou = used_refreshed ? refreshed.clone() : refreshed
                            used_refreshed = true
                            return next_kyou
                        })
                    },
                })
            }
        })();
        (async (): Promise<void> => {
            if (focused_kyou.value && focused_kyou.value.id === kyou.id) {
                const refreshed = await refresh_kyou(kyou, find_column_query_for(kyou.id), requested_at)
                if (refreshed) {
                    focused_kyou.value = refreshed
                }
            }
        })();
        (async (): Promise<void> => {
            const target_dialogs = opened_dialogs.value.filter(dialog => dialog.kyou.id === kyou.id)
            for (const target_dialog of target_dialogs) {
                const refreshed = await refresh_kyou(kyou, find_column_query_for(kyou.id), requested_at)
                if (!refreshed) {
                    continue
                }
                // await中にダイアログの開閉で並びが変わりうるので、書き戻し先はIDで引き直す。
                // 位置のまま書くと別のダイアログのkyouを差し替えてしまう
                const current_index = opened_dialogs.value.findIndex(dialog => dialog.id === target_dialog.id)
                if (current_index === -1) {
                    continue
                }
                opened_dialogs.value[current_index] = { ...opened_dialogs.value[current_index], kyou: refreshed }
            }
        })();
    }

    async function update_check_kyous(_kyou: Array<Kyou>, _is_checked: boolean): Promise<void> {
        throw new Error('Not implemented')
    }

    async function reload_list(column_index: number): Promise<void> {
        return search(column_index, querys.value[column_index], true, false, true)
    }

    async function reload_list_by_query_id(query_id: string): Promise<void> {
        const column_index = querys.value.findIndex(query => query.query_id === query_id)
        if (column_index === -1) {
            return
        }
        return reload_list(column_index)
    }

    // 追加されたKyouは再検索せず、各列の正しい位置へ差し込む。
    // 再検索するとヒット集合もスクロール位置も変わるし、KyouListViewは
    // 配列参照の差し替えでフル再描画する(reload_kyou と同じ理由)
    const { onRegisteredKyou, insert_registered_kyou } = useRegisteredKyouLocalInsert({
        querys: querys,
        match_kyous_list: match_kyous_list,
        reload_list_by_query_id: reload_list_by_query_id,
        onColumnMutated: (query_id: string) => {
            const column_index = querys.value.findIndex(query => query.query_id === query_id)
            if (column_index === -1 || column_index !== focused_column_index.value) {
                return
            }
            if (is_show_kyou_count_calendar.value) {
                update_focused_kyous_list(column_index)
            }
        },
    })

    async function init(): Promise<void> {
        if (inited.value) {
            return
        }
        return nextTick(async () => {
            const wait_promises = new Array<Promise<void>>()
            try {
                // スクロール位置の復元
                match_kyous_list_top_list.value = props.gkill_api.get_saved_mi_scroll_indexs()

                // 前回開いていた列があれば復元する
                skip_search_this_tick.value = true
                const saved_querys = props.gkill_api.get_saved_mi_find_kyou_querys()
                if (saved_querys.length.valueOf() === 0) {
                    const default_query = query_editor_sidebar.value!.get_default_query()!.clone()
                    default_query.query_id = props.gkill_api.generate_uuid()
                    saved_querys.push(default_query)
                }

                if (props.application_config.rykv_hot_reload) {
                    for (let i = 0; i < saved_querys.length; i++) {
                        await nextTick(() => {
                            skip_search_this_tick.value = true
                            wait_promises.push(search(i, saved_querys[i], true).then(async () => {
                                return nextTick(() => {
                                    const kyou_list_view = get_kyou_list_view(saved_querys[i].query_id)
                                    kyou_list_view?.scroll_to(match_kyous_list_top_list.value[i] ?? 0)
                                    kyou_list_view?.set_loading(false)
                                })
                            }))
                        })
                    }
                } else {
                    querys.value = saved_querys.concat()
                    // バックアップは複製で持つ。同一参照だと差分検知が機能しない
                    querys_backup.value = saved_querys.map(query => query.clone())
                    for (let i = 0; i < saved_querys.length; i++) {
                        match_kyous_list.value.push([])
                    }
                }
            } finally {
                Promise.all(wait_promises).then(async () => {
                    focused_column_index.value = 0
                    // サイドバーを列0の保存済み条件へ同期する。hot reload OFFではsearch()を
                    // 通らずfocused_queryが初期値のままになり、検索ボタンがサイドバーの
                    // 既定値から条件を組んで保存済み条件を上書きしてしまう
                    if (querys.value[0]) {
                        focused_query.value = querys.value[0]
                    }
                    inited.value = true
                    drawer_mode_is_mobile.value = !(props.app_content_width.valueOf() >= 760)
                    drawer.value = props.app_content_width.valueOf() >= 760
                    is_loading.value = false
                    skip_search_this_tick.value = false
                })
            }
        })
    }

    async function search(column_index: number, query: FindKyouQuery, force_search?: boolean, update_cache?: boolean, preserve_scroll?: boolean): Promise<void> {
        const query_id = query.query_id
        // この列の最新の検索だけが結果を書き戻せるようにする世代番号。
        // 検索中に同じ列で条件が変更されたら、最後の検索条件の結果だけを表示する
        let seq = -1
        const is_current = () => seq !== -1 && search_seqs.get(query_id) === seq

        // 検索する。Tickでまとめる
        try {
            if (!force_search) {
                if (deep_equals(querys_backup.value[column_index], query)) {
                    return
                }
            }

            seq = (search_seqs.get(query_id) ?? 0) + 1
            search_seqs.set(query_id, seq)

            querys.value[column_index] = query
            // バックアップは複製で持つ。同一参照だと後からの変更が両方に映って差分検知が壊れる
            querys_backup.value[column_index] = query.clone()
            // フォーカス列の検索だけがサイドバーの表示対象(focused_query)を更新する。
            // 無条件に更新すると全列リロード等でサイドバーが別列の条件に乗っ取られ、
            // そのquery_idを引き継いだ編集が列間のquery_id重複(結果の誤配送)を生む。
            // 差し替えはサイドバーのprops同期を誘発するので、機械的な残響を1回分抑止する
            if (column_index === focused_column_index.value) {
                run_with_sidebar_search_suppressed(() => {
                    focused_query.value = query
                })
            }

            props.gkill_api.set_saved_mi_find_kyou_querys(querys.value)

            // 前の検索処理を中断する
            abort_controllers.get(query_id)?.abort()

            if (match_kyous_list.value[column_index]) {
                match_kyous_list.value[column_index] = []
            }

            nextTick(() => {
                if (!is_current()) {
                    return
                }
                const kyou_list_view = get_kyou_list_view(query_id)
                if (kyou_list_view) {
                    if (inited.value && !preserve_scroll) {
                        kyou_list_view.scroll_to(0)
                    }
                    ((async () => kyou_list_view.set_loading(true))());
                }
            })

            const wait_promises = new Array<Promise<unknown>>()

            const req = new GetKyousRequest()
            abort_controllers.set(query_id, req.abort_controller)
            req.query = query.clone()
            req.query.parse_words_and_not_words()
            if (update_cache) {
                wait_promises.push(delete_gkill_kyou_cache(null))
                req.query.update_cache = true
            } else {
                wait_promises.push(props.gkill_api.delete_updated_gkill_caches())
            }

            let res = new GetKyousResponse()
            wait_promises.push(props.gkill_api.get_kyous(req).then(response => res = response))

            await Promise.all(wait_promises)

            // 待っている間に同じ列で新しい検索が始まっていたら、この結果は捨てる。
            // スピナーは新しい検索が所有しているので触らない
            if (!is_current()) {
                return
            }

            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                // エラー時もスピナーを解除する。解除しないと列が読み込み中のまま残る
                nextTick(() => get_kyou_list_view(query_id)?.set_loading(false))
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }

            // 検索後の列位置を取得する
            column_index = -1
            for (let i = 0; i < querys.value.length; i++) {
                const query = querys.value[i]
                if (query.query_id === query_id) {
                    column_index = i
                    break
                }
            }

            if (column_index === -1) {
                return
            }

            match_kyous_list.value[column_index] = res.kyous
            // フォーカス列以外の検索完了でfocused_kyous_list(カレンダー)を汚染しない
            if (column_index === focused_column_index.value && is_show_kyou_count_calendar.value) {
                update_focused_kyous_list(column_index)
            }

            await nextTick(() => {
                if (!is_current()) {
                    return
                }
                const kyou_list_view = get_kyou_list_view(query_id)
                if (kyou_list_view) {
                    ((async () => kyou_list_view.set_loading(false))());
                    if (inited.value) {
                        if (preserve_scroll) {
                            kyou_list_view.scroll_to(match_kyous_list_top_list.value[column_index] ?? 0)
                        } else {
                            kyou_list_view.scroll_to(0)
                        }
                    }
                }
                if (inited.value) {
                    skip_search_this_tick.value = false
                }
            })
        } catch (err: unknown) {
            // abortは握りつぶす
            if (!(err instanceof Error && (err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request")))) {
                // abort以外はエラー出力する
                console.error(err)
            }
            // abort含め例外時はloading状態を解除する（ただし新しいsearchが開始されていない場合のみ）
            if (is_current()) {
                nextTick(() => get_kyou_list_view(query_id)?.set_loading(false))
            }
        }
    }

    async function close_list_view(column_index: number): Promise<void> {
        return nextTick(() => {
            // ここでの書き込み(focused_column_index・列配列のsplice)はサイドバーの
            // props(focused_query)に触れないので抑止は不要。抑止が要るのは
            // focused_queryを差し替える下のnextTick内だけ
            const closed_query = querys.value[column_index]
            const focused_index_before_close = focused_column_index.value
            focused_column_index.value = -1

            querys.value.splice(column_index, 1)
            querys_backup.value.splice(column_index, 1)

            if (closed_query) {
                // 飛行中の検索は中断し、世代表からも消す。遅れて届いた結果は世代照合で破棄される
                abort_controllers.get(closed_query.query_id)?.abort()
                abort_controllers.delete(closed_query.query_id)
                search_seqs.delete(closed_query.query_id)
            }

            match_kyous_list.value.splice(column_index, 1)
            match_kyous_list_top_list.value.splice(column_index, 1)

            props.gkill_api.set_saved_mi_find_kyou_querys(querys.value)
            props.gkill_api.set_saved_mi_scroll_indexs(match_kyous_list_top_list.value)
            nextTick(() => {
                // 閉じた列の最近傍の列へフォーカスを移す
                const next_focused_index = Math.max(0, Math.min(
                    focused_index_before_close > column_index ? focused_index_before_close - 1 : focused_index_before_close,
                    querys.value.length - 1,
                ))
                run_with_sidebar_search_suppressed(() => {
                    focused_column_index.value = next_focused_index
                    focused_query.value = querys.value[next_focused_index]
                })
                if (is_show_kyou_count_calendar.value) {
                    update_focused_kyous_list(next_focused_index)
                }
            })
        })
    }

    function add_list_view(query?: FindKyouQuery): void {
        match_kyous_list.value.push(new Array<Kyou>())
        match_kyous_list_top_list.value.push(0)
        // 初期化されていないときはDefaultQueryがない。
        // その場合は初期値のFindKyouQueryをわたして初期化してもらう
        const default_query = query_editor_sidebar.value?.get_default_query()?.clone()
        if (query) {
            querys.value.push(query)
            focused_query.value = query
        } else if (default_query) {
            default_query.query_id = props.gkill_api.generate_uuid()
            querys.value.push(default_query)
            focused_query.value = default_query
        } else {
            const query = new FindKyouQuery()
            query.query_id = props.gkill_api.generate_uuid()
            querys.value.push(query)
            focused_query.value = query
        }
        if (inited.value) {
            focused_column_index.value = querys.value.length - 1
        }
        props.gkill_api.set_saved_mi_find_kyou_querys(querys.value)
        props.gkill_api.set_saved_mi_scroll_indexs(match_kyous_list_top_list.value)
    }

    function open_or_focus_board(board_name: string): void {
        // 「すべて」の番兵はサイドバーのgenerate_queryが比較に使うキーと揃える。
        // ロケール非依存の MI_ALL_BOARD_KEY（＝ツリーのノードが持つkey）であること
        const all_board_sentinel = MI_ALL_BOARD_KEY
        if (board_name === "") {
            board_name = all_board_sentinel
        }
        const is_all_board = board_name === all_board_sentinel

        let opened = false
        for (let i = 0; i < querys.value.length; i++) {
            const query = querys.value[i]
            // 「すべて」列は mi_board_name === null で表す。
            // 板名の文字列一致だけだと null(すべて)列を拾えないので null 判定で分ける
            const is_match = is_all_board
                ? query.mi_board_name === null
                : (query.mi_board_name !== null && query.mi_board_name === board_name)
            if (is_match) {
                // フォーカス切り替えの他経路と同様、機械的なサイドバー更新を1回分抑止する
                run_with_sidebar_search_suppressed(() => {
                    focused_query.value = querys.value[i]
                    focused_kyous_list.value = match_kyous_list.value[i] ?? []
                    focused_column_index.value = i
                })
                opened = true
                break
            }
        }
        if (opened) {
            return
        }

        const query = query_editor_sidebar.value!.get_default_query()!.clone()
        query.query_id = props.gkill_api.generate_uuid()
        query.mi_board_name = is_all_board ? null : board_name

        // add_list_viewはfocused_queryを差し替えるので抑止で包む
        run_with_sidebar_search_suppressed(() => {
            add_list_view(query)
        })
        if (props.application_config.rykv_hot_reload) {
            search(querys.value.length - 1, query, true)
        }
    }

    async function clicked_kyou_in_list_view(column_index: number, kyou: Kyou): Promise<void> {
        focused_kyou.value = kyou
        focused_column_index.value = column_index

        const update_target_column_indexs = new Array<number>()
        for (let i = 0; i < querys.value.length; i++) {
            if (querys.value[i].is_focus_kyou_in_list_view) {
                update_target_column_indexs.push(i)
            }
        }

        for (let i = 0; i < update_target_column_indexs.length; i++) {
            const target_column_index = update_target_column_indexs[i]
            if (inited.value) {
                get_kyou_list_view(querys.value[target_column_index].query_id)?.scroll_to_time(kyou.related_time)
            }
        }
    }

    // ドロップは板の移動をそのままサーバへ書き込む。前の移動が終わる前に
    // 次をドロップされると、同じタスクに対する更新が重なるので直列化する
    async function onDropBoardTask(e: DragEvent, find_kyou_query: FindKyouQuery) {
        if (is_moving_board_task.value) {
            return
        }
        is_moving_board_task.value = true
        try {
            await move_board_task(e, find_kyou_query)
        } finally {
            is_moving_board_task.value = false
        }
    }

    async function move_board_task(e: DragEvent, find_kyou_query: FindKyouQuery) {
        // MiとMiReKyouのどちらがドロップされたか判定する
        const mi_json = e.dataTransfer!.getData("gkill_mi")
        const mirekyou_json = e.dataTransfer!.getData("gkill_mi_re_kyou")
        let task: Mi | MiReKyou
        try {
            if (mirekyou_json !== "") {
                task = parse_dropped_task(JSON.parse(mirekyou_json), new MiReKyou(), MI_REKYOU_DROP_ALLOWED_KEYS)
            } else if (mi_json !== "") {
                task = parse_dropped_task(JSON.parse(mi_json), new Mi(), MI_DROP_ALLOWED_KEYS)
            } else {
                return
            }
        } catch (e: unknown) {
            console.error(e)
            return
        }

        if (!task.id || task.id == "") {
            return
        }

        e!.preventDefault()
        e!.stopPropagation()

        const before_board_name = task.board_name
        const after_board_name = find_kyou_query.mi_board_name
        // 「すべて」列(mi_board_name === null)へのドロップは移動先の板が定まらないので何もしない
        if (after_board_name === null || before_board_name === after_board_name) {
            return
        }

        // 新しい板名の確認はここでは行わない。移動先の列は板ツリー由来なので、
        // ツリーを正とする判定では必ず既存扱いになり、確認が発火しないため
        task.board_name = after_board_name
        task.update_app = "gkill"
        task.update_device = props.application_config.device
        task.update_time = new Date(Date.now())
        task.update_user = props.application_config.user_id

        let updated_task: Mi | MiReKyou = task
        let updated_kyou: Kyou | null = null
        if (task instanceof MiReKyou) {
            const req = new UpdateMiReKyouRequest()
            req.mirekyou = task
            req.want_response_kyou = true
            const res = await props.gkill_api.update_mirekyou(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }
            if (res.updated_mirekyou && res.updated_mirekyou.id !== "") {
                updated_task = res.updated_mirekyou
            }
            updated_kyou = res.updated_kyou
        } else {
            const req = new UpdateMiRequest()
            req.mi = task
            req.want_response_kyou = true
            const res = await props.gkill_api.update_mi(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }
            if (res.updated_mi && res.updated_mi.id !== "") {
                updated_task = res.updated_mi
            }
            updated_kyou = res.updated_kyou
        }

        apply_board_move_locally(task.id, before_board_name, after_board_name, updated_task)

        if (updated_kyou) {
            emits('updated_kyou', updated_kyou)
        }
    }

    function onDragoverBoardTask(e: DragEvent, _find_kyou_query: FindKyouQuery) {
        e!.dataTransfer!.dropEffect = "move"
        e!.preventDefault()
        e!.stopPropagation()
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
        // 開いた直後にも最新化する。リストのKyouは検索時点のものなので、
        // 別経路で更新されていると古い内容でダイアログが開いてしまう
        ;(async (): Promise<void> => {
            const refreshed = await refresh_kyou(kyou, find_column_query_for(kyou.id))
            if (!refreshed) {
                return
            }
            for (let i = 0; i < opened_dialogs.value.length; i++) {
                if (opened_dialogs.value[i].id === dialog_id) {
                    opened_dialogs.value[i] = { ...opened_dialogs.value[i], kyou: refreshed }
                    return
                }
            }
        })().catch((err: unknown) => console.error(err))
    }

    function close_rykv_dialog(dialog_id: string): void {
        for (let i = 0; i < opened_dialogs.value.length; i++) {
            if (opened_dialogs.value[i].id === dialog_id) {
                opened_dialogs.value.splice(i, 1)
                break
            }
        }
    }

    // ── Template event handlers (extracted from inline) ──

    function toggle_drawer(): void {
        if (inited.value) { drawer.value = !drawer.value }
    }

    async function navigate_to_page(page_name: string): Promise<void> {
        await reset_dialog_history()
        router.replace('/' + page_name + '?loaded=true')
    }

    function onSidebarRequestedSearch(update_cache: boolean): void {
        const column_index = focused_column_index.value
        const base_query = querys.value[column_index]
        if (!base_query) {
            return
        }
        // 検索ボタンはサイドバーに今見えている条件で検索する。
        // hot reload OFFでは編集が列のクエリに保存されていないため、保存済みクエリではなく
        // サイドバーから現在値を組み立てる(query_idは列のものを引き継ぐ)
        const current_query = query_editor_sidebar.value?.generate_query(base_query.query_id) ?? base_query
        nextTick(() => search(column_index, current_query, true, update_cache))
    }

    function onSidebarUpdatedQuery(new_query: FindKyouQuery): void {
        if (!inited.value) {
            return
        }
        if (skip_search_this_tick.value || !props.application_config.rykv_hot_reload) {
            nextTick(() => skip_search_this_tick.value = false)
            return
        }
        // どの列宛ての編集かはquery_idで解決する。focused_column_indexを盲信すると、
        // フォーカス切り替えと編集が交錯したときに別列へ書き込んでしまう
        const column_index = querys.value.findIndex(q => q.query_id === new_query.query_id)
        if (column_index === -1) {
            return
        }
        search(column_index, new_query)
    }

    function onSidebarInited(): void {
        if (!received_init_request.value) { init() }
        received_init_request.value = true
    }

    function onColumnScrollList(index: number, scroll_top: number): void {
        match_kyous_list_top_list.value[index] = scroll_top
        if (inited.value) {
            props.gkill_api.set_saved_mi_scroll_indexs(match_kyous_list_top_list.value)
        }
    }

    function onColumnClickedListView(index: number): void {
        run_with_sidebar_search_suppressed(() => {
            focused_column_index.value = index
            focused_query.value = querys.value[index]
        })
        if (is_show_kyou_count_calendar.value) {
            update_focused_kyous_list(index)
        }
    }

    function onColumnClickedKyou(index: number, kyou: Kyou): void {
        run_with_sidebar_search_suppressed(() => {
            focused_column_index.value = index
            focused_query.value = querys.value[index]
        })
        clicked_kyou_in_list_view(index, kyou)
    }

    function onColumnRequestedChangeFocusKyou(index: number, is_focus: boolean): void {
        run_with_sidebar_search_suppressed(() => {
            focused_column_index.value = index
            const query = querys.value[index].clone()
            query.is_focus_kyou_in_list_view = is_focus
            querys.value.splice(index, 1, query)
            querys_backup.value.splice(index, 1, query.clone())
            // サイドバーの表示対象も差し替え後のクエリへ揃える
            focused_query.value = query
        })
    }

    function onColumnRequestedSearch(index: number): void {
        run_with_sidebar_search_suppressed(() => {
            focused_column_index.value = index
            const query = querys.value[index].clone()
            querys.value.splice(index, 1, query)
            querys_backup.value.splice(index, 1, query.clone())
        })
        reload_list(index)
    }

    function onColumnRequestedChangeImageOnlyView(index: number, is_image_only: boolean): void {
        run_with_sidebar_search_suppressed(() => {
            focused_column_index.value = index
            focused_kyous_list.value = match_kyous_list.value[index]
            const query = querys.value[index].clone()
            query.is_image_only = is_image_only
            querys.value.splice(index, 1, query)
            querys_backup.value.splice(index, 1, query.clone())
        })
        reload_list(index)
    }

    function onRequestedFocusTime(time: Date): void {
        focused_time.value = time
    }

    // ── Dialog show methods ──
    function show_kftl_dialog(): void {
        kftl_dialog.value?.show()
    }

    function show_add_kc_dialog(): void {
        add_kc_dialog.value?.show()
    }

    function show_mkfl_dialog(): void {
        mkfl_dialog.value?.show()
    }

    function show_timeis_dialog(): void {
        add_timeis_dialog.value?.show()
    }

    function show_mi_dialog(): void {
        add_mi_dialog.value?.show()
    }

    function show_nlog_dialog(): void {
        add_nlog_dialog.value?.show()
    }

    function show_lantana_dialog(): void {
        add_lantana_dialog.value?.show()
    }

    function show_urlog_dialog(): void {
        add_urlog_dialog.value?.show()
    }

    function show_upload_file_dialog(): void {
        upload_file_dialog.value?.show()
    }

    function show_save_clipboard_to_file_dialog(): void {
        save_clipboard_to_file_dialog.value?.show()
    }

    function floating_action_button_style() {
        return {
            'bottom': '60px',
            'right': '10px',
            'height': '50px',
            'width': '50px'
        }
    }

    // ── Event relay objects ──
    const crudRelayHandlers = {
        'deleted_kyou': (kyou: Kyou) => onDeletedKyou(kyou),
        'deleted_tag': (tag: Tag) => emits('deleted_tag', tag),
        'deleted_text': (text: Text) => emits('deleted_text', text),
        'deleted_notification': (notification: Notification) => emits('deleted_notification', notification),
        'registered_kyou': (kyou: Kyou) => { onRegisteredKyou(kyou); emits('registered_kyou', kyou) },
        'registered_tag': (tag: Tag) => emits('registered_tag', tag),
        'registered_text': (text: Text) => emits('registered_text', text),
        'registered_notification': (notification: Notification) => emits('registered_notification', notification),
        'updated_kyou': (kyou: Kyou) => { reload_kyou(kyou); emits('updated_kyou', kyou) },
        'updated_tag': (tag: Tag) => emits('updated_tag', tag),
        'updated_text': (text: Text) => emits('updated_text', text),
        'updated_notification': (notification: Notification) => emits('updated_notification', notification),
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
    }

    const allColumnsRequestHandlers = {
        'requested_reload_kyou': (kyou: Kyou) => reload_kyou(kyou),
        'requested_reload_list': () => { for (let i = 0; i < querys.value.length; i++) { reload_list(i) } },
        'requested_update_check_kyous': (kyous: Array<Kyou>, checked: boolean) => update_check_kyous(kyous, checked),
    }

    const rykv_dialog_handler = {
        'requested_open_rykv_dialog': (kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => open_rykv_dialog(kind, kyou, payload),
    }

    // ── Keyboard shortcut ──
    const enable_enter_shortcut = ref(true)
    useScopedEnterForKFTL(mi_root, show_kftl_dialog, enable_enter_shortcut)
    useScopedCtrlVForClipboard(mi_root, show_save_clipboard_to_file_dialog, enable_enter_shortcut)

    // ── Return ──
    return {
        // Template refs
        mi_root,
        query_editor_sidebar,
        add_mi_dialog,
        add_nlog_dialog,
        add_lantana_dialog,
        add_timeis_dialog,
        add_urlog_dialog,
        kftl_dialog,
        add_kc_dialog,
        mkfl_dialog,
        upload_file_dialog,
        save_clipboard_to_file_dialog,
        kyou_list_views,

        // State
        enable_context_menu,
        enable_dialog,
        opened_dialogs,
        querys,
        match_kyous_list,
        focused_query,
        focused_column_index,
        focused_kyous_list,
        focused_kyou,
        is_show_kyou_detail_view,
        is_show_kyou_count_calendar,
        drawer,
        drawer_mode_is_mobile,
        is_loading,
        inited,

        // Computed
        kyou_list_view_height,
        page_list,

        // Template event handlers
        toggle_drawer,
        navigate_to_page,
        onSidebarRequestedSearch,
        onSidebarUpdatedQuery,
        onSidebarInited,
        onColumnScrollList,
        onColumnClickedListView,
        onColumnClickedKyou,
        onColumnRequestedChangeFocusKyou,
        onColumnRequestedSearch,
        onColumnRequestedChangeImageOnlyView,
        onRequestedFocusTime,
        onDropBoardTask,
        onDragoverBoardTask,
        close_list_view,
        open_or_focus_board,
        open_rykv_dialog,
        close_rykv_dialog,
        reload_kyou,
        reload_list,
        insert_registered_kyou,
        update_check_kyous,

        // Dialog show methods
        show_kftl_dialog,
        show_mkfl_dialog,
        show_add_kc_dialog,
        show_urlog_dialog,
        show_timeis_dialog,
        show_mi_dialog,
        show_nlog_dialog,
        show_lantana_dialog,
        show_upload_file_dialog,
        show_save_clipboard_to_file_dialog,
        floating_action_button_style,

        // Event relay objects
        crudRelayHandlers,
        allColumnsRequestHandlers,
        rykv_dialog_handler,
    }
}
