<template>
    <v-card variant="flat" class="pa-2">
        <v-card-title>
            {{ i18n.global.t("DELETE_MI_BOARD_STRUCT_TITLE") }}
        </v-card-title>
        {{ mi_board_struct.name }}
        <v-row class="pa-0 ma-0 gkill-dialog-actions">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" v-if="mi_board_struct && mi_board_struct.id"
                    @click="emits('requested_delete_mi_board', mi_board_struct.id)">{{ i18n.global.t("DELETE_TITLE")
                    }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { ConfirmDeleteMiBoardStructViewProps } from './confirm-delete-mi-board-struct-view-props';
import type { ConfirmDeleteMiBoardStructViewEmits } from './confirm-delete-mi-board-struct-view-emits';

defineProps<ConfirmDeleteMiBoardStructViewProps>()
const emits = defineEmits<ConfirmDeleteMiBoardStructViewEmits>()
</script>
