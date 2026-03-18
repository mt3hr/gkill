<template>
  <Teleport to="body" v-if="is_show_dialog">
    <div class="gkill-float-scrim" :class="ui.isTransparent.value ? 'is-transparent' : ''" />

    <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog"
      :class="ui.isTransparent.value ? 'is-transparent' : ''">
      <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
        @touchstart="ui.onHeaderPointerDown">
        <div class="gkill-floating-dialog__title"></div>
        <div class="gkill-floating-dialog__spacer"></div>
  <v-checkbox v-model="ui.isTransparent.value" color="white"    size="small" variant="flat"
          :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
        <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details :color="'primary'"
          variant="flat">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </div>

      <div class="gkill-floating-dialog__body" ref="dialog_body_ref">
        <v-card style="overflow: hidden">
       <KFTLView :app_content_height="view_height" :app_content_width="view_width"
          :application_config="application_config" :gkill_api="gkill_api"
          @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
          @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)" ref="kftl_view" />
        </v-card>
</div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import { computed, nextTick, onBeforeUnmount, type Ref, ref, watch } from 'vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { KFTLDialogEmits } from './kftl-dialog-emits'
import type { KFTLDialogProps } from './kftl-dialog-props'
import KFTLView from '../views/kftl-view.vue'

const kftl_view = ref<InstanceType<typeof KFTLView> | null>(null);
const dialog_body_ref = ref<HTMLElement | null>(null)

const props = defineProps<KFTLDialogProps>()
const emits = defineEmits<KFTLDialogEmits>()
defineExpose({ show, hide })

const default_view_width = computed(() => Math.min(props.app_content_width.valueOf() * 0.85, 600))
const default_view_height = computed(() => props.app_content_height.valueOf() * 0.75)

const observed_body_width = ref(0)
const observed_body_height = ref(0)

const view_width = computed(() => {
  if (observed_body_width.value > 0) {
    return observed_body_width.value
  }
  return default_view_width.value
})
const view_height = computed(() => {
  if (observed_body_height.value > 0) {
    return observed_body_height.value
  }
  return default_view_height.value
})

let body_ro: ResizeObserver | null = null
watch(dialog_body_ref, (el, oldEl) => {
  if (body_ro && oldEl) { try { body_ro.unobserve(oldEl) } catch { /* noop */ } }
  if (el) {
    if (!body_ro) {
      body_ro = new ResizeObserver((entries) => {
        for (const entry of entries) {
          observed_body_width.value = entry.contentRect.width
          observed_body_height.value = entry.contentRect.height
        }
      })
    }
    body_ro.observe(el)
  }
}, { flush: 'post' })
onBeforeUnmount(() => { body_ro?.disconnect(); body_ro = null })

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { i18n } from '@/i18n'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
import { useFloatingDialog } from "@/classes/use-floating-dialog"
const ui = useFloatingDialog("kftl-dialog", {
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
