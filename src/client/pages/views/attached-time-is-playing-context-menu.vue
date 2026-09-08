<template>
    <v-menu v-model="is_show" :target="menu_target" location="bottom start">
        <v-list class="gkill_context_menu_list">
            <v-list-item @click="show_edit_timeis_dialog()">
                <v-list-item-title>{{ i18n.global.t("TIMEIS_CONTEXTMENU_EDIT_TIMEIS") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_timeis_histories_dialog()">
                <v-list-item-title>{{ i18n.global.t("TIMEIS_CONTEXTMENU_HISTORIES") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="copy_content()">
                <v-list-item-title>{{ i18n.global.t("TIMEIS_CONTEXTMENU_COPY_CONTENT") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="copy_id()">
                <v-list-item-title>{{ i18n.global.t("TIMEIS_CONTEXTMENU_COPY_ID") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_confirm_delete_timeis_dialog()">
                <v-list-item-title>{{ i18n.global.t("TIMEIS_CONTEXTMENU_DELETE") }}</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { AttachedTimeIsPlayingContextMenuProps } from './attached-time-is-playing-context-menu-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { useAttachedTimeIsPlayingContextMenu } from '@/classes/use-attached-time-is-playing-context-menu'

const props = defineProps<AttachedTimeIsPlayingContextMenuProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    is_show,
    menu_target,
    show,
    hide,
    show_edit_timeis_dialog,
    show_timeis_histories_dialog,
    copy_content,
    copy_id,
    show_confirm_delete_timeis_dialog,
} = useAttachedTimeIsPlayingContextMenu({ props, emits })

defineExpose({ show, hide })
</script>