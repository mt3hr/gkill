<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("ADD_MI_TITLE") }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-btn dark color="primary" @click="add_notification()" :disabled="is_requested_submit">{{
                        i18n.global.t("ADD_NOTIFICATION_TITLE") }}</v-btn>
                </v-col>
            </v-row>
        </v-card-title>
        <v-text-field class="input text" type="text" v-model="mi_title" :label="i18n.global.t('MI_TITLE_TITLE')"
            autofocus :readonly="is_requested_submit" />
        <table>
            <tr>
                <td>
                    <v-select class="select" v-model="mi_board_name" :items="mi_board_names"
                        :readonly="is_requested_submit" />
                </td>
                <td>
                    <v-btn color="secondary" class="pt-1" @click="show_new_board_name_dialog()" icon="mdi-plus" dark
                        size="small" :disabled="is_requested_submit"></v-btn>
                </td>
            </tr>
        </table>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <table>
                    <tr>
                        <td>
                            <v-menu v-model="show_start_date_menu" :close-on-content-click="false"
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
                            <v-menu v-model="show_start_time_menu" :close-on-content-click="false"
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
                </table>
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <table class="pt-2">
                    <tr>
                        <td>
                            <v-btn dark color="secondary" @click="clear_estimate_start_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("CLEAR_TITLE") }}</v-btn>
                        </td>
                        <td>
                            <v-btn dark color="primary" @click="now_to_estimate_start_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("CURRENT_DATE_TIME_TITLE") }}</v-btn>
                        </td>
                    </tr>
                </table>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <table>
                    <tr>
                        <td>
                            <v-menu v-model="show_end_date_menu" :close-on-content-click="false"
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
                            <v-menu v-model="show_end_time_menu" :close-on-content-click="false"
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
                </table>
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <table class="pt-2">
                    <tr>
                        <td>
                            <v-btn dark color="secondary" @click="clear_estimate_end_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("CLEAR_TITLE") }}</v-btn>
                        </td>
                        <td>
                            <v-btn dark color="primary" @click="now_to_estimate_end_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("CURRENT_DATE_TIME_TITLE") }}</v-btn>
                        </td>
                    </tr>
                </table>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <table>
                    <tr>
                        <td>
                            <v-menu v-model="show_limit_date_menu" :close-on-content-click="false"
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
                            <v-menu v-model="show_limit_time_menu" :close-on-content-click="false"
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
                </table>
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <table class="pt-2">
                    <tr>
                        <td>
                            <v-btn dark color="secondary" @click="clear_limit_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("CLEAR_TITLE") }}</v-btn>
                        </td>
                        <td>
                            <v-btn dark color="primary" @click="now_to_limit_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("CURRENT_DATE_TIME_TITLE") }}</v-btn>
                        </td>
                    </tr>
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
                            :default_notification="notification" ref="add_notification_views"
                            v-on="crudRelayHandlers" />
                    </v-col>
                </v-row>
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
        <NewBoardNameDialog v-if="mi" :application_config="application_config" :gkill_api="gkill_api"
            v-on="crudRelayHandlers"
            @setted_new_board_name="(board_name: string) => update_board_name(board_name)"
            ref="new_board_name_dialog" />
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import AddNotificationForAddMiView from './add-notification-for-add-mi-view.vue'
import type { AddMiViewProps } from './add-mi-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import NewBoardNameDialog from '../dialogs/new-board-name-dialog.vue'
import { VDatePicker } from 'vuetify/components'
import { VTimePicker } from 'vuetify/components'
import { useAddMiView } from '@/classes/use-add-mi-view'

const props = defineProps<AddMiViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    // Template refs
    new_board_name_dialog,
    add_notification_views,

    // State
    is_requested_submit,
    kyou,
    mi,
    mi_board_names,
    mi_title,
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
    notifications,
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
} = useAddMiView({ props, emits })
</script>
