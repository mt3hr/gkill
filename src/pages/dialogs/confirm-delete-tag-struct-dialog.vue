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
        <ConfirmDeleteTagStructView :application_config="application_config" :gkill_api="gkill_api"
          :tag_struct="tag_struct"
          @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
          @requested_close_dialog="hide"
          @requested_delete_tag="(...id: any[]) => { emits('requested_delete_tag', id[0] as string); hide() }"
          @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" />
      </div>
    </div>
  </Teleport>
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import ConfirmDeleteTagStructView from '../views/confirm-delete-tag-struct-view.vue';
import type { ConfirmDeleteTagStructDialogEmits } from './confirm-delete-tag-struct-dialog-emits';
import type { ConfirmDeleteTagStructDialogProps } from './confirm-delete-tag-struct-dialog-props';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';

defineProps<ConfirmDeleteTagStructDialogProps>()
const emits = defineEmits<ConfirmDeleteTagStructDialogEmits>()
defineExpose({ show, hide })

const tag_struct: Ref<TagStructElementData> = ref(new TagStructElementData())

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { TagStructElementData } from '@/classes/datas/config/tag-struct-element-data';
import { i18n } from '@/i18n'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
import { useFloatingDialog } from "@/classes/use-floating-dialog"
const ui = useFloatingDialog("confirm-delete-tag-struct-dialog", {
  centerMode: "always",
})


async function show(tag_struct_obj: TagStructElementData): Promise<void> {
  tag_struct.value = tag_struct_obj
  is_show_dialog.value = true
}
async function hide(): Promise<void> {
  is_show_dialog.value = false
  tag_struct.value = new TagStructElementData()
}
</script>

