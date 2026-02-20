<template>
  <Teleport to="body" v-if="is_show_dialog">
    <div class="gkill-float-scrim" :class="ui.isTransparent.value ? 'is-transparent' : ''" />

    <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog is-floating kyou_dialog"
      :class="ui.isTransparent.value ? 'is-transparent' : ''">
      <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
        @touchstart="ui.onHeaderPointerDown">
        <div class="gkill-floating-dialog__title"></div>
        <div class="gkill-floating-dialog__spacer"></div>
  <v-checkbox v-model="ui.isTransparent.value" color="white"    size="small" variant="flat"  :label="i18n.global.t('TRANSPARENT_TITLE')"
          hide-details />
        <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details :color="'primary'" variant="flat"> 
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </div>

      <div class="gkill-floating-dialog__body"> 

        <v-card class="pa-2">
          <KyouView :application_config="application_config" :gkill_api="gkill_api"
            :is_image_request_to_thumb_size="false" :highlight_targets="highlight_targets" :is_image_view="false"
            :kyou="kyou" :last_added_tag="last_added_tag" :show_checkbox="false" :show_content_only="false"
            :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
            :show_mi_limit_time="true" :show_timeis_elapsed_time="true"
            :show_timeis_plaing_end_button="show_timeis_plaing_end_button" :height="'80%'" :width="'100%'"
            :is_readonly_mi_check="is_readonly_mi_check" :enable_context_menu="enable_context_menu"
            :enable_dialog="enable_dialog" :show_attached_timeis="true" :show_rep_name="true"
            :force_show_latest_kyou_info="false" :show_update_time="false" :show_related_time="true"
            :show_attached_tags="true" :show_attached_texts="true" :show_attached_notifications="true"
            @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0] as Kyou)"
            @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
            @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
            @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
            @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
            @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
            @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
            @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
            @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
            @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
            @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
            @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
            @requested_reload_list="emits('requested_reload_list')"
            @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)"
            @requested_open_rykv_dialog="(...params: any[]) => emits('requested_open_rykv_dialog', params[0], params[1], params[2])" />
        </v-card>
      </div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import { computed, type Ref, ref } from 'vue'
import type { KyouDialogProps } from './kyou-dialog-props'
import type { KyouDialogEmits } from '../views/kyou-dialog-emits'
import KyouView from '../views/kyou-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { Kyou } from '@/classes/datas/kyou'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';

defineProps<KyouDialogProps>()
const emits = defineEmits<KyouDialogEmits>()

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
  emits('closed')
}
</script>
