import { i18n } from '@/i18n'
import { computed, type Ref, ref } from 'vue'
import type { EditURLogViewProps } from '@/pages/views/edit-ur-log-view-props'
import { URLog } from '@/classes/datas/ur-log'
import moment from 'moment'
import { GkillError } from '@/classes/api/gkill-error'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { AddURLogRequest } from '@/classes/api/req_res/add-ur-log-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { ComponentRef } from '@/classes/component-ref'
import { useConfirmUnknownTag } from '@/classes/use-confirm-unknown-tag'
import { add_tags_to_target } from '@/classes/kyou-tags'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

export function useAddURLogView(options: {
    props: EditURLogViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const kyou_tags_view = ref<ComponentRef | null>(null)

    // ── Confirm unknown tag ──
    const confirm_unknown_tag = useConfirmUnknownTag({ application_config: () => props.application_config })

    // ── State refs ──
    const is_requested_submit = ref(false)

    const urlog: Ref<URLog> = ref((() => {
        const urlog = new URLog()
        urlog.related_time = new Date(Date.now())
        return urlog
    })())
    const title: Ref<string> = ref(urlog.value.title)
    const url: Ref<string> = ref(urlog.value.url)
    const related_date_typed: Ref<Date> = ref(moment(urlog.value.related_time).toDate())
    const related_date_string: Ref<string> = computed(() => moment(related_date_typed.value).format("YYYY-MM-DD"))
    const related_time_string: Ref<string> = ref(moment(urlog.value.related_time).format("HH:mm:ss"))

    const show_related_date_menu = ref(false)
    const show_related_time_menu = ref(false)

    // ── Business logic ──
    function reset(): void {
        title.value = urlog.value.title
        url.value = urlog.value.url
        related_date_typed.value = moment(urlog.value.related_time).toDate()
        related_time_string.value = moment(urlog.value.related_time).format("HH:mm:ss")
        kyou_tags_view.value?.reset()
    }

    async function save(): Promise<void> {
        try {
            is_requested_submit.value = true
            // データがちゃんとあるか確認。なければエラーメッセージを出力する
            if (!urlog.value) {
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

            // タグツリーに無いタグ名なら、保存する前に確認を取る
            const tag_names = kyou_tags_view.value?.get_tag_names() ?? []
            const unknown_tags = confirm_unknown_tag.collect_unknown_tags(tag_names)
            if (unknown_tags.length !== 0) {
                confirm_unknown_tag.open_confirm(unknown_tags)
                return
            }

            await execute_save(tag_names)
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
            // 確認ダイアログは非モーダルなので、確認中にタグ欄を書き換えられる。取り直す
            await execute_save(kyou_tags_view.value?.get_tag_names() ?? [])
        } finally {
            is_requested_submit.value = false
        }
    }

    async function execute_save(tag_names: Array<string>): Promise<void> {
        try {
            is_requested_submit.value = true

            // 更新後URLog情報を用意する
            const new_urlog = urlog.value.clone()
            new_urlog.id = props.gkill_api.generate_uuid()
            new_urlog.title = title.value
            new_urlog.url = url.value
            new_urlog.related_time = moment(related_date_string.value + " " + related_time_string.value).toDate()
            new_urlog.create_app = "gkill"
            new_urlog.create_device = props.application_config.device
            new_urlog.create_time = new Date(Date.now())
            new_urlog.create_user = props.application_config.user_id
            new_urlog.update_app = "gkill"
            new_urlog.update_device = props.application_config.device
            new_urlog.update_time = new Date(Date.now())
            new_urlog.update_user = props.application_config.user_id

            // 追加リクエストを飛ばす
            await delete_gkill_kyou_cache(new_urlog.id)
            const req = new AddURLogRequest()
            req.urlog = new_urlog
            req.want_response_kyou = true
            const res = await props.gkill_api.add_urlog(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }

            // タグは registered_kyou より必ず先に付ける。
            // 局所挿入は渡したKyouをそのまま使わず refresh_kyou で引き直すので、
            // この時点でサーバにタグが入っていれば attached_tags 込みで差し込まれる。
            // 逆に先に emit すると、タグで絞り込んだ列が空のタグ列を見て
            // 「一致しない」と判定し、エラーも出ないまま行が現れない
            const tag_result = await add_tags_to_target(props.gkill_api, props.application_config, new_urlog.id, tag_names)
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

    function now_to_related_date_time(): void {
        related_date_typed.value = moment().toDate()
        related_time_string.value = moment().format("HH:mm:ss")
    }

    function reset_related_date_time(): void {
        related_date_typed.value = moment(urlog.value.related_time).toDate()
        related_time_string.value = moment(urlog.value.related_time).format("HH:mm:ss")
    }

    // ── CRUD relay handlers ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

    // ── Template event handlers ──
    function onCloseDateMenu(): void {
        show_related_date_menu.value = false
    }

    function onCloseTimeMenu(): void {
        show_related_time_menu.value = false
    }

    return {
        // Template refs
        kyou_tags_view,
        confirm_unknown_tag_dialog: confirm_unknown_tag.confirm_unknown_tag_dialog,

        // Confirm unknown tag
        unknown_tags: confirm_unknown_tag.unknown_tags,
        cancel_save,
        confirm_save,

        // State
        is_requested_submit,
        title,
        url,
        related_date_typed,
        related_date_string,
        related_time_string,
        show_related_date_menu,
        show_related_time_menu,

        // Business logic
        reset,
        save,
        now_to_related_date_time,
        reset_related_date_time,

        // Template event handlers
        onCloseDateMenu,
        onCloseTimeMenu,

        // Event relay objects
        crudRelayHandlers,
    }
}
