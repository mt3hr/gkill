import { computed, nextTick, type Ref, ref, watch } from 'vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { Kyou } from '@/classes/datas/kyou'
import type { PlaingTimeIsViewProps } from '@/pages/views/plaing-time-is-view-props'
import type { PlaingTimeIsViewEmits } from '@/pages/views/plaing-time-is-view-emits'
import { GetKyousRequest } from '@/classes/api/req_res/get-kyous-request'
import generate_get_plaing_timeis_kyous_query from '@/classes/api/generate-get-plaing-timeis-kyous-query'
import moment from 'moment'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { Tag } from '@/classes/datas/tag'
import type { OpenedRykvDialog, RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import { useScopedEnterForKFTL } from '@/classes/use-scoped-enter-for-kftl'
import { useScopedCtrlVForClipboard } from '@/classes/use-scoped-ctrl-v-for-clipboard'
import { new_reload_batch, refresh_kyou, refresh_kyou_in_list } from '@/classes/kyou-reload'
import type { ComponentRef } from '@/classes/component-ref'

export function usePlaingTimeIsView(options: {
    props: PlaingTimeIsViewProps,
    emits: PlaingTimeIsViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const plaing_timeis_root = ref<HTMLElement | null>(null)
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

    const query: Ref<FindKyouQuery> = ref(new FindKyouQuery())
    const match_kyous_list: Ref<Array<Kyou>> = ref(new Array<Kyou>())
    const focused_column_index: Ref<number> = ref(0)
    const focused_kyous_list: Ref<Array<Kyou>> = ref(new Array<Kyou>())
    const focused_kyou: Ref<Kyou | null> = ref(null)
    const focused_time: Ref<Date> = ref(moment().toDate())
    const last_added_request_time: Ref<Date | null> = ref(null)
    const is_loading = ref(false)
    const skip_search_this_tick = ref(false)
    const abort_controller: Ref<AbortController> = ref(new AbortController())

    // ── Computed ──
    const kyou_list_view_height = computed(() => props.app_content_height)
    const timeis_kyou_list_view_width = computed(() => {
        const app_content_width = props.app_content_width
        if ((typeof app_content_width) !== "number") {
            return app_content_width
        }
        return app_content_width.valueOf() - 8/* --gkill-scrollbar-size */
    })

    // ── Watchers ──
    if (props.application_config.is_loaded) {
        nextTick(() => {
            search(false)
        })
    }

    watch(() => props.application_config.is_loaded, () => {
        nextTick(async () => {
            await nextTick(async () => {
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
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
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const kyou_list_view = kyou_list_views.value[focused_column_index.value] as any
        if (!kyou_list_view) {
            return
        }
        kyou_list_view.scroll_to_time(focused_time.value)
    })

    // ── Internal helpers ──
    function remove_kyou_from_list_by_id(list: Array<Kyou>, deleted_id: string): void {
        for (let i = list.length - 1; i >= 0; i--) {
            if (list[i].id === deleted_id) {
                list.splice(i, 1)
            }
        }
    }

    // ── Business logic ──
    function onDeletedKyou(deleted_kyou: Kyou): void {
        remove_kyou_from_list_by_id(match_kyous_list.value, deleted_kyou.id)
        remove_kyou_from_list_by_id(focused_kyous_list.value, deleted_kyou.id)
        if (focused_kyou.value?.id === deleted_kyou.id) {
            focused_kyou.value = null
        }
        emits('deleted_kyou', deleted_kyou)
    }

    async function reload_kyou(kyou: Kyou): Promise<void> {
        // 以前は3ブロックとも load_all の force_attached が無く添付タグを引き直せていなかった。
        // focused の分岐だけ reload(false) になっていて「更新後の最新版を取る」意図とも
        // 食い違っていたが、共通関数に寄せて reload(true) に揃う
        // 3ブロックは同じ更新から派生しているので、同じ値を渡して1往復に合流させる
        const requested_at = new_reload_batch()
        await refresh_kyou_in_list(match_kyous_list.value, kyou, {
            requested_at: requested_at,
            replace: (next_list) => { match_kyous_list.value = next_list },
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
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
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

            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            const kyou_list_view = kyou_list_views.value as any
            if (kyou_list_view) {
                kyou_list_view.scroll_to(1)
            }
            await nextTick(() => {
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                const kyou_list_view = kyou_list_views.value as any
                if (!kyou_list_view) {
                    return
                }
                kyou_list_view.scroll_to(0)
                kyou_list_view.set_loading(false)
                skip_search_this_tick.value = false
            })
        } catch (err: unknown) {
            // abortは握りつぶす
            if (!(err instanceof Error && (err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request")))) {
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
    useScopedCtrlVForClipboard(plaing_timeis_root, show_save_clipboard_to_file_dialog, enable_enter_shortcut)

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
    // Note: this view uses reload_list(false) for registered/updated_kyou, NOT reload_kyou
    const crudRelayHandlers = {
        'deleted_kyou': (kyou: Kyou) => onDeletedKyou(kyou),
        'deleted_tag': (tag: Tag) => emits('deleted_tag', tag),
        'deleted_text': (text: Text) => emits('deleted_text', text),
        'deleted_notification': (notification: Notification) => emits('deleted_notification', notification),
        'registered_kyou': (kyou: Kyou) => { reload_list(false); emits('registered_kyou', kyou) },
        'registered_tag': (tag: Tag) => emits('registered_tag', tag),
        'registered_text': (text: Text) => emits('registered_text', text),
        'registered_notification': (notification: Notification) => emits('registered_notification', notification),
        'updated_kyou': (kyou: Kyou) => { reload_list(false); emits('updated_kyou', kyou) },
        'updated_tag': (tag: Tag) => emits('updated_tag', tag),
        'updated_text': (text: Text) => emits('updated_text', text),
        'updated_notification': (notification: Notification) => emits('updated_notification', notification),
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
    }

    const reloadListRequestHandlers = {
        'requested_reload_kyou': () => reload_list(false),
        'requested_reload_list': () => reload_list(false),
    }

    const dialogReloadRequestHandlers = {
        'requested_reload_kyou': (kyou: Kyou) => reload_kyou(kyou),
        'requested_reload_list': () => reload_list(false),
    }

    const rykv_dialog_handler = {
        'requested_open_rykv_dialog': (kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => open_rykv_dialog(kind, kyou, payload),
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
        save_clipboard_to_file_dialog,
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
        timeis_kyou_list_view_width,

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
        show_save_clipboard_to_file_dialog,
        floating_action_button_style,

        // Event relay objects
        crudRelayHandlers,
        reloadListRequestHandlers,
        dialogReloadRequestHandlers,
        rykv_dialog_handler,
    }
}
