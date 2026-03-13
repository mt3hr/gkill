<template>
    <v-menu v-model="is_show" :style="context_menu_style">
        <v-list class="gkill_context_menu_list">
            <v-list-item @click="show_edit_tag_dialog()">
                <v-list-item-title>{{ i18n.global.t("TAG_CONTEXTMENU_EDIT_TAG") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_tag_histories_dialog()">
                <v-list-item-title>{{ i18n.global.t("TAG_CONTEXTMENU_HISTORIES") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="copy_id()">
                <v-list-item-title>{{ i18n.global.t("TAG_CONTEXTMENU_COPY_ID") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_confirm_delete_tag_dialog()">
                <v-list-item-title>{{ i18n.global.t("TAG_CONTEXTMENU_DELETE") }}</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { AttachedTagContextMenuProps } from './attached-tag-context-menu-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { useAttachedTagContextMenu } from '@/classes/use-attached-tag-context-menu'

const props = defineProps<AttachedTagContextMenuProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    is_show,
    context_menu_style,
    show,
    hide,
    show_edit_tag_dialog,
    show_tag_histories_dialog,
    copy_id,
    show_confirm_delete_tag_dialog,
} = useAttachedTagContextMenu({ props, emits })

defineExpose({ show, hide })
</script>
