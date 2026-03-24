<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t("ADD_KC_TITLE") }}</span>
                </v-col>
                <v-spacer />
            </v-row>
        </v-card-title>
        <v-text-field v-model="title" :label="i18n.global.t('KC_TITLE_TITLE')" autofocus
            :readonly="is_requested_submit" :rules="[(v: string) => !!v || i18n.global.t('REQUIRED_FIELD_MESSAGE')]" />
        <v-text-field type="number" v-model="num_value" :label="i18n.global.t('KC_NUM_VALUE_TITLE')"
            :readonly="is_requested_submit" />
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <table>
                    <tr>
                        <td>
                            <v-menu v-model="show_related_date_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="related_date_string" :label="i18n.global.t('KC_DATE_TITLE')"
                                        readonly v-bind="props" min-width="120" />
                                </template>
                                <v-date-picker v-model="related_date_typed"
                                    @update:model-value="show_related_date_menu = false" locale="ja-JP" />
                            </v-menu>
                        </td>
                        <td>
                            <v-menu v-model="show_related_time_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="related_time_string" :label="i18n.global.t('KC_TIME_TITLE')"
                                        readonly min-width="120" v-bind="props" />
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
import type { EditKCViewProps } from './edit-kc-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { VDatePicker } from 'vuetify/components'
import { VTimePicker } from 'vuetify/components'
import { useAddKcView } from '@/classes/use-add-kc-view'

const props = defineProps<EditKCViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    // State
    is_requested_submit,
    kc,
    title,
    num_value,
    related_date_typed,
    related_date_string,
    related_time_string,
    show_related_date_menu,
    show_related_time_menu,

    // Business logic
    reset,
    reset_related_date_time,
    now_to_related_date_time,
    save,

    // Template event handlers
    onUpdateRelatedDateMenu,
    onUpdateRelatedTimeMenu,
} = useAddKcView({ props, emits })
</script>
