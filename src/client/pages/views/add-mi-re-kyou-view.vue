<template>
    <v-card class="pa-2" variant="flat">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("ADD_MI_REKYOU_TITLE") }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="add_notification()" :disabled="is_requested_submit">{{
                        i18n.global.t("ADD_NOTIFICATION_TITLE") }}</v-btn>
                </v-col>
            </v-row>
        </v-card-title>
        <table>
            <tbody>
                <tr>
                    <td>
                        <v-select class="select" v-model="mi_board_name" :items="mi_board_names"
                            :readonly="is_requested_submit" />
                    </td>
                    <td>
                        <v-tooltip :text="i18n.global.t('TOOLTIP_ADD_BOARD')">
                            <template v-slot:activator="{ props }">
                                <v-btn v-bind="props" color="secondary" class="pt-1" @click="show_new_board_name_dialog()"
                                    icon="mdi-plus" dark size="small" :disabled="is_requested_submit"></v-btn>
                            </template>
                        </v-tooltip>
                    </td>
                </tr>
            </tbody>
        </table>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <table>
                    <tbody>
                        <tr>
                            <td>
                                <v-menu :disabled="is_requested_submit" v-model="show_start_date_menu" :close-on-content-click="false"
                                    transition="scale-transition" offset-y min-width="auto">
                                    <template #activator="{ props }">
                                        <v-text-field v-model="mi_estimate_start_date_string"
                                            :label="i18n.global.t('MI_START_DATE_TITLE')" readonly v-bind="props"
                                            min-width="120" />
                                    </template>
                                    <v-date-picker v-model="mi_estimate_start_date_typed"
                                        @update:model-value="show_start_date_menu = false" locale="ja-JP" />
                                </v-menu>
                            </td>
                            <td>
                                <v-menu :disabled="is_requested_submit" v-model="show_start_time_menu" :close-on-content-click="false"
                                    transition="scale-transition" offset-y min-width="auto">
                                    <template #activator="{ props }">
                                        <v-text-field v-model="mi_estimate_start_time_string"
                                            :label="i18n.global.t('MI_START_TIME_TITLE')" min-width="120" readonly
                                            v-bind="props" />
                                    </template>
                                    <v-time-picker v-model="mi_estimate_start_time_string" format="24hr"
                                        @update:minute="show_start_time_menu = false" />
                                </v-menu>
                            </td>
                        </tr>
                    </tbody>
                </table>
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <table class="gkill-field-side-buttons">
                    <tbody>
                        <tr>
                            <td>
                                <v-btn dark color="secondary" @click="clear_estimate_start_date_time()"
                                    :disabled="is_requested_submit">{{ i18n.global.t("CLEAR_TITLE") }}</v-btn>
                            </td>
                            <td>
                                <v-btn dark color="primary" @click="now_to_estimate_start_date_time()"
                                    :disabled="is_requested_submit">{{ i18n.global.t("CURRENT_DATE_TIME_TITLE") }}</v-btn>
                            </td>
                        </tr>
                    </tbody>
                </table>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <table>
                    <tbody>
                        <tr>
                            <td>
                                <v-menu :disabled="is_requested_submit" v-model="show_end_date_menu" :close-on-content-click="false"
                                    transition="scale-transition" offset-y min-width="auto">
                                    <template #activator="{ props }">
                                        <v-text-field v-model="mi_estimate_end_date_string"
                                            :label="i18n.global.t('MI_END_DATE_TITLE')" readonly v-bind="props"
                                            min-width="120" />
                                    </template>
                                    <v-date-picker v-model="mi_estimate_end_date_typed"
                                        @update:model-value="show_end_date_menu = false" locale="ja-JP" />
                                </v-menu>
                            </td>
                            <td>
                                <v-menu :disabled="is_requested_submit" v-model="show_end_time_menu" :close-on-content-click="false"
                                    transition="scale-transition" offset-y min-width="auto">
                                    <template #activator="{ props }">
                                        <v-text-field v-model="mi_estimate_end_time_string"
                                            :label="i18n.global.t('MI_END_TIME_TITLE')" min-width="120" readonly
                                            v-bind="props" />
                                    </template>
                                    <v-time-picker v-model="mi_estimate_end_time_string" format="24hr"
                                        @update:minute="show_end_time_menu = false" />
                                </v-menu>
                            </td>
                        </tr>
                    </tbody>
                </table>
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <table class="gkill-field-side-buttons">
                    <tbody>
                        <tr>
                            <td>
                                <v-btn dark color="secondary" @click="clear_estimate_end_date_time()"
                                    :disabled="is_requested_submit">{{ i18n.global.t("CLEAR_TITLE") }}</v-btn>
                            </td>
                            <td>
                                <v-btn dark color="primary" @click="now_to_estimate_end_date_time()"
                                    :disabled="is_requested_submit">{{ i18n.global.t("CURRENT_DATE_TIME_TITLE") }}</v-btn>
                            </td>
                        </tr>
                    </tbody>
                </table>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <table>
                    <tbody>
                        <tr>
                            <td>
                                <v-menu :disabled="is_requested_submit" v-model="show_limit_date_menu" :close-on-content-click="false"
                                    transition="scale-transition" offset-y min-width="auto">
                                    <template #activator="{ props }">
                                        <v-text-field v-model="mi_limit_date_string"
                                            :label="i18n.global.t('MI_LIMIT_DATE_TITLE')" readonly v-bind="props"
                                            min-width="120" />
                                    </template>
                                    <v-date-picker v-model="mi_limit_date_typed"
                                        @update:model-value="show_limit_date_menu = false" locale="ja-JP" />
                                </v-menu>
                            </td>
                            <td>
                                <v-menu :disabled="is_requested_submit" v-model="show_limit_time_menu" :close-on-content-click="false"
                                    transition="scale-transition" offset-y min-width="auto">
                                    <template #activator="{ props }">
                                        <v-text-field v-model="mi_limit_time_string"
                                            :label="i18n.global.t('MI_LIMIT_TIME_TITLE')" min-width="120" readonly
                                            v-bind="props" />
                                    </template>
                                    <v-time-picker v-model="mi_limit_time_string" format="24hr"
                                        @update:minute="show_limit_time_menu = false" />
                                </v-menu>
                            </td>
                        </tr>
                    </tbody>
                </table>
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <table class="gkill-field-side-buttons">
                    <tbody>
                        <tr>
                            <td>
                                <v-btn dark color="secondary" @click="clear_limit_date_time()"
                                    :disabled="is_requested_submit">{{ i18n.global.t("CLEAR_TITLE") }}</v-btn>
                            </td>
                            <td>
                                <v-btn dark color="primary" @click="now_to_limit_date_time()"
                                    :disabled="is_requested_submit">{{ i18n.global.t("CURRENT_DATE_TIME_TITLE") }}</v-btn>
                            </td>
                        </tr>
                    </tbody>
                </table>
            </v-col>
        </v-row>
        <v-row v-for="notification, index in notifications" :key="notification.id" class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-row class="pa-0 ma-0">
                    <v-col cols="auto" class="pa-0 ma-0">
                        <div>{{ i18n.global.t("NOTIFICATION_TITLE") }}</div>
                    </v-col>
                    <v-spacer />
                    <v-col cols="auto" class="pa-0 ma-0">
                        <v-btn class="rounded-sm mx-auto" icon @click.prevent="delete_notification(index)"
                            :disabled="is_requested_submit">
                            <v-icon>mdi-close</v-icon>
                        </v-btn>
                    </v-col>
                </v-row>
                <v-row class="pa-0 ma-0">
                    <v-col cols="auto" class="pa-0 ma-0">
                        <AddNotificationForAddMiView :application_config="application_config" :gkill_api="gkill_api"
                            :enable_context_menu="false" :enable_dialog="true" :highlight_targets="[]" :kyou="kyou"
                            :default_notification="notification" :is_readonly="is_requested_submit"
                            ref="add_notification_views" v-on="crudRelayHandlers" />
                    </v-col>
                </v-row>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0 flex-row-reverse gkill-dialog-actions">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="() => save()" :disabled="is_requested_submit">{{
                    i18n.global.t("SAVE_TITLE") }}</v-btn>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="secondary" @click="reset()" :disabled="is_requested_submit">{{
                    i18n.global.t("RESET_TITLE") }}</v-btn>
            </v-col>
        </v-row>
        <v-checkbox v-model="show_kyou" :readonly="is_requested_submit"
            :label="i18n.global.t('SHOW_TARGET_KYOU_TITLE')" hide-details color="primary" />
        <v-card v-if="show_kyou">
            <KyouView :application_config="application_config" :gkill_api="gkill_api"
                :is_image_request_to_thumb_size="false" :highlight_targets="[kyou.generate_info_identifier()]"
                :is_image_view="false" :kyou="kyou" :show_checkbox="false" :show_content_only="false"
                :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                :show_mi_limit_time="true" :show_timeis_elapsed_time="true" :show_timeis_plaing_end_button="true"
                :height="'unset'" :width="'100%'" :enable_context_menu="false" :enable_dialog="enable_dialog"
                :is_readonly_mi_check="true" :show_attached_timeis="true" :show_rep_name="true"
                :force_show_latest_kyou_info="true" :show_update_time="false" :show_related_time="true"
                :show_attached_tags="true" :show_attached_texts="true" :show_attached_notifications="true"
                v-on="crudRelayHandlers" />
        </v-card>
        <NewBoardNameDialog :application_config="application_config" :gkill_api="gkill_api"
            v-on="crudRelayHandlers"
            @setted_new_board_name="(board_name: string) => update_board_name(board_name)"
            ref="new_board_name_dialog" />
        <ConfirmUnknownMiBoardDialog :unknown_mi_boards="unknown_mi_boards"
            :is_requested_submit="is_requested_submit"
            @requested_confirm="confirm_save()" @requested_cancel="cancel_save()"
            ref="confirm_unknown_mi_board_dialog" />
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import AddNotificationForAddMiView from './add-notification-for-add-mi-view.vue'
import KyouView from './kyou-view.vue'
import NewBoardNameDialog from '../dialogs/new-board-name-dialog.vue'
import ConfirmUnknownMiBoardDialog from '../dialogs/confirm-unknown-mi-board-dialog.vue'
import type { AddMiReKyouViewProps } from './add-mi-re-kyou-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { VDatePicker } from 'vuetify/components'
import { VTimePicker } from 'vuetify/components'
import { useAddMiReKyouView } from '@/classes/use-add-mi-re-kyou-view'

