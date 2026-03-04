<template>
    <div>
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-checkbox v-model="use_period_of_time" :label="i18n.global.t('PERIOD_OF_TIME_QUERY_TITLE')"
                    hide-details />
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pb-0 mb-0 pr-0">
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
import { nextTick, type Ref, ref, watch } from 'vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import type { PeriodOfTimeQueryEmits } from './period-of-time-query-emits'
import type { PeriodOfTimeQueryProps } from './period-of-time-query-props'
import moment from 'moment'
import { WeekOfDays } from '@/classes/api/find_query/week-of-days'

const props = defineProps<PeriodOfTimeQueryProps>()
const emits = defineEmits<PeriodOfTimeQueryEmits>()
defineExpose({ get_use_period_of_time, get_period_of_time_start_time_second, get_period_of_time_end_time_second, get_period_of_time_week_of_days })

const use_period_of_time: Ref<boolean> = ref(false)
const show_period_of_time_start_time_menu: Ref<boolean> = ref(false)
const show_period_of_time_end_time_menu: Ref<boolean> = ref(false)
const period_of_time_start_time_string: Ref<string> = ref("")
const period_of_time_end_time_string: Ref<string> = ref("")
const week_of_days = ref<number[]>([])

const cloned_application_config: Ref<ApplicationConfig> = ref(props.application_config.clone())
const skip_emits_this_tick = ref(false)

watch(() => props.application_config, () => {
    skip_emits_this_tick.value = true
    nextTick(() => skip_emits_this_tick.value = false)
    cloned_application_config.value = props.application_config.clone()
})
watch(() => props.find_kyou_query.use_period_of_time, (new_value: boolean, old_value: boolean) => {
    if (new_value === old_value) {
        return
    }
    skip_emits_this_tick.value = true
    nextTick(() => skip_emits_this_tick.value = false)
    use_period_of_time.value = props.find_kyou_query.use_period_of_time
})
watch(
    () => props.find_kyou_query.period_of_time_start_time_second,
    (newSec, oldSec) => {
        if (newSec === oldSec) return

        skip_emits_this_tick.value = true
        nextTick(() => (skip_emits_this_tick.value = false))

        period_of_time_start_time_string.value =
            newSec == null ? "" : moment.unix(newSec).format("HH:mm")
    }
)
watch(
    () => props.find_kyou_query.period_of_time_end_time_second,
    (newSec, oldSec) => {
        if (newSec === oldSec) return

        skip_emits_this_tick.value = true
        nextTick(() => (skip_emits_this_tick.value = false))

        period_of_time_end_time_string.value =
            newSec == null ? "" : moment.unix(newSec).format("HH:mm")
    }
)
watch(() => props.find_kyou_query.period_of_time_week_of_days, () => {
    week_of_days.value.splice(0)
    week_of_days.value.push(...props.find_kyou_query.period_of_time_week_of_days)
})

watch(() => use_period_of_time.value, () => {
    if (skip_emits_this_tick.value) {
        return
    }
    emits('request_update_use_period_of_time', use_period_of_time.value)
})
watch(() => period_of_time_start_time_string.value, () => {
    if (skip_emits_this_tick.value) {
        return
    }
    emits('request_update_period_of_time', get_period_of_time_start_time_second(), get_period_of_time_end_time_second(), get_period_of_time_week_of_days())
})
watch(() => period_of_time_end_time_string.value, () => {
    if (skip_emits_this_tick.value) {
        return
    }
    emits('request_update_period_of_time', get_period_of_time_start_time_second(), get_period_of_time_end_time_second(), get_period_of_time_week_of_days())
})
watch(() => week_of_days.value, () => {
    if (skip_emits_this_tick.value) {
        return
    }
    emits('request_update_period_of_time', get_period_of_time_start_time_second(), get_period_of_time_end_time_second(), get_period_of_time_week_of_days())
})

function get_use_period_of_time(): boolean {
    return use_period_of_time.value
}
function get_period_of_time_start_time_second(): number | null {
    if (period_of_time_start_time_string.value === "") return null
    const [h, m] = period_of_time_start_time_string.value.split(":").map(Number)
    return moment().startOf("day").hour(h).minute(m).second(0).unix()
}
function get_period_of_time_end_time_second(): number | null {
    if (period_of_time_end_time_string.value === "") return null
    const [h, m] = period_of_time_end_time_string.value.split(":").map(Number)
    return moment().startOf("day").hour(h).minute(m).second(0).unix()
}

function get_period_of_time_week_of_days(): Array<number> {
    return week_of_days.value.concat()
}

function to_week_of_days_label(num: WeekOfDays): string {
    switch (num) {
        case WeekOfDays.SunDay:
            return "SUNDAY_TITLE"
        case WeekOfDays.MonDay:
            return "MONDAY_TITLE"
        case WeekOfDays.TuesDay:
            return "TUESDAY_TITLE"
        case WeekOfDays.WednesDay:
            return "WEDNESDAY_TITLE"
        case WeekOfDays.ThrusDay:
            return "THURSDAY_TITLE"
        case WeekOfDays.FriDay:
            return "FRIDAY_TITLE"
        case WeekOfDays.SaturDay:
            return "SATURDAY_TITLE"
    }
    return ""
}
</script>
