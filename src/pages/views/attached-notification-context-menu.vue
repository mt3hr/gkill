<template>
    <v-menu v-model="is_show" :style="context_menu_style">
        <v-list>
            <v-list-item @click="show_edit_notification_dialog()">
                <v-list-item-title>{{ $t("NOTIFICATION_CONTEXTMENU_EDIT_NOTIFICATION") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_notification_histories_dialog()">
                <v-list-item-title>{{ $t("NOTIFICATION_CONTEXTMENU_HISTORIES") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="copy_id()">
                <v-list-item-title>{{ $t("NOTIFICATION_CONTEXTMENU_COPY_ID") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_confirm_delete_notification_dialog()">
                <v-list-item-title>{{ $t("NOTIFICATION_CONTEXTMENU_DELETE") }}</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
    <EditNotificationDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        :notification="notification" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
        @deleted_kyou="(deleted_kyou) => emits('deleted_kyou', deleted_kyou)"
        @deleted_tag="(deleted_tag) => emits('deleted_tag', deleted_tag)"
        @deleted_text="(deleted_text) => emits('deleted_text', deleted_text)"
        @deleted_notification="(deleted_notification) => emits('deleted_notification', deleted_notification)"
        @registered_kyou="(registered_kyou) => emits('registered_kyou', registered_kyou)"
        @registered_tag="(registered_tag) => emits('registered_tag', registered_tag)"
        @registered_text="(registered_text) => emits('registered_text', registered_text)"
        @registered_notification="(registered_notification) => emits('registered_notification', registered_notification)"
        @updated_kyou="(updated_kyou) => emits('updated_kyou', updated_kyou)"
        @updated_tag="(updated_tag) => emits('updated_tag', updated_tag)"
        @updated_text="(updated_text) => emits('updated_text', updated_text)"
        @updated_notification="(updated_notification) => emits('updated_notification', updated_notification)"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
        ref="edit_notification_dialog" />
    <ConfirmDeleteNotificationDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        :notification="notification" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
        @deleted_kyou="(deleted_kyou) => emits('deleted_kyou', deleted_kyou)"
        @deleted_tag="(deleted_tag) => emits('deleted_tag', deleted_tag)"
        @deleted_text="(deleted_text) => emits('deleted_text', deleted_text)"
        @deleted_notification="(deleted_notification) => emits('deleted_notification', deleted_notification)"
        @registered_kyou="(registered_kyou) => emits('registered_kyou', registered_kyou)"
        @registered_tag="(registered_tag) => emits('registered_tag', registered_tag)"
        @registered_text="(registered_text) => emits('registered_text', registered_text)"
        @registered_notification="(registered_notification) => emits('registered_notification', registered_notification)"
        @updated_kyou="(updated_kyou) => emits('updated_kyou', updated_kyou)"
        @updated_tag="(updated_tag) => emits('updated_tag', updated_tag)"
        @updated_text="(updated_text) => emits('updated_text', updated_text)"
        @updated_notification="(updated_notification) => emits('updated_notification', updated_notification)"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
        ref="confirm_delete_notification_dialog" />
    <NotificationHistoriesDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        :notification="notification" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
        @deleted_kyou="(deleted_kyou) => emits('deleted_kyou', deleted_kyou)"
        @deleted_tag="(deleted_tag) => emits('deleted_tag', deleted_tag)"
        @deleted_text="(deleted_text) => emits('deleted_text', deleted_text)"
        @deleted_notification="(deleted_notification) => emits('deleted_notification', deleted_notification)"
        @registered_kyou="(registered_kyou) => emits('registered_kyou', registered_kyou)"
        @registered_tag="(registered_tag) => emits('registered_tag', registered_tag)"
        @registered_text="(registered_text) => emits('registered_text', registered_text)"
        @registered_notification="(registered_notification) => emits('registered_notification', registered_notification)"
        @updated_kyou="(updated_kyou) => emits('updated_kyou', updated_kyou)"
        @updated_tag="(updated_tag) => emits('updated_tag', updated_tag)"
        @updated_text="(updated_text) => emits('updated_text', updated_text)"
        @updated_notification="(updated_notification) => emits('updated_notification', updated_notification)"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
        ref="notification_histories_dialog" />
</template>
<script lang="ts" setup>
import EditNotificationDialog from '../dialogs/edit-notification-dialog.vue';
import ConfirmDeleteNotificationDialog from '../dialogs/confirm-delete-notification-dialog.vue';
import NotificationHistoriesDialog from '../dialogs/notification-histories-dialog.vue';
import type { KyouViewEmits } from './kyou-view-emits'
import { computed, type Ref, ref } from 'vue'
import { GkillMessage } from '@/classes/api/gkill-message'
import type { AttachedNotificationContextMenuProps } from './attached-notification-context-menu-props';
import { GkillMessageCodes } from '@/classes/api/message/gkill_message';
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const edit_notification_dialog = ref<InstanceType<typeof EditNotificationDialog> | null>(null);
const confirm_delete_notification_dialog = ref<InstanceType<typeof ConfirmDeleteNotificationDialog> | null>(null);
const notification_histories_dialog = ref<InstanceType<typeof NotificationHistoriesDialog> | null>(null);

const props = defineProps<AttachedNotificationContextMenuProps>()
const emits = defineEmits<KyouViewEmits>()
defineExpose({ show, hide })

const is_show: Ref<boolean> = ref(false)
const position_x: Ref<Number> = ref(0)
const position_y: Ref<Number> = ref(0)
const context_menu_style = computed(() => `{ position: absolute; left: ${Math.min(document.defaultView!.innerWidth - 130, position_x.value.valueOf())}px; top: ${Math.min(document.defaultView!.innerHeight - (props.application_config.session_is_local ? 500 : 400), position_y.value.valueOf())}px; }`)

async function show(e: PointerEvent): Promise<void> {
    position_x.value = e.clientX
    position_y.value = e.clientY
    is_show.value = true
}

async function hide(): Promise<void> {
    is_show.value = false
}

async function show_edit_notification_dialog(): Promise<void> {
    edit_notification_dialog.value?.show()
}

async function show_notification_histories_dialog(): Promise<void> {
    notification_histories_dialog.value?.show()
}

async function copy_id(): Promise<void> {
    navigator.clipboard.writeText(props.notification.id)
    const message = new GkillMessage()
    message.message_code = GkillMessageCodes.copied_notification_id
    message.message = t("COPIED_ID_MESSAGE")
    const messages = new Array<GkillMessage>()
    messages.push(message)
    emits('received_messages', messages)
}

async function show_confirm_delete_notification_dialog(): Promise<void> {
    confirm_delete_notification_dialog.value?.show()
}
</script>
<style lang="css" scoped></style>
