<template>
    <div class="login_wrap">
        <v-container class="pa-0 ma-0">
            <v-row class="pa-0 ma-0">
                <v-spacer />
                <v-col cols="auto">
                    <div class="welcome">{{ welcome_emoji + i18n.global.t("WELCOME_TITLE") + welcome_emoji }}</div>
                </v-col>
                <v-spacer />
            </v-row>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <div class="welcome_message">{{ i18n.global.t("WELCOME_MESSAGE") }}</div>
                </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
                <v-col cols="12">
                    <v-text-field id="username" :label="i18n.global.t('USER_ID_TITLE')" v-model="user_id"
                        name="new-username" autocomplete="new-username"
                        :readonly="RegistStatus.added_account <= regist_state" />
                </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
                <v-col cols="12">
                    <v-text-field id="password" :label="i18n.global.t('PASSWORD_TITLE')" :type="'password'"
                        v-model="password" name="new-password" autocomplete="new-password"
                        :readonly="RegistStatus.reseted_account_password <= regist_state" />
                </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
                <v-col cols="12">
                    <v-text-field :label="i18n.global.t('PASSWORD_RETYPE_TITLE')" :type="'password'"
                        name="retype-password" autocomplete="retype-password" v-model="password_retype"
                        :readonly="RegistStatus.reseted_account_password <= regist_state" />
                </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
                <v-col cols="12">
                    <v-text-field :label="i18n.global.t('ADMIN_PASSWORD_TITLE')" :type="'password'"
                        v-model="admin_password" :readonly="RegistStatus.reseted_admin_password <= regist_state" />
                </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
                <v-col cols="12">
                    <v-text-field :label="i18n.global.t('ADMIN_PASSWORD_RETYPE_TITLE')" :type="'password'"
                        v-model="admin_password_retype"
                        :readonly="RegistStatus.reseted_admin_password <= regist_state" />
                </v-col>
            </v-row>

            <v-row class="pa-0 ma-0">
                <v-spacer />
                <v-col cols="auto">
                    <v-btn dark class="login_button" color="primary" @click="try_regist_account()"
                        :disable="is_submiting">
                        {{ i18n.global.t("REGIST_ACCOUNT_TITLE") }}
                    </v-btn>
                </v-col>
            </v-row>
        </v-container>
    </div>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { computed, nextTick, ref, type Ref } from 'vue'
import router from '@/router';
import { GkillError } from '@/classes/api/gkill-error';
import { useRoute } from 'vue-router';
import { SetNewPasswordRequest } from '@/classes/api/req_res/set-new-password-request';
import { GkillErrorCodes } from '@/classes/api/message/gkill_error';

import type { RegistFirstAccountViewProps } from './regist-first-account-view-props';
import type { RegistFirstAccountViewEmits } from './regist-first-account-view-emits';
import { AddAccountRequest } from '@/classes/api/req_res/add-account-request';
import { Account } from '@/classes/datas/config/account';
import { LoginRequest } from '@/classes/api/req_res/login-request';
import { GetServerConfigsRequest } from '@/classes/api/req_res/get-server-configs-request';
import { GkillMessage } from '@/classes/api/gkill-message';
import { GkillMessageCodes } from '@/classes/api/message/gkill_message';
import { useTheme } from 'vuetify';
const theme = useTheme()

const welcome_emoji = computed(() => theme.current.value.dark ? "⭐️" : "❄️")
const admin_account_password_reset_token: Ref<string> = ref(useRoute().query.reset_token ? useRoute().query.reset_token!.toString() : "")
const user_id: Ref<string> = ref(useRoute().query.user_id ? useRoute().query.user_id!.toString() : "")
const password: Ref<string> = ref("")
const password_retype: Ref<string> = ref("")
const admin_password: Ref<string> = ref("")
const admin_password_retype: Ref<string> = ref("")

const props = defineProps<RegistFirstAccountViewProps>()
const emits = defineEmits<RegistFirstAccountViewEmits>()

// nextTick(() => document.getElementById("username")?.focus()).then(() => nextTick(() => document.getElementById("password")?.focus())).then(() => nextTick(() => document.getElementById("username")?.focus()))

const app_content_height_px = computed(() => props.app_content_height + 'px')
const app_content_width_px = computed(() => props.app_content_width + 'px')
// eslint-disable-next-line vue/no-async-in-computed-properties
const password_sha256 = computed(async () => {
    const encoder = new TextEncoder();
    const msgUint8 = encoder.encode(password.value);
    // eslint-disable-next-line vue/no-async-in-computed-properties
    const hashBuffer = await crypto.subtle.digest('SHA-256', msgUint8);

    const hashArray = Array.from(new Uint8Array(hashBuffer));
    const hashHex = hashArray
        .map((b) => b.toString(16).padStart(2, '0'))
        .join('');
    return hashHex;
})
// eslint-disable-next-line vue/no-async-in-computed-properties
const admin_password_sha256 = computed(async () => {
    const encoder = new TextEncoder();
    const msgUint8 = encoder.encode(admin_password.value);
    // eslint-disable-next-line vue/no-async-in-computed-properties
    const hashBuffer = await crypto.subtle.digest('SHA-256', msgUint8);

    const hashArray = Array.from(new Uint8Array(hashBuffer));
    const hashHex = hashArray
        .map((b) => b.toString(16).padStart(2, '0'))
        .join('');
    return hashHex;
})

enum RegistStatus {
    none = 0,
    reseted_admin_password = 1,
    added_account = 2,
    reseted_account_password = 3,
    done = 4,
}
const regist_state = ref(RegistStatus.none)
const is_submiting = ref(false)

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
        // eslint-disable-next-line no-fallthrough
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
        // eslint-disable-next-line no-fallthrough
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
            const account = server_config?.accounts.find((account) => account.user_id === user_id.value)!

            const set_new_password_req = new SetNewPasswordRequest()
            set_new_password_req.user_id = user_id.value
            set_new_password_req.reset_token = account?.password_reset_token!
            set_new_password_req.new_password_sha256 = await password_sha256.value

            const set_new_password_res = await props.gkill_api.set_new_password(set_new_password_req)
            if (set_new_password_res.errors && set_new_password_res.errors.length !== 0) {
                emits('received_errors', set_new_password_res.errors)
                is_submiting.value = false
                return false
            }
            regist_state.value = RegistStatus.reseted_account_password
        }
        // eslint-disable-next-line no-fallthrough
        case RegistStatus.reseted_account_password: {
            const message = new GkillMessage()
            message.message = "登録が完了しました"
            message.message_code = GkillMessageCodes.added_account
            emits('received_messages', [message])
            regist_state.value = RegistStatus.done
        }
        // eslint-disable-next-line no-fallthrough
        default:
            await sleep(2500)
            router.replace("/")
            return true
    }
}
</script>

<style lang="css" scoped>
.login_wrap {
    height: v-bind(app_content_height_px);
    max-height: v-bind(app_content_height_px);
    min-height: v-bind(app_content_height_px);
    width: v-bind(app_content_width_px);
    max-width: v-bind(app_content_width_px);
    min-width: v-bind(app_content_width_px);
    display: flex;
    justify-content: center;
    align-items: center;
}

.welcome {
    font-size: x-large;
}
</style>
