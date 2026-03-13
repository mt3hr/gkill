<template>
    <div class="kftl_view_wrap">
        <v-app-bar :height="app_title_bar_height.valueOf()" class="app_bar" color="primary" app flat>
            <v-btn icon="mdi-menu" :ripple="false" link="false" :style="{ opacity: 0, cursor: 'unset', }" />
            <v-toolbar-title>
                <div>
                    <span>
                        {{ i18n.global.t("KFTL_APP_NAME") }}
                    </span>
                    <v-menu activator="parent">
                        <v-list>
                            <v-list-item :key="index" :value="index" v-for="page, index in [
                                { app_name: i18n.global.t('RYKV_APP_NAME'), page_name: 'rykv' },
                                { app_name: i18n.global.t('MI_APP_NAME'), page_name: 'mi' },
                                { app_name: i18n.global.t('KFTL_APP_NAME'), page_name: 'kftl' },
                                { app_name: i18n.global.t('PLAING_TIMEIS_APP_NAME'), page_name: 'plaing' },
                                { app_name: i18n.global.t('MKFL_APP_NAME'), page_name: 'mkfl' },
                                { app_name: i18n.global.t('SAIHATE_APP_NAME'), page_name: 'saihate' },
                            ]">
                                <v-list-item-title
                                    @click="async () => { await resetDialogHistory(); router.replace('/' + page.page_name + '?loaded=true') }">
                                    {{ page.app_name }}</v-list-item-title>
                            </v-list-item>
                        </v-list>
                    </v-menu>
                </div>
            </v-toolbar-title>
            <v-spacer />
            <v-divider vertical />
            <v-btn icon="mdi-cog" :disabled="!application_config.is_loaded" @click="show_application_config_dialog()" />
        </v-app-bar>
        <v-main class="main">
            <div class="overlay_target">
                <v-overlay v-model="is_loading" class="align-center justify-center" persistent contained>
                    <v-progress-circular indeterminate color="primary" />
                </v-overlay>
            </div>
            <kftlView :app_content_height="app_content_height" :app_content_width="app_content_width"
                :application_config="application_config" :gkill_api="gkill_api"
                @received_errors="onReceivedErrors"
                @received_messages="onReceivedMessages"
                ref="kftl_view" />
            <ApplicationConfigDialog :application_config="application_config" :gkill_api="gkill_api"
                :app_content_height="app_content_height" :app_content_width="app_content_width"
                :is_show="is_show_application_config_dialog"
                @received_errors="onReceivedErrors"
                @received_messages="onReceivedMessages"
                @requested_reload_application_config="onRequestedReloadApplicationConfig" ref="application_config_dialog" />
        </v-main>
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
    </div>
</template>

<script lang="ts" setup>
import { i18n } from '@/i18n'
import router from '@/router'
import ApplicationConfigDialog from './dialogs/application-config-dialog.vue'
import kftlView from './views/kftl-view.vue'
import { resetDialogHistory } from '@/classes/use-dialog-history-stack'
import { useKftlPage } from '@/classes/use-kftl-page'

const {
    // Template refs
    application_config_dialog,
    kftl_view,

    // State
    actual_height,
    app_title_bar_height,
    gkill_api,
    application_config,
    app_content_height,
    app_content_width,
    is_show_application_config_dialog,
    is_loading,
    messages,

    // Methods
    show_application_config_dialog,

    // Event handlers
    onCloseMessage,
    onReceivedErrors,
    onReceivedMessages,
    onRequestedReloadApplicationConfig,
} = useKftlPage()
</script>
<style lang="css" scoped>
.overlay_target {
    z-index: -10000;
    position: absolute;
    min-height: calc(v-bind('app_content_height.toString().concat("px")'));
    min-width: v-bind("is_loading == true ? 'calc(100vw)' : '0px'");
}
</style>
<style scoped>
:root {
    --actual_height: v-bind(actual_height)
}
</style>
