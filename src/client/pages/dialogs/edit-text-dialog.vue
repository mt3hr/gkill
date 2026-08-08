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
       <EditTextView :application_config="application_config" :gkill_api="gkill_api"
          :highlight_targets="text_highlight_targets" :kyou="kyou" :text="text"
          :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
           @requested_close_dialog="hide()" v-on="crudRelayHandlers" />
        </v-card>
</div>
    </div>
  </Teleport>
</template>
<script setup lang="ts">
import type { EditTextDialogProps } from './edit-text-dialog-props'
import type { KyouDialogEmits } from '../views/kyou-dialog-emits'
import EditTextView from '../views/edit-text-view.vue'
import type { Kyou } from '@/classes/datas/kyou'
import { i18n } from '@/i18n'
import { useEditTextDialog } from '@/classes/use-edit-text-dialog'
import { build_kyou_dialog_relay } from '@/classes/kyou-view-relay'

const props = defineProps<EditTextDialogProps>()
const emits = defineEmits<KyouDialogEmits>()

// クリックはフォーカス移動も伴う
const crudRelayHandlers = build_kyou_dialog_relay(emits, {
  'clicked_kyou': (kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) },
})

const { is_show_dialog, ui, text_highlight_targets, show, hide } = useEditTextDialog({ props, emits })

defineExpose({ show, hide })
</script>


