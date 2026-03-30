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
        const kyous_list = match_kyous.value
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
            await updated_kyou.reload(false, true)
            await updated_kyou.load_all()
            focused_kyou.value = updated_kyou
        }
    }

    function removeKyouFromListById(list: Array<Kyou>, deletedId: string): void {
        for (let i = list.length - 1; i >= 0; i--) {
            if (list[i].id === deletedId) {
                list.splice(i, 1)
            }
        }
    }

    function onDeletedKyou(deletedKyou: Kyou): void {
        removeKyouFromListById(match_kyous.value, deletedKyou.id)
        if (focused_kyou.value?.id === deletedKyou.id) {
            focused_kyou.value = null
        }
        emits('deleted_kyou', deletedKyou)
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

    const rykvDialogHandler = {
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
        rykvDialogHandler,
    }
}
