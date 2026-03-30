<template>
    <div style="position: relative; min-height: 100px;">
        <v-overlay v-model="is_loading" class="align-center justify-center" contained persistent>
            <v-progress-circular indeterminate color="primary" />
        </v-overlay>
    <v-card v-if="cloned_kyou.typed_urlog" class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("EDIT_URLOG_TITLE") }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" :label="i18n.global.t('SHOW_TARGET_KYOU_TITLE')" hide-details
                        color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
        <v-text-field class="input text" type="text" v-model="url" :label="i18n.global.t('URL_TITLE')" autofocus :readonly="is_requested_submit" />
        <v-text-field class="input text" type="text" v-model="title" :label="i18n.global.t('URLOG_TITLE_TITLE')" :readonly="is_requested_submit" />
        <v-checkbox v-model="re_get_urlog_content" :label="i18n.global.t('URLOG_REGET_TITLE')" hide-details color="primary" />
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <table>
                    <tr>
                        <td>
                            <v-menu v-model="show_related_date_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="related_date_string"
                                        :label="i18n.global.t('URLOG_DATE_TITLE')" readonly v-bind="props"
                                        min-width="120" />
                                </template>
                                <v-date-picker v-model="related_date_typed"
                                    @update:model-value="show_related_date_menu = false" locale="ja-JP" />
                            </v-menu>
                        </td>
                        <td>
                            <v-menu v-model="show_related_time_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="related_time_string"
                                        :label="i18n.global.t('URLOG_TIME_TITLE')" readonly min-width="120"
                                        v-bind="props" />
                                </template>
                                <v-time-picker v-model="related_time_string" format="24hr"
                                    @update:minute="show_related_time_menu = false" />
                            </v-menu>
                        </td>
                    </tr>
                </table>
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <table class="pt-2">
                    <tr>
                        <td>
                            <v-btn dark color="secondary" @click="reset_related_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("RESET_TITLE") }}</v-btn>
                        </td>
                        <td>
                            <v-btn dark color="primary" @click="now_to_related_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("CURRENT_DATE_TIME_TITLE") }}</v-btn>
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
            <KyouView :application_config="application_config" :gkill_api="gkill_api" :show_timeis_elapsed_time="true" :is_image_request_to_thumb_size="false"
                :show_timeis_plaing_end_button="true" :highlight_targets="highlight_targets" :is_image_view="false"
                :kyou="cloned_kyou" :show_checkbox="false" :show_content_only="false"
                :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                :show_mi_limit_time="true" :show_urlog_plaing_end_button="true" :height="'100%'" :width="'100%'"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" :is_readonly_mi_check="true"
                :show_rep_name="true" :force_show_latest_kyou_info="true" :show_attached_timeis="true"
                :show_attached_tags="true" :show_attached_texts="true" :show_attached_notifications="true"
                :show_update_time="false" :show_related_time="true"
                @deleted_kyou="crudRelayHandlers['deleted_kyou']"
                @deleted_tag="crudRelayHandlers['deleted_tag']"
                @deleted_text="crudRelayHandlers['deleted_text']"
                @deleted_notification="crudRelayHandlers['deleted_notification']"
                @registered_kyou="crudRelayHandlers['registered_kyou']"
                @registered_tag="crudRelayHandlers['registered_tag']"
                @registered_text="crudRelayHandlers['registered_text']"
                @registered_notification="crudRelayHandlers['registered_notification']"
                @updated_kyou="crudRelayHandlers['updated_kyou']"
                @updated_tag="crudRelayHandlers['updated_tag']"
                @updated_text="crudRelayHandlers['updated_text']"
                @updated_notification="crudRelayHandlers['updated_notification']"
                @received_errors="crudRelayHandlers['received_errors']"
                @received_messages="crudRelayHandlers['received_messages']"
                @requested_reload_kyou="crudRelayHandlers['requested_reload_kyou']"
                @requested_reload_list="crudRelayHandlers['requested_reload_list']"
                @requested_update_check_kyous="crudRelayHandlers['requested_update_check_kyous']" />
        </v-card>
    </v-card>
    </div>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { EditURLogViewProps } from './edit-ur-log-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import KyouView from './kyou-view.vue'
import { VDatePicker } from 'vuetify/components'
import { VTimePicker } from 'vuetify/components'
import { useEditUrLogView } from '@/classes/use-edit-ur-log-view'

const props = defineProps<EditURLogViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    is_loading,
    is_requested_submit,
    cloned_kyou,
    title,
    url,
    related_date_typed,
    related_date_string,
    related_time_string,
    re_get_urlog_content,
    show_kyou,
    show_related_date_menu,
    show_related_time_menu,
    save,
    reset,
    now_to_related_date_time,
    reset_related_date_time,
    crudRelayHandlers,
} = useEditUrLogView({ props, emits })

defineExpose({
    save,
    reset,
})
</script>
