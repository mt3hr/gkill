<template>
  <Teleport to="body" v-if="is_show_dialog" >
    <div class="gkill-float-scrim" :class="ui.isTransparent.value ? 'is-transparent' : ''" />

    <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog"
      :class="ui.isTransparent.value ? 'is-transparent' : ''">
      <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
        @touchstart="ui.onHeaderPointerDown">
        <div class="gkill-floating-dialog__title"></div>
        <div class="gkill-floating-dialog__spacer"></div>
  <v-checkbox v-model="ui.isTransparent.value" color="white"    size="small" variant="flat"
          :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
                <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details :color="'primary'" variant="flat">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </div>

      <div class="gkill-floating-dialog__body">
        <v-card>
          <FindQueryEditorView v-if="model_value" :application_config="received_application_config"
            :gkill_api="gkill_api" :find_kyou_query="model_value" :inited="inited"
            @updated_query="(...query: any[]) => model_value = (query[0] as FindKyouQuery)" @inited="inited = true"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_close_dialog="hide()"
            @requested_apply="(...find_kyou_query: any[]) => model_value = find_kyou_query[0] as FindKyouQuery"
            ref="find_query_editor_view" />
        </v-card>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import FindQueryEditorView from '../views/find-query-editor-view.vue'
import type FindQueryEditorDialogProps from './find-query-editor-dialog-props'
import type FindQueryEditorDialogEmits from './find-query-editor-dialog-emits'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { i18n } from '@/i18n'
import { useFindQueryEditorDialog } from '@/classes/use-find-query-editor-dialog'

const model_value = defineModel<FindKyouQuery>()
const props = defineProps<FindQueryEditorDialogProps>()
const emits = defineEmits<FindQueryEditorDialogEmits>()

const { is_show_dialog, ui, inited, received_application_config, show, hide } = useFindQueryEditorDialog({ props, emits, model_value })

defineExpose({ show, hide })
</script>

