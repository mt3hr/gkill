<template>
    <div class="mkfl_view_wrap">
        <v-app-bar :height="app_title_bar_height.valueOf()" class="app_bar" color="primary" app flat>
            <v-btn icon="mdi-menu" :ripple="false" link="false" :style="{ opacity: 0, cursor: 'unset', }" />
            <v-toolbar-title>
                <div>
                    <span>
                        {{ i18n.global.t("MKFL_APP_NAME") }}
                    </span>
                    <v-menu activator="parent">
                        <v-list>
                            <v-list-item :key="index" :value="index" v-for="page, index in page_list">
                                <v-list-item-title @click="navigateToPage(page.page_name)">
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
            <MkflView :app_content_height="app_content_height" :app_content_width="app_content_width"
                :application_config="application_config" :gkill_api="gkill_api"
                @received_errors="onMkflViewReceivedErrors"
                @received_messages="onMkflViewReceivedMessages"
                @deleted_kyou="onMkflViewDeletedKyou"
                @deleted_tag="onMkflViewDeletedTag"
                @deleted_text="onMkflViewDeletedText"
                @deleted_notification="onMkflViewDeletedNotification"
                @registered_kyou="onMkflViewRegisteredKyou"
                @registered_tag="onMkflViewRegisteredTag"
                @registered_text="onMkflViewRegisteredText"
                @registered_notification="onMkflViewRegisteredNotification"
                @updated_kyou="onMkflViewUpdatedKyou"
                @updated_tag="onMkflViewUpdatedTag"
                ref="mkfl_view" />
            <ApplicationConfigDialog :application_config="application_config" :gkill_api="gkill_api"
                :app_content_height="app_content_height" :app_content_width="app_content_width"
                :is_show="is_show_application_config_dialog"
                @received_errors="onApplicationConfigReceivedErrors"
                @received_messages="onApplicationConfigReceivedMessages"
                @requested_reload_application_config="load_application_config" ref="application_config_dialog" />
        </v-main>
        <div class="alert_container">
            <v-slide-y-transition group>
                <v-tooltip :text="(message.is_error ? 'エラーコード' : 'メッセージコード') + ':' + message.code"
                    v-for="message in messages" :key="message.id">
                    <template v-slot:activator="{ props }">
                        <v-alert v-bind="props" :color="message.is_error ? 'error' : undefined"
                            :closable="message.closable" @click:close="onAlertClickClose(message.id)">
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
import ApplicationConfigDialog from './dialogs/application-config-dialog.vue'
import MkflView from './views/mkfl-view.vue'
import { useMkflPage } from '@/classes/use-mkfl-page'

const {
    // Template refs
    mkfl_view,
    application_config_dialog,

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

    // Computed
    page_list,

    // Methods
    load_application_config,
    show_application_config_dialog,

    // Template event handlers
    navigateToPage,
    onMkflViewReceivedErrors,
    onMkflViewReceivedMessages,
    onMkflViewDeletedKyou,
    onMkflViewDeletedTag,
    onMkflViewDeletedText,
    onMkflViewDeletedNotification,
    onMkflViewRegisteredKyou,
    onMkflViewRegisteredTag,
    onMkflViewRegisteredText,
    onMkflViewRegisteredNotification,
    onMkflViewUpdatedKyou,
    onMkflViewUpdatedTag,
    onApplicationConfigReceivedErrors,
    onApplicationConfigReceivedMessages,
    onAlertClickClose,
} = useMkflPage()
</script>
<style lang="css" scoped>
.overlay_target {
    z-index: -10000;
    position: absolute;
    min-height: calc(v-bind('app_content_height.toString().concat("px")'));
    min-width: v-bind("is_loading ? 'calc(100vw)' : '0px'");
}
</style>
<style scoped>
:root {
    --actual_height: v-bind(actual_height)
}
</style>
