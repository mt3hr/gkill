import { i18n } from '@/i18n'
import { computed, onUnmounted, type Ref, ref, watch } from 'vue'
import moment from 'moment'
import { GkillMessage } from '@/classes/api/gkill-message'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'
import { ResetPasswordRequest } from '@/classes/api/req_res/reset-password-request'
import type { ShowPasswordResetLinkViewProps } from '@/pages/views/show-password-reset-link-view-props'
import type { ShowPasswordResetLinkViewEmits } from '@/pages/views/show-password-reset-link-view-emits'

export function useShowPasswordResetLinkView(options: {
    props: ShowPasswordResetLinkViewProps,
    emits: ShowPasswordResetLinkViewEmits,
}) {
    const { props, emits } = options

    const local_password_reset_url: Ref<string> = ref("")
    const lan_password_reset_url: Ref<string> = ref("")
    const over_lan_password_reset_url: Ref<string> = ref("")
    const is_reissuing: Ref<boolean> = ref(false)

    // 期限切れの判定は今の時刻に依存するので、ダイアログを開いている間だけ
    // 定期的に進める。開きっぱなしのまま期限を跨いでも表示が追随する
    const now: Ref<number> = ref(Date.now())
    const now_timer = setInterval(() => { now.value = Date.now() }, 30 * 1000)
    onUnmounted(() => clearInterval(now_timer))

    watch(() => props.account, () => update_password_reset_urls())
    watch(() => props.server_configs, () => update_password_reset_urls())

    update_password_reset_urls()

    // ── Computed ──
    // 有効期限。サーバはリセットトークンがないときnullを返すので、その場合は空文字にする
    const password_reset_token_expiration_label = computed(() => {
        const expiration = props.account.password_reset_token_expiration
        if (!expiration) return ""
        const parsed = moment(expiration)
        if (!parsed.isValid()) return ""
        return parsed.format("YYYY-MM-DD HH:mm:ss")
    })

    const is_password_reset_link_expired = computed(() => {
        const expiration = props.account.password_reset_token_expiration
        if (!expiration) return false
        const parsed = moment(expiration)
        if (!parsed.isValid()) return false
        return parsed.valueOf() < now.value
    })

    function update_password_reset_urls(): void {
        const current_server_config = props.server_configs.filter((server_config) => server_config.enable_this_device)[0]
        if (!current_server_config) return
        const token = props.account.password_reset_token
        const http = current_server_config.enable_tls ? "https://" : "http://"
        const port = current_server_config.address
        // user_id を載せておくと設定画面のユーザ名欄が自動で埋まる。
        // 載せないと利用者が手入力することになり、打ち間違えると
        // 「トークンは正しいのに失敗する」状態になる
        const query = `?user_id=${encodeURIComponent(props.account.user_id)}&reset_token=${encodeURIComponent(token ?? "")}`
        local_password_reset_url.value = `${http}localhost${port}/set_new_password${query}`
        const lan_host = (current_server_config.lan_hostname && current_server_config.lan_hostname !== "") ? current_server_config.lan_hostname : props.application_config.private_ip
        lan_password_reset_url.value = (lan_host && lan_host !== "") ? `${http}${lan_host}${port}/set_new_password${query}` : ""
        const global_host = (current_server_config.global_hostname && current_server_config.global_hostname !== "") ? current_server_config.global_hostname : ""
        over_lan_password_reset_url.value = (global_host !== "") ? `${http}${global_host}${port}/set_new_password${query}` : ""
    }

    // reissue_password_reset_link はリセットトークンを発行しなおす。
    //
    // トークンの有効期限は72時間で切れるが、管理画面はトークンがある間
    // このダイアログしか出さないので、ここに再発行がないと期限切れから
    // 復帰する手段がサーバと同じマシンでの `reset_password` コマンドだけになる。
    async function reissue_password_reset_link(): Promise<void> {
        if (is_reissuing.value) return
        is_reissuing.value = true
        try {
            const req = new ResetPasswordRequest()
            req.target_user_id = props.account.user_id
            const res = await props.gkill_api.reset_password(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }

            emits('requested_reload_server_config')
            // 新しいトークンは親がサーバから引き直してこのダイアログに渡しなおす。
            // レスポンスのトークンを自分で持つと、有効期限だけ古いままになる
            emits('requested_show_show_password_reset_dialog', props.account)
        } finally {
            is_reissuing.value = false
        }
    }

    function copy_local_password_reset_url(): void {
        navigator.clipboard.writeText(local_password_reset_url.value)
        const message = new GkillMessage()
        message.message_code = GkillMessageCodes.copied_lan_set_password_link
        message.message = i18n.global.t("COPIED_MESSAGE")
        emits('received_messages', [message])
    }

    function copy_lan_password_reset_url(): void {
        navigator.clipboard.writeText(lan_password_reset_url.value)
        const message = new GkillMessage()
        message.message_code = GkillMessageCodes.copied_lan_set_password_link
        message.message = i18n.global.t("COPIED_MESSAGE")
        emits('received_messages', [message])
    }

    function copy_over_lan_password_reset_url(): void {
        navigator.clipboard.writeText(over_lan_password_reset_url.value)
        const message = new GkillMessage()
        message.message_code = GkillMessageCodes.copied_over_lan_set_password_link
        message.message = i18n.global.t("COPIED_MESSAGE")
        emits('received_messages', [message])
    }

    return {
        local_password_reset_url,
        lan_password_reset_url,
        over_lan_password_reset_url,
        is_reissuing,
        password_reset_token_expiration_label,
        is_password_reset_link_expired,
        copy_local_password_reset_url,
        copy_lan_password_reset_url,
        copy_over_lan_password_reset_url,
        reissue_password_reset_link,
    }
}
