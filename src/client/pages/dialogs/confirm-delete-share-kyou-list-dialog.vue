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
        <ConfirmDeleteShareKyousListView v-if="share_kyou_list_info" :application_config="application_config"
          :gkill_api="gkill_api" :share_kyou_list_info="share_kyou_list_info"
          @requested_delete_share_kyou_link_info="(...share_kyou_link_info: any[]) => emits('requested_delete_share_kyou_link_info', share_kyou_link_info[0] as ShareKyousInfo)"
          @requested_close_dialog="hide()"
          @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
          @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" />
      </div>
    </div>
  </Teleport>
</template>

<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { ConfirmDeleteShareKyousLinkDialogEmits } from './confirm-delete-share-kyou-link-dialog-emits'
import type { ConfirmDeleteShareKyousLinkDialogProps } from './confirm-delete-share-kyou-link-dialog-props'
import ConfirmDeleteShareKyousListView from '../views/confirm-delete-share-task-list-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { ShareKyousInfo } from '@/classes/datas/share-kyous-info'

defineProps<ConfirmDeleteShareKyousLinkDialogProps>()
const emits = defineEmits<ConfirmDeleteShareKyousLinkDialogEmits>()
defineExpose({ show, hide })

const share_kyou_list_info: Ref<ShareKyousInfo | null> = ref(null)

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { i18n } from '@/i18n'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
import { useFloatingDialog } from "@/classes/use-floating-dialog"
const ui = useFloatingDialog("confirm-delete-share-kyou-list-dialog", {
  centerMode: "always",
})


async function show(share_kyou_list_info_: ShareKyousInfo): Promise<void> {
  share_kyou_list_info.value = share_kyou_list_info_
  is_show_dialog.value = true
}
async function hide(): Promise<void> {
  is_show_dialog.value = false
}
</script>

