<template>
    <v-card class="pa-0 ma-0 aggregate_amount_view">
        <v-row class="pa-0 ma-0">
            <v-col class="aggregate_amount_related_time pa-0 ma-0" cols="auto">
                {{ format_time(aggregate_amount.related_time) }}
            </v-col>
        </v-row>
        <div>
            {{ aggregate_amount.title }}
        </div>
        <div v-if="aggregate_amount.amount > 0" class="aggregate_amount_plus">
            {{ aggregate_amount.amount }} 円
        </div>
        <div v-if="aggregate_amount.amount <= 0" class="aggregate_amount_minus">
            {{ aggregate_amount.amount }} 円
        </div>
    </v-card>
</template>
<script lang="ts" setup>
import type { AggregateAmountViewProps } from './aggregate-amount-view-props';
import type { KyouViewEmits } from './kyou-view-emits';
import { useI18n } from 'vue-i18n'

import { i18n } from '@/i18n'

defineProps<AggregateAmountViewProps>()
defineEmits<KyouViewEmits>()

function format_time(time: Date) {
    let year: string | number = time.getFullYear()
    let month: string | number = time.getMonth() + 1
    let date: string | number = time.getDate()
    let hour: string | number = time.getHours()
    let minute: string | number = time.getMinutes()
    let second: string | number = time.getSeconds()
    const day_of_week = [i18n.global.t("SUNDAY_TITLE"), i18n.global.t("MONDAY_TITLE"), i18n.global.t("TUESDAY_TITLE"), i18n.global.t("WEDNESDAY_TITLE"), i18n.global.t("THURSDAY_TITLE"), i18n.global.t("FRIDAY_TITLE"), i18n.global.t("SATURDAY_TITLE")][time.getDay()]
    month = ('0' + month).slice(-2)
    date = ('0' + date).slice(-2)
    hour = ('0' + hour).slice(-2)
    minute = ('0' + minute).slice(-2)
    second = ('0' + second).slice(-2)
    return year + '/' + month + '/' + date + '(' + day_of_week + ')' + ' ' + hour + ':' + minute + ':' + second
}



</script>

<style lang="css" scoped>
.aggregate_amount_related_time {
    font-size: small;
    color: silver;
}

.aggregate_amount_plus {
    color: limegreen;
}

.aggregate_amount_minus {
    color: crimson;
}

.aggregate_amount_view {
    border-top: 1px solid silver;
}
</style>