import { computed, watch, type Ref, ref, nextTick, onUnmounted } from 'vue'
import type { RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import { format_time } from '@/classes/format-date-time'
import { Kyou } from '@/classes/datas/kyou'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import type { KyouViewProps } from '@/pages/views/kyou-view-props'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type KmemoView from '@/pages/views/kmemo-view.vue'
import type KCView from '@/pages/views/kc-view.vue'
import type MiKyouView from '@/pages/views/mi-kyou-view.vue'
import type NlogView from '@/pages/views/nlog-view.vue'
import type LantanaView from '@/pages/views/lantana-view.vue'
import type TimeIsView from '@/pages/views/time-is-view.vue'
import type UrLogView from '@/pages/views/ur-log-view.vue'
import type IdfKyouView from '@/pages/views/idf-kyou-view.vue'
import type ReKyouView from '@/pages/views/re-kyou-view.vue'
import type GitCommitLogView from '@/pages/views/git-commit-log-view.vue'

export function useKyouView(options: {
    props: KyouViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const kmemo_view = ref<InstanceType<typeof KmemoView> | null>(null)
    const kc_view = ref<InstanceType<typeof KCView> | null>(null)
    const mi_view = ref<InstanceType<typeof MiKyouView> | null>(null)
    const nlog_view = ref<InstanceType<typeof NlogView> | null>(null)
    const lantana_view = ref<InstanceType<typeof LantanaView> | null>(null)
    const timeis_view = ref<InstanceType<typeof TimeIsView> | null>(null)
    const urlog_view = ref<InstanceType<typeof UrLogView> | null>(null)
    const idf_kyou_view = ref<InstanceType<typeof IdfKyouView> | null>(null)
    const rekyou_view = ref<InstanceType<typeof ReKyouView> | null>(null)
    const git_commit_log_view = ref<InstanceType<typeof GitCommitLogView> | null>(null)

    // ── State refs ──
    const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())

    // ── Lifecycle ──
    onUnmounted(() => {
        cloned_kyou.value.abort_controller.abort()
        cloned_kyou.value.abort_controller = new AbortController()
    })

    // ── Computed ──
    const related_time = computed(() => format_time(props.kyou.related_time))
    const update_time = computed(() => format_time(props.kyou.update_time))
    const rep_name = computed(() => props.kyou.rep_name)

    const kyou_class = computed(() => {
        let highlighted = false
        for (let i = 0; i < props.highlight_targets.length; i++) {
            if (props.highlight_targets[i].id === props.kyou.id
                && props.highlight_targets[i].create_time.getTime() === props.kyou.create_time.getTime()
                && props.highlight_targets[i].update_time.getTime() === props.kyou.update_time.getTime()) {
                highlighted = true
                break
            }
        }
        if (highlighted) {
            return "highlighted_kyou"
        }
        return ""
    })

    // ── Watchers ──
    watch(() => props.kyou, async () => {
        cloned_kyou.value.abort_controller.abort()
        cloned_kyou.value = props.kyou.clone()
        cloned_kyou.value.abort_controller = new AbortController()
        if (props.force_show_latest_kyou_info) {
            await cloned_kyou.value.reload(true, props.force_show_latest_kyou_info);//最新を読み込むためにReload
        }
        (() => load_attached_infos())(); //非同期で実行してほしい
    })

    // ── Initialization (IIFE) ──
    ;(async () => {
        if (props.force_show_latest_kyou_info) {
            await cloned_kyou.value.reload(true, props.force_show_latest_kyou_info);//最新を読み込むためにReload
        }
        load_attached_infos()
    })(); //非同期で実行してほしい

    // ── Internal helpers ──
    async function load_attached_infos(): Promise<void> {
        try {
            const awaitPromises = new Array<Promise<Array<GkillError>>>()
            try {
                awaitPromises.push(cloned_kyou.value.load_typed_datas())
                if (props.show_attached_tags) {
                    awaitPromises.push(cloned_kyou.value.load_attached_tags())
                }
                if (props.show_attached_texts) {
                    awaitPromises.push(cloned_kyou.value.load_attached_texts())
                }
                if (props.show_attached_notifications) {
                    awaitPromises.push(cloned_kyou.value.load_attached_notifications())
                }
                if (props.show_attached_timeis) {
                    awaitPromises.push(cloned_kyou.value.load_attached_timeis())
                }
                await Promise.all(awaitPromises)
            } catch (err: unknown) {
                // abortは握りつぶす
                if (!(err instanceof Error && (err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request")))) {
                    // abort以外はエラー出力する
                    console.error(err)
                }
            }
        } catch (err: unknown) {
            // abortは握りつぶす
            if (!(err instanceof Error && (err.message.includes("signal is aborted without reason") || err.message.includes("user aborted a request")))) {
                // abort以外はエラー出力する
                console.error(err)
            }
        }
    }

    // ── Business logic ──
    async function show_context_menu(e: PointerEvent): Promise<void> {
        if (!props.enable_context_menu) {
            return
        }
        kmemo_view.value?.show_context_menu(e)
        kc_view.value?.show_context_menu(e)
        mi_view.value?.show_context_menu(e)
        nlog_view.value?.show_context_menu(e)
        lantana_view.value?.show_context_menu(e)
        timeis_view.value?.show_context_menu(e)
        urlog_view.value?.show_context_menu(e)
        idf_kyou_view.value?.show_context_menu(e)
        rekyou_view.value?.show_context_menu(e)
        git_commit_log_view.value?.show_context_menu(e)
    }

    function show_kyou_dialog(): void {
        if (props.enable_dialog) {
            emits('requested_open_rykv_dialog', 'kyou', cloned_kyou.value)
        }
    }

    // ── Event relay objects ──
    const crudRelayHandlers = {
        'deleted_kyou': (kyou: Kyou) => emits('deleted_kyou', kyou),
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
        'requested_reload_kyou': (kyou: Kyou) => emits('requested_reload_kyou', kyou),
        'requested_reload_list': () => emits('requested_reload_list'),
        'requested_update_check_kyous': (kyous: Array<Kyou>, checked: boolean) => emits('requested_update_check_kyous', kyous, checked),
        'requested_open_rykv_dialog': (kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => emits('requested_open_rykv_dialog', kind, kyou, payload),
    }

    // ── Template event handlers ──
    function onRootClick(): void {
        nextTick(() => {
            emits('focused_kyou', cloned_kyou.value)
            emits('clicked_kyou', cloned_kyou.value)
        })
    }

    // ── Return ──
    return {
        // Template refs
        kmemo_view,
        kc_view,
        mi_view,
        nlog_view,
        lantana_view,
        timeis_view,
        urlog_view,
        idf_kyou_view,
        rekyou_view,
        git_commit_log_view,

        // State
        cloned_kyou,

        // Computed
        related_time,
        update_time,
        rep_name,
        kyou_class,

        // Business logic
        show_context_menu,
        show_kyou_dialog,
        onRootClick,

        // Event relay objects
        crudRelayHandlers,
    }
}
