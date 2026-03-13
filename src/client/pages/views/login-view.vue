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
                    <v-text-field id="username" @keydown.enter="try_login(user_id)" name="username"
                        autocomplete="username" :label="i18n.global.t('USER_ID_TITLE')" v-model="user_id" />
                </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
                <v-col cols="12">
                    <v-text-field id="password" @keydown.enter="try_login(user_id)"
                        name="current-password" autocomplete="current-password" :label="i18n.global.t('PASSWORD_TITLE')"
                        :type="'password'" v-model="password" />
                </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
                <v-spacer />
                <v-col cols="auto">
                    <v-btn dark class="login_button" color="primary" @click="try_login(user_id)">
                        {{ i18n.global.t("LOGIN_TITLE") }}</v-btn>
                </v-col>
            </v-row>
        </v-container>
    </div>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type LoginViewProps } from './login-view-props'
import type LoginViewEmits from './login-view-emits'
import { useLoginView } from '@/classes/use-login-view'

const props = defineProps<LoginViewProps>()
const emits = defineEmits<LoginViewEmits>()

const {
    // State
    welcome_emoji,
    user_id,
    password,
    app_content_height_px,
    app_content_width_px,

    // Business logic
    try_login,
} = useLoginView({ props, emits })
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
