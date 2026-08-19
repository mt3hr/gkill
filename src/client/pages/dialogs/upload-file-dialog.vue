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

        <UploadFileView :app_content_height="app_content_height" :app_content_width="app_content_width"
          :application_config="application_config" :gkill_api="gkill_api"
          v-on="crudRelayHandlers" />
        </v-card>
</div>
    </div>
  </Teleport>
</template>

<script lang="ts" setup>
import type { UploadFileDialogProps } from './upload-file-dialog-props'
import UploadFileView from '../views/upload-file-view.vue'
import type { KyouViewEmits } from '../views/kyou-view-emits';
import { i18n } from '@/i18n'
import { useUploadFileDialog } from '@/classes/use-upload-file-dialog'

const props = defineProps<UploadFileDialogProps>()
const emits = defineEmits<KyouViewEmits>()
const { is_show_dialog, ui, crudRelayHandlers, show, hide } = useUploadFileDialog({ props, emits })
defineExpose({ show, hide })
</script>

