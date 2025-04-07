<template>
    <v-card class="pa-2">
        <v-card-title>
            {{ $t("DELETE_REP_STRUCT_TITLE") }}
        </v-card-title>
        {{ rep_struct.rep_name }}
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="emits('requested_delete_rep', rep_struct.id)">{{ $t("DELETE_TITLE")
                    }}</v-btn>
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
