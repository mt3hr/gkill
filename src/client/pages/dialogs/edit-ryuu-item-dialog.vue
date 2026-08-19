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
        <v-card variant="flat" class="pa-2">

        <EditRyuuItemView v-model="model_value" :application_config="application_config" :gkill_api="gkill_api"
          @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
          @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
          @requested_close_dialog="hide()" />
        </v-card>
        <HelpDialog screen_name="ryuu" ref="help_dialog" />
</div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import EditRyuuItemView from '../views/edit-ryuu-item-view.vue';
import HelpDialog from './help-dialog.vue'
import type EditRyuuItemDialogEmits from './edit-ryuu-item-dialog-emits';
import type EditRyuuItemDialogProps from './edit-ryuu-item-dialog-props';
import RelatedKyouQuery from '@/classes/dnote/related-kyou-query';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';
import { i18n } from '@/i18n'
import { useEditRyuuItemDialog } from '@/classes/use-edit-ryuu-item-dialog'

const model_value = defineModel<RelatedKyouQuery>()
const props = defineProps<EditRyuuItemDialogProps>()
const emits = defineEmits<EditRyuuItemDialogEmits>()
const { is_show_dialog, ui, help_dialog, show, hide } = useEditRyuuItemDialog({ props, emits })
defineExpose({ show, hide })
</script>

