<template>
  <Teleport to="body" v-if="is_show_dialog">
    <div class="gkill-float-scrim" :class="ui.isTransparent.value ? 'is-transparent' : ''" />

    <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog"
      :class="ui.isTransparent.value ? 'is-transparent' : ''">
      <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
        @touchstart="ui.onHeaderPointerDown">
        <div class="gkill-floating-dialog__title">{{ i18n.global.t('EXPORT_RYKV_HTML_TITLE') }}</div>
        <div class="gkill-floating-dialog__spacer"></div>
        <v-checkbox v-model="ui.isTransparent.value" color="white" size="small" variant="flat"
          :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
        <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details :color="'primary'"
          variant="flat">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </div>

      <div class="gkill-floating-dialog__body">
        <ExportRykvHTMLView :application_config="application_config" :gkill_api="gkill_api" :columns="columns"
          :focused_column_index="focused_column_index"
          @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
          @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
          @requested_close_dialog="hide()" />
      </div>
    </div>
  </Teleport>
</template>

<script lang="ts" setup>
import type { ExportRykvHTMLDialogEmits } from './export-rykv-html-dialog-emits'
import type { ExportRykvHTMLDialogProps } from './export-rykv-html-dialog-props'
import ExportRykvHTMLView from '../views/export-rykv-html-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { i18n } from '@/i18n'
import { useExportRykvHTMLDialog } from '@/classes/use-export-rykv-html-dialog'

const props = defineProps<ExportRykvHTMLDialogProps>()
const emits = defineEmits<ExportRykvHTMLDialogEmits>()
const { is_show_dialog, ui, show, hide } = useExportRykvHTMLDialog({ props, emits })
defineExpose({ show, hide })
</script>
