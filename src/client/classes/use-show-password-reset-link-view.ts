import { i18n } from '@/i18n'
import { type Ref, ref, watch } from 'vue'
import { GkillMessage } from '@/classes/api/gkill-message'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'
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

    watch(() => props.account, () => update_password_reset_urls())
    watch(() => props.server_configs, () => update_password_reset_urls())

    update_password_reset_urls()

    function update_password_reset_urls(): void {
        const current_server_config = props.server_configs.filter((server_config) => server_config.enable_this_device)[0]
        if (!current_server_config) return
        const token = props.account.password_reset_token
        const http = current_server_config.enable_tls ? "https://" : "http://"
        const port = current_server_config.address
        local_password_reset_url.value = `${http}localhost${port}/set_new_password?reset_token=${token}`
        const lan_host = (current_server_config.lan_hostname && current_server_config.lan_hostname !== "") ? current_server_config.lan_hostname : props.application_config.private_ip
        lan_password_reset_url.value = (lan_host && lan_host !== "") ? `${http}${lan_host}${port}/set_new_password?reset_token=${token}` : ""
        const global_host = (current_server_config.global_hostname && current_server_config.global_hostname !== "") ? current_server_config.global_hostname : ""
        over_lan_password_reset_url.value = (global_host !== "") ? `${http}${global_host}${port}/set_new_password?reset_token=${token}` : ""
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
        copy_local_password_reset_url,
        copy_lan_password_reset_url,
        copy_over_lan_password_reset_url,
    }
}
