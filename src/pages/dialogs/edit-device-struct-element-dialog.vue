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
        <EditDeviceStructElementView :application_config="application_config" :gkill_api="gkill_api"
          :struct_obj="device_struct"
          @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
          @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
          @requested_update_device_struct="(...device_struct: any[]) => emits('requested_update_device_struct', device_struct[0] as DeviceStructElementData)"
          @requested_close_dialog="hide" />
      </div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { EditDeviceStructElementDialogEmits } from './edit-device-struct-element-dialog-emits.ts'
import type { EditDeviceStructElementDialogProps } from './edit-device-struct-element-dialog-props.ts'
import EditDeviceStructElementView from '../views/edit-device-struct-element-view.vue'
import type { GkillError } from '@/classes/api/gkill-error.js'
import type { GkillMessage } from '@/classes/api/gkill-message.js'

defineProps<EditDeviceStructElementDialogProps>()
const emits = defineEmits<EditDeviceStructElementDialogEmits>()
defineExpose({ show, hide })

const device_struct: Ref<DeviceStructElementData> = ref(new DeviceStructElementData())
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { DeviceStructElementData } from '@/classes/datas/config/device-struct-element-data.js'
import { i18n } from '@/i18n'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
import { useFloatingDialog } from "@/classes/use-floating-dialog"
const ui = useFloatingDialog("", {
  centerMode: "always",
})


async function show(device_struct_obj: DeviceStructElementData): Promise<void> {
  device_struct.value = device_struct_obj
  is_show_dialog.value = true
}
async function hide(): Promise<void> {
  is_show_dialog.value = false
}
</script>
