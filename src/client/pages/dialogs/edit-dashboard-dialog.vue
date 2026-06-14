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
        <v-card class="pa-2">
          <v-row class="pa-0 ma-0">
            <v-col class="pa-1 ma-0">
              <span>{{ i18n.global.t('DASHBOARD_DNOTE_FIND_KYOU_QUERY_TITLE') }}</span>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-1 ma-0">
              <v-btn color="primary" @click="openDnoteQueryEditor">
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
              <v-btn color="primary" @click="openMiQueryEditor">
                {{ i18n.global.t('DASHBOARD_DEFAULT_FIND_KYOU_QUERY_TITLE') }}
              </v-btn>
            </v-col>
          </v-row>
          <v-row class="pa-0 ma-0 pt-2">
            <v-col cols="auto" class="pa-0 ma-0">
              <v-btn color="secondary" @click="onCancel">{{ i18n.global.t('CANCEL_TITLE') }}</v-btn>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
              <v-btn color="primary" @click="onSave">{{ i18n.global.t('SAVE_TITLE') }}</v-btn>
            </v-col>
          </v-row>
        </v-card>
        <FindQueryEditorDialog v-model="current_dnote_query" :application_config="props.application_config"
          :gkill_api="props.gkill_api"
          @requested_apply="(query) => { current_dnote_query = query; emitCurrentConfig() }"
          @received_errors="(errors) => emits('received_errors', errors)"
          @received_messages="(messages) => emits('received_messages', messages)"
          ref="dnote_query_editor_dialog" />
        <MiFindQueryEditorDialog v-model="current_mi_query" :application_config="props.application_config"
          :gkill_api="props.gkill_api"
          @requested_apply="(query) => { current_mi_query = query; emitCurrentConfig() }"
          @received_errors="(errors) => emits('received_errors', errors)"
          @received_messages="(messages) => emits('received_messages', messages)"
          ref="mi_query_editor_dialog" />
        <HelpDialog screen_name="dashboard" ref="help_dialog" />
      </div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import { ref } from 'vue'
import { i18n } from '@/i18n'
import HelpDialog from './help-dialog.vue'
import FindQueryEditorDialog from './find-query-editor-dialog.vue'
import MiFindQueryEditorDialog from './mi-find-query-editor-dialog.vue'
import { DashboardConfig } from '@/classes/datas/config/dashboard-config'
import type { EditDashboardDialogProps } from './edit-dashboard-dialog-props'
import type { EditDashboardDialogEmits } from './edit-dashboard-dialog-emits'
import { useEditDashboardDialog } from '@/classes/use-edit-dashboard-dialog'

const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)
const dnote_query_editor_dialog = ref<InstanceType<typeof FindQueryEditorDialog> | null>(null)
const mi_query_editor_dialog = ref<InstanceType<typeof MiFindQueryEditorDialog> | null>(null)
const props = defineProps<EditDashboardDialogProps>()
const emits = defineEmits<EditDashboardDialogEmits>()

const { is_show_dialog, ui, current_dnote_query, current_mi_query, show, hide } = useEditDashboardDialog({ props, emits })

function openDnoteQueryEditor(): void {
    dnote_query_editor_dialog.value?.show(current_dnote_query.value)
}

function openMiQueryEditor(): void {
    mi_query_editor_dialog.value?.show(current_mi_query.value)
}

function emitCurrentConfig(): void {
    const config = new DashboardConfig()
    config.dashboard_dnote_find_kyou_query = current_dnote_query.value
    config.dashboard_mi_find_kyou_query = current_mi_query.value
    emits('requested_apply_dashboard_struct', config.to_json())
}

function onSave(): void {
    emitCurrentConfig()
    hide()
}

function onCancel(): void {
    hide()
}

defineExpose({ show, hide })
</script>
