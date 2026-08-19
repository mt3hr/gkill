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
       <ConfirmLogoutView :close_database="close_database_value"
          @requested_logout="(close_database: boolean) => emits('requested_logout', close_database)"
          @requested_close_dialog="hide" />
        </v-card>
</div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import type { ConfirmLogoutDialogEmits } from './confirm-logout-dialog-emits'
import type { ConfirmLogoutDialogProps } from './confirm-logout-dialog-props'
import ConfirmLogoutView from '../views/confirm-logout-view.vue'
import { i18n } from '@/i18n'
import { useConfirmLogoutDialog } from '@/classes/use-confirm-logout-dialog'

const props = defineProps<ConfirmLogoutDialogProps>()
const emits = defineEmits<ConfirmLogoutDialogEmits>()
const { is_show_dialog, ui, close_database_value, show, hide } = useConfirmLogoutDialog({ props, emits })
defineExpose({ show, hide })
</script>
