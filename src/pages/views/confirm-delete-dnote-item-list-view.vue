<template>
    <v-card class="pa-2">
        <v-card-title>
            {{ $t("DELETE_DNOTE_LIST_ITEM_TITLE") }}
        </v-card-title>
        {{ dnote_item.title }}
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="emits('requested_delete_dnote_item_list', dnote_item.id)">{{
                    $t("DELETE_TITLE")
                }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import type { ConfirmDeleteDnoteItemListViewEmits } from './confirm-delete-dnote-item-list-view-emits';
import type { ConfirmDeleteDnoteItemListViewProps } from './confirm-delete-dnote-item-list-view-props';

defineProps<ConfirmDeleteDnoteItemListViewProps>()
const emits = defineEmits<ConfirmDeleteDnoteItemListViewEmits>()
</script>
