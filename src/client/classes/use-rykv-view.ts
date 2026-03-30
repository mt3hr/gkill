import { i18n } from '@/i18n'
import router from '@/router'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { computed, nextTick, type Ref, ref, watch } from 'vue'
import { Kyou } from '@/classes/datas/kyou'
import type { rykvViewEmits } from '@/pages/views/rykv-view-emits'
import type { rykvViewProps } from '@/pages/views/rykv-view-props'
import { GetKyousRequest } from '@/classes/api/req_res/get-kyous-request'
import { GetKyousResponse } from '@/classes/api/req_res/get-kyous-response'
import moment from 'moment'
import { deepEquals } from '@/classes/deep-equals'
import { useScopedEnterForKFTL } from '@/classes/use-scoped-enter-for-kftl'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { Tag } from '@/classes/datas/tag'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import { resetDialogHistory } from '@/classes/use-dialog-history-stack'
import type { OpenedRykvDialog, RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import type { ComponentRef } from '@/classes/component-ref'

export function useRykvView(options: {
    props: rykvViewProps,
    emits: rykvViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const rykv_root = ref<HTMLElement | null>(null)
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
    const dnote_view = ref<ComponentRef | null>(null)
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
    const focused_column_checked_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
    const gps_log_map_start_time: Ref<Date> = ref(moment().toDate())
    const gps_log_map_end_time: Ref<Date> = ref(moment().toDate())
    const gps_log_map_marker_time: Ref<Date> = ref(moment().toDate())
    const is_show_kyou_detail_view: Ref<boolean> = ref(false)
    const is_show_kyou_count_calendar: Ref<boolean> = ref(false)
    const is_show_gps_log_map: Ref<boolean> = ref(false)
    const is_show_dnote: Ref<boolean> = ref(false)
    const drawer: Ref<boolean | null> = ref(false)
    const drawer_mode_is_mobile: Ref<boolean | null> = ref(false)
    const default_query: Ref<FindKyouQuery> = ref(new FindKyouQuery())
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
        { app_name: i18n.global.t('SAIHATE_APP_NAME'), page_name: 'saihate' },
    ])

    // ── Watchers ──
    watch(() => focused_time.value, () => {
        if (!kyou_list_views.value) {
            return
        }
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const kyou_list_view = kyou_list_views.value[focused_column_index.value] as any
        if (!kyou_list_view) {
            return
        }
        if (inited.value) {
            kyou_list_view.scroll_to_time(focused_time.value)
        }
    })

    watch(() => is_show_kyou_count_calendar.value, () => {
        if (props.is_shared_rykv_view) {
            return
        }
        if (is_show_kyou_count_calendar.value) {
            update_focused_kyous_list(focused_column_index.value)
        }
    })

    watch(() => is_show_dnote.value, async () => {
        if (props.is_shared_rykv_view) {
            return
        }
        dnote_view.value?.abort()
        if (is_show_dnote.value) {
            update_focused_kyous_list(focused_column_index.value)

            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const kyou_list_view = kyou_list_views.value[focused_column_index.value] as any
            if (!kyou_list_view) {
                return
            }
            while (kyou_list_view.get_is_loading()) {
                await sleep(500)
            }
            nextTick(() => {
                dnote_view.value?.reload(focused_kyous_list.value, focused_query.value)
            })
        } else {
            dnote_view.value?.abort()
        }
    })

    // ── Shared view init ──
    if (props.is_shared_rykv_view) {
        nextTick(async () => {
            is_loading.value = false
            inited.value = true
            await props.gkill_api.delete_updated_gkill_caches()
            const kyous = (await props.gkill_api.get_kyous(new GetKyousRequest())).kyous
            const wait_promises = new Array<Promise<unknown>>()
            for (let i = 0; i < kyous.length; i++) {
                wait_promises.push(kyous[i].load_all())
            }
            await Promise.all(wait_promises)
            match_kyous_list.value = [kyous]
            focused_kyous_list.value = kyous
            focused_column_index.value = 0
        })
    }

    // ── Internal helpers ──
    const sleep = (time: number) => new Promise<void>((r) => setTimeout(r, time))

    function update_focused_kyous_list(column_index: number): void {
        if (props.is_shared_rykv_view) {
            return
        }
        if (!match_kyous_list.value || match_kyous_list.value.length === 0) {
            return
        }
        focused_kyous_list.value = match_kyous_list.value[column_index]
    }

    function removeKyouFromListById(list: Array<Kyou>, deletedId: string): void {
        for (let i = list.length - 1; i >= 0; i--) {
            if (list[i].id === deletedId) {
                list.splice(i, 1)
            }
        }
    }

    function removeKyouFromMultiColumnLists(lists: Array<Array<Kyou>>, deletedId: string): void {
        for (let i = 0; i < lists.length; i++) {
            removeKyouFromListById(lists[i], deletedId)
        }
    }

    // ── Business logic ──
    function onDeletedKyou(deletedKyou: Kyou): void {
        removeKyouFromMultiColumnLists(match_kyous_list.value, deletedKyou.id)
        removeKyouFromListById(focused_kyous_list.value, deletedKyou.id)
        if (focused_kyou.value?.id === deletedKyou.id) {
            focused_kyou.value = null
        }
        emits('deleted_kyou', deletedKyou)
    }

    async function reload_kyou(kyou: Kyou): Promise<void> {
        (async (): Promise<void> => {
            for (let i = 0; i < match_kyous_list.value.length; i++) {
                const kyous_list = match_kyous_list.value[i]
                for (let j = 0; j < kyous_list.length; j++) {
                    const kyou_in_list = kyous_list[j]
                    if (kyou.id === kyou_in_list.id) {
                        const updated_kyou = kyou.clone()
                        await updated_kyou.reload(false, true)
                        await updated_kyou.load_all()
                        kyous_list.splice(j, 1, updated_kyou)
                    }
                }
            }
        })();
        (async (): Promise<void> => {
            if (focused_kyou.value && focused_kyou.value.id === kyou.id) {
                const updated_kyou = kyou.clone()
                await updated_kyou.reload(false, true)
                await updated_kyou.load_all()
                focused_kyou.value = updated_kyou
            }
        })();
    }

    async function reload_list(column_index: number): Promise<void> {
        return search(column_index, querys.value[column_index], true)
    }

    async function init(): Promise<void> {
        if (inited.value) {
            return
        }
        return nextTick(async () => {
            const waitPromises = new Array<Promise<unknown>>()
            try {
                // スクロール位置の復元
                match_kyous_list_top_list.value = props.gkill_api.get_saved_rykv_scroll_indexs()

                // 前回開いていた列があれば復元する
                skip_search_this_tick.value = true
                const saved_querys = props.gkill_api.get_saved_rykv_find_kyou_querys()
                default_query.value = query_editor_sidebar.value!.get_default_query()!.clone()
                default_query.value.query_id = props.gkill_api.generate_uuid()
                if (saved_querys.length.valueOf() === 0) {
                    const cloned_default_query = default_query.value.clone()
                    cloned_default_query.query_id = props.gkill_api.generate_uuid()
                    saved_querys.push(cloned_default_query)
                }

                if (props.application_config.rykv_hot_reload) {
                    for (let i = 0; i < saved_querys.length; i++) {
                        await nextTick(() => {
                            skip_search_this_tick.value = true
                            waitPromises.push(search(i, saved_querys[i], true).then(async () => {
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
                Promise.all(waitPromises).then(async () => {
                    focused_column_index.value = 0
                    if (querys.value[focused_column_index.value].use_calendar && querys.value[focused_column_index.value].calendar_start_date && querys.value[focused_column_index.value].calendar_end_date) {
                        gps_log_map_start_time.value = querys.value[focused_column_index.value].calendar_start_date!
                        gps_log_map_end_time.value = querys.value[focused_column_index.value].calendar_end_date!
                    }

                    inited.value = true
                    drawer_mode_is_mobile.value = !(props.app_content_width.valueOf() >= 760)
                    drawer.value = props.app_content_width.valueOf() >= 760
                    is_loading.value = false
                    skip_search_this_tick.value = false
                })
                nextTick(() => default_query.value = query_editor_sidebar.value!.get_default_query()!.clone())
            }
        })
    }

    async function search(column_index: number, query: FindKyouQuery, force_search?: boolean, update_cache?: boolean): Promise<void> {
        const query_id = query.query_id
        await dnote_view.value?.abort()
        // 検索する。Tickでまとめる
        try {
            if (!force_search) {
                if (deepEquals(querys_backup.value[column_index], query)) {
                    return
                }
            }

            querys.value[column_index] = query
            querys_backup.value[column_index] = query
            focused_query.value = query

            props.gkill_api.set_saved_rykv_find_kyou_querys(querys.value)

            focused_column_checked_kyous.value = []

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
                    if (inited.value) {
                        kyou_list_view.scroll_to(0)
                    }
                    ((async () => kyou_list_view.set_loading(true))());
                }
            })

            const waitPromises = new Array<Promise<unknown>>()

            const req = new GetKyousRequest()
            abort_controllers.value[column_index] = req.abort_controller
            req.query = query.clone()
            req.query.parse_words_and_not_words()
            if (update_cache) {
                waitPromises.push(delete_gkill_kyou_cache(null))
                req.query.update_cache = true
            } else {
                waitPromises.push(props.gkill_api.delete_updated_gkill_caches())
            }

            let res = new GetKyousResponse()
            waitPromises.push(props.gkill_api.get_kyous(req).then(response => res = response))

            await Promise.all(waitPromises)

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
            if (!props.is_shared_rykv_view) {
                if (is_show_kyou_count_calendar.value || is_show_dnote.value) {
                    update_focused_kyous_list(column_index)
                }
            }
            await nextTick(() => {
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                const kyou_list_view = kyou_list_views.value.filter((kyou_list_view: any) => kyou_list_view.get_query_id() === query.query_id)[0] as any
                if (kyou_list_view) {
                    ((async () => kyou_list_view.set_loading(false))());
                }

                if (inited.value) {
                    kyou_list_view.scroll_to(0)
                    skip_search_this_tick.value = false
                }
                dnote_view.value?.reload(focused_kyous_list.value, focused_query.value)
            })
        } catch (err: unknown) {
            // abortは握りつぶす
            if (!(err instanceof Error && (err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request")))) {
                // abort以外はエラー出力する
                console.error(err)
            }
        }
    }

    async function close_list_view(column_index: number): Promise<void> {
        return nextTick(() => {
            skip_search_this_tick.value = true
            focused_column_index.value = -1
            focused_query.value = querys.value[focused_column_index.value]

            querys.value.splice(column_index, 1)
            querys_backup.value.splice(column_index, 1)

            if (abort_controllers.value[column_index]) {
                abort_controllers.value[column_index].abort()
                abort_controllers.value[column_index] = new AbortController()
            }

            match_kyous_list.value.splice(column_index, 1)
            match_kyous_list_top_list.value.splice(column_index, 1)
            abort_controllers.value.splice(column_index, 1)

            match_kyous_list_top_list.value.splice(column_index, 1)
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
            props.gkill_api.set_saved_rykv_find_kyou_querys(querys.value)
            props.gkill_api.set_saved_rykv_scroll_indexs(match_kyous_list_top_list.value)
            nextTick(() => {
                skip_search_this_tick.value = true
                focused_column_index.value = 0
            })
        })
    }

    function add_list_view(query?: FindKyouQuery): void {
        match_kyous_list.value.push(new Array<Kyou>())
        match_kyous_list_top_list.value.push(0)
        // 初期化されていないときはDefaultQueryがない。
        // その場合は初期値のFindKyouQueryをわたして初期化してもらう
        const dq = query_editor_sidebar.value?.get_default_query()?.clone()
        if (query) {
            querys.value.push(query)
            focused_query.value = query
        } else if (dq) {
            dq.query_id = props.gkill_api.generate_uuid()
            querys.value.push(dq)
            focused_query.value = dq
        } else {
            const query = new FindKyouQuery()
            query.query_id = props.gkill_api.generate_uuid()
            querys.value.push(query)
            focused_query.value = query
        }
        if (inited.value) {
            focused_column_index.value = querys.value.length - 1
        }
        props.gkill_api.set_saved_rykv_find_kyou_querys(querys.value)
        props.gkill_api.set_saved_rykv_scroll_indexs(match_kyous_list_top_list.value)
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
            if (inited.value && column_index !== target_column_index) {
                kyou_list_views.value[target_column_index].scroll_to_time(kyou.related_time)
            }
        }
    }

    function onFocusedKyouFromSubView(kyou: Kyou): void {
        focused_kyou.value = kyou
        gps_log_map_start_time.value = kyou.related_time
        gps_log_map_end_time.value = kyou.related_time
        gps_log_map_marker_time.value = kyou.related_time
    }

    async function update_check_kyous(_kyou: Array<Kyou>, _is_checked: boolean): Promise<void> {
        throw new Error('Not implemented')
    }

    // ── Template event handlers ──

    function toggleDrawer(): void {
        if (inited.value) { drawer.value = !drawer.value }
    }

    async function navigateToPage(page_name: string): Promise<void> {
        await resetDialogHistory()
        router.replace('/' + page_name + '?loaded=true')
    }

    async function toggleDnote(): Promise<void> {
        await dnote_view.value?.abort()
        is_show_dnote.value = !is_show_dnote.value
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
        if (new_query.use_calendar && new_query.calendar_start_date && new_query.calendar_end_date) {
            gps_log_map_start_time.value = new_query.calendar_start_date
            gps_log_map_end_time.value = new_query.calendar_end_date
        }
    }

    function onSidebarInited(): void {
        if (!received_init_request.value) { init() }
        received_init_request.value = true
    }

    function onColumnScrollList(index: number, scroll_top: number): void {
        match_kyous_list_top_list.value[index] = scroll_top
        if (inited.value) {
            props.gkill_api.set_saved_rykv_scroll_indexs(match_kyous_list_top_list.value)
        }
    }

    function onColumnClickedListView(index: number): void {
        if (props.is_shared_rykv_view) {
            return
        }
        skip_search_this_tick.value = true
        focused_query.value = querys.value[index]
        if (is_show_kyou_count_calendar.value || is_show_dnote.value) {
            update_focused_kyous_list(index)
        }
        focused_column_index.value = index
        nextTick(() => skip_search_this_tick.value = false)
    }

    function onColumnClickedKyou(index: number, kyou: Kyou): void {
        skip_search_this_tick.value = true
        focused_column_index.value = index
        focused_query.value = querys.value[index]
        clicked_kyou_in_list_view(index, kyou)
        onFocusedKyouFromSubView(kyou)
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
        const query = querys.value[index].clone()
        query.query_id = props.gkill_api.generate_uuid()
        query.is_image_only = is_image_only
        querys.value[index] = query
        querys.value.splice(index, 1, query)
        querys_backup.value.splice(index, 1, query)
        search(index, query, true)
    }

    function onColumnRequestedReloadList(index: number): void {
        const query = querys.value[index].clone()
        query.query_id = props.gkill_api.generate_uuid()
        querys.value[index] = query
        reload_list(index)
    }

    function onRequestedFocusTime(time: Date): void {
        focused_time.value = time
        gps_log_map_start_time.value = time
        gps_log_map_end_time.value = time
        gps_log_map_marker_time.value = time
    }

    function onGpsLogMapRequestedFocusTime(time: Date): void {
        focused_time.value = time
    }

    function onAddColumnClick(): void {
        add_list_view()
        skip_search_this_tick.value = true
        if (props.application_config.rykv_hot_reload) {
            search(querys.value.length - 1, querys.value[querys.value.length - 1], true)
        }
    }

    function open_rykv_dialog(kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload): void {
        opened_dialogs.value.push({
            id: props.gkill_api.generate_uuid(),
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

    function floatingActionButtonStyle() {
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

    const subViewFocusHandlers = {
        'focused_kyou': (kyou: Kyou) => onFocusedKyouFromSubView(kyou),
        'clicked_kyou': (kyou: Kyou) => onFocusedKyouFromSubView(kyou),
    }

    const rykvDialogHandler = {
        'requested_open_rykv_dialog': (kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => open_rykv_dialog(kind, kyou, payload),
    }

    // ── Keyboard shortcut ──
    const enable_enter_shortcut = ref(true)
    useScopedEnterForKFTL(rykv_root, show_kftl_dialog, enable_enter_shortcut)

    // ── Return ──
    return {
        // Template refs
        rykv_root,
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
        dnote_view,
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
        focused_column_checked_kyous,
        gps_log_map_start_time,
        gps_log_map_end_time,
        gps_log_map_marker_time,
        is_show_kyou_detail_view,
        is_show_kyou_count_calendar,
        is_show_gps_log_map,
        is_show_dnote,
        drawer,
        drawer_mode_is_mobile,
        default_query,
        is_loading,
        inited,

        // Computed
        kyou_list_view_height,
        add_kyou_menu_style,
        page_list,

        // Template event handlers
        toggleDrawer,
        navigateToPage,
        toggleDnote,
        onSidebarRequestedSearch,
        onSidebarUpdatedQuery,
        onSidebarInited,
        onColumnScrollList,
        onColumnClickedListView,
        onColumnClickedKyou,
        onColumnRequestedChangeFocusKyou,
        onColumnRequestedSearch,
        onColumnRequestedChangeImageOnlyView,
        onColumnRequestedReloadList,
        onRequestedFocusTime,
        onGpsLogMapRequestedFocusTime,
        onAddColumnClick,
        onFocusedKyouFromSubView,
        close_list_view,
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
        floatingActionButtonStyle,

        // Event relay objects
        crudRelayHandlers,
        allColumnsRequestHandlers,
        subViewFocusHandlers,
        rykvDialogHandler,
    }
}
