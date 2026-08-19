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

        <ConfirmResetPasswordView :application_config="application_config" :gkill_api="gkill_api"
          :server_configs="server_configs" :account="cloned_account"
          @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
          @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
          @requested_close_dialog="hide" @requested_reload_server_config="() => emits('requested_reload_server_config')"
          @requested_show_show_password_reset_dialog="(account: Account) => emits('requested_show_show_password_reset_dialog', account)" />
        </v-card>
</div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import type { ConfirmResetPasswordDialogEmits } from './confirm-reset-password-dialog-emits'
import type { ConfirmResetPasswordDialogProps } from './confirm-reset-password-dialog-props'
import ConfirmResetPasswordView from '../views/confirm-reset-password-view.vue'
import { Account } from '@/classes/datas/config/account';
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { i18n } from '@/i18n'
import { useConfirmResetPasswordDialog } from '@/classes/use-confirm-reset-password-dialog'

const props = defineProps<ConfirmResetPasswordDialogProps>()
const emits = defineEmits<ConfirmResetPasswordDialogEmits>()
const { is_show_dialog, ui, cloned_account, show, hide } = useConfirmResetPasswordDialog({ props, emits })
defineExpose({ show, hide })
</script>

