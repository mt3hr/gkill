<template>
    <v-text-field v-model="title" :label="$t('DNOTE_TITLE_TITLE')" />
    <v-select v-model="aggregate_target" :items="aggregate_targets" item-title="label" item-value="value"
        :label="$t('DNOTE_AGGREGATE_TARGET')"></v-select>
    <PredicateGroup v-model="rootPredicate" />
</template>

<script setup lang="ts">
import { ref, type Ref } from 'vue'
import PredicateGroup from './edit-dnote-predicate-group.vue'
import type PredicateGroupType from '../../classes/dnote/predicate-group-type'
import type Predicate from '../../classes/dnote/predicate'
import { i18n } from '@/i18n'
import type DnoteSelectItem from '../../classes/dnote/dnote-select-item'

const title = ref("")

const rootPredicate = ref<PredicateGroupType>({
    logic: 'AND',
    predicates: []
})

const aggregate_targets: Ref<Array<DnoteSelectItem>> = ref([
    { label: i18n.global.t("DNOTE_AVERAGE_LANTANA_MOOD"), value: "AggregateAverageLantanaMood" },
    { label: i18n.global.t("DNOTE_AVERAGE_NLOG_AMOUNT"), value: "AggregateAverageNlogAmount" },
    { label: i18n.global.t("DNOTE_AVERAGE_TIMEIS_END_TIME"), value: "AggregateAverageTimeIsEndTime" },
    { label: i18n.global.t("DNOTE_AVERAGE_TIMEIS_START_TIME"), value: "AggregateAverageTimeIsStartTime" },
    { label: i18n.global.t("DNOTE_AVERAGE_TIMEIS_TIME"), value: "AggregateAverageTimeIsTime" },
    { label: i18n.global.t("DNOTE_COUNT_KYOU"), value: "AggregateCountKyou" },
    { label: i18n.global.t("DNOTE_SUM_LANTANA_MOOD"), value: "AggregateSumLantanaMood" },
    { label: i18n.global.t("DNOTE_SUM_NLOG_AMOUNT"), value: "AggregateSumNlogAmount" },
    { label: i18n.global.t("DNOTE_SUM_TIMEIS_TIME"), value: "AggregateSumTimeIsTime" },
    { label: i18n.global.t("DNOTE_AVERAGE_GIT_COMMIT_LOG_CODE_COUNT"), value: "AgregateAverageGitCommitLogCode" },
    { label: i18n.global.t("DNOTE_AVERAGE_GIT_COMMIT_LOG_ADDITION_CODE_COUNT"), value: "AgregateAverageGitCommitLogAdditionCode" },
    { label: i18n.global.t("DNOTE_AVERAGE_GIT_COMMIT_LOG_DELETION_CODE_COUNT"), value: "AgregateAverageGitCommitLogDeletionCode" },
    { label: i18n.global.t("DNOTE_SUM_GIT_COMMIT_LOG_CODE_COUNT"), value: "AgregateSumGitCommitLogCode" },
    { label: i18n.global.t("DNOTE_SUM_GIT_COMMIT_LOG_CODE_ADDITION_COUNT"), value: "AgregateSumGitCommitLogAdditionCode" },
    { label: i18n.global.t("DNOTE_SUM_GIT_COMMIT_LOG_CODE_DELETION_COUNT"), value: "AgregateSumGitCommitLogDeletionCode" },
])
const aggregate_target: Ref<DnoteSelectItem> = ref(aggregate_targets.value[0])


function to_json(group: PredicateGroupType | Predicate): any {
    if (is_group(group)) {
        return {
            logic: group.logic,
            predicates: group.predicates.map(p => to_json(p))
        }
    } else {
        return { type: group.type, value: group.value }
    }
}

function from_json(json: any): PredicateGroupType | Predicate {
    if (json.logic && Array.isArray(json.predicates)) {
        return {
            logic: json.logic,
            predicates: json.predicates.map(from_json)
        }
    } else {
        return {
            type: json.type,
            value: json.value
        }
    }
}

function is_group(p: Predicate | PredicateGroupType): p is PredicateGroupType {
    return 'logic' in p && Array.isArray(p.predicates)
}
</script>
