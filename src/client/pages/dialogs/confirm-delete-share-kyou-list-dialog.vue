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

        <ConfirmDeleteShareKyousListView v-if="share_kyou_list_info" :application_config="application_config"
          :gkill_api="gkill_api" :share_kyou_list_info="share_kyou_list_info"
          @requested_delete_share_kyou_link_info="(share_kyou_link_info: ShareKyousInfo) => emits('requested_delete_share_kyou_link_info', share_kyou_link_info)"
          @requested_close_dialog="hide()"
          @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
          @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)" />
        </v-card>
</div>
    </div>
  </Teleport>
</template>

<script lang="ts" setup>
import type { ConfirmDeleteShareKyousLinkDialogEmits } from './confirm-delete-share-kyou-link-dialog-emits'
import type { ConfirmDeleteShareKyousLinkDialogProps } from './confirm-delete-share-kyou-link-dialog-props'
import ConfirmDeleteShareKyousListView from '../views/confirm-delete-share-task-list-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { ShareKyousInfo } from '@/classes/datas/share-kyous-info'
import { i18n } from '@/i18n'
import { useConfirmDeleteShareKyouListDialog } from '@/classes/use-confirm-delete-share-kyou-list-dialog'

const props = defineProps<ConfirmDeleteShareKyousLinkDialogProps>()
const emits = defineEmits<ConfirmDeleteShareKyousLinkDialogEmits>()
const { share_kyou_list_info, is_show_dialog, ui, show, hide } = useConfirmDeleteShareKyouListDialog({ props, emits })
defineExpose({ show, hide })
</script>

