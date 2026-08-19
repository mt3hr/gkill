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
        <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="help_dialog?.show()" hide-details :color="'primary'" variant="flat">
          <v-icon>mdi-help-circle-outline</v-icon>
        </v-btn>
                <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details :color="'primary'" variant="flat">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </div>

      <div class="gkill-floating-dialog__body">
        <v-card variant="flat" class="pa-2">

        <EditKFTLTemplateStructElementView :application_config="application_config" :gkill_api="gkill_api"
          :struct_obj="kftl_template_struct"
          @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
          @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
          @requested_update_kftl_template_struct="(kftl_template_struct: KFTLTemplateElementData) => emits('requested_update_kftl_template_struct', kftl_template_struct)"
          @requested_close_dialog="hide" />
        </v-card>
        <HelpDialog screen_name="kftl" ref="help_dialog" />
</div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import type { EditKFTLTemplateStructElementDialogEmits } from './edit-kftl-template-struct-element-dialog-emits'
import type { EditKFTLTemplateStructElementDialogProps } from './edit-kftl-template-struct-element-dialog-props'
import EditKFTLTemplateStructElementView from '../views/edit-kftl-template-struct-element-view.vue'
import HelpDialog from './help-dialog.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { KFTLTemplateElementData } from '@/classes/datas/kftl-template-element-data'
import { i18n } from '@/i18n'
import { useEditKFTLTemplateStructElementDialog } from '@/classes/use-edit-kftl-template-struct-element-dialog'

const props = defineProps<EditKFTLTemplateStructElementDialogProps>()
const emits = defineEmits<EditKFTLTemplateStructElementDialogEmits>()
const { help_dialog, kftl_template_struct, is_show_dialog, ui, show, hide } = useEditKFTLTemplateStructElementDialog({ props, emits })
defineExpose({ show, hide })
</script>

