import { i18n } from '@/i18n'
import { type Ref, ref, watch } from 'vue'
import { MiReKyou } from '@/classes/datas/mi-re-kyou'
import moment from 'moment'
import { GkillError } from '@/classes/api/gkill-error'
import { AddMiReKyouRequest } from '@/classes/api/req_res/add-mi-re-kyou-request'
import { Notification } from '@/classes/datas/notification'
import { AddNotificationRequest } from '@/classes/api/req_res/add-notification-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { AddMiReKyouViewProps } from '@/pages/views/add-mi-re-kyou-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import type { ComponentRef } from '@/classes/component-ref'
import { useMiReKyouScheduleFields } from '@/classes/use-mi-re-kyou-schedule-fields'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

export function useAddMiReKyouView(options: {
    props: AddMiReKyouViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const add_notification_views = ref<ComponentRef | null>(null)

    // ── State refs ──
    const is_requested_submit = ref(false)
    const show_kyou: Ref<boolean> = ref(true)
    const id: Ref<string> = ref(props.gkill_api.generate_uuid())
    const notifications: Ref<Array<Notification>> = ref(new Array<Notification>())

    // ── 板名・日時のフォーム ──
    const schedule = useMiReKyouScheduleFields({ props, emits })

    // ── Watchers ──
    watch(() => props.application_config, () => schedule.load_mi_board_names())

    // ── Init ──
    schedule.load_mi_board_names()

    // ── Business logic ──
    function reset(): void {
        schedule.clear_estimate_start_date_time()
        schedule.clear_estimate_end_date_time()
        schedule.clear_limit_date_time()
        schedule.mi_board_name.value = props.application_config.mi_default_board
        notifications.value.splice(0)
    }

    async function save(): Promise<void> {
        try {
            is_requested_submit.value = true

            // Notification チェック
            // おかしかったらnullが戻ってくるので中断する
            const notification_results = new Array<Notification>()
            if (add_notification_views.value) {
                for (let i = 0; i < add_notification_views.value.length; i++) {
                    const notification = await add_notification_views.value[i].get_notification()
                    if (!notification) {
                        return
                    }
                    notification_results.push(notification)
                }
            }

            // リポスト対象がなければエラーメッセージを出力する
            if (!props.kyou || props.kyou.id === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.not_found_mi_rekyou_target
                error.error_message = i18n.global.t("NOT_FOUND_MI_REKYOU_TARGET_ERROR_MESSAGE")
                emits('received_errors', [error])
                return
            }

            // 日時の入力チェック
            if (!schedule.validate()) {
                return
            }

            // 作成するMiReKyou情報を用意する
            const times = schedule.resolve_times()
            const new_mirekyou = new MiReKyou()
            new_mirekyou.id = id.value
            new_mirekyou.is_deleted = false
            new_mirekyou.target_id = props.kyou.id
            new_mirekyou.is_checked = false
            new_mirekyou.board_name = schedule.mi_board_name.value
            new_mirekyou.estimate_start_time = times.estimate_start_time
            new_mirekyou.estimate_end_time = times.estimate_end_time
            new_mirekyou.limit_time = times.limit_time
            new_mirekyou.related_time = new Date(Date.now())
            new_mirekyou.create_app = "gkill"
            new_mirekyou.create_device = props.application_config.device
            new_mirekyou.create_time = new Date(Date.now())
            new_mirekyou.create_user = props.application_config.user_id
            new_mirekyou.update_app = "gkill"
            new_mirekyou.update_device = props.application_config.device
            new_mirekyou.update_time = new Date(Date.now())
            new_mirekyou.update_user = props.application_config.user_id

            // 追加リクエストを飛ばす
            await delete_gkill_kyou_cache(new_mirekyou.id)
            const req = new AddMiReKyouRequest()
            req.mirekyou = new_mirekyou
            req.want_response_kyou = true
            const res = await props.gkill_api.add_mirekyou(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }

            // Notification 追加
            for (let i = 0; i < notification_results.length; i++) {
                await delete_gkill_kyou_cache(notification_results[i].id)
                const notif_req = new AddNotificationRequest()
                notif_req.notification = notification_results[i]
                const notif_res = await props.gkill_api.add_notification(notif_req)
                if (notif_res.errors && notif_res.errors.length !== 0) {
                    emits('received_errors', notif_res.errors)
                    return
                }
                if (notif_res.messages && notif_res.messages.length !== 0) {
                    emits('received_messages', notif_res.messages)
                }
            }
            if (res.added_kyou) {
                emits("registered_kyou", res.added_kyou)
            }
            emits('requested_reload_list')
            emits('requested_close_dialog')
            return
        } finally {
            is_requested_submit.value = false
        }
    }

    function add_notification(): void {
        const notification = new Notification()
        notification.id = props.gkill_api.generate_uuid()
        notification.target_id = id.value
        notification.content = i18n.global.t("MI_REKYOU_TITLE")
        notification.notification_time = new Date(0)
        if (schedule.mi_estimate_start_date_string.value !== "" && schedule.mi_estimate_start_time_string.value !== "") {
            notification.notification_time = moment(schedule.mi_estimate_start_date_string.value + " " + schedule.mi_estimate_start_time_string.value).toDate()
        }
        notifications.value.push(notification)
    }

    function delete_notification(index: number): void {
        notifications.value.splice(index, 1)
    }

    // ── CRUD relay handlers ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

    // ── Return ──
    return {
        // Template refs
        add_notification_views,
        new_board_name_dialog: schedule.new_board_name_dialog,

        // State
        is_requested_submit,
        show_kyou,
        id,
        notifications,
        mi_board_names: schedule.mi_board_names,
        mi_board_name: schedule.mi_board_name,
        mi_estimate_start_date_typed: schedule.mi_estimate_start_date_typed,
        mi_estimate_start_date_string: schedule.mi_estimate_start_date_string,
        mi_estimate_start_time_string: schedule.mi_estimate_start_time_string,
        mi_estimate_end_date_typed: schedule.mi_estimate_end_date_typed,
        mi_estimate_end_date_string: schedule.mi_estimate_end_date_string,
        mi_estimate_end_time_string: schedule.mi_estimate_end_time_string,
        mi_limit_date_typed: schedule.mi_limit_date_typed,
        mi_limit_date_string: schedule.mi_limit_date_string,
        mi_limit_time_string: schedule.mi_limit_time_string,
        show_start_date_menu: schedule.show_start_date_menu,
        show_start_time_menu: schedule.show_start_time_menu,
        show_end_date_menu: schedule.show_end_date_menu,
        show_end_time_menu: schedule.show_end_time_menu,
        show_limit_date_menu: schedule.show_limit_date_menu,
        show_limit_time_menu: schedule.show_limit_time_menu,

        // Business logic / template handlers
        update_board_name: schedule.update_board_name,
        show_new_board_name_dialog: schedule.show_new_board_name_dialog,
        clear_estimate_start_date_time: schedule.clear_estimate_start_date_time,
        clear_estimate_end_date_time: schedule.clear_estimate_end_date_time,
        clear_limit_date_time: schedule.clear_limit_date_time,
        now_to_estimate_start_date_time: schedule.now_to_estimate_start_date_time,
        now_to_estimate_end_date_time: schedule.now_to_estimate_end_date_time,
        now_to_limit_date_time: schedule.now_to_limit_date_time,
        reset,
        save,
        add_notification,
        delete_notification,

        // Event relay objects
        crudRelayHandlers,
    }
}
