<template>
    <v-card class="pa-2">
        <v-card-title>
            {{ i18n.global.t("DELETE_REP_TITLE") }}
        </v-card-title>
        <div>{{ i18n.global.t("CONFIRM_DELETE_MESSAGE") }}</div>
        <div>{{ props.repository.file }}</div>
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary"
                    @click="emits('requested_delete_rep', repository); emits('requested_close_dialog')">{{
                        i18n.global.t("DELETE_TITLE") }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { ConfirmDeleteRepViewEmits } from './confirm-delete-rep-view-emits'
import type { ConfirmDeleteRepViewProps } from './confirm-delete-rep-view-props'

const props = defineProps<ConfirmDeleteRepViewProps>()
const emits = defineEmits<ConfirmDeleteRepViewEmits>()
</script>
