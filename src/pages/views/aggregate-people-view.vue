<template>
    <v-card class="pa-0 ma-0">
        <div>
            {{ aggregate_people.title }}
        </div>
        <v-row class="pa-0 ma-0">
            <v-col class="pa-0 ma-0" cols="auto">
                {{ format_duration(aggregate_people.duration_milli_second) }}
            </v-col>
        </v-row>
        <div>
            <span v-for="type in aggregate_people.type" :key="type">
                {{ type }}
            </span>
        </div>
    </v-card>
</template>
<script lang="ts" setup>
import moment from 'moment';
import type { AggregatePeopleViewProps } from './aggregate-people-view-props';
import type { KyouViewEmits } from './kyou-view-emits';

defineProps<AggregatePeopleViewProps>()
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
        diff_str += diff_date.getFullYear() - 1970 + "年"
    }
    if (diff_date.getMonth() !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += (diff_date.getMonth() + 1) + "ヶ月"
    }
    if ((diff_date.getDate() - 1) !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += (diff_date.getDate() - 1) + "日"
    }
    if (diff_date.getHours() !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += (diff_date.getHours()) + "時間"
    }
    if (diff_date.getMinutes() !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += diff_date.getMinutes() + "分"
    }
    if (diff_str === "") {
        diff_str += diff_date.getSeconds() + "秒"
    }
    return diff_str
}
</script>
<style lang="css"></style>
