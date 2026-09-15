<template>
    <div>
        <sharedMiTaskView :app_content_height="app_content_height" :app_content_width="app_content_width"
            :app_title_bar_height="app_title_bar_height" :share_id="share_kyou_id" :share_title="share_title"
            :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(...errors: unknown[]) => write_errors(errors[0] as Array<GkillError>)"
            @received_messages="(...messages: unknown[]) => write_messages(messages[0] as Array<GkillMessage>)" />
        <GkillMessageFeedView />
    </div>
</template>

<script lang="ts" setup>
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import sharedMiTaskView from './views/shared-mi-view.vue'
import type { SharedMiPageProps } from './shared-mi-page-props'
import { useSharedMiPage } from '@/classes/use-shared-mi-page'
import GkillMessageFeedView from './views/gkill-message-feed-view.vue'

const props = defineProps<SharedMiPageProps>()

const {
    // State
    actual_height,
    app_title_bar_height,
    app_content_height,
    app_content_width,
    share_kyou_id,

    // Event handlers
    write_errors,
    write_messages,
} = useSharedMiPage({ props })
</script>
<style scoped>
:root {
    --actual_height: v-bind(actual_height)
}
</style>
