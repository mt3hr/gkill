<template>
    <v-card class="pa-2">
        <v-card-title>
            {{ i18n.global.t("DELETE_RELATED_KYOU_QUERY_TITLE") }}
        </v-card-title>
        {{ related_kyou_query.title }}
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary"
                    @click="emits('requested_delete_related_kyou_query', related_kyou_query.id)">{{
                        i18n.global.t("DELETE_TITLE") }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { ConfirmDeleteRelatedKyouQueryViewEmits } from './confirm-delete-related-kyou-query-view-emits.js';
import type { ConfirmDeleteRelatedKyouQueryViewProps } from './confirm-delete-related-kyou-query-view-props.js';

defineProps<ConfirmDeleteRelatedKyouQueryViewProps>()
const emits = defineEmits<ConfirmDeleteRelatedKyouQueryViewEmits>()
</script>
