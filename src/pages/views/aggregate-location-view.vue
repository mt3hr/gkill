<template>
    <v-card class="pa-0 ma-0 aggregate_location_view">
        <div>
            {{ aggregate_location.title }}
        </div>
        <v-row class="pa-0 ma-0">
            <v-col class="pa-0 ma-0" cols="auto">
                {{ format_duration(aggregate_location.duration_milli_second) }}
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import moment from 'moment';
import type { KyouViewEmits } from './kyou-view-emits';
import type { AggregateLocationViewProps } from './aggregate-location-view-props';
import { useI18n } from 'vue-i18n'

import { i18n } from '@/i18n'

defineProps<AggregateLocationViewProps>()
defineEmits<KyouViewEmits>()

function format_duration(duration_milli_second: number): string {
    if (duration_milli_second === 0) {
        return ""
    }
    let diff_str = ""
    const offset_in_locale_milli_second = new Date().getTimezoneOffset().valueOf() * 60000
    const diff = duration_milli_second
    const diff_date = moment(diff + offset_in_locale_milli_second).toDate()
    if (diff_date.getFullYear() - 1970 !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += diff_date.getFullYear() - 1970 + i18n.global.t("YEAR_SUFFIX")
    }
    if (diff_date.getMonth() !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += (diff_date.getMonth() + 1) + i18n.global.t("MONTH_SUFFIX")
    }
    if ((diff_date.getDate() - 1) !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += (diff_date.getDate() - 1) + i18n.global.t("DAY_SUFFIX")
    }
    if (diff_date.getHours() !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += (diff_date.getHours()) + i18n.global.t("HOUR_SUFFIX")
    }
    if (diff_date.getMinutes() !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += diff_date.getMinutes() + i18n.global.t("MINUTE_SUFFIX")
    }
    if (diff_str === "") {
        diff_str += diff_date.getSeconds() + i18n.global.t("SECOND_SUFFIX")
    }
    return diff_str
}
</script>
<style lang="css">
.aggregate_location_view {
    border-top: 1px solid silver;
}
</style>
