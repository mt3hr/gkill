<template>
  <Teleport to="body" v-if="is_show_dialog">
    <div class="gkill-float-scrim" :class="ui.isTransparent.value ? 'is-transparent' : ''" />

    <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog"
      :class="ui.isTransparent.value ? 'is-transparent' : ''">
      <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
        @touchstart="ui.onHeaderPointerDown">
        <div class="gkill-floating-dialog__title"></div>
        <div class="gkill-floating-dialog__spacer"></div>
        <v-checkbox v-model="ui.isTransparent.value" color="white" size="small" variant="flat"
          :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
        <!-- 出力中は閉じさせない。中断したいときはダイアログの中のキャンセルを押す -->
        <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details :color="'primary'"
          variant="flat" :disabled="is_exporting">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </div>

      <div class="gkill-floating-dialog__body">
        <ExportKyouHTMLView :application_config="application_config" :gkill_api="gkill_api" :columns="columns"
          :focused_column_index="focused_column_index" :screen="screen" :can_choose_dnote="can_choose_dnote"
          :can_choose_map="can_choose_map"
          @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
          @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
          @changed_is_exporting="(value: boolean) => set_is_exporting(value)"
          @requested_close_dialog="hide()" />
      </div>
    </div>
  </Teleport>
</template>

<script lang="ts" setup>
import type { ExportKyouHTMLDialogEmits } from './export-kyou-html-dialog-emits'
import type { ExportKyouHTMLDialogProps } from './export-kyou-html-dialog-props'
import ExportKyouHTMLView from '../views/export-kyou-html-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { i18n } from '@/i18n'
import { useExportKyouHTMLDialog } from '@/classes/use-export-kyou-html-dialog'

const props = defineProps<ExportKyouHTMLDialogProps>()
const emits = defineEmits<ExportKyouHTMLDialogEmits>()
const { is_show_dialog, is_exporting, ui, show, hide, set_is_exporting } = useExportKyouHTMLDialog({ props, emits })
defineExpose({ show, hide })
</script>
