import { i18n } from '@/i18n'
import { computed, type Ref, ref } from 'vue'
import moment from 'moment'
import { GkillError } from '@/classes/api/gkill-error'
import { UpdateURLogRequest } from '@/classes/api/req_res/update-ur-log-request'
import type { Kyou } from '@/classes/datas/kyou'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { EditURLogViewProps } from '@/pages/views/edit-ur-log-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'
import type { ComponentRef } from '@/classes/component-ref'
import { useConfirmUnknownTag } from '@/classes/use-confirm-unknown-tag'
import { apply_kyou_tag_changes } from '@/classes/kyou-tags'

export function useEditUrLogView(options: {
    props: EditURLogViewProps,
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
    const title: Ref<string> = ref(cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.title : "")
    const url: Ref<string> = ref(cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.url : "")
    const related_date_typed: Ref<Date> = ref(moment(cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.related_time : "").toDate())
    const related_date_string: Ref<string> = computed(() => moment(related_date_typed.value).format("YYYY-MM-DD"))
    const related_time_string: Ref<string> = ref(moment(cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.related_time : "").format("HH:mm:ss"))
    const re_get_urlog_content: Ref<boolean> = ref(true)
    const show_kyou: Ref<boolean> = ref(false)
    const show_related_date_menu = ref(false)
    const show_related_time_menu = ref(false)

    // ── CRUD relay handlers ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

    // ── Business logic ──
    async function load(): Promise<void> {
        try {
            is_loading.value = true
            cloned_kyou.value = props.kyou.clone()
            await cloned_kyou.value.load_typed_datas()
            title.value = cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.title : ""
            url.value = cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.url : ""
            related_date_typed.value = moment(cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.related_time : "").toDate()
            related_time_string.value = moment(cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.related_time : "").format("HH:mm:ss")
        } finally {
            is_loading.value = false
        }
    }

    function reset(): void {
        title.value = cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.title : ""
        url.value = cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.url : ""
        related_date_typed.value = moment(cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.related_time : "").toDate()
        related_time_string.value = moment(cloned_kyou.value.typed_urlog ? cloned_kyou.value.typed_urlog.related_time : "").format("HH:mm:ss")
        kyou_tags_view.value?.reset()
    }

    /**
     * URLogの中身か関連日時が変わっているか。
     *
     * タグだけを足したときにこれが偽なら update_urlog は呼ばない。
     * 呼ぶと中身の同じ新しい版が1つ増えてしまう
     */
    function is_body_changed(): boolean {
        const urlog = cloned_kyou.value.typed_urlog
        if (!urlog) {
            return false
        }
        return !(urlog.title === title.value &&
            urlog.url === url.value &&
            moment(urlog.related_time).toDate().getTime() === moment(related_date_string.value + " " + related_time_string.value).toDate().getTime())
    }

    async function save(): Promise<void> {
        try {
            is_requested_submit.value = true
            cloned_kyou.value.abort_controller.abort()
            cloned_kyou.value.abort_controller = new AbortController()

            // データがちゃんとあるか確認。なければエラーメッセージを出力する
            const urlog = cloned_kyou.value.typed_urlog
            if (!urlog) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.client_urlog_is_null
                error.error_message = i18n.global.t("CLIENT_URLOG_IS_NULL_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 日時必須入力チェック
            if (related_date_string.value === "" || related_time_string.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.urlog_related_time_is_blank
                error.error_message = i18n.global.t("URLOG_DATE_TIME_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // URL入力チェック
            if (url.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.urlog_url_is_blank
                error.error_message = i18n.global.t("URLOG_URL_IS_BLANK_MESSAGE")
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
                error.error_code = GkillErrorCodes.urlog_is_no_update
                error.error_message = i18n.global.t("URLOG_IS_NO_UPDATE_MESSAGE")
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
            const urlog = cloned_kyou.value.typed_urlog
            if (!urlog) {
                return
            }

            // 中身が変わったときだけ更新リクエストを飛ばす
            if (is_body_changed()) {
                const updated_urlog = urlog.clone()
                updated_urlog.title = title.value
                updated_urlog.url = url.value
                updated_urlog.related_time = moment(related_date_string.value + " " + related_time_string.value).toDate()
                updated_urlog.update_app = "gkill"
                updated_urlog.update_device = props.application_config.device
                updated_urlog.update_time = new Date(Date.now())
                updated_urlog.update_user = props.application_config.user_id

                // 再取得の場合、URLとタイトル以外をブランクにする
                if (re_get_urlog_content.value) {
                    updated_urlog.description = ""
                    updated_urlog.favicon_image = ""
                    updated_urlog.thumbnail_image = ""
                }

                await delete_gkill_kyou_cache(updated_urlog.id)
                const req = new UpdateURLogRequest()
                req.urlog = updated_urlog
                req.want_response_kyou = true
                const res = await props.gkill_api.update_urlog(req)
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

    function now_to_related_date_time(): void {
        related_date_typed.value = moment().toDate()
        related_time_string.value = moment().format("HH:mm:ss")
    }

    function reset_related_date_time(): void {
        related_date_typed.value = moment(cloned_kyou.value.related_time).toDate()
        related_time_string.value = moment(cloned_kyou.value.related_time).format("HH:mm:ss")
    }

    // ── Initialize ──
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
        title,
        url,
        related_date_typed,
        related_date_string,
        related_time_string,
        re_get_urlog_content,
        show_kyou,
        show_related_date_menu,
        show_related_time_menu,

        // Methods
        save,
        reset,
        now_to_related_date_time,
        reset_related_date_time,

        // Event relay objects
        crudRelayHandlers,
    }
}

