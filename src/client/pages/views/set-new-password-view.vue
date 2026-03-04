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
                    <v-text-field id="username" :label="i18n.global.t('USER_ID_TITLE')" v-model="user_id"
                        name="username" autocomplete="username" :readonly="!(!useRoute().query.user_id)" />
                </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
                <v-col cols="12">
                    <v-text-field id="password" :label="i18n.global.t('PASSWORD_TITLE')" :type="'password'"
                        v-model="password" name="new-password" autocomplete="new-password" />
                </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
                <v-col cols="12">
                    <v-text-field :label="i18n.global.t('PASSWORD_RETYPE_TITLE')" :type="'password'"
                        name="retype-password" autocomplete="retype-password" v-model="password_retype" />
                </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
                <v-spacer />
                <v-col cols="auto">
                    <v-btn dark class="login_button" color="primary" @click="try_set_new_password()">
                        {{ i18n.global.t("RESET_PASSWORD_TITLE") }}
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
import type { SetNewPasswordViewEmits } from './set-new-password-view-emits'
import type { SetNewPasswordViewProps } from './set-new-password-view-props'
import { useRoute } from 'vue-router';
import { SetNewPasswordRequest } from '@/classes/api/req_res/set-new-password-request';
import { GkillMessage } from '@/classes/api/gkill-message';
import { GkillMessageCodes } from '@/classes/api/message/gkill_message';
import { GkillErrorCodes } from '@/classes/api/message/gkill_error';
import { useTheme } from 'vuetify';
import { resetDialogHistory } from '@/classes/use-dialog-history-stack'
const theme = useTheme()

const welcome_emoji = computed(() => theme.current.value.dark ? "⭐️" : "❄️")
const password_reset_token: Ref<string> = ref(useRoute().query.reset_token ? useRoute().query.reset_token!.toString() : "")
const user_id: Ref<string> = ref(useRoute().query.user_id ? useRoute().query.user_id!.toString() : "")
const password: Ref<string> = ref("")
const password_retype: Ref<string> = ref("")

const props = defineProps<SetNewPasswordViewProps>()
const emits = defineEmits<SetNewPasswordViewEmits>()

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

nextTick(() => {
    if (user_id.value === "admin") {
        const message = new GkillMessage()
        message.message_code = GkillMessageCodes.set_admin_password_request
        message.message = i18n.global.t("RESET_SET_ADMIN_PASSWORD_MESSAGE")
        emits('received_messages', [message])
    }
})

const sleep = (time: number) => new Promise<void>((r) => setTimeout(r, time))

async function try_set_new_password(): Promise<boolean> {
    // 未入力チェック
    if (user_id.value === "") {
        const error = new GkillError()
        error.error_code = GkillErrorCodes.user_id_is_blank
        error.error_message = i18n.global.t("REQUEST_INPUT_USER_ID_MESSAGE")
        const errors = new Array<GkillError>()
        errors.push(error)
        emits('received_errors', errors)
        return false
    }
    if (password.value === "" || password_retype.value === "") {
        const error = new GkillError()
        error.error_code = GkillErrorCodes.password_is_blank
        error.error_message = i18n.global.t("REQUEST_INPUT_PASSWORD_MESSAGE")
        const errors = new Array<GkillError>()
        errors.push(error)
        emits('received_errors', errors)
        return false
    }
    if (password.value !== password_retype.value) {
        const error = new GkillError()
        error.error_code = GkillErrorCodes.password_retype_is_blank
        error.error_message = i18n.global.t("INVALID_RETYPED_PASSWORD_MESSAGE")
        const errors = new Array<GkillError>()
        errors.push(error)
        emits('received_errors', errors)
        return false
    }

    const req = new SetNewPasswordRequest()
    req.user_id = user_id.value
    req.reset_token = password_reset_token.value
    req.new_password_sha256 = (await password_sha256.value.then((value) => value))

    const res = await props.gkill_api.set_new_password(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return false
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }

    await sleep(2500)
    await resetDialogHistory()
    router.replace("/")

    return true
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
