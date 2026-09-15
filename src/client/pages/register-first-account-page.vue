<template>
    <v-app-bar :height="app_title_bar_height.valueOf()" class="app_bar" app color="primary" flat>
        <v-toolbar-title>{{ i18n.global.t("REGISTER_FIRST_ACCOUNT_TITLE") }}</v-toolbar-title>
        <v-spacer />
        <span class="gkill_version">{{ i18n.global.t("VERSION_TITLE") }}: {{ gkill_version }}</span>
        <v-tooltip :text="i18n.global.t('TOOLTIP_HELP')">
            <template v-slot:activator="{ props }">
                <v-btn v-bind="props" icon="mdi-help-circle-outline" @click="help_dialog?.show()" />
            </template>
        </v-tooltip>
    </v-app-bar>
    <v-main class="main">
        <RegisterFirstAccountView :app_content_height="app_content_height" :app_content_width="app_content_width"
            :application_config="new ApplicationConfig()" :gkill_api="gkill_api"
            @received_errors="onReceivedErrors"
            @received_messages="onReceivedMessages" />
        <HelpDialog screen_name="getting-started" ref="help_dialog" />
        <GkillMessageFeedView />
    </v-main>
</template>

<script lang="ts" setup>
import { ref } from 'vue'
import { i18n } from '@/i18n'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import RegisterFirstAccountView from './views/register-first-account-view.vue'
import HelpDialog from './dialogs/help-dialog.vue'
import { useRegisterFirstAccountPage } from '@/classes/use-register-first-account-page'
import GkillMessageFeedView from './views/gkill-message-feed-view.vue'

const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)

const {
    // State
    actual_height,
    app_title_bar_height,
    app_title_bar_height_px,
    gkill_api,
    app_content_height,
    app_content_width,
    gkill_version,

    // Event handlers
    onReceivedErrors,
    onReceivedMessages,
} = useRegisterFirstAccountPage()
</script>

<style lang="css" scoped>
.main {
    height: calc(100vh - v-bind(app_title_bar_height_px));
    padding-top: v-bind(app_title_bar_height_px);
    top: v-bind(app_title_bar_height_px)
}

.gkill_version {
    font-size: small;
    margin-right: 15px;
}

</style>
<style scoped>
:root {
    --actual_height: v-bind(actual_height)
}
</style>