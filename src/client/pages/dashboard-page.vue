<template>
    <div>
        <DashboardView :app_content_height="app_content_height" :app_content_width="app_content_width"
            :app_title_bar_height="app_title_bar_height" :application_config="application_config"
            :gkill_api="gkill_api"
            :application_config_load_failed="application_config_load_failed"
            :is_hosted_in_dialog="false"
            :kyou_change_channel="null /* 単独ページ。画面間の伝播はポートの中だけ */"
            v-on="dashboardViewHandlers" />
        <ApplicationConfigDialog :application_config="application_config" :gkill_api="gkill_api"
            :app_content_height="app_content_height" :app_content_width="app_content_width"
            @received_errors="(...errors: unknown[]) => write_errors(errors[0] as Array<GkillError>)"
            @received_messages="(...msgs: unknown[]) => write_messages(msgs[0] as Array<GkillMessage>)"
            @requested_reload_application_config="load_application_config" ref="application_config_dialog" />
        <ConfirmLogoutDialog @requested_logout="(close_database: boolean) => logout(close_database)"
            ref="confirm_logout_dialog" />
        <TutorialDialog :application_config="application_config" :gkill_api="gkill_api"
            ref="tutorial_dialog" />
        <div class="alert_container" role="status" aria-live="polite">
            <v-slide-y-transition group>
                <v-tooltip :text="(message.is_error ? 'エラーコード' : 'メッセージコード') + ':' + message.code"
                    v-for="message in messages" :key="message.id">
                    <template v-slot:activator="{ props }">
                        <v-alert v-bind="props" :color="message.is_error ? 'error' : undefined"
                            :role="message.is_error ? 'alert' : undefined" :closable="message.closable"
                            @click:close="close_message(message.id)">
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
import { useTutorialOnStartup } from '@/classes/use-tutorial-on-startup'
import DashboardView from './views/dashboard-view.vue'
import ApplicationConfigDialog from './dialogs/application-config-dialog.vue'
import ConfirmLogoutDialog from './dialogs/confirm-logout-dialog.vue'
import TutorialDialog from './dialogs/tutorial-dialog.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { useDashboardPage } from '@/classes/use-dashboard-page'

const tutorial_dialog = ref<InstanceType<typeof TutorialDialog> | null>(null)

const {
    // Template refs
    application_config_dialog,
    confirm_logout_dialog,

    // State
    actual_height,
    app_title_bar_height,
    gkill_api,
    application_config,
    application_config_load_failed,
    app_content_height,
    app_content_width,
    messages,

    // Methods
    write_errors,
    write_messages,
    close_message,
    load_application_config,
    logout,

    // CRUD relay
    dashboardViewHandlers,
} = useDashboardPage()

useTutorialOnStartup(application_config, tutorial_dialog)
</script>
<style scoped>
:root {
    --actual_height: v-bind(actual_height)
}
</style>
