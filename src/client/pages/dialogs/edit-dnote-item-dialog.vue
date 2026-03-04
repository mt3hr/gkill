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
        <EditDnoteItemView :application_config="application_config" :gkill_api="gkill_api" v-model="model_value"
          @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
          @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
          @requested_update_dnote_item="(...dnote_item: any[]) => emits('requested_update_dnote_item', dnote_item[0] as DnoteItem)"
          @requested_close_dialog="hide" />
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import EditDnoteItemView from '@/pages/views/edit-dnote-item-view.vue';
import { ref, type Ref } from 'vue'
import type { GkillError } from '../../classes/api/gkill-error';
import type { GkillMessage } from '../../classes/api/gkill-message';
import type EditDnoteItemDialogEmits from './edit-dnote-item-dialog-emits';
import type EditDnoteItemDialogProps from './edit-dnote-item-dialog-props';
import DnoteItem from '@/classes/dnote/dnote-item';
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { i18n } from '@/i18n'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
import { useFloatingDialog } from "@/classes/use-floating-dialog"
const ui = useFloatingDialog("edit-dnote-item-dialog", {
  centerMode: "always",
})


defineExpose({ show, hide })
defineProps<EditDnoteItemDialogProps>()
const model_value = defineModel<DnoteItem>()
const emits = defineEmits<EditDnoteItemDialogEmits>()

async function show(): Promise<void> {
  is_show_dialog.value = true
}
async function hide(): Promise<void> {
  is_show_dialog.value = false
}
</script>

