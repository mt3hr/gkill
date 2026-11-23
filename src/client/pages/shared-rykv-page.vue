<template>
    <div>
        <SharedRykvView :app_content_height="app_content_height" :app_content_width="app_content_width"
            :app_title_bar_height="app_title_bar_height" :application_config="application_config"
            :gkill_api="gkill_api" :share_title="share_title"
            @received_errors="(...errors: unknown[]) => write_errors(errors[0] as Array<GkillError>)"
            @received_messages="(...messages: unknown[]) => write_messages(messages[0] as Array<GkillMessage>)" />
        <GkillMessageFeedView />
    </div>
</template>

<script lang="ts" setup>
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import SharedRykvView from './views/shared-rykv-view.vue'
import type { KyouViewEmits } from './views/kyou-view-emits'
import type { SharedRYKVPageProps } from './shared-rykv-page-props'
import { useSharedRykvPage } from '@/classes/use-shared-rykv-page'
import GkillMessageFeedView from './views/gkill-message-feed-view.vue'

const props = defineProps<SharedRYKVPageProps>()
defineEmits<KyouViewEmits>()

const {
    // State
    actual_height,
    app_title_bar_height,
    app_content_height,
    app_content_width,

    // Event handlers
    write_errors,
    write_messages,
} = useSharedRykvPage({ props })
</script>
<style scoped>
:root {
    --actual_height: v-bind(actual_height)
}
</style>
