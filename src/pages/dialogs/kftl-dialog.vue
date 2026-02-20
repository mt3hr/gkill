<template>
  <Teleport to="body" v-if="is_show_dialog">
    <div class="gkill-float-scrim" :class="ui.isTransparent.value ? 'is-transparent' : ''" />

    <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog"
      :class="ui.isTransparent.value ? 'is-transparent' : ''">
      <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
        @touchstart="ui.onHeaderPointerDown">
        <div class="gkill-floating-dialog__title"></div>
        <div class="gkill-floating-dialog__spacer"></div>
        <v-checkbox v-model="ui.isTransparent.value" dark  size="small" variant="flat"
          :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
        <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details :color="'primary'"
          variant="flat">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </div>

      <div class="gkill-floating-dialog__body">
        <KFTLView :app_content_height="view_height" :app_content_width="view_width"
          :application_config="application_config" :gkill_api="gkill_api"
          @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
          @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)" ref="kftl_view" />
      </div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import { computed, nextTick, type Ref, ref } from 'vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { KFTLDialogEmits } from './kftl-dialog-emits'
import type { KFTLDialogProps } from './kftl-dialog-props'
import KFTLView from '../views/kftl-view.vue'

const kftl_view = ref<InstanceType<typeof KFTLView> | null>(null);

const props = defineProps<KFTLDialogProps>()
const emits = defineEmits<KFTLDialogEmits>()
defineExpose({ show, hide })

const view_height = computed(() => props.app_content_height.valueOf() * 0.75)
const view_width = computed(() => props.app_content_width.valueOf() * 0.85)

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { i18n } from '@/i18n'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
import { useFloatingDialog } from "@/classes/use-floating-dialog"
const ui = useFloatingDialog("", {
  centerMode: "always",
})


async function show(): Promise<void> {
  is_show_dialog.value = true
  nextTick(() => kftl_view.value?.focus_kftl_text_area())
}
async function hide(): Promise<void> {
  is_show_dialog.value = false
}
</script>