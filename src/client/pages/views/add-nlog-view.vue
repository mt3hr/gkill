<template>
    <v-card class="pa-2" variant="flat">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t('ADD_NLOG_TITLE') }}</span>
                </v-col>
            </v-row>
        </v-card-title>
        <!-- メモ帳（KFTL）の支出と同じ並び: 店名を1つ書き、品名と金額の組を行で増やす。1行が1件の支出になる -->
        <v-text-field v-if="nlog" v-model="nlog_shop_value" :label="i18n.global.t('NLOG_SHOP_NAME_TITLE')"
            :readonly="is_requested_submit" />
        <v-row v-for="(row, index) in nlog_rows" :key="row.row_key" class="pa-0 ma-0 add_nlog_row">
            <v-col class="pa-0 ma-0">
                <v-text-field v-model="row.title" :label="i18n.global.t('NLOG_TITLE_TITLE')"
                    :readonly="is_requested_submit"
                    :rules="[(v: string) => !!v || i18n.global.t('REQUIRED_FIELD_MESSAGE')]" />
            </v-col>
            <v-col cols="4" class="pa-0 ma-0 pl-2">
                <v-text-field v-model="row.amount" type="number" :label="i18n.global.t('NLOG_AMOUNT_TITLE')"
                    :readonly="is_requested_submit" />
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0 gkill-field-side-buttons">
                <v-btn icon="mdi-delete" size="small" variant="text" color="secondary"
                    :title="i18n.global.t('DELETE_TITLE')" :disabled="!can_delete_row || is_requested_submit"
                    @click="delete_row(index)" />
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn color="primary" variant="text" prepend-icon="mdi-plus" :disabled="is_requested_submit"
                    @click="add_row()">{{ i18n.global.t('ADD_TITLE') }}</v-btn>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <table>
                    <tbody>
                        <tr>
                            <td>
                                <v-menu :disabled="is_requested_submit" v-model="show_related_date_menu" :close-on-content-click="false"
                                    transition="scale-transition" offset-y min-width="auto">
                                    <template #activator="{ props }">
                                        <v-text-field v-model="related_date_string"
                                            :label="i18n.global.t('NLOG_DATE_TITLE')" readonly v-bind="props"
                                            min-width="120" />
                                    </template>
                                    <v-date-picker v-model="related_date_typed"
                                        @update:model-value="show_related_date_menu = false" locale="ja-JP" />
                                </v-menu>
                            </td>
                            <td>
                                <v-menu :disabled="is_requested_submit" v-model="show_related_time_menu" :close-on-content-click="false"
                                    transition="scale-transition" offset-y min-width="auto">
                                    <template #activator="{ props }">
                                        <v-text-field v-model="related_time_string"
                                            :label="i18n.global.t('NLOG_TIME_TITLE')" min-width="120" readonly
                                            v-bind="props" />
                                    </template>
                                    <v-time-picker v-model="related_time_string" format="24hr"
                                        @update:minute="show_related_time_menu = false" />
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
                    </tbody>
                </table>
            </v-col>
        </v-row>
        <EditKyouTagsView :application_config="application_config" :gkill_api="gkill_api" :kyou="null"
            :is_readonly="is_requested_submit" ref="kyou_tags_view" v-on="crudRelayHandlers" />
        <v-row class="pa-0 ma-0 flex-row-reverse gkill-dialog-actions">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="() => save()" :disabled="is_requested_submit">{{
                    i18n.global.t("SAVE_TITLE")
                    }}</v-btn>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="secondary" @click="reset()" :disabled="is_requested_submit">{{
                    i18n.global.t("RESET_TITLE")
                    }}</v-btn>
            </v-col>
        </v-row>
        <ConfirmUnknownTagDialog :unknown_tags="unknown_tags" :is_requested_submit="is_requested_submit"
            @requested_confirm="confirm_save()" @requested_cancel="cancel_save()"
            ref="confirm_unknown_tag_dialog" />
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { AddNlogViewProps } from './add-nlog-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import EditKyouTagsView from './edit-kyou-tags-view.vue'
import ConfirmUnknownTagDialog from '../dialogs/confirm-unknown-tag-dialog.vue'
import { VDatePicker } from 'vuetify/components'
import { VTimePicker } from 'vuetify/components'
import { useAddNlogView } from '@/classes/use-add-nlog-view'

const props = defineProps<AddNlogViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    // Template refs
    kyou_tags_view,
    confirm_unknown_tag_dialog,

    // Confirm unknown tag
    unknown_tags,
    cancel_save,
    confirm_save,

    // State
    is_requested_submit,
    nlog,
    nlog_rows,
    nlog_shop_value,
    related_date_typed,
    related_date_string,
    related_time_string,
    show_related_date_menu,
    show_related_time_menu,

    // Computed
    can_delete_row,

    // Business logic / template handlers
    save,
    add_row,
    delete_row,
    reset_related_date_time,
    now_to_related_date_time,
    reset,

    // Event relay objects
    crudRelayHandlers,
} = useAddNlogView({ props, emits })
</script>
