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

        <EditRepTypeStructElementView :application_config="application_config" :gkill_api="gkill_api"
          :struct_obj="rep_type_struct"
          @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
          @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
          @requested_update_rep_type_struct="(rep_type_struct: RepTypeStructElementData) => emits('requested_update_rep_type_struct', rep_type_struct)"
          @requested_close_dialog="hide" />
        </v-card>
</div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import type { EditRepTypeStructElementDialogEmits } from './edit-rep-type-struct-element-dialog-emits'
import type { EditRepTypeStructElementDialogProps } from './edit-rep-type-struct-element-dialog-props'
import EditRepTypeStructElementView from '../views/edit-rep-type-struct-element-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { RepTypeStructElementData } from '@/classes/datas/config/rep-type-struct-element-data'
import { i18n } from '@/i18n'
import { useEditRepTypeStructElementDialog } from '@/classes/use-edit-rep-type-struct-element-dialog'

const props = defineProps<EditRepTypeStructElementDialogProps>()
const emits = defineEmits<EditRepTypeStructElementDialogEmits>()
const { rep_type_struct, is_show_dialog, ui, show, hide } = useEditRepTypeStructElementDialog({ props, emits })
defineExpose({ show, hide })
</script>

