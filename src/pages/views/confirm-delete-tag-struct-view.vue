<template>
    <v-card class="pa-2">
        <v-card-title>
            {{ i18n.global.t("DELETE_TAG_STRUCT_TITLE") }}
        </v-card-title>
        {{ tag_struct.tag_name }}
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary"
                    @click="emits('requested_delete_tag', tag_struct.id)">{{ i18n.global.t("DELETE_TITLE") }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { ConfirmDeleteTagStructViewProps } from './confirm-delete-tag-struct-view-props';
import type { ConfirmDeleteTagStructViewEmits } from './confirm-delete-tag-struct-view-emits';

defineProps<ConfirmDeleteTagStructViewProps>()
const emits = defineEmits<ConfirmDeleteTagStructViewEmits>()
</script>
