<template>
    <v-menu v-model="is_show" :target="menu_target" location="bottom start">
        <v-list class="gkill_context_menu_list">
            <v-list-item @click="emits('requested_edit_dnote_correlation_graph', id)">
                <v-list-item-title>{{ i18n.global.t("EDIT_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="emits('requested_delete_dnote_correlation_graph', id)">
                <v-list-item-title>{{ i18n.global.t("DELETE_TITLE") }}</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { DnoteCorrelationGraphContextMenuEmits } from './dnote-correlation-graph-context-menu-emits';
import type { DnoteCorrelationGraphContextMenuProps } from './dnote-correlation-graph-context-menu-props';
import { useDnoteCorrelationGraphContextMenu } from '@/classes/use-dnote-correlation-graph-context-menu';

const props = defineProps<DnoteCorrelationGraphContextMenuProps>()
const emits = defineEmits<DnoteCorrelationGraphContextMenuEmits>()

const {
    id,
    is_show,
    menu_target,
    show,
    hide,
} = useDnoteCorrelationGraphContextMenu({ props, emits })

defineExpose({ show, hide })
</script>
