import { i18n } from '@/i18n'
import { computed, type Ref, ref } from 'vue'
import type { EditTimeIsViewProps } from '@/pages/views/edit-time-is-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import type { Kyou } from '@/classes/datas/kyou'
import moment from 'moment'
import { GkillError } from '@/classes/api/gkill-error'
import { UpdateTimeisRequest } from '@/classes/api/req_res/update-timeis-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'
import type { ComponentRef } from '@/classes/component-ref'
import { useConfirmUnknownTag } from '@/classes/use-confirm-unknown-tag'
import { apply_kyou_tag_changes } from '@/classes/kyou-tags'

export function useEditTimeIsView(options: {
    props: EditTimeIsViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const kyou_tags_view = ref<ComponentRef | null>(null)

    // ── Confirm unknown tag ──
    const confirm_unknown_tag = useConfirmUnknownTag({ application_config: () => props.application_config })

    // ── State refs ──
    const is_loading = ref(true)
    const is_requested_submit = ref(false)
    const is_busy = computed(() => is_loading.value || is_requested_submit.value)

    const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
    const show_kyou: Ref<boolean> = ref(false)

    const timeis_title: Ref<string> = ref(cloned_kyou.value.typed_timeis ? cloned_kyou.value.typed_timeis.title : "")
    const timeis_start_date_typed: Ref<Date> = ref(cloned_kyou.value.typed_timeis?.start_time ? moment(cloned_kyou.value.typed_timeis?.start_time).toDate() : new Date(0))
    const timeis_start_date_string: Ref<string> = computed(() => moment(timeis_start_date_typed.value).format("YYYY-MM-DD"))
    const timeis_start_time_string: Ref<string> = ref(cloned_kyou.value.typed_timeis ? moment(cloned_kyou.value.typed_timeis.start_time).format("HH:mm:ss") : "")
    const timeis_end_date_typed: Ref<Date | null> = ref(cloned_kyou.value.typed_timeis?.end_time ? moment(cloned_kyou.value.typed_timeis.end_time).toDate() : null)
    const timeis_end_date_string: Ref<string> = computed(() => timeis_end_date_typed.value ? moment(timeis_end_date_typed.value).format("YYYY-MM-DD") : "")
    const timeis_end_time_string: Ref<string> = ref(timeis_end_date_typed.value ? moment(timeis_end_date_typed.value).format("HH:mm:ss") : "")

    const show_start_date_menu = ref(false)
    const show_start_time_menu = ref(false)
    const show_end_date_menu = ref(false)
    const show_end_time_menu = ref(false)

    // ── Business logic ──
    async function load(): Promise<void> {
        try {
            is_loading.value = true
            cloned_kyou.value = props.kyou.clone()
            await cloned_kyou.value.load_typed_datas()
            timeis_title.value = cloned_kyou.value.typed_timeis ? cloned_kyou.value.typed_timeis.title : ""
            timeis_start_date_typed.value = moment(cloned_kyou.value.typed_timeis ? cloned_kyou.value.typed_timeis.start_time : "").toDate()
            timeis_start_time_string.value = moment(cloned_kyou.value.typed_timeis ? cloned_kyou.value.typed_timeis.start_time : "").format("HH:mm:ss")
            timeis_end_date_typed.value = cloned_kyou.value.typed_timeis?.end_time ? moment(cloned_kyou.value.typed_timeis.end_time).toDate() : null
            timeis_end_time_string.value = cloned_kyou.value.typed_timeis?.end_time ? moment(cloned_kyou.value.typed_timeis.end_time).format("HH:mm:ss") : ""
        } finally {
            is_loading.value = false
        }
    }

    function reset(): void {
        timeis_title.value = cloned_kyou.value.typed_timeis ? cloned_kyou.value.typed_timeis.title : ""
        reset_start_date_time()
        reset_end_date_time()
        kyou_tags_view.value?.reset()
    }

    function reset_start_date_time(): void {
        timeis_start_date_typed.value = cloned_kyou.value.typed_timeis?.start_time ? moment(cloned_kyou.value.typed_timeis.start_time).toDate() : new Date(0)
        timeis_start_time_string.value = cloned_kyou.value.typed_timeis?.start_time ? moment(cloned_kyou.value.typed_timeis.start_time).format("HH:mm:ss") : ""
    }

    function reset_end_date_time(): void {
        timeis_end_date_typed.value = cloned_kyou.value.typed_timeis?.end_time ? moment(cloned_kyou.value.typed_timeis.end_time).toDate() : null
        timeis_end_time_string.value = cloned_kyou.value.typed_timeis?.end_time ? moment(cloned_kyou.value.typed_timeis.end_time).format("HH:mm:ss") : ""
    }

    function now_to_start_date_time(): void {
        timeis_start_date_typed.value = moment().toDate()
        timeis_start_time_string.value = moment().format("HH:mm:ss")
    }

    function now_to_end_date_time(): void {
        timeis_end_date_typed.value = moment().toDate()
        timeis_end_time_string.value = moment().format("HH:mm:ss")
    }

    function clear_end_date_time(): void {
        timeis_end_date_typed.value = null
        timeis_end_time_string.value = ""
    }

    /**
     * TimeIsの中身が変わっているか。
     *
     * タグだけを足したときにこれが偽なら update_timeis は呼ばない。
     * 呼ぶと中身の同じ新しい版が1つ増えてしまう
     */
    function is_body_changed(): boolean {
        const timeis = cloned_kyou.value.typed_timeis
        if (!timeis) {
            return false
        }
        return !(timeis.title === timeis_title.value &&
            (moment(timeis.start_time).toDate().getTime() === moment(timeis_start_date_string.value + " " + timeis_start_time_string.value).toDate().getTime()) &&
            (moment(timeis.end_time).toDate().getTime() === moment(timeis_end_date_string.value + " " + timeis_end_time_string.value).toDate().getTime() || (timeis.end_time === null && timeis_end_date_string.value === "" && timeis_end_time_string.value === "")))
    }

    async function save(): Promise<void> {
        try {
            is_requested_submit.value = true
            cloned_kyou.value.abort_controller.abort()
            cloned_kyou.value.abort_controller = new AbortController()

            // データがちゃんとあるか確認。なければエラーメッセージを出力する
            const timeis = cloned_kyou.value.typed_timeis
            if (!timeis) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.client_timeis_is_null
                error.error_message = i18n.global.t("CLIENT_TIMEIS_IS_NULL_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 開始日時必須入力チェック
            if (timeis_start_date_string.value === "" || timeis_start_time_string.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.timeis_start_time_is_blank
                error.error_message = i18n.global.t("TIMEIS_START_TIME_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 終了日時 片方だけ入力されていたらエラーチェック
            if (timeis_end_date_string.value === "" || timeis_end_time_string.value === "") {//どっちも入力されていなければOK。nullとして扱う
                if ((timeis_end_date_string.value === "" && timeis_end_time_string.value !== "") ||
                    (timeis_end_date_string.value !== "" && timeis_end_time_string.value === "")) { // 片方入力されていなかったらエラーメッセージ出力
                    const error = new GkillError()
                    error.error_code = GkillErrorCodes.timeis_end_time_is_blank
                    error.error_message = i18n.global.t("TIMEIS_END_TIME_IS_BLANK_MESSAGE")
                    const errors = new Array<GkillError>()
                    errors.push(error)
                    emits('received_errors', errors)
                    return
                }
            }

            // タイトル入力チェック
            if (timeis_title.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.timeis_title_is_blank
                error.error_message = i18n.global.t("TIMEIS_TITLE_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 更新がなかったらエラーメッセージを出力する。
            // タグだけを足した/外したときもここで弾かれないよう、タグの変更も更新とみなす
            const has_tag_changes = kyou_tags_view.value?.has_pending_changes() ?? false
            if (!is_body_changed() && !has_tag_changes) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.timeis_is_no_update
                error.error_message = i18n.global.t("TIMEIS_IS_NO_UPDATE_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // タグツリーに無いタグ名なら、保存する前に確認を取る
            const tag_names = kyou_tags_view.value?.get_tag_names() ?? []
            const unknown_tags = confirm_unknown_tag.collect_unknown_tags(tag_names)
            if (unknown_tags.length !== 0) {
                confirm_unknown_tag.open_confirm(unknown_tags)
                return
            }

            await execute_save()
        } finally {
            is_requested_submit.value = false
        }
    }

    function cancel_save(): void {
        confirm_unknown_tag.close_confirm()
    }

    async function confirm_save(): Promise<void> {
        confirm_unknown_tag.close_confirm()
        try {
            is_requested_submit.value = true
            await execute_save()
        } finally {
            is_requested_submit.value = false
        }
    }

    async function execute_save(): Promise<void> {
        try {
            is_requested_submit.value = true
            const timeis = cloned_kyou.value.typed_timeis
            if (!timeis) {
                return
            }

            // 中身が変わったときだけ更新リクエストを飛ばす
            if (is_body_changed()) {
                let end_time: Date | null = null
                if (timeis_end_date_string.value !== "" && timeis_end_time_string.value !== "") {
                    end_time = moment(timeis_end_date_string.value + " " + timeis_end_time_string.value).toDate()
                }
                const updated_timeis = timeis.clone()
                updated_timeis.title = timeis_title.value
                updated_timeis.start_time = moment(timeis_start_date_string.value + " " + timeis_start_time_string.value).toDate()
                updated_timeis.end_time = end_time
                updated_timeis.update_app = "gkill"
                updated_timeis.update_device = props.application_config.device
                updated_timeis.update_time = new Date(Date.now())
                updated_timeis.update_user = props.application_config.user_id

                await delete_gkill_kyou_cache(updated_timeis.id)
                const req = new UpdateTimeisRequest()
                req.timeis = updated_timeis
                req.want_response_kyou = true
                const res = await props.gkill_api.update_timeis(req)
                if (res.errors && res.errors.length !== 0) {
                    emits('received_errors', res.errors)
                    return
                }
                if (res.messages && res.messages.length !== 0) {
                    emits('received_messages', res.messages)
                }
                emits("updated_kyou", res.updated_kyou!)
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

    // ── Event relay objects ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

    // ── Init calls ──
    load()

    // ── Return ──
    return {
        // Template refs
        kyou_tags_view,
        confirm_unknown_tag_dialog: confirm_unknown_tag.confirm_unknown_tag_dialog,

        // Confirm unknown tag
        unknown_tags: confirm_unknown_tag.unknown_tags,
        cancel_save,
        confirm_save,

        // State
        is_loading,
        is_requested_submit,
        is_busy,
        cloned_kyou,
        show_kyou,
        timeis_title,
        timeis_start_date_typed,
        timeis_start_date_string,
        timeis_start_time_string,
        timeis_end_date_typed,
        timeis_end_date_string,
        timeis_end_time_string,
        show_start_date_menu,
        show_start_time_menu,
        show_end_date_menu,
        show_end_time_menu,

        // Business logic
        reset,
        reset_start_date_time,
        reset_end_date_time,
        now_to_start_date_time,
        now_to_end_date_time,
        clear_end_date_time,
        save,

        // Event relay objects
        crudRelayHandlers,
    }
}

