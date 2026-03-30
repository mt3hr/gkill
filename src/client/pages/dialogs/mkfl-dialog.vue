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
          @deleted_kyou="(deleted_kyou: Kyou) => { emits('deleted_kyou', deleted_kyou) }"
          @deleted_tag="(deleted_tag: Tag) => emits('deleted_tag', deleted_tag)"
          @deleted_text="(deleted_text: Text) => emits('deleted_text', deleted_text)"
          @deleted_notification="(deleted_notification: Notification) => emits('deleted_notification', deleted_notification)"
          @registered_kyou="(registered_kyou: Kyou) => { emits('registered_kyou', registered_kyou) }"
          @registered_tag="(registered_tag: Tag) => emits('registered_tag', registered_tag)"
          @registered_text="(registered_text: Text) => emits('registered_text', registered_text)"
          @registered_notification="(registered_notification: Notification) => emits('registered_notification', registered_notification)"
          @updated_kyou="(updated_kyou: Kyou) => { emits('updated_kyou', updated_kyou) }"
          @updated_tag="(updated_tag: Tag) => emits('updated_tag', updated_tag)"
          @updated_text="(updated_text: Text) => emits('updated_text', updated_text)"
          @updated_notification="(updated_notification: Notification) => emits('updated_notification', updated_notification)"
          @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
          @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)" />
        <HelpDialog screen_name="mkfl" ref="help_dialog" />
      </div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import { ref, type ComponentPublicInstance } from 'vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { MKFLDialogEmits } from './mkfl-dialog-emits'
import type { MKFLDialogProps } from './mkfl-dialog-props'
import MKFLView from '../views/mkfl-view.vue'
import HelpDialog from './help-dialog.vue'
import type { Kyou } from "@/classes/datas/kyou"
import type { Tag } from "@/classes/datas/tag"
import type { Text } from "@/classes/datas/text"
import type { Notification } from "@/classes/datas/notification"
import { useMKFLDialog } from '@/classes/use-mkfl-dialog'
import { i18n } from '@/i18n'

const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)

const props = defineProps<MKFLDialogProps>()
const emits = defineEmits<MKFLDialogEmits>()

const { is_show_dialog, ui, dialog_body_ref, view_width, view_height, show, hide } = useMKFLDialog({ props, emits })

defineExpose({ show, hide })
</script>
