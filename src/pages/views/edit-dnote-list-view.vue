<template>
    <v-card class="pa-2">
        <v-text-field v-model="title" :label="$t('DNOTE_TITLE_TITLE')" />
        <v-text-field v-model="prefix" :label="$t('DNOTE_PREFIX_TITLE')" />
        <v-text-field v-model="suffix" :label="$t('DNOTE_SUFFIX_TITLE')" />
        <v-select v-model="aggregate_target" :items="aggregate_targets" item-title="label" item-value="value"
            :label="$t('DNOTE_AGGREGATE_TARGET')"></v-select>
        <v-select v-model="key_getter" :items="key_getters" item-title="label" item-value="value"
            :label="$t('DNOTE_LIST_KEY')"></v-select>
        <PredicateGroup v-model="root_predicate" />
        <v-row class="pa-0 ma-0">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="secondary" @click="reset()">{{ $t("RESET_TITLE") }}</v-btn>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="() => save()">{{ $t("SAVE_TITLE") }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>

<script setup lang="ts">
import { ref, type Ref } from 'vue'
import PredicateGroup from './edit-dnote-predicate-group.vue'
import type PredicateGroupType from '../../classes/dnote/predicate-group-type'
import type Predicate from '../../classes/dnote/predicate'
import { i18n } from '@/i18n'
import type DnoteSelectItem from '../../classes/dnote/dnote-select-item'
import DnoteListQuery from '@/pages/views/dnote-list-query'
import AggregateTargetDictionary from '@/classes/dnote/serialize/dnote-aggregate-target-dictionary'
import type EditDnoteListViewEmits from './edit-dnote-list-view-emits'
import type EditDnoteListViewProps from './edit-dnote-list-view-props'
import DnoteKeyGetterDictionary from '@/classes/dnote/serialize/dnote-key-getter-dictionary'
import { build_dnote_predicate_from_json } from '@/classes/dnote/serialize/regist-dictionary'

const props = defineProps<EditDnoteListViewProps>()
const emits = defineEmits<EditDnoteListViewEmits>()

load_props()

async function load_props(): Promise<void> {
    id.value = props.dnote_list_query.id
    title.value = props.dnote_list_query.title
    prefix.value = props.dnote_list_query.prefix
    suffix.value = props.dnote_list_query.suffix
    root_predicate.value = predicate_struct_from_json(props.dnote_list_query.predicate.predicate_struct_to_json()) as PredicateGroupType
    key_getter.value = key_getters.value.find((key_getter) => key_getter.value === props.dnote_list_query.key_getter.to_json().type)!.value
    aggregate_target.value = aggregate_targets.value.find((aggregate_target) => aggregate_target.value === props.dnote_list_query.aggregate_target.to_json().type)!.value
}

async function reset(): Promise<void> {
    return load_props()
}

async function save(): Promise<void> {
    const new_dnote_list_query = new DnoteListQuery()
    new_dnote_list_query.id = id.value
    new_dnote_list_query.prefix = prefix.value
    new_dnote_list_query.suffix = suffix.value
    new_dnote_list_query.title = title.value
    new_dnote_list_query.aggregate_target = AggregateTargetDictionary.get(aggregate_target.value)()
    new_dnote_list_query.key_getter = DnoteKeyGetterDictionary.get(key_getter.value)()
    new_dnote_list_query.predicate = build_dnote_predicate_from_json(predicate_struct_to_json(root_predicate.value))

    emits('requested_update_dnote_list_query', new_dnote_list_query)
    emits('requested_close_dialog')
}

const id = ref(props.gkill_api.generate_uuid())
const title = ref("")
const prefix = ref("")
const suffix = ref("")

const root_predicate = ref<PredicateGroupType>({
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
const aggregate_target: Ref<string> = ref(aggregate_targets.value[0].value)

const key_getters: Ref<Array<DnoteSelectItem>> = ref([
    { label: i18n.global.t("DNOTE_DATA_TYPE_GETTER"), value: "DataTypeGetter" },
    { label: i18n.global.t("DNOTE_LANTANA_MOOD_GETTER"), value: "LantanaMoodGetter" },
    { label: i18n.global.t("DNOTE_NLOG_SHOP_NAME_GETTER"), value: "NlogShopNameGetter" },
    { label: i18n.global.t("DNOTE_RELATED_MONTH_GETTER"), value: "RelatedMonthGetter" },
    { label: i18n.global.t("DNOTE_RELATED_WEEK_DAY_GETTER"), value: "RelatedWeekDayGetter" },
    { label: i18n.global.t("DNOTE_RELATED_WEEK_GETTER"), value: "RelatedWeekGetter" },
    { label: i18n.global.t("DNOTE_RELATED_DATE_GETTER"), value: "RelatedDateGetter" },
    { label: i18n.global.t("DNOTE_TAG_GETTER"), value: "TagGetter" },
    { label: i18n.global.t("DNOTE_TITLE_GETTER"), value: "TitleGetter" },
])
const key_getter: Ref<string> = ref(key_getters.value[0].value)

function predicate_struct_to_json(group: PredicateGroupType | Predicate): any {
    if (is_group(group)) {
        return {
            logic: group.logic,
            predicates: group.predicates.map(p => predicate_struct_to_json(p))
        }
    } else {
        return { type: group.type, value: group.value }
    }
}

function predicate_struct_from_json(json: any): PredicateGroupType | Predicate {
    if (json.logic && Array.isArray(json.predicates)) {
        return {
            logic: json.logic,
            predicates: json.predicates.map(predicate_struct_from_json)
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
