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
        <v-card class="pa-2">
          <v-card-title>
            <v-row class="pa-0 ma-0">
              <v-col cols="auto" class="pa-0 ma-0">
                <span>{{ i18n.global.t('TAG_CONTEXTMENU_HISTORIES') }}</span>
              </v-col>
              <v-spacer />
              <v-col cols="auto" class="pa-0 ma-0">
                <v-checkbox v-model="show_kyou" :label="i18n.global.t('SHOW_TARGET_KYOU_TITLE')" hide-details
                  color="primary" />
              </v-col>
            </v-row>
          </v-card-title>
          <tagHistoriesView :application_config="application_config" :gkill_api="gkill_api" :tag="tag" :kyou="kyou"
            @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
            :highlight_targets="tag_highlight_targets" :enable_context_menu="enable_context_menu"
            :enable_dialog="enable_dialog"
            @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
          @focused_kyou="(kyou: Kyou) => emits('focused_kyou', kyou)"
          @clicked_kyou="(kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) }"
            @deleted_kyou="(deleted_kyou: Kyou) => emits('deleted_kyou', deleted_kyou)"
            @deleted_tag="(deleted_tag: Tag) => emits('deleted_tag', deleted_tag)"
            @deleted_text="(deleted_text: Text) => emits('deleted_text', deleted_text)"
            @deleted_notification="(deleted_notification: Notification) => emits('deleted_notification', deleted_notification)"
            @registered_kyou="(registered_kyou: Kyou) => emits('registered_kyou', registered_kyou)"
            @registered_tag="(registered_tag: Tag) => emits('registered_tag', registered_tag)"
            @registered_text="(registered_text: Text) => emits('registered_text', registered_text)"
            @registered_notification="(registered_notification: Notification) => emits('registered_notification', registered_notification)"
            @updated_kyou="(updated_kyou: Kyou) => emits('updated_kyou', updated_kyou)"
            @updated_tag="(updated_tag: Tag) => emits('updated_tag', updated_tag)"
            @updated_text="(updated_text: Text) => emits('updated_text', updated_text)"
            @updated_notification="(updated_notification: Notification) => emits('updated_notification', updated_notification)"
            @requested_reload_kyou="(kyou: Kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => emits('requested_update_check_kyous', kyous, is_checked)" />
          <v-card v-if="show_kyou">
            <KyouView :application_config="application_config" :gkill_api="gkill_api"
              :is_image_request_to_thumb_size="false" :highlight_targets="tag_highlight_targets" :is_image_view="false"
              :kyou="kyou" :show_checkbox="false" :show_content_only="false"
              :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
              :show_mi_limit_time="true" :show_timeis_elapsed_time="true" :show_timeis_plaing_end_button="true"
              :height="'100%'" :width="'100%'" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
              :show_attached_timeis="true" :is_readonly_mi_check="true" :show_rep_name="true"
              :force_show_latest_kyou_info="true" :show_update_time="false" :show_related_time="true"
              :show_attached_tags="true" :show_attached_texts="true" :show_attached_notifications="true"
              @deleted_kyou="(deleted_kyou: Kyou) => emits('deleted_kyou', deleted_kyou)"
              @deleted_tag="(deleted_tag: Tag) => emits('deleted_tag', deleted_tag)"
              @deleted_text="(deleted_text: Text) => emits('deleted_text', deleted_text)"
              @deleted_notification="(deleted_notification: Notification) => emits('deleted_notification', deleted_notification)"
              @registered_kyou="(registered_kyou: Kyou) => emits('registered_kyou', registered_kyou)"
              @registered_tag="(registered_tag: Tag) => emits('registered_tag', registered_tag)"
              @registered_text="(registered_text: Text) => emits('registered_text', registered_text)"
              @registered_notification="(registered_notification: Notification) => emits('registered_notification', registered_notification)"
              @updated_kyou="(updated_kyou: Kyou) => emits('updated_kyou', updated_kyou)"
              @updated_tag="(updated_tag: Tag) => emits('updated_tag', updated_tag)"
              @updated_text="(updated_text: Text) => emits('updated_text', updated_text)"
              @updated_notification="(updated_notification: Notification) => emits('updated_notification', updated_notification)"
              @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
              @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
          @focused_kyou="(kyou: Kyou) => emits('focused_kyou', kyou)"
          @clicked_kyou="(kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) }"
              @requested_reload_kyou="(kyou: Kyou) => emits('requested_reload_kyou', kyou)"
              @requested_reload_list="emits('requested_reload_list')"
              @requested_update_check_kyous="(kyous: Array<Kyou>, is_checked: boolean) => emits('requested_update_check_kyous', kyous, is_checked)" />
          </v-card>
        </v-card>
      </div>
    </div>
  </Teleport>
</template>
<script setup lang="ts">
import type { TagHistoriesDialogProps } from './tag-histories-dialog-props'
import type { KyouDialogEmits } from '../views/kyou-dialog-emits'
import { computed, type Ref, ref } from 'vue'
import KyouView from '../views/kyou-view.vue'
import tagHistoriesView from '../views/tag-histories-view.vue'
import type { InfoIdentifier } from '@/classes/datas/info-identifier'
import type { GkillError } from '@/classes/api/gkill-error'
import type { Kyou } from '@/classes/datas/kyou'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';

const props = defineProps<TagHistoriesDialogProps>()
const emits = defineEmits<KyouDialogEmits>()
defineExpose({ show, hide })

const tag_highlight_targets = computed<Array<InfoIdentifier>>(() => {
  const info_identifer = props.tag.generate_info_identifer()
  return [info_identifer]
})

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { i18n } from '@/i18n'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
import { useFloatingDialog } from "@/classes/use-floating-dialog"
const ui = useFloatingDialog("tag-histories-dialog", {
  centerMode: "always",
  onEscape: () => hide(),
})

const show_kyou: Ref<boolean> = ref(false)

async function show(): Promise<void> {
  is_show_dialog.value = true
}
async function hide(): Promise<void> {
  is_show_dialog.value = false
  emits('closed')
}
</script>


