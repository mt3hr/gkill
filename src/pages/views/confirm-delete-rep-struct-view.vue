<template>
    <v-card class="pa-2">
        <v-card-title>
            記録保管場所構造削除
        </v-card-title>
        {{ rep_struct.rep_name }}
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn color="primary" @click="emits('requested_delete_rep', rep_struct.id)">削除</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import type { ConfirmDeleteRepStructViewProps } from './confirm-delete-rep-struct-view-props';
import type { ConfirmDeleteRepStructViewEmits } from './confirm-delete-rep-struct-view-emits';

defineProps<ConfirmDeleteRepStructViewProps>()
const emits = defineEmits<ConfirmDeleteRepStructViewEmits>()
</script>
