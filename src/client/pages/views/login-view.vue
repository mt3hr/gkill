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
                <v-col cols="12">
                    <v-text-field id="username" @keydown.enter="try_login(user_id, password_sha256)" name="username"
                        autocomplete="username" :label="i18n.global.t('USER_ID_TITLE')" v-model="user_id" />
                </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
                <v-col cols="12">
                    <v-text-field id="password" @keydown.enter="try_login(user_id, password_sha256)"
                        name="current-password" autocomplete="current-password" :label="i18n.global.t('PASSWORD_TITLE')"
                        :type="'password'" v-model="password" />
                </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
                <v-spacer />
                <v-col cols="auto">
                    <v-btn dark class="login_button" color="primary" @click="try_login(user_id, password_sha256)">
                        {{ i18n.global.t("LOGIN_TITLE") }}</v-btn>
                </v-col>
            </v-row>
        </v-container>
    </div>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { computed, ref, type Ref } from 'vue'
import { type LoginViewProps } from './login-view-props'
import type LoginViewEmits from './login-view-emits'
import { LoginRequest } from '@/classes/api/req_res/login-request';
import router from '@/router';
import { GkillError } from '@/classes/api/gkill-error';
import { GkillErrorCodes } from '@/classes/api/message/gkill_error';
import { useTheme } from 'vuetify';
import { resetDialogHistory } from '@/classes/use-dialog-history-stack'
const theme = useTheme()

const welcome_emoji = computed(() => theme.current.value.dark ? "⭐️" : "❄️")
const user_id: Ref<string> = ref("")
const password: Ref<string> = ref("")

const props = defineProps<LoginViewProps>()
const emits = defineEmits<LoginViewEmits>()

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

check_logined()

async function check_logined(): Promise<void> {
    const session_id = props.gkill_api.get_session_id()
    const default_page = props.gkill_api.get_default_page_from_cookie()
    if (session_id && session_id !== "" && default_page && default_page !== "") {
        await resetDialogHistory()
        router.replace("/" + default_page)
    }
}

async function try_login(user_id: string, password_sha256: Promise<string>): Promise<boolean> {
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
        req.password_sha256 = (await password_sha256.then((value) => value))

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
        error.error_code = GkillErrorCodes.requeired_certificate
        error.error_message = i18n.global.t("REQUEST_CERTIFICATE_REQUIRED_MESSAGE")
        const errors = new Array<GkillError>()
        errors.push(error)
        emits('received_errors', errors)
        return false
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