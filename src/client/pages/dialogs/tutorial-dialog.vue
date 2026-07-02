<template>
  <Teleport to="body" v-if="is_show_dialog">
    <div class="gkill-float-scrim" />

    <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog tutorial-dialog">
      <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
        @touchstart="ui.onHeaderPointerDown">
        <div class="gkill-floating-dialog__title">{{ $t('TUTORIAL_TITLE') }}</div>
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
import { computed, type Ref, ref, watch } from 'vue'
import { useTheme } from 'vuetify'
import type { TutorialDialogProps } from './tutorial-dialog-props'
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'
import { i18n } from '@/i18n'
import { UpdateApplicationConfigRequest } from '@/classes/api/req_res/update-application-config-request'

const props = defineProps<TutorialDialogProps>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
const ui = useFloatingDialog("tutorial-dialog", {
  centerMode: "always",
  onEscape: () => hide(),
})

const theme = useTheme()
const dont_show_again: Ref<boolean> = ref(false)

const tutorial_url = computed(() => {
  const locale = i18n.global.locale || 'ja'
  const isDark = theme.global.name.value === 'gkill_dark_theme'
  return `/resources/manual/${locale}/tutorial.html${isDark ? '?theme=dark' : ''}`
})

async function show(): Promise<void> {
  dont_show_again.value = false
  is_show_dialog.value = true
}
async function hide(): Promise<void> {
  is_show_dialog.value = false
}

watch(dont_show_again, async (checked) => {
  if (!checked) return
  const config = props.application_config.clone()
  config.show_tutorial_on_startup = false
  const req = new UpdateApplicationConfigRequest()
  req.session_id = props.gkill_api.get_session_id()
  req.application_config = config
  await props.gkill_api.update_application_config(req)
  // ブラウザ側キャッシュも更新
  config.show_tutorial_on_startup = false
  props.gkill_api.set_saved_application_config(props.application_config)
})

async function close_dialog(): Promise<void> {
  await hide()
}
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
