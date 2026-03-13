<template>
    <v-menu v-model="is_show" :style="context_menu_style">
        <v-list class="gkill_context_menu_list">
            <v-list-item @click="emits('requested_edit_dnote_list_query', id)">
                <v-list-item-title>{{ i18n.global.t("EDIT_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="emits('requested_delete_dnote_list_query', id)">
                <v-list-item-title>{{ i18n.global.t("DELETE_TITLE") }}</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { DnoteListQueryContextMenuEmits } from './dnote-list-query-context-menu-emits';
import type { DnoteListQueryContextMenuProps } from './dnote-list-query-context-menu-props';
import { useDnoteListQueryContextMenu } from '@/classes/use-dnote-list-query-context-menu';

const props = defineProps<DnoteListQueryContextMenuProps>()
const emits = defineEmits<DnoteListQueryContextMenuEmits>()

const {
    id,
    is_show,
    context_menu_style,
    show,
    hide,
} = useDnoteListQueryContextMenu({ props, emits })

defineExpose({ show, hide })
</script>
