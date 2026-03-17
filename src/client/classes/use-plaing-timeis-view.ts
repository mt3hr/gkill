import { computed, nextTick, type Ref, ref, watch } from 'vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { Kyou } from '@/classes/datas/kyou'
import type { PlaingTimeIsViewProps } from '@/pages/views/plaing-timeis-view-props'
import type { PlaingTimeIsViewEmits } from '@/pages/views/plaing-timeis-emits'
import { GetKyousRequest } from '@/classes/api/req_res/get-kyous-request'
import generate_get_plaing_timeis_kyous_query from '@/classes/api/generate-get-plaing-timeis-kyous-query'
import moment from 'moment'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { Tag } from '@/classes/datas/tag'
import type { OpenedRykvDialog, RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import { useScopedEnterForKFTL } from '@/classes/use-scoped-enter-for-kftl'

export function usePlaingTimeisView(options: {
    props: PlaingTimeIsViewProps,
    emits: PlaingTimeIsViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const plaing_timeis_root = ref<HTMLElement | null>(null)
    const add_mi_dialog = ref<any>(null)
    const add_nlog_dialog = ref<any>(null)
    const add_lantana_dialog = ref<any>(null)
    const add_timeis_dialog = ref<any>(null)
    const add_urlog_dialog = ref<any>(null)
    const kftl_dialog = ref<any>(null)
    const add_kc_dialog = ref<any>(null)
    const mkfl_dialog = ref<any>(null)
    const upload_file_dialog = ref<any>(null)
    const kyou_list_views = ref()

    // ── State refs ──
    const enable_context_menu = ref(true)
    const enable_dialog = ref(true)
    const opened_dialogs: Ref<Array<OpenedRykvDialog>> = ref([])

    const query: Ref<FindKyouQuery> = ref(new FindKyouQuery())
    const match_kyous_list: Ref<Array<Kyou>> = ref(new Array<Kyou>())
    const focused_column_index: Ref<number> = ref(0)
    const focused_kyous_list: Ref<Array<Kyou>> = ref(new Array<Kyou>())
    const focused_kyou: Ref<Kyou | null> = ref(null)
    const focused_time: Ref<Date> = ref(moment().toDate())
    const last_added_request_time: Ref<Date | null> = ref(null)
    const position_x: Ref<Number> = ref(0)
    const position_y: Ref<Number> = ref(0)
    const is_loading = ref(false)
    const skip_search_this_tick = ref(false)
    const abort_controller: Ref<AbortController> = ref(new AbortController())

    // ── Computed ──
    const kyou_list_view_height = computed(() => props.app_content_height)
    const add_kyou_menu_style = computed(() => `{ position: absolute; left: ${position_x.value}px; top: ${position_y.value}px; }`)

    // ── Watchers ──
    if (props.application_config.is_loaded) {
        nextTick(() => {
            search(false)
        })
    }

    watch(() => props.application_config.is_loaded, () => {
        nextTick(async () => {
            await nextTick(async () => {
                const kyou_list_view = kyou_list_views.value as any
                if (!kyou_list_view) {
                    return
                }
                kyou_list_view.set_loading(true)
                return nextTick(() => { }) // loading表記切り替え待ち
            })
            search(false)
        })
    })

    watch(() => focused_time.value, () => {
        if (!kyou_list_views.value) {
            return
        }
        const kyou_list_view = kyou_list_views.value[focused_column_index.value] as any
        if (!kyou_list_view) {
            return
        }
        kyou_list_view.scroll_to_time(focused_time.value)
    })

    // ── Internal helpers ──
    function removeKyouFromListById(list: Array<Kyou>, deletedId: string): void {
        for (let i = list.length - 1; i >= 0; i--) {
            if (list[i].id === deletedId) {
                list.splice(i, 1)
            }
        }
    }

    // ── Business logic ──
    function onDeletedKyou(deletedKyou: Kyou): void {
        removeKyouFromListById(match_kyous_list.value, deletedKyou.id)
        removeKyouFromListById(focused_kyous_list.value, deletedKyou.id)
        if (focused_kyou.value?.id === deletedKyou.id) {
            focused_kyou.value = null
        }
        emits('deleted_kyou', deletedKyou)
    }

    async function reload_kyou(kyou: Kyou): Promise<void> {
        const kyous_list = match_kyous_list.value
        for (let j = 0; j < kyous_list.length; j++) {
            const kyou_in_list = kyous_list[j]
            if (kyou.id === kyou_in_list.id) {
                const updated_kyou = kyou.clone()
                await updated_kyou.reload(false, true)
                await updated_kyou.load_all()
                kyous_list.splice(j, 1, updated_kyou)
            }
        }
        if (focused_kyou.value && focused_kyou.value.id === kyou.id) {
            const updated_kyou = kyou.clone()
            await updated_kyou.reload(false, false)
            await updated_kyou.load_all()
            focused_kyou.value = updated_kyou
        }
    }

    async function search(update_cache: boolean): Promise<void> {
        if (is_loading.value) {
            return
        }
        is_loading.value = true
        // 検索する。Tickでまとめる
        query.value = generate_get_plaing_timeis_kyous_query(last_added_request_time.value)
        try {
            if (abort_controller.value) {
                abort_controller.value.abort()
                abort_controller.value = new AbortController()
            }

            if (match_kyous_list.value) {
                match_kyous_list.value.splice(0)
            }

            match_kyous_list.value.splice(0)
            focused_kyous_list.value.splice(0)

            await nextTick(async () => {
                const kyou_list_view = kyou_list_views.value as any
                if (!kyou_list_view) {
                    return
                }
                kyou_list_view.set_loading(true)
                return nextTick(() => { }) // loading表記切り替え待ち
            })

            const req = new GetKyousRequest()
            abort_controller.value = req.abort_controller
            req.query = query.value.clone()
            req.query.parse_words_and_not_words()
            if (update_cache) {
                req.query.update_cache = true
            }

            await props.gkill_api.delete_updated_gkill_caches()
            const res = await props.gkill_api.get_kyous(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }
            match_kyous_list.value.push(...res.kyous)
            focused_kyous_list.value.push(...res.kyous)

            const kyou_list_view = kyou_list_views.value as any
            if (kyou_list_view) {
                kyou_list_view.scroll_to(1)
            }
            await nextTick(() => {
                const kyou_list_view = kyou_list_views.value as any
                if (!kyou_list_view) {
                    return
                }
                kyou_list_view.scroll_to(0)
                kyou_list_view.set_loading(false)
                skip_search_this_tick.value = false
            })
        } catch (err: any) {
            // abortは握りつぶす
            if (!(err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request"))) {
                // abort以外はエラー出力する
                console.error(err)
            }
        } finally {
            is_loading.value = false
        }
    }

    async function reload_list(update_cache: boolean): Promise<void> {
        // nextTickでまとめる
        match_kyous_list.value.splice(0)

        await search(update_cache)
        if (!kyou_list_views.value) {
            return
        }
        kyou_list_views.value.scroll_to(0)
    }

    function set_last_added_request_time(time: Date): void {
        last_added_request_time.value = time
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

    // ── Enter key → KFTL dialog ──
    const enable_enter_shortcut = ref(true)
    useScopedEnterForKFTL(plaing_timeis_root, show_kftl_dialog, enable_enter_shortcut)

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
    // Note: this view uses reload_list(false) for registered/updated_kyou, NOT reload_kyou
    const crudRelayHandlers = {
        'deleted_kyou': (...args: any[]) => onDeletedKyou(args[0] as Kyou),
        'deleted_tag': (...args: any[]) => emits('deleted_tag', args[0] as Tag),
        'deleted_text': (...args: any[]) => emits('deleted_text', args[0] as Text),
        'deleted_notification': (...args: any[]) => emits('deleted_notification', args[0] as Notification),
        'registered_kyou': (...args: any[]) => { reload_list(false); emits('registered_kyou', args[0] as Kyou) },
        'registered_tag': (...args: any[]) => emits('registered_tag', args[0] as Tag),
        'registered_text': (...args: any[]) => emits('registered_text', args[0] as Text),
        'registered_notification': (...args: any[]) => emits('registered_notification', args[0] as Notification),
        'updated_kyou': (...args: any[]) => { reload_list(false); emits('updated_kyou', args[0] as Kyou) },
        'updated_tag': (...args: any[]) => emits('updated_tag', args[0] as Tag),
        'updated_text': (...args: any[]) => emits('updated_text', args[0] as Text),
        'updated_notification': (...args: any[]) => emits('updated_notification', args[0] as Notification),
        'received_errors': (...args: any[]) => emits('received_errors', args[0] as Array<GkillError>),
        'received_messages': (...args: any[]) => emits('received_messages', args[0] as Array<GkillMessage>),
    }

    const reloadListRequestHandlers = {
        'requested_reload_kyou': () => reload_list(false),
        'requested_reload_list': () => reload_list(false),
    }

    const dialogReloadRequestHandlers = {
        'requested_reload_kyou': (...args: any[]) => reload_kyou(args[0] as Kyou),
        'requested_reload_list': () => reload_list(false),
    }

    const rykvDialogHandler = {
        'requested_open_rykv_dialog': (...args: any[]) => open_rykv_dialog(args[0], args[1], args[2]),
    }

    // ── Return ──
    return {
        // Template refs
        plaing_timeis_root,
        add_mi_dialog,
        add_nlog_dialog,
        add_lantana_dialog,
        add_timeis_dialog,
        add_urlog_dialog,
        kftl_dialog,
        add_kc_dialog,
        mkfl_dialog,
        upload_file_dialog,
        kyou_list_views,

        // State
        enable_context_menu,
        enable_dialog,
        opened_dialogs,
        query,
        match_kyous_list,
        focused_kyou,
        is_loading,

        // Computed
        kyou_list_view_height,
        add_kyou_menu_style,

        // Business logic
        reload_list,
        search,
        set_last_added_request_time,
        open_rykv_dialog,
        close_rykv_dialog,

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
        reloadListRequestHandlers,
        dialogReloadRequestHandlers,
        rykvDialogHandler,
    }
}
