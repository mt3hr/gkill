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
          <v-row class="pa-0 ma-0" align="center">
            <v-col class="pa-1 ma-0">
              <span>{{ title }}</span>
            </v-col>
          </v-row>
          <v-row v-for="(item, index) in editing_items" :key="item.id" class="pa-0 ma-0" align="center">
            <v-col class="pa-1 ma-0">
              <v-text-field v-model="editing_items[index].title" :label="i18n.global.t('SAVED_FIND_QUERY_NAME_LABEL')"
                density="compact" hide-details />
            </v-col>
            <v-col cols="auto" class="pa-1 ma-0">
              <v-btn color="primary" @click="open_query_editor(index)">
                {{ i18n.global.t('EDIT_SAVED_FIND_QUERY_QUERY_TITLE') }}
              </v-btn>
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
              <v-btn icon="mdi-arrow-up" size="small" variant="text" :disabled="index === 0"
                @click="move_item(index, 'up')" :title="i18n.global.t('MOVE_UP_TITLE')" />
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
              <v-btn icon="mdi-arrow-down" size="small" variant="text" :disabled="index === editing_items.length - 1"
                @click="move_item(index, 'down')" :title="i18n.global.t('MOVE_DOWN_TITLE')" />
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
              <v-btn icon="mdi-delete" size="small" variant="text" @click="delete_item(index)"
                :title="i18n.global.t('DELETE_SAVED_FIND_QUERY_TITLE')" />
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
        <!-- 追加は他画面と同じ右下FAB。
             v-card の中に置いてはいけない: Vuetify の .v-card は position:relative なので
             包含ブロックがスクロール箱そのものになり、一覧と一緒に流れてしまう
             (ryuu-view の FAB がそうなっている)。
             ここ(.gkill-floating-dialog__body 直下)なら body に position が無いので
             包含ブロックはスクロール箱の外側 .gkill-floating-dialog になり、右下に固定される -->
        <v-avatar :style="floating_action_button_style()" color="primary" class="position-fixed-saved-find-query">
          <v-btn color="white" icon="mdi-plus" variant="text" @click="add_item"
            :title="i18n.global.t('ADD_SAVED_FIND_QUERY_TITLE')" />
        </v-avatar>
        <FindQueryEditorDialog v-if="props.query_type === 'rykv'" v-model="current_editing_query"
          :application_config="props.application_config" :gkill_api="props.gkill_api"
          @requested_apply="(query) => onAppliedQuery(query)"
          @received_errors="(errors) => emits('received_errors', errors)"
          @received_messages="(messages) => emits('received_messages', messages)"
          ref="find_query_editor_dialog" />
        <MiFindQueryEditorDialog v-if="props.query_type === 'mi'" v-model="current_editing_query"
          :application_config="props.application_config" :gkill_api="props.gkill_api"
          @requested_apply="(query) => onAppliedQuery(query)"
          @received_errors="(errors) => emits('received_errors', errors)"
          @received_messages="(messages) => emits('received_messages', messages)"
          ref="mi_find_query_editor_dialog" />
        <HelpDialog screen_name="application-config" ref="help_dialog" />
      </div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import HelpDialog from './help-dialog.vue'
import FindQueryEditorDialog from './find-query-editor-dialog.vue'
import MiFindQueryEditorDialog from './mi-find-query-editor-dialog.vue'
import type { EditSavedFindQueryListDialogProps } from './edit-saved-find-query-list-dialog-props'
import type { EditSavedFindQueryListDialogEmits } from './edit-saved-find-query-list-dialog-emits'
import { useEditSavedFindQueryListDialog } from '@/classes/use-edit-saved-find-query-list-dialog'

const props = defineProps<EditSavedFindQueryListDialogProps>()
const emits = defineEmits<EditSavedFindQueryListDialogEmits>()
const { is_show_dialog, ui, editing_items, current_editing_query, title, show, hide, add_item, delete_item, move_item, floating_action_button_style, onSave, onCancel, help_dialog, find_query_editor_dialog, mi_find_query_editor_dialog, open_query_editor, onAppliedQuery } = useEditSavedFindQueryListDialog({ props, emits })

defineExpose({ show, hide })
</script>
