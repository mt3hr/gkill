import { i18n } from '@/i18n'
import { computed, type Ref, ref } from 'vue'
import { GkillError } from '@/classes/api/gkill-error'
import moment from 'moment'
import { UpdateNlogRequest } from '@/classes/api/req_res/update-nlog-request'
import type { EditNlogViewProps } from '@/pages/views/edit-nlog-view-props'
import type { Kyou } from '@/classes/datas/kyou'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'
import type { ComponentRef } from '@/classes/component-ref'
import { useConfirmUnknownTag } from '@/classes/use-confirm-unknown-tag'
import { apply_kyou_tag_changes } from '@/classes/kyou-tags'

export function useEditNlogView(options: {
    props: EditNlogViewProps,
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
    const nlog_title_value: Ref<string> = ref(cloned_kyou.value.typed_nlog ? cloned_kyou.value.typed_nlog.title : "")
    const nlog_amount_value: Ref<number> = ref(cloned_kyou.value.typed_nlog ? cloned_kyou.value.typed_nlog.amount : 0)
    const nlog_shop_value: Ref<string> = ref(cloned_kyou.value.typed_nlog ? cloned_kyou.value.typed_nlog.shop : "")
    const related_date_typed: Ref<Date> = ref(moment(cloned_kyou.value.related_time).toDate())
    const related_date_string: Ref<string> = computed(() => moment(related_date_typed.value).format("YYYY-MM-DD"))
    const related_time_string: Ref<string> = ref(moment(cloned_kyou.value.related_time).format("HH:mm:ss"))
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
            nlog_title_value.value = cloned_kyou.value.typed_nlog ? cloned_kyou.value.typed_nlog.title : ""
            nlog_amount_value.value = cloned_kyou.value.typed_nlog ? cloned_kyou.value.typed_nlog.amount : 0
            nlog_shop_value.value = cloned_kyou.value.typed_nlog ? cloned_kyou.value.typed_nlog.shop : ""
            related_date_typed.value = moment(cloned_kyou.value.related_time).toDate()
            related_time_string.value = moment(cloned_kyou.value.related_time).format("HH:mm:ss")
        } finally {
            is_loading.value = false
        }
    }

    /**
     * Nlogの中身か関連日時が変わっているか。
     *
     * タグだけを足したときにこれが偽なら update_nlog は呼ばない。
     * 呼ぶと中身の同じ新しい版が1つ増えてしまう
     */
    function is_body_changed(): boolean {
        const nlog = cloned_kyou.value.typed_nlog
        if (!nlog) {
            return false
        }
        return !(nlog_amount_value.value === nlog.amount &&
            nlog_shop_value.value === nlog.shop &&
            nlog_title_value.value === nlog.title &&
            moment(related_date_string.value + " " + related_time_string.value).toDate().getTime() === moment(nlog.related_time).toDate().getTime())
    }

    async function save(): Promise<void> {
        try {
            is_requested_submit.value = true
            cloned_kyou.value.abort_controller.abort()
            cloned_kyou.value.abort_controller = new AbortController()

            // データがちゃんとあるか確認。なければエラーメッセージを出力する
            const nlog = cloned_kyou.value.typed_nlog
            if (!nlog) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.client_nlog_is_null
                error.error_message = i18n.global.t("CLIENT_NLOG_IS_NULL_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 日時必須入力チェック
            if (related_date_string.value === "" || related_time_string.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.nlog_related_time_is_blank
                error.error_message = i18n.global.t("NLOG_DATE_TIME_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 金額入力チェック
            if (Number.isNaN(nlog_amount_value.value)) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.nlog_amount_is_blank
                error.error_message = i18n.global.t("NLOG_AMOUNT_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 店名入力チェック
            if (nlog_shop_value.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.nlog_shop_name_is_blank
                error.error_message = i18n.global.t("NLOG_SHOP_NAME_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // タイトル入力チェック
            if (nlog_title_value.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.nlog_title_is_blank
                error.error_message = i18n.global.t("NLOG_TITLE_IS_BLANK_MESSAGE")
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
                error.error_code = GkillErrorCodes.nlog_is_no_update
                error.error_message = i18n.global.t("NLOG_IS_NO_UPDATE_MESSAGE")
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
            const nlog = cloned_kyou.value.typed_nlog
            if (!nlog) {
                return
            }

            // 中身が変わったときだけ更新リクエストを飛ばす
            if (is_body_changed()) {
                const updated_nlog = nlog.clone()
                updated_nlog.amount = nlog_amount_value.value
                updated_nlog.shop = nlog_shop_value.value
                updated_nlog.title = nlog_title_value.value
                updated_nlog.related_time = moment(related_date_string.value + " " + related_time_string.value).toDate()
                updated_nlog.update_app = "gkill"
                updated_nlog.update_device = props.application_config.device
                updated_nlog.update_time = new Date(Date.now())
                updated_nlog.update_user = props.application_config.user_id

                await delete_gkill_kyou_cache(updated_nlog.id)
                const req = new UpdateNlogRequest()
                req.want_response_kyou = true
                req.nlog = updated_nlog

                const res = await props.gkill_api.update_nlog(req)
                if (res.errors && res.errors.length !== 0) {
                    emits('received_errors', res.errors)
                    return
                }
                if (res.messages && res.messages.length !== 0) {
                    emits('received_messages', res.messages)
                }
                emits('updated_kyou', res.updated_kyou!)
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

    function reset(): void {
        nlog_title_value.value = cloned_kyou.value.typed_nlog ? cloned_kyou.value.typed_nlog.title : ""
        nlog_amount_value.value = cloned_kyou.value.typed_nlog ? cloned_kyou.value.typed_nlog.amount : 0
        nlog_shop_value.value = cloned_kyou.value.typed_nlog ? cloned_kyou.value.typed_nlog.shop : ""
        related_date_typed.value = moment(cloned_kyou.value.related_time).toDate()
        related_time_string.value = moment(cloned_kyou.value.related_time).format("HH:mm:ss")
        kyou_tags_view.value?.reset()
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
        nlog_title_value,
        nlog_amount_value,
        nlog_shop_value,
        related_date_typed,
        related_date_string,
        related_time_string,
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

