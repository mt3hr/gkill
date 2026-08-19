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
        <v-card variant="flat" class="pa-2">

        <ApplicationConfigView :application_config="application_config" :gkill_api="gkill_api"
          :app_content_height="app_content_height" :app_content_width="app_content_width"
          @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
          @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
          @requested_close_dialog="() => hide()" ref="application_config_view" />
        </v-card>
        <HelpDialog screen_name="application-config" ref="help_dialog" />
</div>
    </div>
  </Teleport>
</template>

<script lang="ts" setup>
import type { ApplicationConfigDialogProps } from './application-config-dialog-props'
import type { ApplicationConfigDialogEmits } from './application-config-dialog-emits'
import ApplicationConfigView from '../views/application-config-view.vue'
import HelpDialog from './help-dialog.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import { GkillMessage } from '@/classes/api/gkill-message'
import { i18n } from '@/i18n'
import { useApplicationConfigDialog } from '@/classes/use-application-config-dialog'

const props = defineProps<ApplicationConfigDialogProps>()
const emits = defineEmits<ApplicationConfigDialogEmits>()
const { application_config_view, help_dialog, is_show_dialog, ui, show, hide } = useApplicationConfigDialog({ props, emits })
defineExpose({ show, hide })
</script>

