<template>
    <div>
        <rykvView :app_content_height="app_content_height" :app_content_width="app_content_width"
            :app_title_bar_height="app_title_bar_height" :application_config="application_config" :gkill_api="gkill_api"
            :is_shared_rykv_view="true" :share_title="share_title"
            @received_errors="(...errors: any[]) => write_errors(errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => write_messages(messages[0] as Array<GkillMessage>)" />
        <div class="alert_container">
            <v-slide-y-transition group>
                <v-tooltip :text="(message.is_error ? 'エラーコード' : 'メッセージコード') + ':' + message.code"
                    v-for="message in messages" :key="message.id">
                    <template v-slot:activator="{ props }">
                        <v-alert v-bind="props" :color="message.is_error ? 'error' : undefined"
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
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import rykvView from './views/rykv-view.vue'
import type { KyouViewEmits } from './views/kyou-view-emits'
import type { SharedRYKVPageProps } from './shared-rykv-page-props'
import { useSharedRykvPage } from '@/classes/use-shared-rykv-page'

const props = defineProps<SharedRYKVPageProps>()
defineEmits<KyouViewEmits>()

const {
    // State
    actual_height,
    app_title_bar_height,
    app_content_height,
    app_content_width,
    messages,

    // Event handlers
    write_errors,
    write_messages,
    close_message,
} = useSharedRykvPage({ props })
</script>
<style scoped>
:root {
    --actual_height: v-bind(actual_height)
}
</style>
