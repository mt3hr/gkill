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
                <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details :color="'primary'" variant="flat"> 
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </div>

      <div class="gkill-floating-dialog__body">
        <v-card variant="flat" class="pa-2">

        <ConfirmGenerateTLSFilesView :application_config="application_config" :gkill_api="gkill_api"
          :server_config="server_config"
          @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
          @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
          @generated_tls_files="emits('generated_tls_files')" @requested_close_dialog="hide" />
        </v-card>
</div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import type { ConfirmGenerateTLSFilesDialogEmits } from './confirm-generate-tls-files-dialog-emits'
import type { ConfirmGenerateTLSFilesDialogProps } from './confirm-generate-tls-files-dialog-props'
import ConfirmGenerateTLSFilesView from '../views/confirm-generate-tls-files-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { i18n } from '@/i18n'
import { useConfirmGenerateTLSFilesDialog } from '@/classes/use-confirm-generate-tls-files-dialog'

const props = defineProps<ConfirmGenerateTLSFilesDialogProps>()
const emits = defineEmits<ConfirmGenerateTLSFilesDialogEmits>()
const { is_show_dialog, ui, show, hide } = useConfirmGenerateTLSFilesDialog({ props, emits })
defineExpose({ show, hide })
</script>

