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
        <GkillMessageFeedView />
    </div>
</template>

<script lang="ts" setup>
import SharedMiPage from './shared-mi-page.vue'
import SharedRYKVPage from './shared-rykv-page.vue'
import { useSharedPage } from '@/classes/use-shared-page'
import { useTheme } from 'vuetify'
import { i18n } from '@/i18n'
import GkillMessageFeedView from './views/gkill-message-feed-view.vue'

const theme = useTheme()

const {
    // State
    share_id,
    view_type,
    share_title,
    gkill_api_for_share,
    application_config,
    is_loading,

    // Event handlers
} = useSharedPage()

function open_help(): void {
    const locale = i18n.global.locale || 'ja'
    const is_dark = theme.global.name.value === 'gkill_dark_theme'
    const url = `/resources/manual/${locale}/shared-page.html${is_dark ? '?theme=dark' : ''}`
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
