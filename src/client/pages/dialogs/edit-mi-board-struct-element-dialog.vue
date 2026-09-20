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

        <EditMiBoardStructElementView :application_config="application_config" :gkill_api="gkill_api"
          :struct_obj="mi_board_struct"
          @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
          @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
          @requested_update_mi_board_struct="(mi_board_struct: MiBoardStructElementData) => emits('requested_update_mi_board_struct', mi_board_struct)"
          @requested_close_dialog="hide" />
        </v-card>
</div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import type { EditMiBoardStructElementDialogEmits } from './edit-mi-board-struct-element-dialog-emits'
import type { EditMiBoardStructElementDialogProps } from './edit-mi-board-struct-element-dialog-props'
import EditMiBoardStructElementView from '../views/edit-mi-board-struct-element-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { MiBoardStructElementData } from '@/classes/datas/config/mi-board-struct-element-data'
import { i18n } from '@/i18n'
import { useEditMiBoardStructElementDialog } from '@/classes/use-edit-mi-board-struct-element-dialog'

const props = defineProps<EditMiBoardStructElementDialogProps>()
const emits = defineEmits<EditMiBoardStructElementDialogEmits>()
const { mi_board_struct, is_show_dialog, ui, show, hide } = useEditMiBoardStructElementDialog({ props, emits })
defineExpose({ show, hide })
</script>
