<template>
    <v-card v-if="cloned_kyou.typed_mi" class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("EDIT_MI_TITLE") }}</span>
                </v-col>
                <v-spacer />
                <v-col cols="auto" class="pa-0 ma-0">
                    <v-checkbox v-model="show_kyou" :label="i18n.global.t('SHOW_TARGET_KYOU_TITLE')" hide-details
                        color="primary" />
                </v-col>
            </v-row>
        </v-card-title>
        <v-text-field class="input text" type="text" v-model="mi_title" :label="i18n.global.t('MI_TITLE_TITLE')"
            autofocus :readonly="is_requested_submit" />
        <table>
            <tr>
                <td>
                    <v-select class="select" v-model="mi_board_name" :items="mi_board_names"
                        :readonly="is_requested_submit" :label="i18n.global.t('MI_BOARD_NAME_TITLE')" />
                </td>
                <td>
                    <v-btn color="primary" @click="show_new_board_name_dialog()" icon="mdi-plus" dark size="small"
                        :disabled="is_requested_submit"></v-btn>
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
                            <v-btn dark color="secondary" @click="reset_estimate_start_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("RESET_TITLE") }}</v-btn>
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
                            <v-btn dark color="secondary" @click="reset_estimate_end_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("RESET_TITLE") }}</v-btn>
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
                            <v-btn dark color="secondary" @click="reset_limit_date_time()"
                                :disabled="is_requested_submit">{{
                                    i18n.global.t("RESET_TITLE") }}</v-btn>
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
                :show_timeis_plaing_end_button="false" :highlight_targets="highlight_targets" :is_image_view="false"
                :kyou="cloned_kyou" :show_checkbox="false" :show_content_only="false"
                :show_mi_create_time="true" :show_mi_estimate_end_time="true" :show_mi_estimate_start_time="true"
                :show_mi_limit_time="true" :show_mi_plaing_end_button="true" :height="'100%'" :width="'100%'"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" :is_readonly_mi_check="true"
                :show_rep_name="true" :force_show_latest_kyou_info="true" :show_attached_timeis="true"
                :show_attached_tags="true" :show_attached_texts="true" :show_attached_notifications="true"
                :show_update_time="false" :show_related_time="true"
                v-on="crudRelayHandlers" />
        </v-card>
        <NewBoardNameDialog v-if="cloned_kyou.typed_mi" :application_config="application_config" :gkill_api="gkill_api"
            v-on="newBoardNameDialogHandlers"
            ref="new_board_name_dialog" />
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { EditMiViewProps } from './edit-mi-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import KyouView from './kyou-view.vue'
import NewBoardNameDialog from '../dialogs/new-board-name-dialog.vue'
import { VDatePicker } from 'vuetify/components'
import { VTimePicker } from 'vuetify/components'
import { useEditMiView } from '@/classes/use-edit-mi-view'

const props = defineProps<EditMiViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    // Template refs
    new_board_name_dialog,

    // State
    is_requested_submit,
    cloned_kyou,
    show_kyou,
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
    show_start_date_menu,
    show_start_time_menu,
    show_end_date_menu,
    show_end_time_menu,
    show_limit_date_menu,
    show_limit_time_menu,

    // Business logic
    show_new_board_name_dialog,
    clear_estimate_start_date_time,
    clear_estimate_end_date_time,
    clear_limit_date_time,
    reset_estimate_start_date_time,
    reset_estimate_end_date_time,
    reset_limit_date_time,
    now_to_estimate_start_date_time,
    now_to_estimate_end_date_time,
    now_to_limit_date_time,
    reset,
    save,

    // Event relay objects
    crudRelayHandlers,
    newBoardNameDialogHandlers,
} = useEditMiView({ props, emits })
</script>
