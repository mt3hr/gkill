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

        <ConfirmDeleteRepView :application_config="application_config" :gkill_api="gkill_api"
          :repository="cloned_repository" :server_configs="server_configs"
          @requested_delete_rep="(rep: Repository) => emits('requested_delete_rep', rep)"
          @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
          @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
          @requested_close_dialog="hide()" />
        </v-card>
</div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import type { ConfirmDeleteRepDialogEmits } from './confirm-delete-rep-dialog-emits'
import type { ConfirmDeleteRepDialogProps } from './confirm-delete-rep-dialog-props'
import ConfirmDeleteRepView from '../views/confirm-delete-rep-view.vue'
import { Repository } from '@/classes/datas/config/repository';
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { i18n } from '@/i18n'
import { useConfirmDeleteRepDialog } from '@/classes/use-confirm-delete-rep-dialog'

const props = defineProps<ConfirmDeleteRepDialogProps>()
const emits = defineEmits<ConfirmDeleteRepDialogEmits>()
const { is_show_dialog, ui, cloned_repository, show, hide } = useConfirmDeleteRepDialog({ props, emits })
defineExpose({ show, hide })
</script>

