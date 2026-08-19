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

        <EditKCView :application_config="application_config" :gkill_api="gkill_api"
          :highlight_targets="[kyou.generate_info_identifier()]" :kyou="kyou"
          :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
           @requested_close_dialog="hide()" v-on="crudRelayHandlers" />
        </v-card>
</div>
    </div>
  </Teleport>
</template>
<script setup lang="ts">
import type { EditKCDialogProps } from './edit-kc-dialog-props'
import type { KyouDialogEmits } from '../views/kyou-dialog-emits'
import EditKCView from '../views/edit-kc-view.vue'
import { i18n } from '@/i18n'
import { useEditKCDialog } from '@/classes/use-edit-kc-dialog'

const props = defineProps<EditKCDialogProps>()
const emits = defineEmits<KyouDialogEmits>()
const { crudRelayHandlers, is_show_dialog, ui, show, hide } = useEditKCDialog({ props, emits })
defineExpose({ show, hide })
</script>


