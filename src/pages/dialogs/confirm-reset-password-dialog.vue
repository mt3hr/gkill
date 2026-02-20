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
        <ConfirmResetPasswordView :application_config="application_config" :gkill_api="gkill_api"
          :server_configs="server_configs" :account="cloned_account"
          @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
          @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
          @requested_close_dialog="hide" @requested_reload_server_config="() => emits('requested_reload_server_config')"
          @requested_show_show_password_reset_dialog="(...account: any[]) => emits('requested_show_show_password_reset_dialog', account[0] as Account)" />
      </div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { ConfirmResetPasswordDialogEmits } from './confirm-reset-password-dialog-emits'
import type { ConfirmResetPasswordDialogProps } from './confirm-reset-password-dialog-props'
import ConfirmResetPasswordView from '../views/confirm-reset-password-view.vue'
import { Account } from '@/classes/datas/config/account';
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

defineProps<ConfirmResetPasswordDialogProps>()
const emits = defineEmits<ConfirmResetPasswordDialogEmits>()
defineExpose({ show, hide })

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { i18n } from '@/i18n'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
import { useFloatingDialog } from "@/classes/use-floating-dialog"
const ui = useFloatingDialog("", {
  centerMode: "always",
})

const cloned_account: Ref<Account> = ref(new Account())

async function show(account: Account): Promise<void> {
  cloned_account.value = account
  is_show_dialog.value = true
}
async function hide(): Promise<void> {
  is_show_dialog.value = false
  cloned_account.value = new Account()
}
</script>
