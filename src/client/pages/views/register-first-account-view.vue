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
                        :readonly="REGISTER_STATUS.added_account <= register_state" />
                </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
                <v-col cols="12">
                    <v-text-field id="password" :label="i18n.global.t('PASSWORD_TITLE')" :type="'password'"
                        v-model="password" name="new-password" autocomplete="new-password"
                        :readonly="REGISTER_STATUS.reseted_account_password <= register_state" />
                </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
                <v-col cols="12">
                    <v-text-field :label="i18n.global.t('PASSWORD_RETYPE_TITLE')" :type="'password'"
                        name="retype-password" autocomplete="retype-password" v-model="password_retype"
                        :readonly="REGISTER_STATUS.reseted_account_password <= register_state" />
                </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
                <v-col cols="12">
                    <v-text-field :label="i18n.global.t('ADMIN_PASSWORD_TITLE')" :type="'password'"
                        v-model="admin_password" :readonly="REGISTER_STATUS.reseted_admin_password <= register_state" />
                </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
                <v-col cols="12">
                    <v-text-field :label="i18n.global.t('ADMIN_PASSWORD_RETYPE_TITLE')" :type="'password'"
                        v-model="admin_password_retype"
                        :readonly="REGISTER_STATUS.reseted_admin_password <= register_state" />
                </v-col>
            </v-row>

            <v-row class="pa-0 ma-0">
                <v-spacer />
                <v-col cols="auto">
                    <v-btn dark class="login_button" color="primary" @click="try_register_account()"
                        :disable="is_submiting">
                        {{ i18n.global.t("REGISTER_ACCOUNT_TITLE") }}
                    </v-btn>
                </v-col>
            </v-row>
        </v-container>
    </div>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'

import type { RegisterFirstAccountViewProps } from './register-first-account-view-props'
import type { RegisterFirstAccountViewEmits } from './register-first-account-view-emits'
import { useRegisterFirstAccountView } from '@/classes/use-register-first-account-view'

const props = defineProps<RegisterFirstAccountViewProps>()
const emits = defineEmits<RegisterFirstAccountViewEmits>()

const {
    // State
    welcome_emoji,
    user_id,
    password,
    password_retype,
    admin_password,
    admin_password_retype,
    register_state,
    is_submiting,

    // Computed
    app_content_height_px,
    app_content_width_px,

    // Constants
    RegisterStatus: REGISTER_STATUS,

    // Business logic
    try_register_account,
} = useRegisterFirstAccountView({ props, emits })
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
