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
          <!-- 見出しは h タグにしない。useFloatingDialog は本文の最初の見出しをダイアログの名前（aria）にするので、
               h にするとダイアログ全体の名前が「検索ショートカット」になる -->
          <div class="saved_find_query_shortcut_section">
            <div class="search_condition_section_title text-subtitle-1 font-weight-medium pa-1">
              {{ i18n.global.t('SAVED_FIND_QUERY_SHORTCUT_TITLE') }}
            </div>
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
          </div>
          <v-divider class="my-2" />
          <div class="playing_timeis_query_section">
            <div class="search_condition_section_title text-subtitle-1 font-weight-medium pa-1">
              {{ i18n.global.t('EDIT_PLAYING_TIMEIS_TITLE') }}
            </div>
            <v-row class="pa-0 ma-0">
              <v-col cols="auto" class="pa-0 ma-0">
                <v-checkbox v-model="is_use_custom_find_kyou_query"
                  :label="i18n.global.t('CUSTOM_FIND_QUERY_TITLE')" hide-details />
              </v-col>
              <v-spacer />
              <v-col cols="auto" class="pa-1 ma-0 align-self-center">
                <v-btn v-if="is_use_custom_find_kyou_query" dark color="primary"
                  @click="open_playing_timeis_query_editor">
                  {{ i18n.global.t('FIND_QUERY_TITLE') }}
                </v-btn>
              </v-col>
            </v-row>
          </div>
          <v-divider class="my-2" />
          <div class="dashboard_query_section">
            <div class="search_condition_section_title text-subtitle-1 font-weight-medium pa-1">
              {{ i18n.global.t('EDIT_DASHBOARD_TITLE') }}
            </div>
            <v-row class="pa-0 ma-0">
              <v-col class="pa-1 ma-0">
                <span>{{ i18n.global.t('DASHBOARD_DNOTE_FIND_KYOU_QUERY_TITLE') }}</span>
              </v-col>
              <v-spacer />
              <v-col cols="auto" class="pa-1 ma-0">
                <v-btn color="primary" @click="open_dnote_query_editor">
                  {{ i18n.global.t('DASHBOARD_DNOTE_FIND_KYOU_QUERY_TITLE') }}
                </v-btn>
              </v-col>
            </v-row>
            <v-row class="pa-0 ma-0">
              <v-col class="pa-1 ma-0">
                <span>{{ i18n.global.t('DASHBOARD_DEFAULT_FIND_KYOU_QUERY_TITLE') }}</span>
              </v-col>
              <v-spacer />
              <v-col cols="auto" class="pa-1 ma-0">
                <v-btn color="primary" @click="open_mi_query_editor">
                  {{ i18n.global.t('DASHBOARD_DEFAULT_FIND_KYOU_QUERY_TITLE') }}
                </v-btn>
              </v-col>
            </v-row>
          </div>
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
        <FindTimeIsQueryEditorDialog v-model="playing_timeis_editor_model"
          :application_config="props.application_config" :gkill_api="props.gkill_api"
          @requested_apply="(query) => onAppliedPlayingTimeIsQuery(query)"
          @received_errors="(errors) => emits('received_errors', errors)"
          @received_messages="(messages) => emits('received_messages', messages)"
          ref="find_time_is_query_editor_dialog" />
        <FindQueryEditorDialog v-model="current_dnote_query" :application_config="props.application_config"
          :gkill_api="props.gkill_api"
          @requested_apply="(query) => onAppliedDnoteQuery(query)"
          @received_errors="(errors) => emits('received_errors', errors)"
          @received_messages="(messages) => emits('received_messages', messages)"
          ref="dnote_query_editor_dialog" />
        <MiFindQueryEditorDialog v-model="current_mi_query" :application_config="props.application_config"
          :gkill_api="props.gkill_api"
          @requested_apply="(query) => onAppliedMiQuery(query)"
          @received_errors="(errors) => emits('received_errors', errors)"
          @received_messages="(messages) => emits('received_messages', messages)"
          ref="mi_query_editor_dialog" />
        <HelpDialog screen_name="application-config" ref="help_dialog" />
      </div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import HelpDialog from './help-dialog.vue'
import EditSavedFindQueryListDialog from './edit-saved-find-query-list-dialog.vue'
import FindTimeIsQueryEditorDialog from './find-time-is-query-editor-dialog.vue'
import FindQueryEditorDialog from './find-query-editor-dialog.vue'
import MiFindQueryEditorDialog from './mi-find-query-editor-dialog.vue'
import type { EditSavedFindQueryDialogProps } from './edit-saved-find-query-dialog-props'
import type { EditSavedFindQueryDialogEmits } from './edit-saved-find-query-dialog-emits'
import { useEditSavedFindQueryDialog } from '@/classes/use-edit-saved-find-query-dialog'

const props = defineProps<EditSavedFindQueryDialogProps>()
const emits = defineEmits<EditSavedFindQueryDialogEmits>()
const {
    is_show_dialog, ui, show, hide,
    help_dialog, rykv_list_dialog, mi_list_dialog,
    find_time_is_query_editor_dialog, dnote_query_editor_dialog, mi_query_editor_dialog,
    playing_timeis_editor_model, is_use_custom_find_kyou_query, current_dnote_query, current_mi_query,
    open_rykv_list_dialog, open_mi_list_dialog, onAppliedRykvItems, onAppliedMiItems,
    open_playing_timeis_query_editor, onAppliedPlayingTimeIsQuery,
    open_dnote_query_editor, open_mi_query_editor, onAppliedDnoteQuery, onAppliedMiQuery,
    onSave, onCancel,
} = useEditSavedFindQueryDialog({ props, emits })

defineExpose({ show, hide })
</script>
