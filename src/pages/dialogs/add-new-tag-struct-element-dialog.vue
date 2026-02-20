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
        <AddNewTagStructElementView :application_config="application_config" :gkill_api="gkill_api"
          @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
          @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
          @requested_close_dialog="hide"
          @requested_add_tag_struct_element="(...tag_struct_element: any[]) => emits('requested_add_tag_struct_element', tag_struct_element[0] as TagStructElementData)" />
      </div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { AddNewTagStructElementDialogEmits } from './add-new-tag-struct-element-dialog-emits'
import type { AddNewTagStructElementDialogProps } from './add-new-tag-struct-element-dialog-props'
import AddNewTagStructElementView from '../views/add-new-tag-struct-element-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { TagStructElementData } from '@/classes/datas/config/tag-struct-element-data'

const add_new_tag_struct_element_view = ref<InstanceType<typeof AddNewTagStructElementView> | null>(null);

defineProps<AddNewTagStructElementDialogProps>()
const emits = defineEmits<AddNewTagStructElementDialogEmits>()
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
  add_new_tag_struct_element_view.value?.reset_tag_name()
  is_show_dialog.value = true
}
async function hide(): Promise<void> {
  is_show_dialog.value = false
  add_new_tag_struct_element_view.value?.reset_tag_name()
}
</script>
