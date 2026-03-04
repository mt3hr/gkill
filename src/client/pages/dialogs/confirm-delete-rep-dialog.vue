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
        <ConfirmDeleteRepView :application_config="application_config" :gkill_api="gkill_api"
          :repository="cloned_repository" :server_configs="server_configs"
          @requested_delete_rep="(...rep: any[]) => emits('requested_delete_rep', rep[0] as Repository)"
          @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
          @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
          @requested_close_dialog="hide()" />
      </div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { ConfirmDeleteRepDialogEmits } from './confirm-delete-rep-dialog-emits'
import type { ConfirmDeleteRepDialogProps } from './confirm-delete-rep-dialog-props'
import ConfirmDeleteRepView from '../views/confirm-delete-rep-view.vue'
import { Repository } from '@/classes/datas/config/repository';
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

defineProps<ConfirmDeleteRepDialogProps>()
const emits = defineEmits<ConfirmDeleteRepDialogEmits>()
defineExpose({ show, hide })

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { i18n } from '@/i18n'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
import { useFloatingDialog } from "@/classes/use-floating-dialog"
const ui = useFloatingDialog("confirm-delete-rep-dialog", {
  centerMode: "always",
})

const cloned_repository: Ref<Repository> = ref(new Repository())

async function show(repository: Repository): Promise<void> {
  cloned_repository.value = repository
  is_show_dialog.value = true
}
async function hide(): Promise<void> {
  is_show_dialog.value = false
  cloned_repository.value = new Repository()
}
</script>

