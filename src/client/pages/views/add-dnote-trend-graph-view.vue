<template>
    <v-card class="pa-2" variant="flat">
        <v-text-field v-model="title" :label="i18n.global.t('DNOTE_TITLE_TITLE')" />
        <v-select v-model="aggregate_target" :items="aggregate_targets" item-title="label" item-value="value"
            :label="i18n.global.t('DNOTE_AGGREGATE_TARGET')"></v-select>
        <v-select v-model="granularity" :items="granularities" item-title="label" item-value="value"
            :label="i18n.global.t('DNOTE_TREND_GRANULARITY')"></v-select>
        <v-select v-model="chart_type" :items="chart_types" item-title="label" item-value="value"
            :label="i18n.global.t('DNOTE_TREND_CHART_TYPE')"></v-select>
        <PredicateGroup v-model="root_predicate" :is_root="true" />
        <v-row class="pa-0 ma-0 flex-row-reverse gkill-dialog-actions">
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="() => save()">{{ i18n.global.t("SAVE_TITLE") }}</v-btn>
            </v-col>
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="secondary" @click="reset()">{{ i18n.global.t("RESET_TITLE") }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>

<script setup lang="ts">
import { i18n } from '@/i18n'
import PredicateGroup from './edit-dnote-predicate-group.vue'
import type AddDnoteTrendGraphViewEmits from './add-dnote-trend-graph-view-emits'
import type AddDnoteTrendGraphViewProps from './add-dnote-trend-graph-view-props'
import { useAddDnoteTrendGraphView } from '@/classes/use-add-dnote-trend-graph-view'

const props = defineProps<AddDnoteTrendGraphViewProps>()
const emits = defineEmits<AddDnoteTrendGraphViewEmits>()

const {
    title,
    root_predicate,
    aggregate_targets,
    aggregate_target,
    granularities,
    granularity,
    chart_types,
    chart_type,
    reset,
    save,
} = useAddDnoteTrendGraphView({ props, emits })
</script>
