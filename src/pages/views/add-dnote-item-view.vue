<template>
    <v-card class="pa-2">
        <v-text-field v-model="title" :label="$t('DNOTE_TITLE_TITLE')" />
        <v-text-field v-model="prefix" :label="$t('DNOTE_PREFIX_TITLE')" />
        <v-text-field v-model="suffix" :label="$t('DNOTE_SUFFIX_TITLE')" />
        <v-select v-model="aggregate_target" :items="aggregate_targets" item-title="label" item-value="value"
            :label="$t('DNOTE_AGGREGATE_TARGET')"></v-select>
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
import { nextTick, ref, type Ref } from 'vue'
import PredicateGroup from './edit-dnote-predicate-group.vue'
import type PredicateGroupType from '../../classes/dnote/predicate-group-type'
import type Predicate from '../../classes/dnote/predicate'
import { i18n } from '@/i18n'
import type DnoteSelectItem from '../../classes/dnote/dnote-select-item'
import DnoteItem from '@/classes/dnote/dnote-item'
import type AddDnoteItemViewEmits from './add-dnote-item-view-emits'
import type AddDnoteItemViewProps from './add-dnote-item-view-props'
import { build_dnote_aggregate_target_from_json, build_dnote_predicate_from_json } from '@/classes/dnote/serialize/regist-dictionary'

const props = defineProps<AddDnoteItemViewProps>()
const emits = defineEmits<AddDnoteItemViewEmits>()

nextTick(() => reset())

async function reset(): Promise<void> {
    id.value = props.gkill_api.generate_uuid()
    title.value = ""
    prefix.value = ""
    suffix.value = ""
    root_predicate.value = {
        logic: 'AND',
        predicates: []
    }
    aggregate_target.value = aggregate_targets.value[0].value

}

async function save(): Promise<void> {
    const new_dnote_item = new DnoteItem()
    new_dnote_item.id = id.value
    new_dnote_item.prefix = prefix.value
    new_dnote_item.suffix = suffix.value
    new_dnote_item.title = title.value
    new_dnote_item.agregate_target = build_dnote_aggregate_target_from_json({ type: aggregate_target.value })
    new_dnote_item.predicate = build_dnote_predicate_from_json(predicate_struct_to_json(root_predicate.value))

    emits('requested_add_dnote_item', new_dnote_item)
    emits('requested_close_dialog')
}

const id = ref("")
const title = ref("")
const prefix = ref("")
const suffix = ref("")

const root_predicate = ref<PredicateGroupType>({
    logic: 'AND',
    predicates: []
})

const aggregate_targets: Ref<Array<DnoteSelectItem>> = ref([
    { label: i18n.global.t("DNOTE_AVERAGE_LANTANA_MOOD"), value: "AgregateAverageLantanaMood" },
    { label: i18n.global.t("DNOTE_AVERAGE_NLOG_AMOUNT"), value: "AgregateAverageNlogAmount" },
    { label: i18n.global.t("DNOTE_AVERAGE_TIMEIS_END_TIME"), value: "AgregateAverageTimeIsEndTime" },
    { label: i18n.global.t("DNOTE_AVERAGE_TIMEIS_START_TIME"), value: "AgregateAverageTimeIsStartTime" },
    { label: i18n.global.t("DNOTE_AVERAGE_TIMEIS_TIME"), value: "AgregateAverageTimeIsTime" },
    { label: i18n.global.t("DNOTE_COUNT_KYOU"), value: "AgregateCountKyou" },
    { label: i18n.global.t("DNOTE_SUM_LANTANA_MOOD"), value: "AgregateSumLantanaMood" },
    { label: i18n.global.t("DNOTE_SUM_NLOG_AMOUNT"), value: "AgregateSumNlogAmount" },
    { label: i18n.global.t("DNOTE_SUM_TIMEIS_TIME"), value: "AgregateSumTimeIsTime" },
    { label: i18n.global.t("DNOTE_AVERAGE_GIT_COMMIT_LOG_CODE_COUNT"), value: "AgregateAverageGitCommitLogCode" },
    { label: i18n.global.t("DNOTE_AVERAGE_GIT_COMMIT_LOG_ADDITION_CODE_COUNT"), value: "AgregateAverageGitCommitLogAdditionCode" },
    { label: i18n.global.t("DNOTE_AVERAGE_GIT_COMMIT_LOG_DELETION_CODE_COUNT"), value: "AgregateAverageGitCommitLogDeletionCode" },
    { label: i18n.global.t("DNOTE_SUM_GIT_COMMIT_LOG_CODE_COUNT"), value: "AgregateSumGitCommitLogCode" },
    { label: i18n.global.t("DNOTE_SUM_GIT_COMMIT_LOG_CODE_ADDITION_COUNT"), value: "AgregateSumGitCommitLogAdditionCode" },
    { label: i18n.global.t("DNOTE_SUM_GIT_COMMIT_LOG_CODE_DELETION_COUNT"), value: "AgregateSumGitCommitLogDeletionCode" },
])
const aggregate_target: Ref<string> = ref(aggregate_targets.value[0].value)

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

function is_group(p: Predicate | PredicateGroupType): p is PredicateGroupType {
    return 'logic' in p && Array.isArray(p.predicates)
}
</script>
