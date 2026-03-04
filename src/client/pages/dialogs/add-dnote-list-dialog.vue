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
        <AddDnoteListView :application_config="application_config" :gkill_api="gkill_api"
          @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
          @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
          @requested_add_dnote_list_query="(...dnote_list_query: any[]) => emits('requested_add_dnote_list_query', dnote_list_query[0] as DnoteListQuery)"
          @requested_close_dialog="hide()" />


      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import AddDnoteListView from '@/pages/views/add-dnote-list-view.vue';
import { ref, type Ref } from 'vue'
import type { GkillError } from '../../classes/api/gkill-error';
import type { GkillMessage } from '../../classes/api/gkill-message';
import type AddDnoteListDialogEmits from './add-dnote-list-dialog-emits';
import type AddDnoteListDialogProps from './add-dnote-list-dialog-props';
import type DnoteListQuery from '../views/dnote-list-query';
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { i18n } from '@/i18n'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
import { useFloatingDialog } from "@/classes/use-floating-dialog"
const ui = useFloatingDialog("add-dnote-list-dialog", {
  centerMode: "always",
})


defineExpose({ show, hide })
defineProps<AddDnoteListDialogProps>()
const emits = defineEmits<AddDnoteListDialogEmits>()

async function show(): Promise<void> {
  is_show_dialog.value = true
}
async function hide(): Promise<void> {
  is_show_dialog.value = false
}
</script>

