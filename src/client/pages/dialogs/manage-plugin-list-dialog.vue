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
                <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="help_dialog?.show()" hide-details
                    :color="'primary'" variant="flat">
                    <v-icon>mdi-help-circle-outline</v-icon>
                </v-btn>
                <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details
                    :color="'primary'" variant="flat">
                    <v-icon>mdi-close</v-icon>
                </v-btn>
            </div>

            <div class="gkill-floating-dialog__body">
                <v-card variant="flat" class="pa-2">
                    <ManagePluginListView :application_config="application_config" :gkill_api="gkill_api"
                        :target_user_id="cloned_account.user_id" :plugins="plugins" :is_loading="is_loading"
                        @requested_show_plugin_config="(plugin: PluginInfo) => show_plugin_config_dialog(plugin)" />
                    <PluginConfigDialog :application_config="application_config" :gkill_api="gkill_api"
                        ref="plugin_config_dialog" />
                </v-card>
                <HelpDialog screen_name="plugin" ref="help_dialog" />
            </div>
        </div>
    </Teleport>
</template>
<script lang="ts" setup>
import type { ManagePluginListDialogEmits } from './manage-plugin-list-dialog-emits'
import type { ManagePluginListDialogProps } from './manage-plugin-list-dialog-props'
import ManagePluginListView from '../views/manage-plugin-list-view.vue'
import PluginConfigDialog from './plugin-config-dialog.vue'
import HelpDialog from './help-dialog.vue'
import type { PluginInfo } from '@/classes/api/req_res/get-plugin-list-response'
import { i18n } from '@/i18n'
import { useManagePluginListDialog } from '@/classes/use-manage-plugin-list-dialog'

const props = defineProps<ManagePluginListDialogProps>()
const emits = defineEmits<ManagePluginListDialogEmits>()
const {
    plugin_config_dialog,
    help_dialog,
    cloned_account,
    plugins,
    is_loading,
    is_show_dialog,
    ui,
    show,
    hide,
    show_plugin_config_dialog,
} = useManagePluginListDialog({ props, emits })
defineExpose({ show, hide })
</script>
