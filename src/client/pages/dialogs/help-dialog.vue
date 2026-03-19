<template>
  <Teleport to="body" v-if="is_show_dialog">
    <div class="gkill-float-scrim" />

    <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog">
      <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
        @touchstart="ui.onHeaderPointerDown">
        <div class="gkill-floating-dialog__title"></div>
        <div class="gkill-floating-dialog__spacer"></div>
        <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details :color="'primary'"
          variant="flat">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </div>

      <div class="gkill-floating-dialog__body">
        <iframe :src="help_url" class="help-dialog-iframe" />
      </div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import { computed, type Ref, ref } from 'vue'
import { useTheme } from 'vuetify'
import type { HelpDialogProps } from './help-dialog-props'
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'
import { i18n } from '@/i18n'

const props = defineProps<HelpDialogProps>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
const ui = useFloatingDialog("help-dialog", {
  centerMode: "always",
})

const theme = useTheme()

const help_url = computed(() => {
  const locale = i18n.global.locale || 'ja'
  const isDark = theme.global.name.value === 'gkill_dark_theme'
  return `/resources/manual/${locale}/${props.screen_name}.html${isDark ? '?theme=dark' : ''}`
})

async function show(): Promise<void> {
  is_show_dialog.value = true
}
async function hide(): Promise<void> {
  is_show_dialog.value = false
}
</script>
<style scoped>
.help-dialog-iframe {
  width: 100%;
  height: 100%;
  border: none;
  min-height: 400px;
}
</style>
