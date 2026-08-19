<template>
  <Teleport to="body" v-if="is_show_dialog">
    <div class="gkill-float-scrim" :class="ui.isTransparent.value ? 'is-transparent' : ''" />

    <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog"
      :class="ui.isTransparent.value ? 'is-transparent' : ''">
      <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
        @touchstart="ui.onHeaderPointerDown">
        <div class="gkill-floating-dialog__spacer"></div>
        <v-checkbox v-model="ui.isTransparent.value" color="white" size="small" variant="flat"
          :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
        <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="help_dialog?.show()" hide-details
          :color="'primary'" variant="flat">
          <v-icon>mdi-help-circle-outline</v-icon>
        </v-btn>
        <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details :color="'primary'"
          variant="flat">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </div>

      <div class="gkill-floating-dialog__body">
        <v-card variant="flat" class="pa-2">
          <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
              <v-checkbox v-model="is_use_custom_find_kyou_query"
                :label="i18n.global.t('CUSTOM_FIND_QUERY_TITLE')" />
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pt-3 ma-0">
              <v-btn v-if="is_use_custom_find_kyou_query" dark color="primary" @click="open_query_editor">
                {{ i18n.global.t('FIND_QUERY_TITLE') }}
              </v-btn>
            </v-col>
          </v-row>
          <v-row class="pa-0 ma-0 pt-2 flex-row-reverse gkill-dialog-actions">
            <v-col cols="auto" class="pa-0 ma-0">
              <v-btn color="primary" @click="onSave">{{ i18n.global.t('APPLY_TITLE') }}</v-btn>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
              <v-btn color="secondary" @click="onCancel">{{ i18n.global.t('CANCEL_TITLE') }}</v-btn>
            </v-col>
          </v-row>
        </v-card>
        <FindTimeIsQueryEditorDialog v-model="editor_model" :application_config="props.application_config"
          :gkill_api="props.gkill_api"
          @requested_apply="(query) => onAppliedQuery(query)"
          @received_errors="(errors) => emits('received_errors', errors)"
          @received_messages="(messages) => emits('received_messages', messages)"
          ref="find_time_is_query_editor_dialog" />
        <HelpDialog screen_name="plaing" ref="help_dialog" />
      </div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import HelpDialog from './help-dialog.vue'
import FindTimeIsQueryEditorDialog from './find-time-is-query-editor-dialog.vue'
import type { EditPlaingTimeIsDialogProps } from './edit-plaing-time-is-dialog-props'
import type { EditPlaingTimeIsDialogEmits } from './edit-plaing-time-is-dialog-emits'
import { useEditPlaingTimeIsDialog } from '@/classes/use-edit-plaing-time-is-dialog'

const props = defineProps<EditPlaingTimeIsDialogProps>()
const emits = defineEmits<EditPlaingTimeIsDialogEmits>()
const { is_show_dialog, ui, editor_model, is_use_custom_find_kyou_query, show, hide, help_dialog, find_time_is_query_editor_dialog, open_query_editor, onAppliedQuery, onSave, onCancel } = useEditPlaingTimeIsDialog({ props, emits })

defineExpose({ show, hide })
</script>
