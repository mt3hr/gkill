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
        <ApplicationConfigView :application_config="application_config" :gkill_api="gkill_api"
          :app_content_height="app_content_height" :app_content_width="app_content_width"
          @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
          @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
          @requested_close_dialog="() => hide()" ref="application_config_view" />
      </div>
    </div>
  </Teleport>
</template>

<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import { defineProps, defineEmits } from 'vue'
import type { ApplicationConfigDialogProps } from './application-config-dialog-props'
import type { ApplicationConfigDialogEmits } from './application-config-dialog-emits'
import ApplicationConfigView from '../views/application-config-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import { GkillMessage } from '@/classes/api/gkill-message'

const application_config_view = ref<InstanceType<typeof ApplicationConfigView> | null>(null);

const _props = defineProps<ApplicationConfigDialogProps>()
const emits = defineEmits<ApplicationConfigDialogEmits>()
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
  application_config_view.value?.reload_cloned_application_config()
}
async function hide(): Promise<void> {
  is_show_dialog.value = false
}
</script>
