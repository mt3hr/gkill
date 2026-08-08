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

        <AddUrlogView :application_config="application_config" :gkill_api="gkill_api"
          :highlight_targets="highlight_targets" :kyou="kyou"
          :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
          @requested_close_dialog="hide()" v-on="crudRelayHandlers" />
        </v-card>
</div>
    </div>
  </Teleport>
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import type { EditLantanaDialogProps } from './edit-lantana-dialog-props'
import type { KyouDialogEmits } from '../views/kyou-dialog-emits'
import AddUrlogView from '../views/add-ur-log-view.vue';
import type { Kyou } from '@/classes/datas/kyou';

defineProps<EditLantanaDialogProps>()
const emits = defineEmits<KyouDialogEmits>()

// クリックはフォーカス移動も伴う
const crudRelayHandlers = build_kyou_dialog_relay(emits, {
  'clicked_kyou': (kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) },
})
defineExpose({ show, hide })

import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { i18n } from '@/i18n'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
import { useFloatingDialog } from "@/classes/use-floating-dialog"
import { build_kyou_dialog_relay } from '@/classes/kyou-view-relay'
const ui = useFloatingDialog("add-urlog-dialog", {
  centerMode: "always",
  onEscape: () => hide(),
})


async function show(): Promise<void> {
  is_show_dialog.value = true
}
async function hide(): Promise<void> {
  close_dialog_via_history(is_show_dialog)
}
</script>


