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
        <v-card variant="flat" class="pa-2">

        <ConfirmDeleteDnoteListQueryView :application_config="application_config" :gkill_api="gkill_api"
          :dnote_list_query="dnote_list_query"
          @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
          @requested_close_dialog="hide"
          @requested_delete_dnote_list_query="(id: string) => { emits('requested_delete_dnote_list_query', id); hide() }"
          @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)" />
        </v-card>
</div>
    </div>
  </Teleport>
</template>
<script setup lang="ts">
import ConfirmDeleteDnoteListQueryView from '../views/confirm-delete-dnote-list-query-view.vue';
import type { ConfirmDeleteDnoteListQueryDialogEmits } from './confirm-delete-dnote-list-query-dialog-emits';
import type { ConfirmDeleteDnoteListQueryDialogProps } from './confirm-delete-dnote-list-query-dialog-props';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';
import { i18n } from '@/i18n'
import { useConfirmDeleteDnoteListQueryDialog } from '@/classes/use-confirm-delete-dnote-list-query-dialog'

const props = defineProps<ConfirmDeleteDnoteListQueryDialogProps>()
const emits = defineEmits<ConfirmDeleteDnoteListQueryDialogEmits>()
const { dnote_list_query, is_show_dialog, ui, show, hide } = useConfirmDeleteDnoteListQueryDialog({ props, emits })
defineExpose({ show, hide })
</script>

