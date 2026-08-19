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
          <v-card-title>
            <v-row class="pa-0 ma-0">
              <v-col cols="auto" class="pa-0 ma-0">
                <span>{{ i18n.global.t('NOTIFICATION_CONTEXTMENU_HISTORIES') }}</span>
              </v-col>
              <v-spacer />
              <v-col cols="auto" class="pa-0 ma-0">
                <v-checkbox v-model="show_kyou" :label="i18n.global.t('SHOW_TARGET_KYOU_TITLE')" hide-details
                  color="primary" />
              </v-col>
            </v-row>
          </v-card-title>
          <NotificationHistoriesView :application_config="application_config" :gkill_api="gkill_api"
            :notification="notification" :kyou="kyou"
            :highlight_targets="notification_highlight_targets" :enable_context_menu="enable_context_menu"
            :enable_dialog="enable_dialog" v-on="crudRelayHandlers" />
          <v-card v-if="show_kyou">
            <KyouView :application_config="application_config" :gkill_api="gkill_api"
              :is_image_request_to_thumb_size="false" :highlight_targets="notification_highlight_targets"
              :is_image_view="false" :kyou="kyou" :show_checkbox="false"
              :show_content_only="false" :show_mi_create_time="true" :show_mi_estimate_end_time="true"
              :show_mi_estimate_start_time="true" :show_mi_limit_time="true" :show_timeis_elapsed_time="true"
              :show_timeis_plaing_end_button="true" :height="'unset'" :width="'100%'"
              :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" :show_attached_timeis="true"
              :is_readonly_mi_check="true" :show_rep_name="true" :force_show_latest_kyou_info="true"
              :show_update_time="false" :show_related_time="true" :show_attached_tags="true" :show_attached_texts="true"
              :show_attached_notifications="true"
               />
          </v-card>
        </v-card>
      </div>
    </div>
  </Teleport>
</template>
<script setup lang="ts">
import NotificationHistoriesView from '../views/notification-histories-view.vue'
import type { KyouDialogEmits } from '../views/kyou-dialog-emits'
import KyouView from '../views/kyou-view.vue'
import type { NotificationHistoriesDialogProps } from './notification-histories-dialog-props'
import { i18n } from '@/i18n'
import { useNotificationHistoriesDialog } from '@/classes/use-notification-histories-dialog'

const props = defineProps<NotificationHistoriesDialogProps>()
const emits = defineEmits<KyouDialogEmits>()
const { crudRelayHandlers, notification_highlight_targets, is_show_dialog, ui, show_kyou, show, hide } = useNotificationHistoriesDialog({ props, emits })
defineExpose({ show, hide })
</script>


