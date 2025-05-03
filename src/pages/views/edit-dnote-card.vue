<!-- PredicateCard.vue -->
<template>
    <v-card class="predicate-card">
        <v-select v-model="predicate_type" :items="predicate_types" :label="$t('DNOTE_PREDICATE_TITLE')"
            @update:modelValue="onTypeChange" item-title="label" item-value="value" />
        <v-text-field v-model="model_value!.value" :label="$t('DNOTE_PREDICATE_VALUE_TITLE')" />
        <v-row>
            <v-spacer />
            <v-col cols="auto">
                <v-btn :color="'secondary'" @click="emits('remove')" class="mt-2">{{ $t('DELETE_TITLE') }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script setup lang="ts">
const emits = defineEmits(['remove'])
import { ref, watch, type Ref } from 'vue'
import { defineModel } from 'vue'
import type Predicate from '../../classes/dnote/predicate'
import { i18n } from '@/i18n'
import type DnoteSelectItem from '../../classes/dnote/dnote-select-item'

const model_value = defineModel<Predicate>()

const predicate_types: Ref<Array<DnoteSelectItem>> = ref([
    { label: i18n.global.t("DNOTE_DATA_TYPE_PREFIX_PREDICATE"), value: "DataTypePrefixPredicate" },
    { label: i18n.global.t("DNOTE_GIT_COMMIT_LOG_CODE_GREATER_THAN_PREDICATE"), value: "GitCommitLogCodeGreaterThanPredicate" },
    { label: i18n.global.t("DNOTE_GIT_COMMIT_LOG_CODE_LESS_THAN_PREDICATE"), value: "GitCommitLogCodeLessThanPredicate" },
    { label: i18n.global.t("DNOTE_GIT_COMMIT_LOG_ADDITION_CODE_GREATER_THAN_PREDICATE"), value: "GitCommitLogCodeAdditionGreaterThanPredicate" },
    { label: i18n.global.t("DNOTE_GIT_COMMIT_LOG_ADDITION_CODE_LESS_THAN_PREDICATE"), value: "GitCommitLogCodeDeletionLessThanPredicate" },
    { label: i18n.global.t("DNOTE_GIT_COMMIT_LOG_DELETION_CODE_GREATER_THAN_PREDICATE"), value: "GitCommitLogCodeAdditionGreaterThanPredicate" },
    { label: i18n.global.t("DNOTE_GIT_COMMIT_LOG_DELETION_CODE_LESS_THAN_PREDICATE"), value: "GitCommitLogCodeDeletionLessThanPredicate" },
    { label: i18n.global.t("DNOTE_KMEMO_CONTENT_CONTAINS_PREDICATE"), value: "KmemoContentContainsPredicate" },
    { label: i18n.global.t("DNOTE_KMEMO_CONTENT_EQUAL_PREDICATE"), value: "KmemoContentEqualPredicate" },
    { label: i18n.global.t("DNOTE_LANTANA_MOOD_EQUAL_PREDICATE"), value: "LantanaMoodEqualPredicate" },
    { label: i18n.global.t("DNOTE_LANTANA_MOOD_GREATER_THAN_PREDICATE"), value: "LantanaMoodGreaterThanPredicate" },
    { label: i18n.global.t("DNOTE_LANTANA_MOOD_LESS_THAN_PREDICATE"), value: "LantanaMoodLessThanPredicate" },
    { label: i18n.global.t("DNOTE_MI_TITLE_CONTAINS_PREDICATE"), value: "MiTitleEqualPredicate" },
    { label: i18n.global.t("DNOTE_MI_TITLE_EQUAL_PREDICATE"), value: "MiTitleEqualPredicate" },
    { label: i18n.global.t("DNOTE_NLOG_AMOUNT_GREATER_THAN_PREDICATE"), value: "NlogAmountGreaterThanPredicate" },
    { label: i18n.global.t("DNOTE_NLOG_AMOUNT_LESS_THAN_PREDICATE"), value: "NlogAmountLessThanPredicate" },
    { label: i18n.global.t("DNOTE_NLOG_SHOP_CONTAINS_PREDICATE"), value: "NlogShopContainsPredicate" },
    { label: i18n.global.t("DNOTE_NLOG_SHOP_EQUAL_PREDICATE"), value: "NlogShopEqualPredicate" },
    { label: i18n.global.t("DNOTE_NLOG_TITLE_CONTAINS_PREDICATE"), value: "NlogTitleContainsPredicate" },
    { label: i18n.global.t("DNOTE_NLOG_TITLE_EQUAL_PREDICATE"), value: "NlogTitleEqualPredicate" },
    { label: i18n.global.t("DNOTE_RELATED_TIME_WEEK_PREDICATE"), value: "RelatedTimeWeekPredicate" },
    { label: i18n.global.t("DNOTE_TAG_EQUAL_PREDICATE"), value: "TagEqualPredicate" },
    { label: i18n.global.t("DNOTE_TIMEIS_TITLE_CONTAINS_PREDICATE"), value: "TimeIsTitleContainsPredicate" },
    { label: i18n.global.t("DNOTE_TIMEIS_TITLE_EQUAL_PREDICATE"), value: "TimeIsTitleEqualPredicate" },
])

const predicate_type = ref(model_value.value?.type)

watch(predicate_type, () => {
    onTypeChange()
})

function onTypeChange(): void {
    model_value.value!.value = ''
}
</script>
