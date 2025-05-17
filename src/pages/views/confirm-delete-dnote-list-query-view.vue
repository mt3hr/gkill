<template>
    <v-card class="pa-2">
        <v-card-title>
            {{ i18n.global.t("DELETE_DNOTE_LIST_QUERY_TITLE") }}
        </v-card-title>
        {{ dnote_list_query.title }}
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="emits('requested_delete_dnote_list_query', dnote_list_query.id)">{{
                    i18n.global.t("DELETE_TITLE")
                }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { ConfirmDeleteDnoteListQueryViewEmits } from './confirm-delete-dnote-list-query-emits';
import type { ConfirmDeleteDnoteListQueryViewProps } from './confirm-delete-dnote-list-query-view-props';


defineProps<ConfirmDeleteDnoteListQueryViewProps>()
const emits = defineEmits<ConfirmDeleteDnoteListQueryViewEmits>()
</script>
