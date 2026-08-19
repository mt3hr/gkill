<template>
    <v-dialog v-model="show" max-width="800">
        <v-card variant="flat">
            <v-card-title>{{ rep_name }} {{ i18n.global.t('TOOLTIP_SETTINGS') }}</v-card-title>
            <v-card-text>
                <div v-if="is_loading" class="d-flex justify-center pa-4">
                    <v-progress-circular indeterminate />
                </div>
                <div v-else-if="error_message" class="plugin-error">
                    {{ error_message }}
                </div>
                <!-- プラグインの設定フォームHTMLをiframeで表示。
                     allow-same-originを付けないことでセッションcookieを隔離。
                     保存はフォーム送信ではなくpostMessageで親に依頼する（下の onWindowMessage）。 -->
                <iframe
                    v-else-if="html"
                    ref="iframe_ref"
                    :srcdoc="html"
                    sandbox="allow-scripts allow-forms"
                    class="plugin-config-iframe"
                    @load="send_theme_to_iframe"
                />
            </v-card-text>
            <v-card-actions class="gkill-dialog-actions">
                <v-spacer />
                <v-btn @click="show = false">{{ i18n.global.t('CLOSE_TITLE') }}</v-btn>
            </v-card-actions>
        </v-card>
    </v-dialog>
</template>

<script setup lang="ts">
import { i18n } from '@/i18n'
import type { PluginConfigDialogProps } from './plugin-config-dialog-props'
import { usePluginConfigDialog } from '@/classes/use-plugin-config-dialog'

const props = defineProps<PluginConfigDialogProps>()
const show = defineModel<boolean>('show', { default: false })
const { html, is_loading, error_message, iframe_ref, send_theme_to_iframe } = usePluginConfigDialog({ props, show })
</script>

<style>
.plugin-config-iframe {
    width: 100%;
    height: 500px;
    border: none;
    display: block;
}
.plugin-error {
    color: red;
    padding: 8px;
}
</style>
