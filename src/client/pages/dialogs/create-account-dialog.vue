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

        <CreateAccountView :application_config="application_config" :gkill_api="gkill_api"
          :server_configs="server_configs"
          @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
          @requested_close_dialog="hide"
          @created_account="(account: Account) => emits('added_account', account)"
          @requested_reload_server_config="() => emits('requested_reload_server_config')"
          @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)" />
        </v-card>
</div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import type { CreateAccountDialogEmits } from './create-account-dialog-emits'
import type { CreateAccountDialogProps } from './create-account-dialog-props'
import CreateAccountView from '../views/create-account-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Account } from '@/classes/datas/config/account'
import { i18n } from '@/i18n'
import { useCreateAccountDialog } from '@/classes/use-create-account-dialog'

const props = defineProps<CreateAccountDialogProps>()
const emits = defineEmits<CreateAccountDialogEmits>()
const { is_show_dialog, ui, show, hide } = useCreateAccountDialog({ props, emits })
defineExpose({ show, hide })
</script>

