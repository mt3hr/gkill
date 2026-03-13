<template>
    <v-menu v-model="is_show" :style="context_menu_style">
        <v-list class="gkill_context_menu_list">
            <v-list-item @click="emits('requested_edit_dnote_item_list', id)">
                <v-list-item-title>{{ i18n.global.t("EDIT_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="emits('requested_delete_dnote_item_list', id)">
                <v-list-item-title>{{ i18n.global.t("DELETE_TITLE") }}</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { DnoteItemListContextMenuEmits } from './dnote-item-list-context-menu-emits';
import type { DnoteItemListContextMenuProps } from './dnote-item-list-context-menu-props';
import { useDnoteItemListContextMenu } from '@/classes/use-dnote-item-list-context-menu';

const props = defineProps<DnoteItemListContextMenuProps>()
const emits = defineEmits<DnoteItemListContextMenuEmits>()

const {
    id,
    is_show,
    context_menu_style,
    show,
    hide,
} = useDnoteItemListContextMenu({ props, emits })

defineExpose({ show, hide })
</script>
