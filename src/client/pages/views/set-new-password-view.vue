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
import { useRoute } from 'vue-router'
import type { SetNewPasswordViewEmits } from './set-new-password-view-emits'
import type { SetNewPasswordViewProps } from './set-new-password-view-props'
import { useSetNewPasswordView } from '@/classes/use-set-new-password-view'

const props = defineProps<SetNewPasswordViewProps>()
const emits = defineEmits<SetNewPasswordViewEmits>()

const {
    // State
    welcome_emoji,
    user_id,
    password,
    password_retype,

    // Computed
    app_content_height_px,
    app_content_width_px,

    // Business logic
    try_set_new_password,
} = useSetNewPasswordView({ props, emits })
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