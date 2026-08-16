import { i18n } from '@/i18n'
import { computed, type Ref, ref } from 'vue'
import type { EditIDFKyouViewProps } from '@/pages/views/edit-idf-kyou-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { Kyou } from '@/classes/datas/kyou'
import { GkillError } from '@/classes/api/gkill-error'
import moment from 'moment'
import { UpdateIDFKyouRequest } from '@/classes/api/req_res/update-idf-kyou-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'
import type { ComponentRef } from '@/classes/component-ref'
import { useConfirmUnknownTag } from '@/classes/use-confirm-unknown-tag'
import { apply_kyou_tag_changes } from '@/classes/kyou-tags'

export function useEditIDFKyouView(options: {
    props: EditIDFKyouViewProps,
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
    const related_date_typed: Ref<Date> = ref(moment(props.kyou.related_time).toDate())
    const related_date_string: Ref<string> = computed(() => moment(related_date_typed.value).format("YYYY-MM-DD"))
    const related_time_string: Ref<string> = ref(moment(props.kyou.related_time).format("HH:mm:ss"))
    const show_kyou: Ref<boolean> = ref(true)
    const show_related_date_menu = ref(false)
    const show_related_time_menu = ref(false)

    // ── Init calls ──
    load()

    // ── Methods ──
    async function load(): Promise<void> {
        try {
            is_loading.value = true
            cloned_kyou.value = props.kyou.clone()
            await cloned_kyou.value.load_typed_datas()
            related_date_typed.value = moment(cloned_kyou.value.related_time).toDate()
            related_time_string.value = moment(cloned_kyou.value.related_time).format("HH:mm:ss")
        } finally {
            is_loading.value = false
        }
    }

    /**
     * 関連日時が変わっているか。
     *
     * タグだけを足したときにこれが偽なら update_idf_kyou は呼ばない。
     * 呼ぶと中身の同じ新しい版が1つ増えてしまう
     */
    function is_body_changed(): boolean {
        const idf_kyou = cloned_kyou.value.typed_idf_kyou
        if (!idf_kyou) {
            return false
        }
        return moment(idf_kyou.related_time).toDate().getTime() !== moment(related_date_string.value + " " + related_time_string.value).toDate().getTime()
    }

    async function save(): Promise<void> {
        try {
            is_requested_submit.value = true
            cloned_kyou.value.abort_controller.abort()
            cloned_kyou.value.abort_controller = new AbortController()

            // データがちゃんとあるか確認。なければエラーメッセージを出力する
            const idf_kyou = cloned_kyou.value.typed_idf_kyou?.clone()
            if (!idf_kyou) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.client_idf_kyou_is_null
                error.error_message = i18n.global.t("CLIENT_IDF_KYOU_IS_NULL_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 日時必須入力チェック
            if (related_date_string.value === "" || related_time_string.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.idf_kyou_related_time_is_blank
                error.error_message = i18n.global.t("IDF_KYOU_DATE_TIME_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 更新がなかったらエラーメッセージを出力する。
            // この画面で編集できるのは関連日時だけなので、それだけを比較する。
            // タグだけを足した/外したときもここで弾かれないよう、タグの変更も更新とみなす
            const has_tag_changes = kyou_tags_view.value?.has_pending_changes() ?? false
            if (!is_body_changed() && !has_tag_changes) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.idf_kyou_is_no_update
                error.error_message = i18n.global.t("IDF_KYOU_IS_NO_UPDATE_MESSAGE")
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
            const idf_kyou = cloned_kyou.value.typed_idf_kyou?.clone()
            if (!idf_kyou) {
                return
            }

            // 関連日時が変わったときだけ更新リクエストを飛ばす
            if (is_body_changed()) {
                const updated_idf_kyou = idf_kyou.clone()
                updated_idf_kyou.related_time = moment(related_date_string.value + " " + related_time_string.value).toDate()
                updated_idf_kyou.update_app = "gkill"
                updated_idf_kyou.update_device = props.application_config.device
                updated_idf_kyou.update_time = new Date(Date.now())
                updated_idf_kyou.update_user = props.application_config.user_id

                await delete_gkill_kyou_cache(updated_idf_kyou.id)
                const req = new UpdateIDFKyouRequest()
                req.want_response_kyou = true
                req.idf_kyou = updated_idf_kyou

                const res = await props.gkill_api.update_idf_kyou(req)
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

    // ── CRUD relay handlers ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

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
        related_date_typed,
        related_date_string,
        related_time_string,
        show_kyou,
        show_related_date_menu,
        show_related_time_menu,

        // Methods
        save,
        now_to_related_date_time,
        reset_related_date_time,

        // Event relay
        crudRelayHandlers,
    }
}

