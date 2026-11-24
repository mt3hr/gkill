<template>
    <Teleport to="body" v-if="is_show_dialog">
        <div class="gkill-float-scrim" :class="ui.isTransparent.value ? 'is-transparent' : ''" />

        <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog"
            :class="ui.isTransparent.value ? 'is-transparent' : ''">
            <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
                @touchstart="ui.onHeaderPointerDown">
                <div class="gkill-floating-dialog__title"></div>
                <div class="gkill-floating-dialog__spacer"></div>
                <v-checkbox v-model="ui.isTransparent.value" color="white" size="small" variant="flat"
                    :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
                <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details
                    :color="'primary'" variant="flat">
                    <v-icon>mdi-close</v-icon>
                </v-btn>
            </div>

            <div class="gkill-floating-dialog__body">
                <v-card variant="flat" class="pa-2">
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
                             保存はフォーム送信ではなくpostMessageで親に依頼する（use-plugin-config-dialog.ts の onWindowMessage）。 -->
                        <iframe
                            v-else-if="html"
                            ref="iframe_ref"
                            :srcdoc="html"
                            sandbox="allow-scripts allow-forms"
                            class="plugin-config-iframe"
                            @load="send_theme_to_iframe"
                        />
                    </v-card-text>
                </v-card>
            </div>
        </div>
    </Teleport>
</template>

<script setup lang="ts">
import { i18n } from '@/i18n'
import type { PluginConfigDialogProps } from './plugin-config-dialog-props'
import { usePluginConfigDialog } from '@/classes/use-plugin-config-dialog'

const props = defineProps<PluginConfigDialogProps>()
const {
    html,
    is_loading,
    error_message,
    iframe_ref,
    rep_name,
    is_show_dialog,
    ui,
    show,
    hide,
    send_theme_to_iframe,
} = usePluginConfigDialog({ props })
defineExpose({ show, hide })
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
