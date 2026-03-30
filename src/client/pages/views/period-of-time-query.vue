<template>
    <div>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-checkbox v-model="use_period_of_time" :label="i18n.global.t('PERIOD_OF_TIME_QUERY_TITLE')"
                    hide-details />
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pb-0 mb-0 pr-0 pt-2">
                <v-btn dark color="secondary" @click="emits('request_clear_use_period_of_time_query')" hide-details>{{
                    i18n.global.t("CLEAR_TITLE") }}</v-btn>
            </v-col>
        </v-row>
        <v-row class="pa-0 ma-0" v-if="use_period_of_time">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-menu v-model="show_period_of_time_start_time_menu" :close-on-content-click="false"
                    transition="scale-transition" offset-y min-width="auto">
                    <template #activator="{ props }">
                        <v-text-field v-model="period_of_time_start_time_string" hide-details
                            :label="i18n.global.t('PERIOD_OF_TIME_QUERY_START_TIME_TITLE')" readonly min-width="120"
                            v-bind="props" />
                    </template>
                    <v-time-picker v-model="period_of_time_start_time_string" format="24hr"
                        @update:minute="show_period_of_time_start_time_menu = false" />
                </v-menu>
            </v-col>
            <v-col cols="auto" class="pa-0 ma-0">
                <v-menu v-model="show_period_of_time_end_time_menu" :close-on-content-click="false"
                    transition="scale-transition" offset-y min-width="auto">
                    <template #activator="{ props }">
                        <v-text-field v-model="period_of_time_end_time_string" hide-details
                            :label="i18n.global.t('PERIOD_OF_TIME_QUERY_END_TIME_TITLE')" readonly min-width="120"
                            v-bind="props" />
                    </template>
                    <v-time-picker v-model="period_of_time_end_time_string" format="24hr"
                        @update:minute="show_period_of_time_end_time_menu = false" />
                </v-menu>
            </v-col>
            <v-col cols="auto" class="pt-2 pa-0 ma-0">
                <v-item-group v-model="week_of_days" multiple class="pa-0 ma-0">
                    <v-item v-for="w in [0, 1, 2, 3, 4, 5, 6]" :key="w" :value="w" v-slot="{ isSelected, toggle }">
                        <v-btn color="primary" class="pa-0 ma-0" min-width="40" :active="isSelected" @click="toggle">
                            {{ i18n.global.t(to_week_of_days_label(w)) }}
                        </v-btn>
                    </v-item>
                </v-item-group>
            </v-col>
        </v-row>
    </div>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { PeriodOfTimeQueryEmits } from './period-of-time-query-emits'
import type { PeriodOfTimeQueryProps } from './period-of-time-query-props'
import { usePeriodOfTimeQuery } from '@/classes/use-period-of-time-query'

const props = defineProps<PeriodOfTimeQueryProps>()
const emits = defineEmits<PeriodOfTimeQueryEmits>()

const {
    use_period_of_time,
    show_period_of_time_start_time_menu,
    show_period_of_time_end_time_menu,
    period_of_time_start_time_string,
    period_of_time_end_time_string,
    week_of_days,
    get_use_period_of_time,
    get_period_of_time_start_time_second,
    get_period_of_time_end_time_second,
    get_period_of_time_week_of_days,
    to_week_of_days_label,
} = usePeriodOfTimeQuery({ props, emits })

defineExpose({ get_use_period_of_time, get_period_of_time_start_time_second, get_period_of_time_end_time_second, get_period_of_time_week_of_days })
</script>
