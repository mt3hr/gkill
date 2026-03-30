<template>
    <div class="replist">
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-checkbox readonly v-model="use_rep" :label="i18n.global.t('REP_QUERY_TITLE')" hide-details />
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pb-0 mb-0 pr-0 pt-2">
                <v-btn dark color="secondary" @click="emits('request_clear_rep_query')" hide-details>{{
                    i18n.global.t("CLEAR_TITLE") }}</v-btn>
            </v-col>
        </v-row>
        <v-tabs v-show="use_rep" v-model="tab">
            <v-tab key="summary">{{ i18n.global.t('REP_QUERY_SUMMARY_TITLE') }}</v-tab>
            <v-tab key="detail">{{ i18n.global.t('REP_QUERY_DETAIL_TITLE') }}</v-tab>
        </v-tabs>
        <v-window v-model="tab">
            <v-window-item key="summary" :eager="true">
                <h2>{{ i18n.global.t("REP_QUERY_DEVIUCES_TITLE") }}</h2>
                <table class="devicelist">
                    <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
                        :is_editable="false" :is_root="true" :is_show_checkbox="true" :is_open="false"
                        :struct_obj="cloned_application_config.device_struct"
                        @requested_update_check_state="update_devices"
                        @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
                        @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
                        @clicked_items="clicked_devices" ref="foldable_struct_devices" />
                </table>
                <h2>{{ i18n.global.t("REP_QUERY_TYPES_TITLE") }}</h2>
                <table class="typelist">
                    <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
                        :is_editable="false" :is_root="true" :is_show_checkbox="true" :is_open="true"
                        :struct_obj="cloned_application_config.rep_type_struct"
                        @requested_update_check_state="update_rep_types"
                        @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
                        @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
                        @clicked_items="clicked_rep_types" ref="foldable_struct_rep_types" />
                </table>
            </v-window-item>
            <v-window-item key="detail" :eager="true">
                <h2>{{ i18n.global.t("REP_QUERY_REPS_TITLE") }}</h2>
                <table>
                    <FoldableStruct :application_config="application_config" :folder_name="''" :gkill_api="gkill_api"
                        :is_editable="false" :is_root="true" :is_show_checkbox="true" :is_open="true"
                        :struct_obj="cloned_application_config.rep_struct"
                        @requested_update_check_state="update_reps"
                        @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
                        @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
                        @clicked_items="clicked_reps" ref="foldable_struct_reps" />
                </table>
            </v-window-item>
        </v-window>
    </div>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import FoldableStruct from './foldable-struct.vue'
import type { RepQueryEmits } from './rep-query-emits'
import type { RepQueryProps } from './rep-query-props'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { useRepQuery } from '@/classes/use-rep-query'

const props = defineProps<RepQueryProps>()
const emits = defineEmits<RepQueryEmits>()

const {
    // Template refs
    foldable_struct_reps,
    foldable_struct_devices,
    foldable_struct_rep_types,

    // State
    cloned_application_config,
    tab,
    use_rep,

    // Exposed methods
    get_checked_reps,
    get_checked_devices,
    get_checked_rep_types,
    update_check_devices,
    update_check_rep_types,
    update_check_reps,

    // Template event handlers
    clicked_reps,
    update_reps,
    clicked_devices,
    update_devices,
    clicked_rep_types,
    update_rep_types,
} = useRepQuery({ props, emits })

defineExpose({ get_checked_reps, get_checked_devices, get_checked_rep_types, update_check_devices, update_check_rep_types, update_check_reps })
</script>
