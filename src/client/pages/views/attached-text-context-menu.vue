<template>
    <v-menu v-model="is_show" :style="context_menu_style">
        <v-list class="gkill_context_menu_list">
            <v-list-item @click="show_edit_text_dialog()">
                <v-list-item-title>{{ i18n.global.t("TEXT_CONTEXTMENU_EDIT_TEXT") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_text_histories_dialog()">
                <v-list-item-title>{{ i18n.global.t("TEXT_CONTEXTMENU_HISTORIES") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="copy_id()">
                <v-list-item-title>{{ i18n.global.t("TEXT_CONTEXTMENU_COPY_ID") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_confirm_delete_text_dialog()">
                <v-list-item-title>{{ i18n.global.t("TEXT_CONTEXTMENU_DELETE") }}</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { AttachedTextContextMenuProps } from './attached-text-context-menu-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { useAttachedTextContextMenu } from '@/classes/use-attached-text-context-menu'

const props = defineProps<AttachedTextContextMenuProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    is_show,
    context_menu_style,
    show,
    hide,
    show_edit_text_dialog,
    show_text_histories_dialog,
    copy_id,
    show_confirm_delete_text_dialog,
} = useAttachedTextContextMenu({ props, emits })

defineExpose({ show, hide })
</script>
