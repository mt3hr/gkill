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
        <EditRyuuItemView v-model="model_value" :application_config="application_config" :gkill_api="gkill_api"
          @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
          @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
          @requested_close_dialog="hide()" />
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, type Ref } from 'vue'
import EditRyuuItemView from '../views/edit-ryuu-item-view.vue';
import type EditRyuuItemDialogEmits from './edit-ryuu-item-dialog-emits';
import type EditRyuuItemDialogProps from './edit-ryuu-item-dialog-props';
import RelatedKyouQuery from '@/classes/dnote/related-kyou-query';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { i18n } from '@/i18n'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
import { useFloatingDialog } from "@/classes/use-floating-dialog"
const ui = useFloatingDialog("edit-ryuu-item-dialog", {
  centerMode: "always",
})


const model_value = defineModel<RelatedKyouQuery>()
defineExpose({ show, hide })
defineProps<EditRyuuItemDialogProps>()
const emits = defineEmits<EditRyuuItemDialogEmits>()

async function show(): Promise<void> {
  is_show_dialog.value = true
}
async function hide(): Promise<void> {
  is_show_dialog.value = false
}
</script>

