<template>
  <Teleport to="body" v-if="is_show_dialog">
    <div class="gkill-float-scrim" :class="ui.isTransparent.value ? 'is-transparent' : ''" />

    <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog"
      :class="ui.isTransparent.value ? 'is-transparent' : ''">
      <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
        @touchstart="ui.onHeaderPointerDown">
        <div class="gkill-floating-dialog__title"></div>
        <div class="gkill-floating-dialog__spacer"></div>
        <v-checkbox v-model="ui.isTransparent.value" color="white" size="small" variant="flat"
          :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
        <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details :color="'primary'"
          variant="flat">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </div>

      <div class="gkill-floating-dialog__body">
        <v-card variant="flat" class="pa-2">
          <ConfirmDeleteMiBoardStructView :application_config="application_config" :gkill_api="gkill_api"
            :mi_board_struct="mi_board_struct"
            @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
            @requested_close_dialog="hide"
            @requested_delete_mi_board="(id: string) => { emits('requested_delete_mi_board', id); hide() }"
            @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)" />
        </v-card>
      </div>
    </div>
  </Teleport>
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import ConfirmDeleteMiBoardStructView from '../views/confirm-delete-mi-board-struct-view.vue';
import type { ConfirmDeleteMiBoardStructDialogEmits } from './confirm-delete-mi-board-struct-dialog-emits';
import type { ConfirmDeleteMiBoardStructDialogProps } from './confirm-delete-mi-board-struct-dialog-props';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { MiBoardStructElementData } from '@/classes/datas/config/mi-board-struct-element-data';
import { i18n } from '@/i18n'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

defineProps<ConfirmDeleteMiBoardStructDialogProps>()
const emits = defineEmits<ConfirmDeleteMiBoardStructDialogEmits>()
defineExpose({ show, hide })

const mi_board_struct: Ref<MiBoardStructElementData> = ref(new MiBoardStructElementData())

const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
const ui = useFloatingDialog("confirm-delete-mi-board-struct-dialog", {
  centerMode: "always",
  onEscape: () => hide(),
})

async function show(mi_board_struct_obj: MiBoardStructElementData): Promise<void> {
  mi_board_struct.value = mi_board_struct_obj
  is_show_dialog.value = true
}
async function hide(): Promise<void> {
  close_dialog_via_history(is_show_dialog)
  mi_board_struct.value = new MiBoardStructElementData()
}
</script>
