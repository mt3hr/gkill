<template>
  <Teleport to="body" v-if="is_show_dialog" class="mkfl_dialog">
    <div class="gkill-float-scrim" :class="ui.isTransparent.value ? 'is-transparent' : ''" />

    <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog"
      :class="ui.isTransparent.value ? 'is-transparent' : ''">
      <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
        @touchstart="ui.onHeaderPointerDown">
        <div class="gkill-floating-dialog__title"></div>
        <div class="gkill-floating-dialog__spacer"></div>
        <v-checkbox v-model="ui.isTransparent.value" color="white" size="small" variant="flat"
          :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
        <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="help_dialog?.show()" hide-details :color="'primary'" variant="flat">
          <v-icon>mdi-help-circle-outline</v-icon>
        </v-btn>
        <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details :color="'primary'"
          variant="flat">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </div>

      <div class="gkill-floating-dialog__body" :ref="(el: Element | ComponentPublicInstance | null) => { dialog_body_ref = el as HTMLElement | null }">
        <MKFLView :app_content_height="view_height" :app_content_width="view_width"
          :application_config="application_config" :gkill_api="gkill_api"
          :is_hosted_in_dialog="true /* 呼び出し元のページが自前のFABを持っているので、内包する実行中ビューのFABは出さない */"
          v-on="crudRelayHandlers"
          @saved_kyou_by_kftl="(last_added_request_time: Date) => emits('saved_kyou_by_kftl', last_added_request_time)" />
        <HelpDialog screen_name="mkfl" ref="help_dialog" />
      </div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import { ref, type ComponentPublicInstance } from 'vue'
import type { MKFLDialogEmits } from './mkfl-dialog-emits'
import type { MKFLDialogProps } from './mkfl-dialog-props'
import MKFLView from '../views/mkfl-view.vue'
import HelpDialog from './help-dialog.vue'
import { useMKFLDialog } from '@/classes/use-mkfl-dialog'
import { i18n } from '@/i18n'

const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)

const props = defineProps<MKFLDialogProps>()
const emits = defineEmits<MKFLDialogEmits>()

const { is_show_dialog, ui, dialog_body_ref, view_width, view_height, show, hide, crudRelayHandlers } = useMKFLDialog({ props, emits })

defineExpose({ show, hide })
</script>
