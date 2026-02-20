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
        <v-card class="edit_ryuu_dialog_view">
          <RyuuListView v-model="model_value" :application_config="application_config" :gkill_api="gkill_api"
            :editable="true" :find_kyou_query_default="new FindKyouQuery()" :target_kyou="new Kyou()"
            @requested_apply_ryuu_struct="(...ryuu_data: any[]) => { emits('requested_apply_ryuu_struct', ryuu_data[0]) }"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_close_dialog="hide()" />
        </v-card>
      </div>
    </div>
  </Teleport>
</template>
<script lang="ts" setup>
import { nextTick, type Ref, ref } from 'vue'
import Dnote from '../views/dnote-view.vue'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import RyuuListView from '../views/ryuu-list-view.vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import type { EditRyuuDialogEmits } from './edit-ryuu-dialog-emits'
import type { EditRyuuDialogProps } from './edit-ryuu-dialog-props'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { Kyou } from "@/classes/datas/kyou";
const dnote_view = ref<InstanceType<typeof Dnote> | null>(null);

const model_value = defineModel<ApplicationConfig>()
defineProps<EditRyuuDialogProps>()
const emits = defineEmits<EditRyuuDialogEmits>()
defineExpose({ show, hide })

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { i18n } from '@/i18n'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
import { useFloatingDialog } from "@/classes/use-floating-dialog"
const ui = useFloatingDialog("", {
  centerMode: "always",
})


async function show(): Promise<void> {
  is_show_dialog.value = true
  nextTick(() => dnote_view.value?.reload([], new FindKyouQuery()))
}
async function hide(): Promise<void> {
  is_show_dialog.value = false
}
</script>

<style scoped lang="css">
.edit_ryuu_dialog_view {
  overflow-x: scroll;
}
</style>