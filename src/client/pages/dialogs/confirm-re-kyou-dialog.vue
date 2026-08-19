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
       <ConfirmReKyouView :application_config="application_config" :gkill_api="gkill_api"
          :highlight_targets="[kyou.generate_info_identifier()]" :kyou="kyou"
          :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
          @requested_close_dialog="hide()" v-on="crudRelayHandlers" />
        </v-card>
</div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import type { ConfirmReKyouDialogProps } from './confirm-re-kyou-dialog-props'
import type { KyouDialogEmits } from '../views/kyou-dialog-emits'
import ConfirmReKyouView from '../views/confirm-re-kyou-view.vue'
import { i18n } from '@/i18n'
import { useConfirmReKyouDialog } from '@/classes/use-confirm-re-kyou-dialog'

defineProps<ConfirmReKyouDialogProps>()
const emits = defineEmits<KyouDialogEmits>()
const { is_show_dialog, ui, show, hide, crudRelayHandlers } = useConfirmReKyouDialog({ emits })
defineExpose({ show, hide })
</script>


