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
        <v-card variant="flat" class="pa-2">

        <DecideRelatedTimeUploadedFileView :application_config="application_config" :gkill_api="gkill_api"
          :app_content_height="app_content_height" :app_content_width="app_content_width"
          :uploaded_kyous="uploaded_kyous"
          v-on="crudRelayHandlers" />
        </v-card>
</div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import type { DecideRelatedTimeUploadedFileDialogEmits } from './decide-related-time-uploaded-file-dialog-emits'
import type { DecideRelatedTimeUploadedFileDialogProps } from './decide-related-time-uploaded-file-dialog-props'
import DecideRelatedTimeUploadedFileView from '../views/decide-related-time-uploaded-file-view.vue'
import { i18n } from '@/i18n'
import { useDecideRelatedTimeUploadedFileDialog } from '@/classes/use-decide-related-time-uploaded-file-dialog'

const props = defineProps<DecideRelatedTimeUploadedFileDialogProps>()
const emits = defineEmits<DecideRelatedTimeUploadedFileDialogEmits>()
const { is_show_dialog, crudRelayHandlers, ui, show, hide } = useDecideRelatedTimeUploadedFileDialog({ props, emits })
defineExpose({ show, hide })
</script>

