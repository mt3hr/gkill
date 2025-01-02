<template>
    <v-card class="pa-2">
        <v-card-title>
            タグ構造削除
        </v-card-title>
        {{ tag_struct.tag_name }}
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn color="primary" @click="emits('requested_delete_tag', tag_struct.id)">削除</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import type { ConfirmDeleteTagStructViewProps } from './confirm-delete-tag-struct-view-props';
import type { ConfirmDeleteTagStructViewEmits } from './confirm-delete-tag-struct-view-emits';

defineProps<ConfirmDeleteTagStructViewProps>()
const emits = defineEmits<ConfirmDeleteTagStructViewEmits>()
</script>
