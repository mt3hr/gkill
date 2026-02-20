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
        <AddNewDeviceStructElementView :application_config="application_config" :gkill_api="gkill_api"
          @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
          @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
          @requested_close_dialog="hide"
          @requested_add_device_struct_element="(...device_struct_element: any[]) => emits('requested_add_device_struct_element', device_struct_element[0] as DeviceStructElementData)" />
      </div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { AddNewDeviceStructElementDialogEmits } from './add-new-device-struct-element-dialog-emits'
import type { AddNewDeviceStructElementDialogProps } from './add-new-device-struct-element-dialog-props'
import AddNewDeviceStructElementView from '../views/add-new-device-struct-element-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { DeviceStructElementData } from '@/classes/datas/config/device-struct-element-data'

const add_new_device_struct_element_view = ref<InstanceType<typeof AddNewDeviceStructElementView> | null>(null);

defineProps<AddNewDeviceStructElementDialogProps>()
const emits = defineEmits<AddNewDeviceStructElementDialogEmits>()
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
  add_new_device_struct_element_view.value?.reset_device_name()
  is_show_dialog.value = true
}
async function hide(): Promise<void> {
  is_show_dialog.value = false
  add_new_device_struct_element_view.value?.reset_device_name()
}
</script>
