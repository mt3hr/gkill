<template>
    <v-menu v-model="is_show" :style="context_menu_style">
        <v-list class="gkill_context_menu_list">
            <v-list-item @click="emits('requested_edit_tag', id)">
                <v-list-item-title>{{ i18n.global.t("EDIT_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="emits('requested_delete_tag', id)">
                <v-list-item-title>{{ i18n.global.t("DELETE_TITLE") }}</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { TagStructContextMenuEmits } from './tag-struct-context-menu-emits';
import type { TagStructContextMenuProps } from './tag-struct-context-menu-props';
import { useTagStructContextMenu } from '@/classes/use-tag-struct-context-menu';

const props = defineProps<TagStructContextMenuProps>()
const emits = defineEmits<TagStructContextMenuEmits>()

const {
    id,
    is_show,
    context_menu_style,
    show,
    hide,
} = useTagStructContextMenu({ props, emits })

defineExpose({ show, hide })
</script>
