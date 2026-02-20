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
        <ConfirmDeleteDnoteListQueryView :application_config="application_config" :gkill_api="gkill_api"
          :dnote_list_query="dnote_list_query"
          @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
          @requested_close_dialog="hide"
          @requested_delete_dnote_list_query="(...id: any[]) => { emits('requested_delete_dnote_list_query', id[0] as string); hide() }"
          @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" />
      </div>
    </div>
  </Teleport>
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import DnoteListQuery from '../views/dnote-list-query';
import ConfirmDeleteDnoteListQueryView from '../views/confirm-delete-dnote-list-query-view.vue';
import type { ConfirmDeleteDnoteListQueryDialogEmits } from './confirm-delete-dnote-list-query-dialog-emits';
import type { ConfirmDeleteDnoteListQueryDialogProps } from './confirm-delete-dnote-list-query-dialog-props';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';

defineProps<ConfirmDeleteDnoteListQueryDialogProps>()
const emits = defineEmits<ConfirmDeleteDnoteListQueryDialogEmits>()
defineExpose({ show, hide })

const dnote_list_query: Ref<DnoteListQuery> = ref(new DnoteListQuery())

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { i18n } from '@/i18n'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
import { useFloatingDialog } from "@/classes/use-floating-dialog"
const ui = useFloatingDialog("", {
  centerMode: "always",
})


async function show(_dnote_item: DnoteListQuery): Promise<void> {
  dnote_list_query.value = _dnote_item
  is_show_dialog.value = true
}
async function hide(): Promise<void> {
  is_show_dialog.value = false
  dnote_list_query.value = new DnoteListQuery()
}
</script>
