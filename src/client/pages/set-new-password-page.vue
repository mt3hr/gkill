<template>
    <v-app-bar :height="app_title_bar_height.valueOf()" class="app_bar" app color="primary" flat>
        <v-toolbar-title>{{ i18n.global.t("RESET_PASSWORD_TITLE") }}</v-toolbar-title>
        <v-spacer />
        <span class="gkill_version">{{ i18n.global.t("VERSION_TITLE") }}: {{ gkill_version }}</span>
    </v-app-bar>
    <v-main class="main">
        <SetNewPasswordView :app_content_height="app_content_height" :app_content_width="app_content_width"
            :application_config="new ApplicationConfig()" :gkill_api="gkill_api"
            @received_errors="onReceivedErrors"
            @received_messages="onReceivedMessages" />
        <div class="alert_container">
            <v-slide-y-transition group>
                <v-tooltip :text="(message.is_error ? 'エラーコード' : 'メッセージコード') + ':' + message.code"
                    v-for="message in messages" :key="message.id">
                    <template v-slot:activator="{ props }">
                        <v-alert v-bind="props" :color="message.is_error ? 'error' : undefined"
                            :closable="message.closable" @click:close="onCloseMessage(message.id)">
                            {{ message.message }}
                        </v-alert>
                    </template>
                </v-tooltip>
            </v-slide-y-transition>
        </div>
    </v-main>
</template>

<script lang="ts" setup>
import { i18n } from '@/i18n'
import SetNewPasswordView from './views/set-new-password-view.vue'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { useSetNewPasswordPage } from '@/classes/use-set-new-password-page'

const {
    // State
    actual_height,
    app_title_bar_height,
    app_title_bar_height_px,
    gkill_api,
    app_content_height,
    app_content_width,
    gkill_version,
    messages,

    // Event handlers
    onReceivedErrors,
    onReceivedMessages,
    onCloseMessage,
} = useSetNewPasswordPage()
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

.alert_container>div {
    width: fit-content;
}

.alert_container {
    justify-items: end;
    position: fixed;
    top: 60px;
    right: 10px;
    display: grid;
    grid-gap: .5em;
    z-index: 99;
}
</style>
<style scoped>
:root {
    --actual_height: v-bind(actual_height)
}
</style>