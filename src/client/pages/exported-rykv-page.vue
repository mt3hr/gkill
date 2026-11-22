<template>
    <div>
        <rykvView v-if="is_ready" :app_content_height="app_content_height" :app_content_width="app_content_width"
            :app_title_bar_height="app_title_bar_height" :application_config="application_config"
            :gkill_api="gkill_api" :is_shared_rykv_view="false" :share_title="share_title"
            :application_config_load_failed="false /* 設定も焼き込み済みなので待たない */"
            :is_hosted_in_dialog="false"
            :kyou_change_channel="null /* 単独ファイル。画面間の伝播は無い */"
            :column_state_instance_key="''"
            @received_errors="(...errors: unknown[]) => write_errors(errors[0] as Array<GkillError>)"
            @received_messages="(...messages: unknown[]) => write_messages(messages[0] as Array<GkillMessage>)" />
        <!--
            焼き込みが読めないとき。ガワだけの template.html を直接開いた場合もここに来る。
            出力 HTML は通信をしないので、サーバへ取りに行くフォールバックは置かない
        -->
        <v-container v-else class="exported_rykv_page_empty">
            <v-alert type="info" variant="tonal" :text="i18n.global.t('EXPORTED_RYKV_PAGE_NO_DATA_MESSAGE')" />
        </v-container>
        <GkillMessageFeedView />
    </div>
</template>

<script lang="ts" setup>
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { i18n } from '@/i18n'
import rykvView from './views/rykv-view.vue'
import { useExportedRykvPage } from '@/classes/use-exported-rykv-page'
import GkillMessageFeedView from './views/gkill-message-feed-view.vue'

const {
    // State
    actual_height,
    app_title_bar_height,
    app_content_height,
    app_content_width,
    is_ready,
    share_title,
    application_config,
    gkill_api,

    // Event handlers
    write_errors,
    write_messages,
} = useExportedRykvPage()
</script>
<style scoped>
:root {
    --actual_height: v-bind(actual_height)
}
</style>
