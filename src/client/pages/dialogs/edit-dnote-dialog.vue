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
        <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="help_dialog?.show()" hide-details :color="'primary'" variant="flat">
          <v-icon>mdi-help-circle-outline</v-icon>
        </v-btn>
                <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details :color="'primary'" variant="flat">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </div>

      <div class="gkill-floating-dialog__body">
        <v-card class="edit_dnote_dialog_view">
          <Dnote :app_content_height="app_content_height" :app_content_width="app_content_width"
            :application_config="application_config" :gkill_api="gkill_api" :query="new FindKyouQuery()"
            :checked_kyous="[]" :editable="true"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_apply_dnote="(...dnote_data: any[]) => { emits('requested_apply_dnote', dnote_data[0]) }"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @requested_close_dialog="hide()"
            @requested_reload_application_config="(...application_config: any[]) => emits('requested_reload_application_config', application_config[0] as ApplicationConfig)"
            ref="dnote_view" />
        </v-card>
        <HelpDialog screen_name="dnote" ref="help_dialog" />
      </div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import { ref } from 'vue'
import Dnote from '../views/dnote-view.vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import HelpDialog from './help-dialog.vue'
import { type EditDnoteDialogProps } from './edit-dnote-dialog-props'
import { type EditDnoteDialogEmits } from './edit-dnote-dialog-emits'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import { useEditDnoteDialog } from '@/classes/use-edit-dnote-dialog'
import { i18n } from '@/i18n'

const dnote_view = ref<InstanceType<typeof Dnote> | null>(null);
const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)
const props = defineProps<EditDnoteDialogProps>()
const emits = defineEmits<EditDnoteDialogEmits>()

const { is_show_dialog, ui, show, hide } = useEditDnoteDialog({ props, emits, dnote_view })

defineExpose({ show, hide })
</script>