<template>
    <div class="kyou_view_wrap">
        <v-app-bar :height="app_title_bar_height.valueOf()" class="app_bar" color="primary" app flat>
            <v-btn icon="mdi-menu" :ripple="false" link="false" :style="{ opacity: 0, cursor: 'unset', }" />
            <v-toolbar-title>
                <div>
                    <span>
                        {{ i18n.global.t("KYOU_APP_NAME") }}
                    </span>
                    <v-menu activator="parent">
                        <v-list>
                            <v-list-item :key="index" :value="index" v-for="page, index in page_list">
                                <v-list-item-title
                                    @click="navigateToPage(page.page_name)">
                                    {{ page.app_name }}</v-list-item-title>
                            </v-list-item>
                        </v-list>
                    </v-menu>
                </div>
            </v-toolbar-title>
            <v-spacer />
            <v-divider vertical />
            <v-tooltip :text="i18n.global.t('TOOLTIP_HELP')">
                <template v-slot:activator="{ props }">
                    <v-btn v-bind="props" icon="mdi-help-circle-outline" @click="help_dialog?.show()" />
                </template>
            </v-tooltip>
            <v-tooltip :text="i18n.global.t('TOOLTIP_SETTINGS')">
                <template v-slot:activator="{ props }">
                    <v-btn v-bind="props" icon="mdi-cog" :disabled="!application_config.is_loaded" @click="show_application_config_dialog()" />
                </template>
            </v-tooltip>
        </v-app-bar>
        <v-main class="main">
            <div class="overlay_target">
                <v-overlay v-model="is_loading" class="align-center justify-center" persistent contained>
                    <v-progress-circular indeterminate color="primary" />
                </v-overlay>
            </div>
            <KyouView :application_config="application_config" :gkill_api="gkill_api"
                :is_image_request_to_thumb_size="false" :highlight_targets="hightlight_targets"
                :is_image_view="is_image_view" :kyou="kyou" :show_checkbox="false"
                :show_content_only="false" :show_mi_create_time="true" :show_mi_estimate_end_time="true"
                :show_mi_estimate_start_time="true" :show_mi_limit_time="true" :show_timeis_elapsed_time="true"
                :show_timeis_plaing_end_button="true" :height="'fit-content'" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" :show_attached_timeis="true" :show_update_time="false"
                :show_related_time="true" :width="'fit-content'" :is_readonly_mi_check="false"
                :show_attached_tags="true" :show_attached_texts="true" :show_attached_notifications="true"
                :show_rep_name="true" :force_show_latest_kyou_info="true"
                @received_errors="(...errors: any[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => write_messages(messages[0] as Array<GkillMessage>)" />
            <ApplicationConfigDialog :application_config="application_config" :gkill_api="gkill_api"
                :app_content_height="app_content_height" :app_content_width="app_content_width"
                :is_show="is_show_application_config_dialog"
                @received_errors="(...errors: any[]) => write_errors(errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => write_messages(messages[0] as Array<GkillMessage>)"
                @requested_reload_application_config="load_application_config" ref="application_config_dialog" />
            <HelpDialog screen_name="kyou" ref="help_dialog" />
        </v-main>
        <div class="alert_container" role="status" aria-live="polite">
            <v-slide-y-transition group>
                <v-tooltip :text="(message.is_error ? 'エラーコード' : 'メッセージコード') + ':' + message.code"
                    v-for="message in messages" :key="message.id">
                    <template v-slot:activator="{ props }">
                        <v-alert v-bind="props" :color="message.is_error ? 'error' : undefined"
                            :role="message.is_error ? 'alert' : undefined"
                            :closable="message.closable" @click:close="close_message(message.id)">
                            {{ message.message }}
                        </v-alert>
                    </template>
                </v-tooltip>
            </v-slide-y-transition>
        </div>
    </div>
</template>
<script lang="ts" setup>
import { ref } from 'vue'
import { i18n } from '@/i18n'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import ApplicationConfigDialog from './dialogs/application-config-dialog.vue'
import HelpDialog from './dialogs/help-dialog.vue'
import KyouView from './views/kyou-view.vue'
import { useKyouPage } from '@/classes/use-kyou-page'

const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)

const {
    // Template refs
    application_config_dialog,

    // State
    enable_context_menu,
    enable_dialog,
    actual_height,
    app_title_bar_height,
    gkill_api,
    application_config,
    app_content_height,
    app_content_width,
    is_show_application_config_dialog,
    hightlight_targets,
    is_image_view,
    kyou,
    is_loading,
    messages,

    // Computed
    page_list,

    // Template event handlers
    navigateToPage,
    write_errors,
    write_messages,
    close_message,
    load_application_config,
    show_application_config_dialog,
} = useKyouPage()
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
