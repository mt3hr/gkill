import { log_unless_aborted } from '@/classes/abort-error'
import { computed, onMounted, ref } from 'vue'
import moment from 'moment'
import { format_duration, to_single_line } from '@/classes/format-date-time'
import type { TimeIsViewProps } from '@/pages/views/time-is-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import type TimeIsContextMenu from '@/pages/views/time-is-context-menu.vue'
import type EndTimeIsPlaingDialog from '@/pages/dialogs/end-time-is-plaing-dialog.vue'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

export function useTimeIsView(options: {
    props: TimeIsViewProps,
    emits: KyouViewEmits,
    context_menu: ReturnType<typeof ref<InstanceType<typeof TimeIsContextMenu> | null>>,
    end_timeis_plaing_dialog: ReturnType<typeof ref<InstanceType<typeof EndTimeIsPlaingDialog> | null>>,
}) {
    const { props, emits, context_menu, end_timeis_plaing_dialog } = options

    // ── Lifecycle ──
    // 表示時点で再生中TimeIsを最新化しておき、終了操作時の読み込み待ちをなくす。
    onMounted(async () => {
        const timeis = props.kyou.typed_timeis
        if (!props.show_timeis_plaing_end_button || !timeis || timeis.end_time) {
            return
        }
        try {
            await props.kyou.reload_with_typed_datas()
        } catch (err: unknown) {
            // 中断（画面を離れた・後発の検索に差し替わった）は正常なので出さない
            log_unless_aborted(err)
        }
    })

    // ── Computed ──
    const duration = computed(() => {
        const time1 = props.timeis.start_time
        let time2 = props.timeis.end_time

        time2 = time2 ? time2 : moment().toDate()
        const diff = Math.abs(time2.getTime() - time1.getTime())
        return to_single_line(format_duration(diff))
    })

    // ── CRUD relay handlers ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

    // ── Methods ──
    function show_context_menu(e: PointerEvent): void {
        if (props.enable_context_menu) {
            context_menu.value?.show(e)
        }
    }

    function show_end_timeis_dialog(): void {
        end_timeis_plaing_dialog.value?.show()
    }

    // ── Return ──
    return {
        // State
        duration,

        // Methods
        show_context_menu,
        show_end_timeis_dialog,

        // Event relay objects
        crudRelayHandlers,
    }
}

