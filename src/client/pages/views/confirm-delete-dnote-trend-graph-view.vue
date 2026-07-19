<template>
    <v-card variant="flat" class="pa-2">
        <v-card-title>
            {{ i18n.global.t("DELETE_DNOTE_TREND_GRAPH_TITLE") }}
        </v-card-title>
        {{ dnote_trend_graph_query.title }}
        <v-row class="pa-0 ma-0 gkill-dialog-actions">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="emits('requested_delete_dnote_trend_graph', dnote_trend_graph_query.id)">{{
                    i18n.global.t("DELETE_TITLE")
                }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { ConfirmDeleteDnoteTrendGraphViewEmits } from './confirm-delete-dnote-trend-graph-emits';
import type { ConfirmDeleteDnoteTrendGraphViewProps } from './confirm-delete-dnote-trend-graph-view-props';


defineProps<ConfirmDeleteDnoteTrendGraphViewProps>()
const emits = defineEmits<ConfirmDeleteDnoteTrendGraphViewEmits>()
</script>
