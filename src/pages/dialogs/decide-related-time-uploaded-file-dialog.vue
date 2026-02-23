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
        <DecideRelatedTimeUploadedFileView :application_config="application_config" :gkill_api="gkill_api"
          :app_content_height="app_content_height" :app_content_width="app_content_width"
          :uploaded_kyous="uploaded_kyous"
          @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
          :last_added_tag="last_added_tag"
          @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
          @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
          @requested_reload_list="emits('requested_reload_list')"
          @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)"
          @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0] as Kyou)"
          @requested_open_rykv_dialog="(...params: any[]) => emits('requested_open_rykv_dialog', params[0], params[1], params[2])" />
      </div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { DecideRelatedTimeUploadedFileDialogEmits } from './decide-related-time-uploaded-file-dialog-emits'
import type { DecideRelatedTimeUploadedFileDialogProps } from './decide-related-time-uploaded-file-dialog-props'
import DecideRelatedTimeUploadedFileView from '../views/decide-related-time-uploaded-file-view.vue'
import type { Kyou } from '@/classes/datas/kyou'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

defineProps<DecideRelatedTimeUploadedFileDialogProps>()
const emits = defineEmits<DecideRelatedTimeUploadedFileDialogEmits>()
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
