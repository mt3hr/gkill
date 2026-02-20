<template>
  <Teleport to="body" v-if="is_show_dialog" >
    <div class="gkill-float-scrim" :class="ui.isTransparent.value ? 'is-transparent' : ''" />

    <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog"
      :class="ui.isTransparent.value ? 'is-transparent' : ''">
      <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
        @touchstart="ui.onHeaderPointerDown">
        <div class="gkill-floating-dialog__title"></div>
        <div class="gkill-floating-dialog__spacer"></div>
        <v-checkbox v-model="ui.isTransparent.value" dark  size="small" variant="flat" 
          :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
                <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details :color="'primary'" variant="flat"> 
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </div>

      <div class="gkill-floating-dialog__body"> 
        <AddNewRepTypeStructElementView :application_config="application_config" :gkill_api="gkill_api"
          @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
          @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
          @requested_close_dialog="hide"
          @requested_add_rep_type_struct_element="(...rep_type_struct_element: any[]) => emits('requested_add_rep_type_struct_element', rep_type_struct_element[0] as RepTypeStructElementData)" />
      </div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { AddNewRepTypeStructElementDialogEmits } from './add-new-rep-type-struct-element-dialog-emits'
import type { AddNewRepTypeStructElementDialogProps } from './add-new-rep-type-struct-element-dialog-props'
import AddNewRepTypeStructElementView from '../views/add-new-rep-type-struct-element-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { RepTypeStructElementData } from '@/classes/datas/config/rep-type-struct-element-data'

const add_new_rep_type_struct_element_view = ref<InstanceType<typeof AddNewRepTypeStructElementView> | null>(null);

defineProps<AddNewRepTypeStructElementDialogProps>()
const emits = defineEmits<AddNewRepTypeStructElementDialogEmits>()
defineExpose({ show, hide })

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { i18n } from '@/i18n'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
import { useFloatingDialog } from "@/classes/use-floating-dialog"
const ui = useFloatingDialog("", {
  centerMode: "always",
})


async function show(): Promise<void> {
  add_new_rep_type_struct_element_view.value?.reset_rep_type_name()
  is_show_dialog.value = true
}
async function hide(): Promise<void> {
  is_show_dialog.value = false
  add_new_rep_type_struct_element_view.value?.reset_rep_type_name()
}
</script>
