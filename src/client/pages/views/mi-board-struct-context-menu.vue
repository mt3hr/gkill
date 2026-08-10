<template>
    <v-menu v-model="is_show" :target="menu_target" location="bottom start">
        <v-list class="gkill_context_menu_list">
            <v-list-item @click="emits('requested_move_up_mi_board', id)">
                <v-list-item-title>{{ i18n.global.t("MOVE_UP_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="emits('requested_move_down_mi_board', id)">
                <v-list-item-title>{{ i18n.global.t("MOVE_DOWN_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="emits('requested_delete_mi_board', id)">
                <v-list-item-title>{{ i18n.global.t("DELETE_TITLE") }}</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { MiBoardStructContextMenuEmits } from './mi-board-struct-context-menu-emits';
import type { MiBoardStructContextMenuProps } from './mi-board-struct-context-menu-props';
import { useMiBoardStructContextMenu } from '@/classes/use-mi-board-struct-context-menu';

const props = defineProps<MiBoardStructContextMenuProps>()
const emits = defineEmits<MiBoardStructContextMenuEmits>()

const {
    id,
    is_show,
    menu_target,
    show,
    hide,
} = useMiBoardStructContextMenu({ props, emits })

defineExpose({ show, hide })
</script>
