<template>
    <v-card class="pa-2">
        <v-card-title>
            {{ i18n.global.t("DELETE_KFTL_TEMPLATE_TITLE") }}
        </v-card-title>
        {{ kftl_template_struct.title }}
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="emits('requested_delete_kftl_template', kftl_template_struct.id)">{{
                    i18n.global.t("DELETE_TITLE") }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { ConfirmDeleteKFTLTemplateStructViewProps } from './confirm-delete-kftl_template-struct-view-props.ts';
import type { ConfirmDeleteKFTLTemplateStructViewEmits } from './confirm-delete-kftl_template-struct-view-emits.ts';

defineProps<ConfirmDeleteKFTLTemplateStructViewProps>()
const emits = defineEmits<ConfirmDeleteKFTLTemplateStructViewEmits>()
</script>
