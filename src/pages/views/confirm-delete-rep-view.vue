<template>
    <v-card class="pa-2">
        <v-card-title>
            Rep削除
        </v-card-title>
        <div>削除しますか？</div>
        <div>{{ props.repository.file }}</div>
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn color="primary" @click="emits('requested_delete_rep', repository); emits('requested_close_dialog')">削除</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import type { ConfirmDeleteRepViewEmits } from './confirm-delete-rep-view-emits'
import type { ConfirmDeleteRepViewProps } from './confirm-delete-rep-view-props'

const props = defineProps<ConfirmDeleteRepViewProps>()
const emits = defineEmits<ConfirmDeleteRepViewEmits>()
</script>
