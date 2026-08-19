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

        <ConfirmDeleteDnoteItemListView :application_config="application_config" :gkill_api="gkill_api"
          :dnote_item="dnote_item"
          @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
          @requested_close_dialog="hide"
          @requested_delete_dnote_item_list="(id: string) => { emits('requested_delete_dnote_list_item', id); hide() }"
          @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)" />
        </v-card>
</div>
    </div>
  </Teleport>
</template>
<script setup lang="ts">
import ConfirmDeleteDnoteItemListView from '../views/confirm-delete-dnote-item-list-view.vue';
import type { ConfirmDeleteDnoteItemListDialogEmits } from './confirm-delete-dnote-item-list-dialog-emits';
import type { ConfirmDeleteDnoteItemListDialogProps } from './confirm-delete-dnote-item-list-dialog-props';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';
import { i18n } from '@/i18n'
import { useConfirmDeleteDnoteItemListDialog } from '@/classes/use-confirm-delete-dnote-item-list-dialog'

const props = defineProps<ConfirmDeleteDnoteItemListDialogProps>()
const emits = defineEmits<ConfirmDeleteDnoteItemListDialogEmits>()
const { dnote_item, is_show_dialog, ui, show, hide } = useConfirmDeleteDnoteItemListDialog({ props, emits })
defineExpose({ show, hide })
</script>

