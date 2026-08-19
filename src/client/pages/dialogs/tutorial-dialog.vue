<template>
  <Teleport to="body" v-if="is_show_dialog">
    <div class="gkill-float-scrim" />

    <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog tutorial-dialog">
      <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
        @touchstart="ui.onHeaderPointerDown">
        <div class="gkill-floating-dialog__title"></div>
        <div class="gkill-floating-dialog__spacer"></div>
        <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="close_dialog" hide-details :color="'primary'"
          variant="flat">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </div>

      <div class="gkill-floating-dialog__body tutorial-dialog__body">
        <iframe :src="tutorial_url" class="tutorial-dialog-iframe" />
      </div>

      <div class="tutorial-dialog-footer pa-2">
        <v-checkbox v-model="dont_show_again" :label="$t('DONT_SHOW_TUTORIAL_AGAIN')" density="compact" hide-details />
      </div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import type { TutorialDialogProps } from './tutorial-dialog-props'
import { useTutorialDialog } from '@/classes/use-tutorial-dialog'

const props = defineProps<TutorialDialogProps>()
const { is_show_dialog, ui, dont_show_again, tutorial_url, show, hide, close_dialog } = useTutorialDialog({ props })
defineExpose({ show, hide })
</script>
<style>
/* Teleport 先では data-v-xxx が付かないため非スコープで定義 */
/* モバイル (≤768px): 80vw × 70vh */
@media (max-width: 768px) {
  .tutorial-dialog:not(.is-user-resized) {
    width: 80vw !important;
    max-width: 80vw !important;
    height: 70vh !important;
  }
  .tutorial-dialog:not(.is-user-resized) .tutorial-dialog__body {
    width: 80vw !important;
    max-width: 80vw !important;
  }
}
/* PC (≥769px): 50vw × 80vh */
@media (min-width: 769px) {
  .tutorial-dialog:not(.is-user-resized) {
    width: 50vw !important;
    max-width: 50vw !important;
    height: 80vh !important;
  }
  .tutorial-dialog:not(.is-user-resized) .tutorial-dialog__body {
    width: 50vw !important;
    max-width: 50vw !important;
  }
}
/* ボディをコンテナ高さに追従させる (ユーザーリサイズ前) */
.tutorial-dialog:not(.is-user-resized) .tutorial-dialog__body {
  max-height: none !important;
  flex: 1 1 auto;
  min-height: 0;
}
.tutorial-dialog-iframe {
  width: 100%;
  flex: 1 1 auto;
  border: none;
  min-height: 0;
}
.tutorial-dialog-footer {
  border-top: 1px solid #e0e0e0;
  background-color: rgb(var(--v-theme-background));
}
</style>
