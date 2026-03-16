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
        <ConfirmLogoutView :close_database="close_database_value"
          @requested_logout="(close_database: boolean) => emits('requested_logout', close_database)"
          @requested_close_dialog="hide" />
      </div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { ConfirmLogoutDialogEmits } from './confirm-logout-dialog-emits'
import type { ConfirmLogoutDialogProps } from './confirm-logout-dialog-props'
import ConfirmLogoutView from '../views/confirm-logout-view.vue'

defineProps<ConfirmLogoutDialogProps>()
const emits = defineEmits<ConfirmLogoutDialogEmits>()
defineExpose({ show, hide })

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { i18n } from '@/i18n'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
import { useFloatingDialog } from "@/classes/use-floating-dialog"
const ui = useFloatingDialog("confirm-logout-dialog", {
  centerMode: "always",
})

const close_database_value: Ref<boolean> = ref(false)

async function show(close_database: boolean): Promise<void> {
  close_database_value.value = close_database
  is_show_dialog.value = true
}
async function hide(): Promise<void> {
  is_show_dialog.value = false
  close_database_value.value = false
}
</script>
