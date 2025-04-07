<template>
    <v-card class="pa-2">
        <v-card-title>
            {{ $t("DELETE_REP_TYPE_TITLE") }}
        </v-card-title>
        {{ rep_type_struct.rep_type_name }}
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="emits('requested_delete_rep_type', rep_type_struct.id)">{{
                    $t("DELETE_TITLE") }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import type { ConfirmDeleteRepTypeStructViewProps } from './confirm-delete-rep-type-struct-view-props';
import type { ConfirmDeleteRepTypeStructViewEmits } from './confirm-delete-rep-type-struct-view-emits';

defineProps<ConfirmDeleteRepTypeStructViewProps>()
const emits = defineEmits<ConfirmDeleteRepTypeStructViewEmits>()
</script>
