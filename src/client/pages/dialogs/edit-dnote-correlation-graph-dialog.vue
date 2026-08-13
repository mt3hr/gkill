<template>
  <Teleport to="body" v-if="is_show_dialog">
    <div class="gkill-float-scrim" :class="ui.isTransparent.value ? 'is-transparent' : ''" />

    <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog"
      :class="ui.isTransparent.value ? 'is-transparent' : ''">
      <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
        @touchstart="ui.onHeaderPointerDown">
        <div class="gkill-floating-dialog__title">{{ i18n.global.t('EDIT_DNOTE_CORRELATION_GRAPH_TITLE') }}</div>
        <div class="gkill-floating-dialog__spacer"></div>
        <v-checkbox v-model="ui.isTransparent.value" color="white" size="small" variant="flat"
          :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
        <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="help_dialog?.show()" hide-details
          :color="'primary'" variant="flat">
          <v-icon>mdi-help-circle-outline</v-icon>
        </v-btn>
        <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details :color="'primary'"
          variant="flat">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </div>

      <div class="gkill-floating-dialog__body">
        <v-card variant="flat" class="pa-2">
          <EditDnoteCorrelationGraphView :application_config="application_config" :gkill_api="gkill_api"
            :dnote_correlation_graph_query="dnote_correlation_graph_query"
            @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
            @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
            @requested_update_dnote_correlation_graph="(dnote_correlation_graph_query: DnoteCorrelationGraphQuery) => emits('requested_update_dnote_correlation_graph', dnote_correlation_graph_query)"
            @requested_close_dialog="hide()" />
        </v-card>
        <HelpDialog screen_name="dnote" ref="help_dialog" />
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import EditDnoteCorrelationGraphView from '@/pages/views/edit-dnote-correlation-graph-view.vue';
import HelpDialog from './help-dialog.vue'
import { ref, type Ref } from 'vue'
import type { GkillError } from '../../classes/api/gkill-error';
import type { GkillMessage } from '../../classes/api/gkill-message';
import type { DnoteCorrelationGraphQuery } from '../../classes/dnote/dnote-correlation';
import type EditDnoteCorrelationGraphDialogEmits from './edit-dnote-correlation-graph-dialog-emits';
import type EditDnoteCorrelationGraphDialogProps from './edit-dnote-correlation-graph-dialog-props';
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { i18n } from '@/i18n'

const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
import { useFloatingDialog } from "@/classes/use-floating-dialog"
const ui = useFloatingDialog("edit-dnote-correlation-graph-dialog", {
  centerMode: "always",
  onEscape: () => hide(),
})

const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)

defineExpose({ show, hide })
defineProps<EditDnoteCorrelationGraphDialogProps>()
const emits = defineEmits<EditDnoteCorrelationGraphDialogEmits>()

async function show(): Promise<void> {
  is_show_dialog.value = true
}
async function hide(): Promise<void> {
  close_dialog_via_history(is_show_dialog)
}
</script>
