<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t('ADD_NLOG_TITLE') }}</span>
                </v-col>
            </v-row>
        </v-card-title>
        <v-text-field v-if="nlog" v-model="nlog_title_value" :label="i18n.global.t('NLOG_TITLE_TITLE')" autofocus
            :readonly="is_requested_submit" :rules="[(v: string) => !!v || i18n.global.t('REQUIRED_FIELD_MESSAGE')]" />
        <v-text-field v-if="nlog" v-model="nlog_shop_value" :label="i18n.global.t('NLOG_SHOP_NAME_TITLE')"
            :readonly="is_requested_submit" />
        <v-text-field v-if="nlog" v-model="nlog_amount_value" type="number" :label="i18n.global.t('NLOG_AMOUNT_TITLE')"
            :readonly="is_requested_submit" />
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <table>
                    <tr>
                        <td>
                            <v-menu v-model="show_related_date_menu" :close-on-content-click="false"
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
                            <v-menu v-model="show_related_time_menu" :close-on-content-click="false"
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
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { AddNlogViewProps } from './add-nlog-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { VDatePicker } from 'vuetify/components'
import { VTimePicker } from 'vuetify/components'
import { useAddNlogView } from '@/classes/use-add-nlog-view'

const props = defineProps<AddNlogViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    // State
    is_requested_submit,
    nlog,
    nlog_title_value,
    nlog_amount_value,
    nlog_shop_value,
    related_date_typed,
    related_date_string,
    related_time_string,
    show_related_date_menu,
    show_related_time_menu,

    // Business logic / template handlers
    save,
    reset_related_date_time,
    now_to_related_date_time,
    reset,
} = useAddNlogView({ props, emits })
</script>
