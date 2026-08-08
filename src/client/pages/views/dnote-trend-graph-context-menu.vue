<template>
    <v-menu v-model="is_show" :target="menu_target" location="bottom start">
        <v-list class="gkill_context_menu_list">
            <v-list-item @click="emits('requested_edit_dnote_trend_graph', id)">
                <v-list-item-title>{{ i18n.global.t("EDIT_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="emits('requested_delete_dnote_trend_graph', id)">
                <v-list-item-title>{{ i18n.global.t("DELETE_TITLE") }}</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { DnoteTrendGraphContextMenuEmits } from './dnote-trend-graph-context-menu-emits';
import type { DnoteTrendGraphContextMenuProps } from './dnote-trend-graph-context-menu-props';
import { useDnoteTrendGraphContextMenu } from '@/classes/use-dnote-trend-graph-context-menu';

const props = defineProps<DnoteTrendGraphContextMenuProps>()
const emits = defineEmits<DnoteTrendGraphContextMenuEmits>()

const {
    id,
    is_show,
    menu_target,
    show,
    hide,
} = useDnoteTrendGraphContextMenu({ props, emits })

defineExpose({ show, hide })
</script>
