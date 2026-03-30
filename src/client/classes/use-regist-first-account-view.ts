import { i18n } from '@/i18n'
import { computed, ref, type Ref } from 'vue'
import router from '@/router'
import { GkillError } from '@/classes/api/gkill-error'
import { useRoute } from 'vue-router'
import { SetNewPasswordRequest } from '@/classes/api/req_res/set-new-password-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { AddAccountRequest } from '@/classes/api/req_res/add-account-request'
import { Account } from '@/classes/datas/config/account'
import { LoginRequest } from '@/classes/api/req_res/login-request'
import { GetServerConfigsRequest } from '@/classes/api/req_res/get-server-configs-request'
import { GkillMessage } from '@/classes/api/gkill-message'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'
import { useTheme } from 'vuetify'
import { resetDialogHistory } from '@/classes/use-dialog-history-stack'
import type { RegistFirstAccountViewProps } from '@/pages/views/regist-first-account-view-props'
import type { RegistFirstAccountViewEmits } from '@/pages/views/regist-first-account-view-emits'

enum RegistStatus {
    none = 0,
    reseted_admin_password = 1,
    added_account = 2,
    reseted_account_password = 3,
    done = 4,
}

export function useRegistFirstAccountView(options: {
    props: RegistFirstAccountViewProps,
    emits: RegistFirstAccountViewEmits,
}) {
    const { props, emits } = options

    const theme = useTheme()

    // ── State refs ──
    const welcome_emoji = computed(() => theme.current.value.dark ? "⭐️" : "❄️")
    const admin_account_password_reset_token: Ref<string> = ref(useRoute().query.reset_token ? useRoute().query.reset_token!.toString() : "")
    const user_id: Ref<string> = ref(useRoute().query.user_id ? useRoute().query.user_id!.toString() : "")
    const password: Ref<string> = ref("")
    const password_retype: Ref<string> = ref("")
    const admin_password: Ref<string> = ref("")
    const admin_password_retype: Ref<string> = ref("")
    const regist_state = ref(RegistStatus.none)
    const is_submiting = ref(false)

    // ── Computed ──
    const app_content_height_px = computed(() => props.app_content_height + 'px')
    const app_content_width_px = computed(() => props.app_content_width + 'px')

     
    const password_sha256 = computed(async () => {
        const encoder = new TextEncoder();
        const msgUint8 = encoder.encode(password.value);
         
        const hashBuffer = await crypto.subtle.digest('SHA-256', msgUint8);

        const hashArray = Array.from(new Uint8Array(hashBuffer));
        const hashHex = hashArray
            .map((b) => b.toString(16).padStart(2, '0'))
            .join('');
        return hashHex;
    })

     
    const admin_password_sha256 = computed(async () => {
        const encoder = new TextEncoder();
        const msgUint8 = encoder.encode(admin_password.value);
         
        const hashBuffer = await crypto.subtle.digest('SHA-256', msgUint8);

        const hashArray = Array.from(new Uint8Array(hashBuffer));
        const hashHex = hashArray
            .map((b) => b.toString(16).padStart(2, '0'))
            .join('');
        return hashHex;
    })

    // ── Business logic ──
    const sleep = (time: number) => new Promise<void>((r) => setTimeout(r, time))

    async function try_regist_account(): Promise<boolean> {
        is_submiting.value = true
        // 未入力チェック
        if (user_id.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.user_id_is_blank
            error.error_message = i18n.global.t("REQUEST_INPUT_USER_ID_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            is_submiting.value = false
            return false
        }
        if (password.value === "" || password_retype.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.password_is_blank
            error.error_message = i18n.global.t("REQUEST_INPUT_PASSWORD_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            is_submiting.value = false
            return false
        }
        if (password.value !== password_retype.value) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.password_retype_is_blank
            error.error_message = i18n.global.t("INVALID_RETYPED_PASSWORD_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            is_submiting.value = false
            return false
        }
        if (admin_password.value === "" || admin_password_retype.value === "") {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.password_is_blank
            error.error_message = i18n.global.t("REQUEST_INPUT_PASSWORD_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            is_submiting.value = false
            return false
        }
        if (admin_password.value !== admin_password_retype.value) {
            const error = new GkillError()
            error.error_code = GkillErrorCodes.password_retype_is_blank
            error.error_message = i18n.global.t("INVALID_RETYPED_PASSWORD_MESSAGE")
            const errors = new Array<GkillError>()
            errors.push(error)
            emits('received_errors', errors)
            is_submiting.value = false
            return false
        }

        // 1.Adminのパスワードリセットトークンを受け取る（クエリパラメータ）
        // 2.Adminのパスワードリセットを実施する
        // 3.Adminのログインセッションを取得する（Adminのパスワード情報は画面が持ってる）
        // 4.アカウント追加を行う
        // 5.アカウントのパスワードリセットトークンを取得する（管理者権限使用）
        // 6.アカウントのパスワードをリセットする
        // 7.Adminのログインセッションを破棄する
        // 8.登録完了メッセージを出してログイン画面に遷移する
        switch (regist_state.value) {
            case RegistStatus.none: {
                // 1.Adminのパスワードリセットトークンを受け取る（クエリパラメータ）
                // 2.Adminのパスワードリセットを実施する
                const set_new_password_admin_req = new SetNewPasswordRequest()
                set_new_password_admin_req.user_id = "admin"
                set_new_password_admin_req.reset_token = admin_account_password_reset_token.value
                set_new_password_admin_req.new_password_sha256 = await admin_password_sha256.value

                const set_new_password_admin_res = await props.gkill_api.set_new_password(set_new_password_admin_req)
                if (set_new_password_admin_res.errors && set_new_password_admin_res.errors.length !== 0) {
                    emits('received_errors', set_new_password_admin_res.errors)
                    is_submiting.value = false
                    return false
                }
                regist_state.value = RegistStatus.reseted_admin_password
            }
             
            case RegistStatus.reseted_admin_password: {
                // 3.Adminのログインセッションを取得する（Adminのパスワード情報は画面が持ってる）
                // 4.アカウント追加を行う
                const login_admin_account_req = new LoginRequest()
                login_admin_account_req.user_id = "admin"
                login_admin_account_req.password_sha256 = await admin_password_sha256.value

                const login_admin_account_res = await props.gkill_api.login(login_admin_account_req)
                if (login_admin_account_res.errors && login_admin_account_res.errors.length !== 0) {
                    emits('received_errors', login_admin_account_res.errors)
                    is_submiting.value = false
                    return false
                }

                const add_account_req = new AddAccountRequest()
                add_account_req.session_id = login_admin_account_res.session_id
                add_account_req.do_initialize = true
                add_account_req.account_info = new Account()
                add_account_req.account_info.is_admin = false
                add_account_req.account_info.is_enable = true
                add_account_req.account_info.user_id = user_id.value

                const add_account_res = await props.gkill_api.add_account(add_account_req)
                if (add_account_res.errors && add_account_res.errors.length !== 0) {
                    emits('received_errors', add_account_res.errors)
                    is_submiting.value = false
                    return false
                }
                regist_state.value = RegistStatus.added_account
            }
             
            case RegistStatus.added_account: {
                // 5.アカウントのパスワードリセットトークンを取得する（管理者権限使用）
                // 6.アカウントのパスワードをリセットする
                const login_admin_account_req = new LoginRequest()
                login_admin_account_req.user_id = "admin"
                login_admin_account_req.password_sha256 = await admin_password_sha256.value

                const login_admin_account_res = await props.gkill_api.login(login_admin_account_req)
                if (login_admin_account_res.errors && login_admin_account_res.errors.length !== 0) {
                    emits('received_errors', login_admin_account_res.errors)
                    is_submiting.value = false
                    return false
                }

                const get_server_configs_req = new GetServerConfigsRequest()
                get_server_configs_req.session_id = login_admin_account_res.session_id

                const get_server_configs_res = await props.gkill_api.get_server_configs(get_server_configs_req)
                if (get_server_configs_res.errors && get_server_configs_res.errors.length !== 0) {
                    emits('received_errors', get_server_configs_res.errors)
                    is_submiting.value = false
                    return false
                }

                const server_config = get_server_configs_res.server_configs.find((server_config) => server_config.enable_this_device)
                const account = server_config?.accounts.find((account) => account.user_id === user_id.value)

                const set_new_password_req = new SetNewPasswordRequest()
                set_new_password_req.user_id = user_id.value
                set_new_password_req.reset_token = account?.password_reset_token ?? ""
                set_new_password_req.new_password_sha256 = await password_sha256.value

                const set_new_password_res = await props.gkill_api.set_new_password(set_new_password_req)
                if (set_new_password_res.errors && set_new_password_res.errors.length !== 0) {
                    emits('received_errors', set_new_password_res.errors)
                    is_submiting.value = false
                    return false
                }
                regist_state.value = RegistStatus.reseted_account_password
            }
             
            case RegistStatus.reseted_account_password: {
                const message = new GkillMessage()
                message.message = i18n.global.t("REGISTERED_ACCOUNT_MESAGE")
                message.message_code = GkillMessageCodes.added_account
                emits('received_messages', [message])
                regist_state.value = RegistStatus.done
            }
             
            default:
                await sleep(2500)
                await resetDialogHistory()
                router.replace("/")
                return true
        }
    }

    // ── Return ──
    return {
        // State
        welcome_emoji,
        user_id,
        password,
        password_retype,
        admin_password,
        admin_password_retype,
        regist_state,
        is_submiting,

        // Computed
        app_content_height_px,
        app_content_width_px,

        // Constants
        RegistStatus,

        // Business logic
        try_regist_account,
    }
}
