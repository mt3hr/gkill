<template>
  <Teleport to="body" v-if="is_show_dialog">
    <div class="gkill-float-scrim" :class="ui.isTransparent.value ? 'is-transparent' : ''" />

    <div :ref="(el) => { ui.containerRef.value = el as HTMLElement | null; set_dialog_el(el as HTMLElement | null) }"
      :style="ui.fixedStyle.value" class="gkill-floating-dialog"
      :class="ui.isTransparent.value ? 'is-transparent' : ''">
      <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
        @touchstart="ui.onHeaderPointerDown">
        <div class="gkill-floating-dialog__title"></div>
        <div class="gkill-floating-dialog__spacer"></div>
        <v-checkbox v-model="ui.isTransparent.value" color="white" size="small" variant="flat"
          :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
        <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details :color="'primary'"
          variant="flat">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </div>

      <div class="gkill-floating-dialog__body">
        <v-card class="pa-2" min-width="320">
          <v-card-title class="pb-1">{{ i18n.global.t('SAVE_CLIPBOARD_TO_FILE_TITLE') }}</v-card-title>

          <!-- Loading -->
          <div v-if="is_loading" class="d-flex justify-center pa-4">
            <v-progress-circular indeterminate color="primary" />
          </div>

          <!-- Error -->
          <v-alert v-if="error_message_key && !is_loading" type="info" variant="tonal" class="ma-2" density="compact">
            {{ i18n.global.t(error_message_key) }}
          </v-alert>

          <!-- Image preview -->
          <v-img v-if="is_image_type() && preview_url && !is_loading"
            :src="preview_url" max-height="200" class="ma-2 rounded" contain />

          <!-- Text preview -->
          <pre v-if="is_text_type() && text_preview && !is_loading"
            class="save-clipboard-text-preview ma-2 pa-2 rounded text-body-2">{{ text_preview }}</pre>

          <!-- Binary / PDF -->
          <div v-if="clipboard_blob && !is_image_type() && !is_text_type() && !is_loading"
            class="ma-2 pa-2 text-body-2 text-medium-emphasis">
            {{ type_display_name() }}  {{ file_size_display() }}
          </div>

          <!-- Type + size -->
          <div v-if="clipboard_blob && !is_loading" class="mx-2 text-caption text-medium-emphasis">
            {{ i18n.global.t('CLIPBOARD_TYPE_LABEL') }}: {{ type_display_name() }}
            <span v-if="!is_text_type()"> / {{ file_size_display() }}</span>
          </div>

          <v-divider class="my-2" />

          <!-- Filename (collapsed) -->
          <div class="mx-2">
            <a class="text-caption text-primary cursor-pointer"
              @click="show_filename_editor = !show_filename_editor">
              {{ i18n.global.t('CLIPBOARD_CHANGE_FILENAME_TITLE') }}
            </a>
            <v-expand-transition>
              <v-text-field v-if="show_filename_editor" v-model="filename"
                density="compact" variant="outlined" class="mt-1" hide-details />
            </v-expand-transition>
          </div>

          <!-- Target rep -->
          <v-select v-model="target_rep_name" :items="target_rep_names" density="compact"
            :label="i18n.global.t('UPLOAD_DESTINATION_TITLE')" variant="outlined" class="ma-2" hide-details />

          <!-- Conflict behavior -->
          <v-radio-group v-model="conflict_behavior" inline density="compact" hide-details class="mx-2">
            <v-radio :label="i18n.global.t('RENAME_TITLE')" :value="FileUploadConflictBehavior.rename" />
            <v-radio :label="i18n.global.t('OVERRIDE_TITLE')" :value="FileUploadConflictBehavior.override" />
          </v-radio-group>

          <!-- Already saved confirm -->
          <v-alert v-if="show_already_saved_confirm" type="warning" variant="tonal" class="ma-2" density="compact">
            {{ i18n.global.t('CLIPBOARD_ALREADY_SAVED_CONFIRM_MESSAGE') }}
            <div class="mt-1">
              <v-btn size="small" color="warning" variant="tonal" @click="force_save">
                {{ i18n.global.t('CLIPBOARD_ALREADY_SAVED_SAVE_AGAIN_TITLE') }}
              </v-btn>
            </div>
          </v-alert>

          <!-- Actions -->
          <v-card-actions class="pt-0">
            <v-spacer />
            <v-btn @click="hide">{{ i18n.global.t('CLOSE_TITLE') }}</v-btn>
            <v-btn ref="save_btn" color="primary" variant="flat"
              :disabled="!clipboard_blob || is_loading"
              @click="save_or_confirm" @keydown.enter.prevent="save_or_confirm">
              {{ i18n.global.t('SAVE_TITLE') }}
            </v-btn>
          </v-card-actions>

          <!-- Shortcut hint -->
          <div class="text-caption text-medium-emphasis text-center pb-1">
            {{ i18n.global.t('SAVE_CLIPBOARD_SHORTCUT_HINT') }}
          </div>
        </v-card>
      </div>
    </div>

    <!-- Saved snackbar -->
    <v-snackbar v-model="show_saved_snackbar" :timeout="2000" color="success" location="bottom">
      {{ i18n.global.t('CLIPBOARD_SAVED_MESSAGE') }}
    </v-snackbar>
  </Teleport>
</template>

<script lang="ts" setup>
import type { SaveClipboardToFileDialogProps } from './save-clipboard-to-file-dialog-props'
import type { KyouViewEmits } from '../views/kyou-view-emits'
import { FileUploadConflictBehavior } from '@/classes/api/req_res/file-upload-conflict-behavior'
import { i18n } from '@/i18n'
import { useFloatingDialog } from '@/classes/use-floating-dialog'
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useSaveClipboardToFileDialog } from '@/classes/use-save-clipboard-to-file-dialog'

const props = defineProps<SaveClipboardToFileDialogProps>()
const emits = defineEmits<KyouViewEmits>()

const ui = useFloatingDialog('save-clipboard-to-file-dialog', {
    centerMode: 'always',
    onEscape: () => hide(),
})

const {
    is_show_dialog,
    is_loading,
    clipboard_blob,
    filename,
    preview_url,
    text_preview,
    error_message_key,
    target_rep_names,
    target_rep_name,
    conflict_behavior,
    show_filename_editor,
    show_already_saved_confirm,
    show_saved_snackbar,
    save_btn,
    set_dialog_el,
    is_image_type,
    is_text_type,
    type_display_name,
    file_size_display,
    save_or_confirm,
    force_save,
    show,
    hide,
} = useSaveClipboardToFileDialog({ props, emits })

useDialogHistoryStack(is_show_dialog)
defineExpose({ show, hide })
</script>

<style scoped>
.save-clipboard-text-preview {
    white-space: pre-wrap;
    word-break: break-all;
    max-height: 120px;
    overflow: hidden;
    background: rgba(0, 0, 0, 0.05);
    font-size: 0.75rem;
}
</style>
