<template>
    <v-menu v-model="is_show" :target="menu_target" location="bottom start">
        <v-list class="gkill_context_menu_list">
            <v-list-item @click="emits('requested_edit_rep', id)">
                <v-list-item-title>{{ i18n.global.t("EDIT_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="emits('requested_move_up_rep', id)">
                <v-list-item-title>{{ i18n.global.t("MOVE_UP_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="emits('requested_move_down_rep', id)">
                <v-list-item-title>{{ i18n.global.t("MOVE_DOWN_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="emits('requested_move_rep_to_folder', id)">
                <v-list-item-title>{{ i18n.global.t("MOVE_TO_FOLDER_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="emits('requested_delete_rep', id)">
                <v-list-item-title>{{ i18n.global.t("DELETE_TITLE") }}</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { RepStructContextMenuProps } from './rep-struct-context-menu-emits';
import type { RepStructContextMenuEmits } from './rep-struct-context-menu-props';
import { useRepStructContextMenu } from '@/classes/use-rep-struct-context-menu';

const props = defineProps<RepStructContextMenuProps>()
const emits = defineEmits<RepStructContextMenuEmits>()

const {
    id,
    is_show,
    menu_target,
    show,
    hide,
} = useRepStructContextMenu({ props, emits })

defineExpose({ show, hide })
</script>
