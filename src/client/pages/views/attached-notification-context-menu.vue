<template>
    <v-menu v-model="is_show" :style="context_menu_style">
        <v-list class="gkill_context_menu_list">
            <v-list-item @click="show_edit_notification_dialog()">
                <v-list-item-title>{{ i18n.global.t("NOTIFICATION_CONTEXTMENU_EDIT_NOTIFICATION") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_notification_histories_dialog()">
                <v-list-item-title>{{ i18n.global.t("NOTIFICATION_CONTEXTMENU_HISTORIES") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="copy_id()">
                <v-list-item-title>{{ i18n.global.t("NOTIFICATION_CONTEXTMENU_COPY_ID") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_confirm_delete_notification_dialog()">
                <v-list-item-title>{{ i18n.global.t("NOTIFICATION_CONTEXTMENU_DELETE") }}</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { AttachedNotificationContextMenuProps } from './attached-notification-context-menu-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { useAttachedNotificationContextMenu } from '@/classes/use-attached-notification-context-menu'

const props = defineProps<AttachedNotificationContextMenuProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    is_show,
    context_menu_style,
    show,
    hide,
    show_edit_notification_dialog,
    show_notification_histories_dialog,
    copy_id,
    show_confirm_delete_notification_dialog,
} = useAttachedNotificationContextMenu({ props, emits })

defineExpose({ show, hide })
</script>
