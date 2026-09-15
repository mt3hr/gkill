<template>
    <div>
        <rykvView :app_content_height="app_content_height" :app_content_width="app_content_width"
            :app_title_bar_height="app_title_bar_height" :application_config="application_config" :gkill_api="gkill_api"
            :is_shared_rykv_view="false" :share_title="''"
            :application_config_load_failed="application_config_load_failed"
            :is_hosted_in_dialog="false"
            :kyou_change_channel="null /* 単独ページ。画面間の伝播はポートの中だけ */" :column_state_instance_key="''"
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
        <GkillMessageFeedView />
    </div>
</template>

<script lang="ts" setup>
import { ref } from 'vue'
import { useTutorialOnStartup } from '@/classes/use-tutorial-on-startup'
import ApplicationConfigDialog from './dialogs/application-config-dialog.vue'
import UploadFileDialog from './dialogs/upload-file-dialog.vue'
import TutorialDialog from './dialogs/tutorial-dialog.vue'
import rykvView from './views/rykv-view.vue'
import { useRykvPage } from '@/classes/use-rykv-page'
import GkillMessageFeedView from './views/gkill-message-feed-view.vue'

const tutorial_dialog = ref<InstanceType<typeof TutorialDialog> | null>(null)

const {
    // Template refs
    application_config_dialog,

    // State
    actual_height,
    app_title_bar_height,
    gkill_api,
    application_config,
    application_config_load_failed,
    app_content_height,
    app_content_width,
    is_show_application_config_dialog,

    // Event handlers
    onReceivedErrors,
    onReceivedMessages,
    onRequestedReloadApplicationConfig,

    // CRUD relay
    rykvViewHandlers,
} = useRykvPage()

useTutorialOnStartup(application_config, tutorial_dialog)
</script>
<style scoped>
:root {
    --actual_height: v-bind(actual_height)
}
</style>
