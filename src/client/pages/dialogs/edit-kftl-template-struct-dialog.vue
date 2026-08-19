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

        <EditKFTLTemplateStructView :application_config="application_config" :gkill_api="gkill_api"
          :kftl_template_struct="application_config.kftl_template_struct"
          @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
          @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
          @requested_reload_application_config="emits('requested_reload_application_config')"
          @requested_apply_kftl_template_struct="(kftl_template_struct_element_data: KFTLTemplateElementData) => emits('requested_apply_kftl_template_struct', kftl_template_struct_element_data)"
          @requested_close_dialog="hide" />
        </v-card>
        <HelpDialog screen_name="kftl-template-struct" ref="help_dialog" />
</div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import type { EditKFTLTemplateStructDialogEmits } from './edit-kftl-template-struct-dialog-emits.ts'
import type { EditKFTLTemplateStructDialogProps } from './edit-kftl-template-struct-dialog-props.ts'
import EditKFTLTemplateStructView from '../views/edit-kftl-template-struct-view.vue'
import HelpDialog from './help-dialog.vue'
import type { GkillError } from '@/classes/api/gkill-error.js'
import type { GkillMessage } from '@/classes/api/gkill-message.js'
import type { KFTLTemplateElementData } from '@/classes/datas/kftl-template-element-data'
import { i18n } from '@/i18n'
import { useEditKFTLTemplateStructDialog } from '@/classes/use-edit-kftl-template-struct-dialog'

const props = defineProps<EditKFTLTemplateStructDialogProps>()
const emits = defineEmits<EditKFTLTemplateStructDialogEmits>()
const { help_dialog, is_show_dialog, ui, show, hide } = useEditKFTLTemplateStructDialog({ props, emits })
defineExpose({ show, hide })
</script>

