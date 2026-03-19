<template>
  <Teleport to="body" v-if="is_show_dialog" >
    <div class="gkill-float-scrim" :class="ui.isTransparent.value ? 'is-transparent' : ''" />

    <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog"
      :class="ui.isTransparent.value ? 'is-transparent' : ''">
      <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
        @touchstart="ui.onHeaderPointerDown">
        <div class="gkill-floating-dialog__title"></div>
        <div class="gkill-floating-dialog__spacer"></div>
  <v-checkbox v-model="ui.isTransparent.value" color="white"    size="small" variant="flat"
          :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
        <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="help_dialog?.show()" hide-details :color="'primary'" variant="flat">
          <v-icon>mdi-help-circle-outline</v-icon>
        </v-btn>
                <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details :color="'primary'" variant="flat">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </div>

      <div class="gkill-floating-dialog__body">
        <v-card class="pa-2">
       <ServerConfigView v-show="server_configs.length !== 0" :application_config="application_config"
          :gkill_api="gkill_api" :server_configs="server_configs"
          @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
          @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
          @requested_reload_server_config="load_server_configs()" @requested_close_dialog="hide" />
        </v-card>
        <HelpDialog screen_name="server-config" ref="help_dialog" />
</div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import { ref } from 'vue'
import type { ServerConfigDialogEmits } from './server-config-dialog-emits'
import type { ServerConfigDialogProps } from './server-config-dialog-props'
import ServerConfigView from '../views/server-config-view.vue'
import HelpDialog from './help-dialog.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { useServerConfigDialog } from '@/classes/use-server-config-dialog'
import { i18n } from '@/i18n'

const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)

const props = defineProps<ServerConfigDialogProps>()
const emits = defineEmits<ServerConfigDialogEmits>()

const { is_show_dialog, ui, server_configs, show, hide, load_server_configs } = useServerConfigDialog({ props, emits })

defineExpose({ show, hide })
</script>

