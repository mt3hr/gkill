import { computed, nextTick, type Ref, ref, watch } from 'vue'
import type { Kyou } from '@/classes/datas/kyou'
import type { SharedMiViewProps } from '@/pages/views/shared-mi-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { GetKyousRequest } from '@/classes/api/req_res/get-kyous-request'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { OpenedRykvDialog, RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import { new_reload_batch, refresh_kyou, refresh_kyou_in_list } from '@/classes/kyou-reload'

export function useSharedMiView(options: {
    props: SharedMiViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const kyou_list_view = ref()

    // ── State refs ──
    const match_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())
    const focused_time: Ref<Date> = ref(new Date())
    const share_title: Ref<string> = ref(props.share_title)
    const is_loading: Ref<boolean> = ref(true)
    const is_show_kyou_detail_view: Ref<boolean> = ref(true)
    const is_show_kyou_count_calendar: Ref<boolean> = ref(true)
    const focused_kyou: Ref<Kyou | null> = ref(null)
    const opened_dialogs: Ref<Array<OpenedRykvDialog>> = ref([])

    // ── Computed ──
    const kyou_list_view_height = computed(() => props.app_content_height)

    // ── Business logic ──
    async function load_content(): Promise<void> {
        const get_kyous_req = new GetKyousRequest()
        await props.gkill_api.delete_updated_gkill_caches()
        const res = await props.gkill_api.get_kyous(get_kyous_req)
        const wait_promises = new Array<Promise<Array<GkillError>>>()
        for (let i = 0; i < res.kyous.length; i++) {
            wait_promises.push(res.kyous[i].load_all())
        }
        await Promise.all(wait_promises)
        match_kyous.value = res.kyous
        is_loading.value = false
    }

    async function reload_kyou(kyou: Kyou): Promise<void> {
        // 以前は3ブロックとも load_all の force_attached が無く添付タグを引き直せていなかった。
        // 3ブロックは同じ更新から派生しているので、同じ値を渡して1往復に合流させる
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

    function remove_kyou_from_list_by_id(list: Array<Kyou>, deleted_id: string): void {
        for (let i = list.length - 1; i >= 0; i--) {
            if (list[i].id === deleted_id) {
                list.splice(i, 1)
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
            const refreshed = await refresh_kyou(kyou)
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

    // ── Watchers ──
    watch(() => focused_time.value, () => {
        if (!kyou_list_view.value) {
            return
        }
        let target_kyou: Kyou | null = null
        for (let i = 0; i < match_kyous.value.length; i++) {
            const kyou = match_kyous.value[i]
            if (kyou.related_time.getTime() >= focused_time.value.getTime()) {
                target_kyou = kyou
                break
            }
        }
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        (kyou_list_view as any).value.scroll_to_kyou(target_kyou)
    })

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
        'updated_kyou': (kyou: Kyou) => emits('updated_kyou', kyou),
        'updated_tag': (tag: Tag) => emits('updated_tag', tag),
        'updated_text': (text: Text) => emits('updated_text', text),
        'updated_notification': (notification: Notification) => emits('updated_notification', notification),
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
    }

    const rykv_dialog_handler = {
        'requested_open_rykv_dialog': (kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => open_rykv_dialog(kind, kyou, payload),
    }

    // ── Init ──
    nextTick(() => load_content())

    // ── Return ──
    return {
        // Template refs
        kyou_list_view,

        // State
        match_kyous,
        focused_time,
        share_title,
        is_loading,
        is_show_kyou_detail_view,
        is_show_kyou_count_calendar,
        focused_kyou,
        opened_dialogs,

        // Computed
        kyou_list_view_height,

        // Business logic
        load_content,
        reload_kyou,
        onDeletedKyou,
        open_rykv_dialog,
        close_rykv_dialog,

        // Event relay objects
        crudRelayHandlers,
        rykv_dialog_handler,
    }
}
