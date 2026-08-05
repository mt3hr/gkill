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
import type { OpenedRykvDialog, RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import type { ComponentRef } from '@/classes/component-ref'

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
    const position_x: Ref<number> = ref(0)
    const position_y: Ref<number> = ref(0)
    const is_loading: Ref<boolean> = ref(true)
    const inited = ref(false)
    const received_init_request = ref(false)
    const skip_search_this_tick = ref(false)
    const abort_controllers: Ref<Array<AbortController>> = ref([])

    // ── Computed ──
    const kyou_list_view_height = computed(() => props.app_content_height)
    const add_kyou_menu_style = computed(() => `{ position: absolute; left: ${position_x.value}px; top: ${position_y.value}px; }`)

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
        if (!kyou_list_views.value) {
            return
        }
        const kyou_list_view = kyou_list_views.value[focused_column_index.value]
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

    function insert_kyou_into_column_if_absent(column_index: number, kyou: Kyou): void {
        const column = match_kyous_list.value[column_index]
        if (!column) {
            return
        }
        for (let i = 0; i < column.length; i++) {
            if (column[i].id === kyou.id) {
                return
            }
        }
        column.push(kyou)
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
            if (query.use_mi_board_name && query.mi_board_name === before_board) {
                remove_kyou_from_column_by_id(i, target_kyou.id)
            }
            if (query.use_mi_board_name && query.mi_board_name === after_board) {
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

    async function reload_kyou(kyou: Kyou): Promise<void> {
        (async (): Promise<void> => {
            for (let i = 0; i < match_kyous_list.value.length; i++) {
                const kyous_list = match_kyous_list.value[i]
                for (let j = 0; j < kyous_list.length; j++) {
                    const kyou_in_list = kyous_list[j]
                    if (kyou.id === kyou_in_list.id) {
                        const updated_kyou = kyou.clone()
                        await delete_gkill_kyou_cache(kyou.id)
                        await updated_kyou.reload(true, querys.value[i])
                        updated_kyou.is_typed_data_loaded = false
                        await updated_kyou.load_all(querys.value[i], true)
                        const new_kyous_list = [...kyous_list]
                        new_kyous_list[j] = updated_kyou
                        match_kyous_list.value[i] = new_kyous_list
                    }
                }
            }
        })();
        (async (): Promise<void> => {
            if (focused_kyou.value && focused_kyou.value.id === kyou.id) {
                const updated_kyou = kyou.clone()
                await updated_kyou.reload(true)
                await updated_kyou.load_all(undefined, true)
                focused_kyou.value = updated_kyou
            }
        })();
        (async (): Promise<void> => {
            for (let i = 0; i < opened_dialogs.value.length; i++) {
                if (opened_dialogs.value[i].kyou.id === kyou.id) {
                    let column_query: FindKyouQuery | undefined
                    for (let j = 0; j < match_kyous_list.value.length; j++) {
                        if (match_kyous_list.value[j].some(k => k.id === kyou.id)) {
                            column_query = querys.value[j]
                            break
                        }
                    }
                    const updated_kyou = kyou.clone()
                    await delete_gkill_kyou_cache(kyou.id)
                    await updated_kyou.reload(true, column_query)
                    updated_kyou.is_typed_data_loaded = false
                    await updated_kyou.load_all(column_query, true)
                    opened_dialogs.value[i] = { ...opened_dialogs.value[i], kyou: updated_kyou }
                }
            }
        })();
    }

    async function update_check_kyous(_kyou: Array<Kyou>, _is_checked: boolean): Promise<void> {
        throw new Error('Not implemented')
    }

    async function reload_list(column_index: number): Promise<void> {
        return search(column_index, querys.value[column_index], true, false, true)
    }

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
                                    kyou_list_views.value[i].scroll_to(match_kyous_list_top_list.value[i])
                                    kyou_list_views.value[i].set_loading(false)
                                })
                            }))
                        })
                    }
                } else {
                    querys.value = saved_querys.concat()
                    querys_backup.value = saved_querys.concat()
                    for (let i = 0; i < saved_querys.length; i++) {
                        match_kyous_list.value.push([])
                    }
                }
            } finally {
                Promise.all(wait_promises).then(async () => {
                    focused_column_index.value = 0
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
        let my_abort_controller: AbortController | null = null

        // 検索する。Tickでまとめる
        try {
            if (!force_search) {
                if (deep_equals(querys_backup.value[column_index], query)) {
                    return
                }
            }
            querys.value[column_index] = query
            querys_backup.value[column_index] = query
            focused_query.value = query

            props.gkill_api.set_saved_mi_find_kyou_querys(querys.value)

            // 前の検索処理を中断する
            if (abort_controllers.value[column_index]) {
                abort_controllers.value[column_index].abort()
                abort_controllers.value[column_index] = new AbortController()
            }

            if (match_kyous_list.value[column_index]) {
                match_kyous_list.value[column_index] = []
            }

            nextTick(() => {
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                const kyou_list_view = kyou_list_views.value.filter((kyou_list_view: any) => kyou_list_view.get_query_id() === query.query_id)[0] as any
                if (kyou_list_view) {
                    if (inited.value && !preserve_scroll) {
                        kyou_list_view.scroll_to(0)
                    }
                    ((async () => kyou_list_view.set_loading(true))());
                }
            })

            const wait_promises = new Array<Promise<unknown>>()

            const req = new GetKyousRequest()
            my_abort_controller = req.abort_controller
            abort_controllers.value[column_index] = my_abort_controller
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

            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
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
            if (is_show_kyou_count_calendar.value) {
                update_focused_kyous_list(column_index)
            }

            await nextTick(() => {
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                const kyou_list_view = kyou_list_views.value.filter((kyou_list_view: any) => kyou_list_view.get_query_id() === query.query_id)[0] as any
                if (kyou_list_view) {
                    ((async () => kyou_list_view.set_loading(false))());
                }

                if (inited.value) {
                    if (preserve_scroll) {
                        kyou_list_view.scroll_to(match_kyous_list_top_list.value[column_index])
                    } else {
                        kyou_list_view.scroll_to(0)
                    }
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
            if (my_abort_controller && abort_controllers.value[column_index] === my_abort_controller) {
                nextTick(() => {
                    // eslint-disable-next-line @typescript-eslint/no-explicit-any
                    const kyou_list_view = kyou_list_views.value?.filter((kyou_list_view: any) => kyou_list_view.get_query_id() === query.query_id)[0] as any
                    if (kyou_list_view) {
                        kyou_list_view.set_loading(false)
                    }
                })
            }
        }
    }

    async function close_list_view(column_index: number): Promise<void> {
        return nextTick(() => {
            skip_search_this_tick.value = true
            focused_column_index.value = -1

            querys.value.splice(column_index, 1)
            querys_backup.value.splice(column_index, 1)

            if (abort_controllers.value[column_index]) {
                abort_controllers.value[column_index].abort()
                abort_controllers.value[column_index] = new AbortController()
            }

            match_kyous_list.value.splice(column_index, 1)
            match_kyous_list_top_list.value.splice(column_index, 1)
            abort_controllers.value.splice(column_index, 1)

            for (let i = column_index; i < querys.value.length; i++) {
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                const kyou_list_view = kyou_list_views.value[i] as any
                if (!kyou_list_view) {
                    continue
                }
                if (inited.value) {
                    kyou_list_view.scroll_to(match_kyous_list_top_list.value[i])
                }
            }
            props.gkill_api.set_saved_mi_find_kyou_querys(querys.value)
            props.gkill_api.set_saved_mi_scroll_indexs(match_kyous_list_top_list.value)
            nextTick(() => {
                skip_search_this_tick.value = true
                focused_column_index.value = 0
                focused_query.value = querys.value[0]
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
        if (board_name === "") {
            board_name = i18n.global.t("MI_ALL_TITLE")
        }

        let opened = false
        for (let i = 0; i < querys.value.length; i++) {
            const query = querys.value[i]
            if (query.mi_board_name === board_name) {
                focused_query.value = querys.value[i].clone()

                for (let j = 0; j < match_kyous_list.value[i].length; j++) {
                    focused_kyous_list.value.push(match_kyous_list.value[i][j])
                }
                focused_column_index.value = i
                opened = true
                break
            }
        }
        if (opened) {
            return
        }

        const query = query_editor_sidebar.value!.get_default_query()!.clone()
        query.query_id = props.gkill_api.generate_uuid()
        query.mi_board_name = board_name
        if (query.mi_board_name !== i18n.global.t("MI_ALL_TITLE")) {
            query.use_mi_board_name = true
        } else {
            query.use_mi_board_name = false
        }

        skip_search_this_tick.value = true
        add_list_view(query)
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
                kyou_list_views.value[target_column_index].scroll_to_time(kyou.related_time)
            }
        }
    }

    async function onDropBoardTask(e: DragEvent, find_kyou_query: FindKyouQuery) {
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
        if (before_board_name === after_board_name || !find_kyou_query.use_mi_board_name) {
            return
        }

        task.board_name = find_kyou_query.mi_board_name
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
        ;(async (): Promise<void> => {
            let column_query: FindKyouQuery | undefined
            for (let i = 0; i < match_kyous_list.value.length; i++) {
                if (match_kyous_list.value[i].some(k => k.id === kyou.id)) {
                    column_query = querys.value[i]
                    break
                }
            }
            const updated_kyou = kyou.clone()
            await updated_kyou.reload(true, column_query)
            updated_kyou.is_typed_data_loaded = false
            await updated_kyou.load_all(column_query, true)
            for (let i = 0; i < opened_dialogs.value.length; i++) {
                if (opened_dialogs.value[i].id === dialog_id) {
                    opened_dialogs.value[i] = { ...opened_dialogs.value[i], kyou: updated_kyou }
                    break
                }
            }
        })()
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
        nextTick(() => search(focused_column_index.value, querys.value[focused_column_index.value], true, update_cache))
    }

    function onSidebarUpdatedQuery(new_query: FindKyouQuery): void {
        if (!inited.value) {
            return
        }
        if (skip_search_this_tick.value || !props.application_config.rykv_hot_reload) {
            nextTick(() => skip_search_this_tick.value = false)
            return
        }
        search(focused_column_index.value, new_query)
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
        skip_search_this_tick.value = true
        focused_column_index.value = index
        focused_query.value = querys.value[index]
        focused_column_index.value = index
        if (is_show_kyou_count_calendar.value) {
            update_focused_kyous_list(index)
        }
        nextTick(() => skip_search_this_tick.value = false)
    }

    function onColumnClickedKyou(index: number, kyou: Kyou): void {
        focused_column_index.value = index
        skip_search_this_tick.value = true
        focused_query.value = querys.value[index]
        clicked_kyou_in_list_view(index, kyou)
    }

    function onColumnRequestedChangeFocusKyou(index: number, is_focus: boolean): void {
        focused_column_index.value = index
        skip_search_this_tick.value = true
        const query = querys.value[index].clone()
        query.is_focus_kyou_in_list_view = is_focus
        querys.value.splice(index, 1, query)
        querys_backup.value.splice(index, 1, query)
    }

    function onColumnRequestedSearch(index: number): void {
        focused_column_index.value = index
        skip_search_this_tick.value = true
        const query = querys.value[index].clone()
        query.query_id = props.gkill_api.generate_uuid()
        querys.value[index] = query
        querys.value.splice(index, 1, query)
        querys_backup.value.splice(index, 1, query)
        reload_list(index)
    }

    function onColumnRequestedChangeImageOnlyView(index: number, is_image_only: boolean): void {
        focused_column_index.value = index
        skip_search_this_tick.value = true
        focused_kyous_list.value = match_kyous_list.value[index]
        const query = querys.value[index].clone()
        query.query_id = props.gkill_api.generate_uuid()
        query.is_image_only = is_image_only
        querys.value[index] = query
        querys.value.splice(index, 1, query)
        querys_backup.value.splice(index, 1, query)
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
        'registered_kyou': (kyou: Kyou) => emits('registered_kyou', kyou),
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
        add_kyou_menu_style,
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