const props = defineProps<AddMiReKyouViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    // Template refs
    new_board_name_dialog,
    confirm_unknown_mi_board_dialog,

    // Confirm unknown mi board
    unknown_mi_boards,
    cancel_save,
    confirm_save,
    add_notification_views,

    // State
    is_requested_submit,
    show_kyou,
    notifications,
    mi_board_names,
    mi_board_name,
    mi_estimate_start_date_typed,
    mi_estimate_start_date_string,
    mi_estimate_start_time_string,
    mi_estimate_end_date_typed,
    mi_estimate_end_date_string,
    mi_estimate_end_time_string,
    mi_limit_date_typed,
    mi_limit_date_string,
    mi_limit_time_string,
    show_start_date_menu,
    show_start_time_menu,
    show_end_date_menu,
    show_end_time_menu,
    show_limit_date_menu,
    show_limit_time_menu,

    // Business logic / template handlers
    update_board_name,
    show_new_board_name_dialog,
    clear_estimate_start_date_time,
    clear_estimate_end_date_time,
    clear_limit_date_time,
    now_to_estimate_start_date_time,
    now_to_estimate_end_date_time,
    now_to_limit_date_time,
    reset,
    save,
    add_notification,
    delete_notification,

    // Event relay objects
    crudRelayHandlers,
} = useAddMiReKyouView({ props, emits })
</script>
