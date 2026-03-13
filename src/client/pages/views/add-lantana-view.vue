<template>
    <v-card class="pa-2">
        <v-card-title>
            <v-row class="pa-0 ma-0">
                <v-col cols="auto" class="pa-0 ma-0">
                    <span>{{ i18n.global.t('ADD_LANTANA_TITLE') }}</span>
                </v-col>
                <v-spacer />
            </v-row>
        </v-card-title>
        <LantanaFlowersView :application_config="application_config" :gkill_api="gkill_api" :mood="mood"
            :editable="!is_requested_submit" ref="edit_lantana_flowers" />
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <table>
                    <tr>
                        <td>
                            <v-menu v-model="show_related_date_menu" :close-on-content-click="false"
                                transition="scale-transition" offset-y min-width="auto">
                                <template #activator="{ props }">
                                    <v-text-field v-model="related_date_string"
                                        :label="i18n.global.t('LANTANA_DATE_TITLE')" readonly v-bind="props"
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
                                        :label="i18n.global.t('LANTANA_TIME_TITLE')" min-width="120" readonly
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
import type { KyouViewEmits } from './kyou-view-emits'
import LantanaFlowersView from './lantana-flowers-view.vue'
import type { AddLantanaViewProps } from './add-lantana-view-props'
import { VDatePicker } from 'vuetify/components'
import { VTimePicker } from 'vuetify/components'
import { useAddLantanaView } from '@/classes/use-add-lantana-view'

const props = defineProps<AddLantanaViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    edit_lantana_flowers,
    is_requested_submit,
    mood,
    related_date_typed,
    related_date_string,
    related_time_string,
    show_related_date_menu,
    show_related_time_menu,
    save,
    reset_related_date_time,
    now_to_related_date_time,
    reset,
} = useAddLantanaView({ props, emits })
</script>
