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
        <ShareKyousListView :application_config="application_config" :gkill_api="gkill_api"
          :find_kyou_query="find_kyou_query"
          @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
          @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
          @regestered_share_kyou_list_info="(...share_kyou_info: any[]) => {
            emits('regestered_share_kyou_list_info', share_kyou_info[0] as ShareKyousInfo)
            emits('requested_show_share_kyou_link_dialog', share_kyou_info[0] as ShareKyousInfo)
          }" @requested_close_dialog="hide()" />
      </div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import type { ShareKyousListDialogEmits } from './share-kyou-list-dialog-emits'
import type { ShareKyousListDialogProps } from './share-kyou-list-dialog-props'
import ShareKyousListView from '../views/share-kyou-view.vue'
import { ref, type Ref } from 'vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { ShareKyousInfo } from '@/classes/datas/share-kyous-info'

defineProps<ShareKyousListDialogProps>()
const emits = defineEmits<ShareKyousListDialogEmits>()
defineExpose({ show, hide })

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { i18n } from '@/i18n'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
import { useFloatingDialog } from "@/classes/use-floating-dialog"
const ui = useFloatingDialog("", {
  centerMode: "always",
})


async function show(): Promise<void> {
  is_show_dialog.value = true
}
async function hide(): Promise<void> {
  is_show_dialog.value = false
}
</script>
