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
        <ConfirmDeleteRepTypeStructView :application_config="application_config" :gkill_api="gkill_api"
          :rep_type_struct="rep_type_struct"
          @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
          @requested_close_dialog="hide"
          @requested_delete_rep_type="(...id: any[]) => { emits('requested_delete_rep_type', id[0] as string); hide() }"
          @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" />
      </div>
    </div>
  </Teleport>
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import ConfirmDeleteRepTypeStructView from '../views/confirm-delete-rep-type-struct-view.vue';
import type { ConfirmDeleteRepTypeStructDialogEmits } from './confirm-delete-rep-type-struct-dialog-emits';
import type { ConfirmDeleteRepTypeStructDialogProps } from './confirm-delete-rep-type-struct-dialog-props';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';

defineProps<ConfirmDeleteRepTypeStructDialogProps>()
const emits = defineEmits<ConfirmDeleteRepTypeStructDialogEmits>()
defineExpose({ show, hide })

const rep_type_struct: Ref<RepTypeStructElementData> = ref(new RepTypeStructElementData())

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { RepTypeStructElementData } from '@/classes/datas/config/rep-type-struct-element-data';
import { i18n } from '@/i18n'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
import { useFloatingDialog } from "@/classes/use-floating-dialog"
const ui = useFloatingDialog("confirm-delete-rep-type-struct-dialog", {
  centerMode: "always",
})


async function show(rep_type_struct_obj: RepTypeStructElementData): Promise<void> {
  rep_type_struct.value = rep_type_struct_obj
  is_show_dialog.value = true
}
async function hide(): Promise<void> {
  is_show_dialog.value = false
  rep_type_struct.value = new RepTypeStructElementData()
}
</script>

