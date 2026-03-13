<template>
    <v-menu v-model="is_show" :style="context_menu_style" :close-on-content-click="false">
        <v-list class="gkill_context_menu_list">
            <v-list-group v-if="tag_history.length > 0">
                <template v-slot:activator="{ props: activatorProps }">
                    <v-list-item v-bind="activatorProps">
                        <v-list-item-title>{{ i18n.global.t("ADD_TAG_FROM_HISTORY_TITLE") }}</v-list-item-title>
                    </v-list-item>
                </template>
                <v-list-item v-for="(history_tag, index) in tag_history" :key="index"
                    @click="add_tag_from_history(history_tag)">
                    <v-list-item-title>{{ history_tag }}</v-list-item-title>
                </v-list-item>
            </v-list-group>
            <v-list-item @click="show_add_tag_dialog()">
                <v-list-item-title>{{ i18n.global.t("ADD_TAG_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_add_text_dialog()">
                <v-list-item-title>{{ i18n.global.t("ADD_TEXT_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_confirm_rekyou_dialog()">
                <v-list-item-title>{{ i18n.global.t("REKYOU_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_add_notification_dialog()">
                <v-list-item-title>{{ i18n.global.t("ADD_NOTIFICATION_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_edit_nlog_dialog()">
                <v-list-item-title>{{ i18n.global.t("EDIT_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_kyou_histories_dialog()">
                <v-list-item-title>{{ i18n.global.t("KYOU_HISTORIES_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="copy_id()">
                <v-list-item-title>{{ i18n.global.t("COPY_ID_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item v-if="application_config.session_is_local" @click="open_folder()">
                <v-list-item-title>{{ i18n.global.t("OPEN_FOLDER_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item v-if="application_config.session_is_local" @click="open_file()">
                <v-list-item-title>{{ i18n.global.t("OPEN_FILE_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_confirm_delete_kyou_dialog()">
                <v-list-item-title>{{ i18n.global.t("DELETE_TITLE") }}</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { NlogContextMenuProps } from './nlog-context-menu-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { useNlogContextMenu } from '@/classes/use-nlog-context-menu'

const props = defineProps<NlogContextMenuProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    // State
    is_show,
    tag_history,
    context_menu_style,

    // Business logic
    show,
    copy_id,
    show_edit_nlog_dialog,
    show_add_tag_dialog,
    show_add_text_dialog,
    show_add_notification_dialog,
    show_confirm_delete_kyou_dialog,
    show_confirm_rekyou_dialog,
    show_kyou_histories_dialog,
    open_folder,
    open_file,
    add_tag_from_history,
} = useNlogContextMenu({ props, emits })

defineExpose({ show })
</script>
