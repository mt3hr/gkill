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
          <ConfirmDeleteDnoteCorrelationGraphView :application_config="application_config" :gkill_api="gkill_api"
            :dnote_correlation_graph_query="dnote_correlation_graph_query"
            @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
            @requested_close_dialog="hide"
            @requested_delete_dnote_correlation_graph="(id: string) => { emits('requested_delete_dnote_correlation_graph', id); hide() }"
            @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)" />
        </v-card>
      </div>
    </div>
  </Teleport>
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import { DnoteCorrelationGraphQuery } from '../../classes/dnote/dnote-correlation';
import ConfirmDeleteDnoteCorrelationGraphView from '../views/confirm-delete-dnote-correlation-graph-view.vue';
import type { ConfirmDeleteDnoteCorrelationGraphDialogEmits } from './confirm-delete-dnote-correlation-graph-dialog-emits';
import type { ConfirmDeleteDnoteCorrelationGraphDialogProps } from './confirm-delete-dnote-correlation-graph-dialog-props';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';

defineProps<ConfirmDeleteDnoteCorrelationGraphDialogProps>()
const emits = defineEmits<ConfirmDeleteDnoteCorrelationGraphDialogEmits>()
defineExpose({ show, hide })

const dnote_correlation_graph_query: Ref<DnoteCorrelationGraphQuery> = ref(new DnoteCorrelationGraphQuery())

import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { i18n } from '@/i18n'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
import { useFloatingDialog } from "@/classes/use-floating-dialog"
const ui = useFloatingDialog("confirm-delete-dnote-correlation-graph-dialog", {
  centerMode: "always",
  onEscape: () => hide(),
})

async function show(target: DnoteCorrelationGraphQuery): Promise<void> {
  dnote_correlation_graph_query.value = target
  is_show_dialog.value = true
}
async function hide(): Promise<void> {
  close_dialog_via_history(is_show_dialog)
}
</script>
