import { i18n } from '@/i18n'
import { computed, ref, type Ref } from 'vue'
import type { LoginViewProps } from '@/pages/views/login-view-props'
import type LoginViewEmits from '@/pages/views/login-view-emits'
import { LoginRequest } from '@/classes/api/req_res/login-request'
import router from '@/router'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { useTheme } from 'vuetify'
import { resetDialogHistory } from '@/classes/use-dialog-history-stack'

export function useLoginView(options: {
    props: LoginViewProps,
    emits: LoginViewEmits,
}) {
    const { props, emits } = options

    // ── Theme ──
    const theme = useTheme()

    // ── State refs ──
    const welcome_emoji = computed(() => theme.current.value.dark ? "⭐️" : "❄️")
    const user_id: Ref<string> = ref("")
    const password: Ref<string> = ref("")

    const app_content_height_px = computed(() => props.app_content_height + 'px')
    const app_content_width_px = computed(() => props.app_content_width + 'px')

    // ── Business logic ──
    async function compute_password_sha256(): Promise<string> {
        const encoder = new TextEncoder();
        const msgUint8 = encoder.encode(password.value);
        const hashBuffer = await crypto.subtle.digest('SHA-256', msgUint8);

        const hashArray = Array.from(new Uint8Array(hashBuffer));
        const hashHex = hashArray
            .map((b) => b.toString(16).padStart(2, '0'))
            .join('');
        return hashHex;
    }

    async function check_logined(): Promise<void> {
        const session_id = props.gkill_api.get_session_id()
        const default_page = props.gkill_api.get_default_page_from_cookie()
        if (session_id && session_id !== "" && default_page && default_page !== "") {
            await resetDialogHistory()
            router.replace("/" + default_page)
        }
    }

    async function try_login(user_id: string): Promise<boolean> {
        // 未入力チェック
        try {
            if (user_id === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.user_id_is_blank
                error.error_message = i18n.global.t("REQUEST_INPUT_USER_ID_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return false
            }
            if (password.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.password_is_blank
                error.error_message = i18n.global.t("REQUEST_INPUT_PASSWORD_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return false
            }

            // クッキーとかキャッシュの削除
            await props.gkill_api.clear_browser_datas()

            // request作成
            const req = new LoginRequest()
            req.user_id = user_id
            req.password_sha256 = await compute_password_sha256()

            // ログインとエラーチェック
            const res = await props.gkill_api.login(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return false
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }

            emits('successed_login', res.session_id)
            return true
        } catch (e) {
            // TLSの場合、サーバ証明書が入っていないとログインできない
            const error = new GkillError()
            error.error_code = GkillErrorCodes.required_certificate
            error.error_message = i18n.global.t("REQUEST_CERTIFICATE_REQUIRED_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            return false
        }
    }

    // ── Init calls ──
    check_logined()

    // ── Return ──
    return {
        // State
        welcome_emoji,
        user_id,
        password,
        app_content_height_px,
        app_content_width_px,

        // Business logic
        try_login,
    }
}
