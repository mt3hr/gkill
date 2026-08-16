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
import { useConfirmUnknownMiBoard } from '@/classes/use-confirm-unknown-mi-board'
import { useConfirmUnknownTag } from '@/classes/use-confirm-unknown-tag'
import { add_tags_to_target } from '@/classes/kyou-tags'

export function useAddMiReKyouView(options: {
    props: AddMiReKyouViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const add_notification_views = ref<ComponentRef | null>(null)
    const kyou_tags_view = ref<ComponentRef | null>(null)

    // ── State refs ──
    const is_requested_submit = ref(false)
    const show_kyou: Ref<boolean> = ref(true)
    const id: Ref<string> = ref(props.gkill_api.generate_uuid())
    const notifications: Ref<Array<Notification>> = ref(new Array<Notification>())

    // ── 板名・日時のフォーム ──
    const schedule = useMiReKyouScheduleFields({ props, emits })

    // ── Confirm unknown mi board ──
    const confirm_unknown_mi_board = useConfirmUnknownMiBoard({ application_config: () => props.application_config })

    // ── Confirm unknown tag ──
    const confirm_unknown_tag = useConfirmUnknownTag({ application_config: () => props.application_config })

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
        kyou_tags_view.value?.reset()
    }

    async function save(): Promise<void> {
        await do_save(false, false)
    }

    /**
     * 保存の本体。確認を挟むたびにここへ戻ってくる。
     *
     * 確認は「タグ → 板名」の順に1つずつ出す（KFTLの do_submit と同じ）。
     * 確認ダイアログは非モーダルなので、確認中に入力を変えられる。
     * 再入のたびに Notification もタグ名も取り直すこと
     */
    async function do_save(skip_unknown_tag_check: boolean, skip_unknown_mi_board_check: boolean): Promise<void> {
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

            // タグツリーに無いタグ名なら、保存する前に確認を取る
            const tag_names = kyou_tags_view.value?.get_tag_names() ?? []
            if (!skip_unknown_tag_check) {
                const unknown_tags = confirm_unknown_tag.collect_unknown_tags(tag_names)
                if (unknown_tags.length !== 0) {
                    confirm_unknown_tag.open_confirm(unknown_tags)
                    return
                }
            }

            // 実在しない板名なら、保存する前に確認を取る。
            // 板はサーバ側で検証されず「その名前のタスクが1件でもあること」で実体化するので、
            // 打ち間違いがそのまま新しい板になってしまう。
            // タグの確認を通した後に改めてここへ来る（確認は1つずつ順に出す）
            if (!skip_unknown_mi_board_check) {
                const unknown_boards = confirm_unknown_mi_board.collect_unknown_mi_boards([schedule.mi_board_name.value])
                if (unknown_boards.length !== 0) {
                    confirm_unknown_mi_board.open_confirm(unknown_boards)
                    return
                }
            }

            await execute_save(notification_results, tag_names)
        } finally {
            is_requested_submit.value = false
        }
    }

    function cancel_tag_save(): void {
        confirm_unknown_tag.close_confirm()
    }

    /** タグの確認を通した。次は板名の確認へ進む */
    async function confirm_tag_save(): Promise<void> {
        confirm_unknown_tag.close_confirm()
        await do_save(true, false)
    }

    function cancel_save(): void {
        confirm_unknown_mi_board.close_confirm()
    }

    async function confirm_save(): Promise<void> {
        confirm_unknown_mi_board.remember_confirmed_mi_boards()
        confirm_unknown_mi_board.close_confirm()
        await do_save(true, true)
    }

    async function execute_save(notification_results: Array<Notification>, tag_names: Array<string>): Promise<void> {
        try {
            is_requested_submit.value = true
            if (!props.kyou || props.kyou.id === "") {
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

            // タグは registered_kyou より必ず先に付ける。
            // 先に emit すると、タグで絞り込んだ列が空のタグ列を見て「一致しない」と判定し、
            // エラーも出ないまま行が現れない
            const tag_result = await add_tags_to_target(props.gkill_api, props.application_config, new_mirekyou.id, tag_names)
            tag_result.added_tags.forEach(added_tag => emits('registered_tag', added_tag))
            if (tag_result.messages.length !== 0) {
                emits('received_messages', tag_result.messages)
            }
            if (tag_result.errors.length !== 0) {
                emits('received_errors', tag_result.errors)
            }

            // 追加した記録は列へ局所挿入されるので、リスト全体の引き直しは要求しない。
            // Kyouが返らなかったときだけ、従来どおり引き直しへ落とす
            if (res.added_kyou) {
                emits("registered_kyou", res.added_kyou)
            } else {
                emits('requested_reload_list')
            }
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
        kyou_tags_view,
        new_board_name_dialog: schedule.new_board_name_dialog,
        confirm_unknown_mi_board_dialog: confirm_unknown_mi_board.confirm_unknown_mi_board_dialog,
        confirm_unknown_tag_dialog: confirm_unknown_tag.confirm_unknown_tag_dialog,

        // Confirm unknown mi board
        unknown_mi_boards: confirm_unknown_mi_board.unknown_mi_boards,
        cancel_save,
        confirm_save,

        // Confirm unknown tag
        unknown_tags: confirm_unknown_tag.unknown_tags,
        cancel_tag_save,
        confirm_tag_save,

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
