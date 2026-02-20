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
        <EditKFTLTemplateStructElementView :application_config="application_config" :gkill_api="gkill_api"
          :struct_obj="kftl_template_struct"
          @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
          @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
          @requested_update_kftl_template_struct="(...kftl_template_struct: any[]) => emits('requested_update_kftl_template_struct', kftl_template_struct[0] as KFTLTemplateStructElementData)"
          @requested_close_dialog="hide" />
      </div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { EditKFTLTemplateStructElementDialogEmits } from './edit-kftl-template-struct-element-dialog-emits'
import type { EditKFTLTemplateStructElementDialogProps } from './edit-kftl-template-struct-element-dialog-props'
import EditKFTLTemplateStructElementView from '../views/edit-kftl-template-struct-element-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

defineProps<EditKFTLTemplateStructElementDialogProps>()
const emits = defineEmits<EditKFTLTemplateStructElementDialogEmits>()
defineExpose({ show, hide })

const kftl_template_struct: Ref<KFTLTemplateStructElementData> = ref(new KFTLTemplateStructElementData())
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { KFTLTemplateStructElementData } from '@/classes/datas/config/kftl-template-struct-element-data'
import { i18n } from '@/i18n'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
import { useFloatingDialog } from "@/classes/use-floating-dialog"
const ui = useFloatingDialog("", {
  centerMode: "always",
})


async function show(kftl_template_struct_obj: KFTLTemplateStructElementData): Promise<void> {
  kftl_template_struct.value = kftl_template_struct_obj
  is_show_dialog.value = true
}
async function hide(): Promise<void> {
  is_show_dialog.value = false
}
</script>
