<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("EDIT_NOTIFICATION_TITLE") }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" :label="i18n.global.t('SHOW_TARGET_KYOU_TITLE')" hide-details
                        color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
        <v-textarea v-model="content_value" :label="i18n.global.t('NOTIFICATION_CONTENT_TITLE')" autofocus
            :readonly="is_requested_submit" />
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <table>
                    <tr>
                        <td>
                            <v-menu v-model="show_notification_date_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="notification_date_string"
                                        :label="i18n.global.t('NOTIFICATION_DATE_TITLE')" readonly v-bind="props"
                                        min-width="120" />
                                </template>
                                <v-date-picker v-model="notification_date_typed"
                                    @update:model-value="show_notification_date_menu = false" locale="ja-JP" />
                            </v-menu>
                        </td>
                        <td>
                            <v-menu v-model="show_notification_time_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="notification_time_string"
                                        :label="i18n.global.t('NOTIFICATION_TIME_TITLE')" readonly min-width="120"
                                        v-bind="props" />
                                </template>
                                <v-time-picker v-model="notification_time_string" format="24hr"
                                    @update:minute="show_notification_time_menu = false" />
                            </v-menu>
                        </td>
                        <td>
                            <v-btn dark color="secondary" @click="reset_notification_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("RESET_TITLE") }}</v-btn>
                        </td>
                    </tr>
                </table>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="secondary" @click="reset()" :disabled="is_requested_submit">{{
                    i18n.global.t("RESET_TITLE")
                    }}</v-btn>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="() => save()" :disabled="is_requested_submit">{{
                    i18n.global.t("SAVE_TITLE")
                    }}</v-btn>
            </v-col>
        </v-row>
        <v-card v-if="show_kyou">
            <KyouView :application_config="application_config" :gkill_api="gkill_api" :is_image_request_to_thumb_size="false"
                :highlight_targets="highlight_targets" :is_image_view="false" :kyou="cloned_kyou"
                :show_checkbox="false" :show_content_only="false"
                :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                :show_mi_limit_time="true" :show_timeis_elapsed_time="true" :show_timeis_plaing_end_button="true"
                :height="'100%'" :width="'100%'" :enable_context_menu="enable_context_menu"
                :enable_dialog="enable_dialog" :is_readonly_mi_check="true" :show_attached_timeis="true"
                :show_rep_name="true" :force_show_latest_kyou_info="true" :show_update_time="false"
                :show_related_time="true" :show_attached_tags="true" :show_attached_texts="true"
                :show_attached_notifications="true"
                v-on="crudRelayHandlers" />
        </v-card>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { EditNotificationViewProps } from './edit-notification-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import KyouView from './kyou-view.vue'
import { VDatePicker } from 'vuetify/components'
import { VTimePicker } from 'vuetify/components'
import { useEditNotificationView } from '@/classes/use-edit-notification-view'

const props = defineProps<EditNotificationViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    // State
    is_requested_submit,
    cloned_kyou,
    content_value,
    notification_date_typed,
    notification_date_string,
    notification_time_string,
    show_kyou,
    show_notification_date_menu,
    show_notification_time_menu,

    // Business logic
    reset,
    reset_notification_date_time,
    save,

    // Event relay objects
    crudRelayHandlers,
} = useEditNotificationView({ props, emits })
</script>
