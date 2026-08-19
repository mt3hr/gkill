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
            <v-col class="pa-1 ma-0">
              <span>{{ i18n.global.t('SAVED_RYKV_FIND_KYOU_QUERY_TITLE') }}</span>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-1 ma-0">
              <v-btn color="primary" @click="open_rykv_list_dialog">
                {{ i18n.global.t('SAVED_RYKV_FIND_KYOU_QUERY_TITLE') }}
              </v-btn>
            </v-col>
          </v-row>
          <v-row class="pa-0 ma-0">
            <v-col class="pa-1 ma-0">
              <span>{{ i18n.global.t('SAVED_MI_FIND_KYOU_QUERY_TITLE') }}</span>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-1 ma-0">
              <v-btn color="primary" @click="open_mi_list_dialog">
                {{ i18n.global.t('SAVED_MI_FIND_KYOU_QUERY_TITLE') }}
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
        <EditSavedFindQueryListDialog :app_content_height="props.app_content_height"
          :app_content_width="props.app_content_width" :application_config="props.application_config"
          :gkill_api="props.gkill_api" :query_type="'rykv'"
          @requested_apply_saved_find_querys="(items) => onAppliedRykvItems(items)"
          @received_errors="(errors) => emits('received_errors', errors)"
          @received_messages="(messages) => emits('received_messages', messages)"
          ref="rykv_list_dialog" />
        <EditSavedFindQueryListDialog :app_content_height="props.app_content_height"
          :app_content_width="props.app_content_width" :application_config="props.application_config"
          :gkill_api="props.gkill_api" :query_type="'mi'"
          @requested_apply_saved_find_querys="(items) => onAppliedMiItems(items)"
          @received_errors="(errors) => emits('received_errors', errors)"
          @received_messages="(messages) => emits('received_messages', messages)"
          ref="mi_list_dialog" />
        <HelpDialog screen_name="application-config" ref="help_dialog" />
      </div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import HelpDialog from './help-dialog.vue'
import EditSavedFindQueryListDialog from './edit-saved-find-query-list-dialog.vue'
import type { EditSavedFindQueryDialogProps } from './edit-saved-find-query-dialog-props'
import type { EditSavedFindQueryDialogEmits } from './edit-saved-find-query-dialog-emits'
import { useEditSavedFindQueryDialog } from '@/classes/use-edit-saved-find-query-dialog'

const props = defineProps<EditSavedFindQueryDialogProps>()
const emits = defineEmits<EditSavedFindQueryDialogEmits>()
const { is_show_dialog, ui, show, hide, help_dialog, rykv_list_dialog, mi_list_dialog, open_rykv_list_dialog, open_mi_list_dialog, onAppliedRykvItems, onAppliedMiItems, onSave, onCancel } = useEditSavedFindQueryDialog({ props, emits })

defineExpose({ show, hide })
</script>
