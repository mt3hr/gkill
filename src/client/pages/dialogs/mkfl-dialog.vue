<template>
  <Teleport to="body" v-if="is_show_dialog" class="mkfl_dialog">
    <div class="gkill-float-scrim" :class="ui.isTransparent.value ? 'is-transparent' : ''" />

    <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog"
      :class="ui.isTransparent.value ? 'is-transparent' : ''">
      <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
        @touchstart="ui.onHeaderPointerDown">
        <div class="gkill-floating-dialog__title"></div>
        <div class="gkill-floating-dialog__spacer"></div>
  <v-checkbox v-model="ui.isTransparent.value" color="white"    size="small" variant="flat"
          :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
        <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details :color="'primary'"
          variant="flat">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </div>

      <div class="gkill-floating-dialog__body">
        <MKFLView :app_content_height="view_height" :app_content_width="view_width"
          :application_config="application_config" :gkill_api="gkill_api"
          @deleted_kyou="(...deleted_kyou: any[]) => { emits('deleted_kyou', deleted_kyou[0] as Kyou) }"
          @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
          @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
          @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
          @registered_kyou="(...registered_kyou: any[]) => { emits('registered_kyou', registered_kyou[0] as Kyou) }"
          @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
          @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
          @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
          @updated_kyou="(...updated_kyou: any[]) => { emits('updated_kyou', updated_kyou[0] as Kyou) }"
          @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
          @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
          @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
          @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
          @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)" />
      </div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import { computed, type Ref, ref } from 'vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { MKFLDialogEmits } from './mkfl-dialog-emits'
import type { MKFLDialogProps } from './mkfl-dialog-props'
import MKFLView from '../views/mkfl-view.vue'
import type { Kyou } from "@/classes/datas/kyou"
import type { Tag } from "@/classes/datas/tag"
import type { Text } from "@/classes/datas/text"
import type { Notification } from "@/classes/datas/notification"

const props = defineProps<MKFLDialogProps>()
const emits = defineEmits<MKFLDialogEmits>()
defineExpose({ show, hide })

const view_width = computed(() => Math.min(props.app_content_width.valueOf() * 0.85, 600))
const view_height = computed(() => props.app_content_height.valueOf() * 0.85)

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { i18n } from '@/i18n'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
import { useFloatingDialog } from "@/classes/use-floating-dialog"
const ui = useFloatingDialog("mkfl-dialog", {
  centerMode: "always",
})


async function show(): Promise<void> {
  is_show_dialog.value = true
}
async function hide(): Promise<void> {
  is_show_dialog.value = false
}
</script>
