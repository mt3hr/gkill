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

        <AddNewTagStructElementView :application_config="application_config" :gkill_api="gkill_api"
          @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
          @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
          @requested_close_dialog="hide"
          @requested_add_tag_struct_element="(tag_struct_element: TagStructElementData) => emits('requested_add_tag_struct_element', tag_struct_element)" />
        </v-card>
</div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import type { AddNewTagStructElementDialogEmits } from './add-new-tag-struct-element-dialog-emits'
import type { AddNewTagStructElementDialogProps } from './add-new-tag-struct-element-dialog-props'
import AddNewTagStructElementView from '../views/add-new-tag-struct-element-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { TagStructElementData } from '@/classes/datas/config/tag-struct-element-data'
import { i18n } from '@/i18n'
import { useAddNewTagStructElementDialog } from '@/classes/use-add-new-tag-struct-element-dialog'

const props = defineProps<AddNewTagStructElementDialogProps>()
const emits = defineEmits<AddNewTagStructElementDialogEmits>()
const { is_show_dialog, ui, show, hide } = useAddNewTagStructElementDialog({ props, emits })
defineExpose({ show, hide })
</script>

