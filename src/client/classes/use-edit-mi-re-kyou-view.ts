import { i18n } from '@/i18n'
import { computed, type Ref, ref } from 'vue'
import moment from 'moment'
import { GkillError } from '@/classes/api/gkill-error'
import { UpdateMiReKyouRequest } from '@/classes/api/req_res/update-mi-re-kyou-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import { Kyou } from '@/classes/datas/kyou'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type { EditMiReKyouViewProps } from '@/pages/views/edit-mi-re-kyou-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { useMiReKyouScheduleFields } from '@/classes/use-mi-re-kyou-schedule-fields'
import { useConfirmUnknownMiBoard } from '@/classes/use-confirm-unknown-mi-board'
import { useConfirmUnknownTag } from '@/classes/use-confirm-unknown-tag'
import { apply_kyou_tag_changes } from '@/classes/kyou-tags'
import type { ComponentRef } from '@/classes/component-ref'

export function useEditMiReKyouView(options: {
    props: EditMiReKyouViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const kyou_tags_view = ref<ComponentRef | null>(null)

    // ── State refs ──
    const is_loading = ref(true)
    const is_requested_submit = ref(false)
    const is_busy = computed(() => is_loading.value || is_requested_submit.value)

    const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
    const show_kyou: Ref<boolean> = ref(false)

    // ── 板名・日時のフォーム ──
    const schedule = useMiReKyouScheduleFields({ props, emits, default_board_name: "" })

    // ── Confirm unknown mi board ──
    const confirm_unknown_mi_board = useConfirmUnknownMiBoard({ application_config: () => props.application_config })

    // ── Confirm unknown tag ──
    const confirm_unknown_tag = useConfirmUnknownTag({ application_config: () => props.application_config })

    // ── Business logic ──
    async function load(): Promise<void> {
        try {
            is_loading.value = true
            cloned_kyou.value = props.kyou.clone()
            await cloned_kyou.value.load_typed_datas()
            reset()
            await schedule.load_mi_board_names()
        } finally {
            is_loading.value = false
        }
    }

    function reset(): void {
        const mirekyou = cloned_kyou.value.typed_mirekyou
        schedule.mi_board_name.value = mirekyou ? mirekyou.board_name : ""
        schedule.apply_times({
            estimate_start_time: mirekyou ? mirekyou.estimate_start_time : null,
            estimate_end_time: mirekyou ? mirekyou.estimate_end_time : null,
            limit_time: mirekyou ? mirekyou.limit_time : null,
        })
        kyou_tags_view.value?.reset()
    }

    /**
     * MiReKyouの中身が変わっているか。
     *
     * タグだけを足したときにこれが偽なら update_mirekyou は呼ばない。
     * 呼ぶと中身の同じ新しい版が1つ増えてしまう
     */
    function is_body_changed(): boolean {
        const mirekyou = cloned_kyou.value.typed_mirekyou
        if (!mirekyou) {
            return false
        }
        const times = schedule.resolve_times()
        const is_same_time = (before: Date | null, after: Date | null): boolean => {
            if (before === null && after === null) {
                return true
            }
            if (before === null || after === null) {
                return false
            }
            return moment(before).toDate().getTime() === moment(after).toDate().getTime()
        }
        return !(mirekyou.board_name === schedule.mi_board_name.value &&
            is_same_time(mirekyou.estimate_start_time, times.estimate_start_time) &&
            is_same_time(mirekyou.estimate_end_time, times.estimate_end_time) &&
            is_same_time(mirekyou.limit_time, times.limit_time))
    }

    async function save(): Promise<void> {
        await do_save(false, false)
    }

    /**
     * 保存の本体。確認を挟むたびにここへ戻ってくる。
     *
     * 確認は「タグ → 板名」の順に1つずつ出す（KFTLの do_submit と同じ）
     */
    async function do_save(skip_unknown_tag_check: boolean, skip_unknown_mi_board_check: boolean): Promise<void> {
        try {
            is_requested_submit.value = true
            cloned_kyou.value.abort_controller.abort()
            cloned_kyou.value.abort_controller = new AbortController()

            // データがちゃんとあるか確認。なければエラーメッセージを出力する
            const mirekyou = cloned_kyou.value.typed_mirekyou
            if (!mirekyou) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.client_mi_rekyou_is_null
                error.error_message = i18n.global.t("CLIENT_MI_REKYOU_IS_NULL_MESSAGE")
                emits('received_errors', [error])
                return
            }

            // 日時の入力チェック
            if (!schedule.validate()) {
                return
            }

            // 更新がなかったらエラーメッセージを出力する。
            // タグだけを足した/外したときもここで弾かれないよう、タグの変更も更新とみなす
            const has_tag_changes = kyou_tags_view.value?.has_pending_changes() ?? false
            if (!is_body_changed() && !has_tag_changes) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.mi_rekyou_is_no_update
                error.error_message = i18n.global.t("MI_REKYOU_IS_NO_UPDATE_MESSAGE")
                emits('received_errors', [error])
                return
            }

            // タグツリーに無いタグ名なら、保存する前に確認を取る
            if (!skip_unknown_tag_check) {
                const unknown_tags = confirm_unknown_tag.collect_unknown_tags(kyou_tags_view.value?.get_tag_names() ?? [])
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

            await execute_save()
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

    async function execute_save(): Promise<void> {
        try {
            is_requested_submit.value = true
            const mirekyou = cloned_kyou.value.typed_mirekyou
            if (!mirekyou) {
                return
            }

            // 中身が変わったときだけ更新リクエストを飛ばす
            if (is_body_changed()) {
                const times = schedule.resolve_times()
                const updated_mirekyou = mirekyou.clone()
                updated_mirekyou.board_name = schedule.mi_board_name.value
                updated_mirekyou.estimate_start_time = times.estimate_start_time
                updated_mirekyou.estimate_end_time = times.estimate_end_time
                updated_mirekyou.limit_time = times.limit_time
                updated_mirekyou.update_app = "gkill"
                updated_mirekyou.update_device = props.application_config.device
                updated_mirekyou.update_time = new Date(Date.now())
                updated_mirekyou.update_user = props.application_config.user_id

                await delete_gkill_kyou_cache(updated_mirekyou.id)
                const req = new UpdateMiReKyouRequest()
                req.mirekyou = updated_mirekyou
                req.want_response_kyou = true
                const res = await props.gkill_api.update_mirekyou(req)
                if (res.errors && res.errors.length !== 0) {
                    emits('received_errors', res.errors)
                    return
                }
                if (res.messages && res.messages.length !== 0) {
                    emits('received_messages', res.messages)
                }
                if (res.updated_kyou) {
                    emits("updated_kyou", res.updated_kyou)
                }
            }

            // 確認ダイアログは非モーダルなので、確認中にタグ欄を書き換えられる。取り直す
            await apply_tag_changes()

            // タグの変更は updated_kyou を出さないので、これが唯一の反映信号になる
            emits('requested_reload_kyou', props.kyou)
            emits('requested_close_dialog')
            return
        } finally {
            is_requested_submit.value = false
        }
    }

    /** タグ欄で足したもの・外したものをサーバへ反映する */
    async function apply_tag_changes(): Promise<void> {
        const tags_view = kyou_tags_view.value
        if (!tags_view) {
            return
        }
        const result = await apply_kyou_tag_changes(props.gkill_api, props.application_config, cloned_kyou.value.id,
            tags_view.get_tag_names(), tags_view.get_removed_tags())
        result.added_tags.forEach(added_tag => emits('registered_tag', added_tag))
        result.removed_tags.forEach(removed_tag => emits('deleted_tag', removed_tag))
        if (result.messages.length !== 0) {
            emits('received_messages', result.messages)
        }
        if (result.errors.length !== 0) {
            emits('received_errors', result.errors)
        }
    }

    // ── Init ──
    load()

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
    }

    // ── Return ──
    return {
        // Template refs
        new_board_name_dialog: schedule.new_board_name_dialog,
        kyou_tags_view,
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
        is_loading,
        is_requested_submit,
        is_busy,
        cloned_kyou,
        show_kyou,
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
        load,
        reset,
        save,
        update_board_name: schedule.update_board_name,
        show_new_board_name_dialog: schedule.show_new_board_name_dialog,
        clear_estimate_start_date_time: schedule.clear_estimate_start_date_time,
        clear_estimate_end_date_time: schedule.clear_estimate_end_date_time,
        clear_limit_date_time: schedule.clear_limit_date_time,
        now_to_estimate_start_date_time: schedule.now_to_estimate_start_date_time,
        now_to_estimate_end_date_time: schedule.now_to_estimate_end_date_time,
        now_to_limit_date_time: schedule.now_to_limit_date_time,

        // Event relay objects
        crudRelayHandlers,
    }
}
