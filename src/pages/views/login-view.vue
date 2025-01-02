<template>
    <div class="login_wrap">
        <v-container class="pa-0 ma-0">
            <v-row class="pa-0 ma-0">
                <v-col cols="auto">
                    <div class="welcome">ようこそ</div>
                </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
                <v-col cols="12">
                    <v-text-field label="ユーザID" v-model="user_id" autofocus />
                </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
                <v-col cols="12">
                    <v-text-field label="パスワード" :type="'password'" v-model="password" />
                </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
                <v-spacer />
                <v-col cols="auto">
                    <v-btn class="login_button" color="primary" @click="try_login(user_id, password_sha256)">
                        ログイン
                    </v-btn>
                </v-col>
            </v-row>
        </v-container>
    </div>
</template>
<script lang="ts" setup>
import { computed, ref, type Ref } from 'vue'
import { type LoginViewProps } from './login-view-props'
import type LoginViewEmits from './login-view-emits'
import { LoginRequest } from '@/classes/api/req_res/login-request';
import router from '@/router';
import { GkillError } from '@/classes/api/gkill-error';

const user_id: Ref<string> = ref("")
const password: Ref<string> = ref("")

const props = defineProps<LoginViewProps>()
const emits = defineEmits<LoginViewEmits>()

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

check_logined()

async function check_logined(): Promise<void> {
    const session_id = props.gkill_api.get_session_id()
    if (session_id && session_id !== "") {
        router.replace("/rykv")
    }
}

async function try_login(user_id: string, password_sha256: Promise<string>): Promise<boolean> {
    // 未入力チェック
    if (user_id === "") {
        const error = new GkillError()
        error.error_code = "//TODO"
        error.error_message = "ユーザIDを入力してください"
        const errors = new Array<GkillError>()
        errors.push(error)
        emits('received_errors', errors)
        return false
    }
    if (password.value === "") {
        const error = new GkillError()
        error.error_code = "//TODO"
        error.error_message = "パスワードを入力してください"
        const errors = new Array<GkillError>()
        errors.push(error)
        emits('received_errors', errors)
        return false
    }

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
</style>