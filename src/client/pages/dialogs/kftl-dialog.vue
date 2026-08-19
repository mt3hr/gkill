<template>
  <Teleport to="body" v-if="is_show_dialog">
    <div class="gkill-float-scrim" :class="ui.isTransparent.value ? 'is-transparent' : ''" />

    <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog kftl_dialog"
      :class="ui.isTransparent.value ? 'is-transparent' : ''">
      <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
        @touchstart="ui.onHeaderPointerDown">
        <!-- タイトルは出さない（他のフローティングダイアログと揃える）。
             空にすると aria-label は useFloatingDialog のキーへフォールバックするので、
             複数枚でも "kftl dialog" / "kftl dialog 2" … と区別は付く -->
        <div class="gkill-floating-dialog__title"></div>
        <div class="gkill-floating-dialog__spacer"></div>
  <v-checkbox v-model="ui.isTransparent.value" color="white"    size="small" variant="flat"
          :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
        <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="help_dialog?.show()" hide-details :color="'primary'" variant="flat">
          <v-icon>mdi-help-circle-outline</v-icon>
        </v-btn>
        <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details :color="'primary'"
          variant="flat">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </div>

      <div class="gkill-floating-dialog__body" ref="dialog_body_ref">
        <v-card variant="flat" style="overflow: hidden">
       <KFTLView :app_content_height="view_height" :app_content_width="view_width"
          :application_config="application_config" :gkill_api="gkill_api"
          @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
          @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
          @registered_kyou="(kyou: Kyou) => emits('registered_kyou', kyou)"
          @updated_kyou="(kyou: Kyou) => emits('updated_kyou', kyou)"
          @requested_reload_list="() => emits('requested_reload_list')"
          @saved_kyou_by_kftl="(last_added_request_time: Date) => emits('saved_kyou_by_kftl', last_added_request_time)"
          ref="kftl_view" />
        </v-card>
        <HelpDialog screen_name="kftl" ref="help_dialog" />
</div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Kyou } from '@/classes/datas/kyou'
import type { KFTLDialogEmits } from './kftl-dialog-emits'
import type { KFTLDialogProps } from './kftl-dialog-props'
import KFTLView from '../views/kftl-view.vue'
import HelpDialog from './help-dialog.vue'
import { i18n } from '@/i18n'
import { useKFTLDialog } from '@/classes/use-kftl-dialog'

const props = defineProps<KFTLDialogProps>()
const emits = defineEmits<KFTLDialogEmits>()
const { kftl_view, help_dialog, dialog_body_ref, view_width, view_height, is_show_dialog, ui, show, hide } = useKFTLDialog({ props, emits })
defineExpose({ show, hide })
</script>
