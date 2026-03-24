<template>
    <div>
        <div class="shared-page-help-button">
            <v-tooltip :text="$t('HELP_TITLE')">
                <template v-slot:activator="{ props }">
                    <v-btn v-bind="props" icon="mdi-help-circle-outline" size="small" variant="text"
                        @click="open_help" />
                </template>
            </v-tooltip>
        </div>
        <div class="overlay_target">
            <v-overlay v-model="is_loading" class="align-center justify-center" persistent contained>
                <v-progress-circular indeterminate color="primary" />
            </v-overlay>
        </div>
        <SharedMiPage
            v-if="!is_loading && view_type === 'mi' && application_config && gkill_api_for_share && share_title"
            :gkill_api="gkill_api_for_share" :application_config="application_config" :share_title="share_title"
            :share_id="share_id" />
        <SharedRYKVPage
            v-if="!is_loading && view_type === 'rykv' && application_config && gkill_api_for_share && share_title"
            :gkill_api="gkill_api_for_share" :application_config="application_config" :share_title="share_title"
            :share_id="share_id" />
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
import SharedMiPage from './shared-mi-page.vue'
import SharedRYKVPage from './shared-rykv-page.vue'
import { useSharedPage } from '@/classes/use-shared-page'
import { useTheme } from 'vuetify'
import { i18n } from '@/i18n'

const theme = useTheme()

const {
    // State
    share_id,
    view_type,
    share_title,
    gkill_api_for_share,
    application_config,
    is_loading,
    messages,

    // Event handlers
    close_message,
} = useSharedPage()

function open_help(): void {
    const locale = i18n.global.locale || 'ja'
    const isDark = theme.global.name.value === 'gkill_dark_theme'
    const url = `/resources/manual/${locale}/shared-page.html${isDark ? '?theme=dark' : ''}`
    window.open(url, '_blank')
}
</script>
<style lang="css" scoped>
.shared-page-help-button {
    position: fixed;
    top: 8px;
    right: 8px;
    z-index: 10;
}

.overlay_target {
    z-index: -10000;
    position: absolute;
    min-height: calc(100vh);
    min-width: v-bind("is_loading ? 'calc(100vw)' : '0px'");
}
</style>
