<template>
    <div>
        <rykvView :app_content_height="app_content_height" :app_content_width="app_content_width"
            :app_title_bar_height="app_title_bar_height" :application_config="application_config" :gkill_api="gkill_api"
            :is_shared_rykv_view="false" :share_title="''"
            v-on="rykvViewHandlers" />
        <ApplicationConfigDialog :application_config="application_config" :gkill_api="gkill_api"
            :app_content_height="app_content_height" :app_content_width="app_content_width"
            :is_show="is_show_application_config_dialog"
            @received_errors="onReceivedErrors"
            @received_messages="onReceivedMessages"
            @requested_reload_application_config="onRequestedReloadApplicationConfig" ref="application_config_dialog" />
        <UploadFileDialog :app_content_height="app_content_height" :app_content_width="app_content_width"
            :application_config="application_config" :gkill_api="gkill_api" />
        <TutorialDialog :application_config="application_config" :gkill_api="gkill_api"
            ref="tutorial_dialog" />
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
import { ref, watch, nextTick } from 'vue'
import ApplicationConfigDialog from './dialogs/application-config-dialog.vue'
import UploadFileDialog from './dialogs/upload-file-dialog.vue'
import TutorialDialog from './dialogs/tutorial-dialog.vue'
import rykvView from './views/rykv-view.vue'
import { useRykvPage } from '@/classes/use-rykv-page'

const tutorial_dialog = ref<InstanceType<typeof TutorialDialog> | null>(null)

const {
    // Template refs
    application_config_dialog,

    // State
    actual_height,
    app_title_bar_height,
    gkill_api,
    application_config,
    app_content_height,
    app_content_width,
    is_show_application_config_dialog,
    messages,

    // Event handlers
    onCloseMessage,
    onReceivedErrors,
    onReceivedMessages,
    onRequestedReloadApplicationConfig,

    // CRUD relay
    rykvViewHandlers,
} = useRykvPage()

watch(application_config, (config) => {
    if (config.is_loaded && config.show_tutorial_on_startup) {
        nextTick(() => tutorial_dialog.value?.show())
    }
})
</script>
<style scoped>
:root {
    --actual_height: v-bind(actual_height)
}
</style>
