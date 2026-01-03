<template>
    <v-menu v-model="is_show" :style="context_menu_style">
        <v-list class="gkill_context_menu_list">
            <v-list-item v-if="gkill_api.get_saved_last_added_tag() !== ''" @click="add_last_added_tag()">
                <v-list-item-title>{{ i18n.global.t("ADD_TAG_TITLE") }} 「{{ gkill_api.get_saved_last_added_tag()
                    }}」</v-list-item-title>
            </v-list-item>
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
            <v-list-item @click="show_edit_urlog_dialog()">
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
    <EditURLogDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
        @deleted_kyou="(...deleted_kyou :any[]) => emits('deleted_kyou', deleted_kyou[0] as Kyou)"
        @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
        @deleted_text="(...deleted_text :any[]) => emits('deleted_text', deleted_text[0] as Text)"
        @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
        @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
        @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
        @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
        @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
        @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
        @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
        @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
        @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
        @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
        @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
        @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)"
        ref="edit_urlog_dialog" />
    <AddTagDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
        @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
        @requested_reload_list="emits('requested_reload_list')" :kyou="kyou" :last_added_tag="last_added_tag"
        @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)" :enable_context_menu="enable_context_menu"
        @deleted_kyou="(...deleted_kyou :any[]) => emits('deleted_kyou', deleted_kyou[0] as Kyou)"
        @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
        @deleted_text="(...deleted_text :any[]) => emits('deleted_text', deleted_text[0] as Text)"
        @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
        @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
        @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
        @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
        @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
        @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
        @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
        @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
        @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
        :enable_dialog="enable_dialog" @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
        ref="add_tag_dialog" />
    <AddTextDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
        @deleted_kyou="(...deleted_kyou :any[]) => emits('deleted_kyou', deleted_kyou[0] as Kyou)"
        @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
        @deleted_text="(...deleted_text :any[]) => emits('deleted_text', deleted_text[0] as Text)"
        @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
        @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
        @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
        @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
        @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
        @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
        @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
        @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
        @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
        @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
        @requested_reload_list="emits('requested_reload_list')"
        @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
        @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" ref="add_text_dialog" />
    <AddNotificationDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
        @deleted_kyou="(...deleted_kyou :any[]) => emits('deleted_kyou', deleted_kyou[0] as Kyou)"
        @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
        @deleted_text="(...deleted_text :any[]) => emits('deleted_text', deleted_text[0] as Text)"
        @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
        @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
        @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
        @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
        @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
        @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
        @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
        @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
        @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
        @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
        @requested_reload_list="emits('requested_reload_list')"
        @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
        @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" ref="add_notification_dialog" />
    <ConfirmDeleteKyouDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
        @deleted_kyou="(...deleted_kyou :any[]) => emits('deleted_kyou', deleted_kyou[0] as Kyou)"
        @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
        @deleted_text="(...deleted_text :any[]) => emits('deleted_text', deleted_text[0] as Text)"
        @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
        @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
        @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
        @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
        @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
        @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
        @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
        @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
        @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
        @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
        @requested_reload_list="emits('requested_reload_list')"
        @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
        @received_messages="(...message: any[]) => emits('received_messages', message[0] as Array<GkillMessage>)" ref="confirm_delete_kyou_dialog" />
    <ConfirmReKyouDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
        @deleted_kyou="(...deleted_kyou :any[]) => emits('deleted_kyou', deleted_kyou[0] as Kyou)"
        @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
        @deleted_text="(...deleted_text :any[]) => emits('deleted_text', deleted_text[0] as Text)"
        @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
        @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
        @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
        @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
        @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
        @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
        @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
        @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
        @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
        @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
        @requested_reload_list="emits('requested_reload_list')"
        @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
        @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" ref="confirm_rekyou_dialog" />
    <kyouHistoriesDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
        @deleted_kyou="(...deleted_kyou :any[]) => emits('deleted_kyou', deleted_kyou[0] as Kyou)"
        @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
        @deleted_text="(...deleted_text :any[]) => emits('deleted_text', deleted_text[0] as Text)"
        @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
        @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
        @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
        @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
        @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
        @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
        @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
        @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
        @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
        @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
        @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)"
        ref="kyou_histories_dialog" />
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { KyouViewEmits } from './kyou-view-emits'
import type { URLogContextMenuProps } from './ur-log-context-menu-props'
import EditURLogDialog from '../dialogs/edit-ur-log-dialog.vue'
import AddTagDialog from '../dialogs/add-tag-dialog.vue'
import AddTextDialog from '../dialogs/add-text-dialog.vue'
import AddNotificationDialog from '../dialogs/add-notification-dialog.vue'
import ConfirmReKyouDialog from '../dialogs/confirm-re-kyou-dialog.vue'
import ConfirmDeleteKyouDialog from '../dialogs/confirm-delete-idf-kyou-dialog.vue'
import { computed, ref, type Ref } from 'vue'
import { GkillMessage } from '@/classes/api/gkill-message'
import kyouHistoriesDialog from '../dialogs/kyou-histories-dialog.vue'
import { OpenDirectoryRequest } from '@/classes/api/req_res/open-directory-request'
import { OpenFileRequest } from '@/classes/api/req_res/open-file-request'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'
import { AddTagRequest } from '@/classes/api/req_res/add-tag-request'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';
import type { GkillError } from '@/classes/api/gkill-error'
import type { Kyou } from '@/classes/datas/kyou'

const edit_urlog_dialog = ref<InstanceType<typeof EditURLogDialog> | null>(null);
const add_tag_dialog = ref<InstanceType<typeof AddTagDialog> | null>(null);
const add_text_dialog = ref<InstanceType<typeof AddTextDialog> | null>(null);
const add_notification_dialog = ref<InstanceType<typeof AddNotificationDialog> | null>(null);
const confirm_delete_kyou_dialog = ref<InstanceType<typeof ConfirmDeleteKyouDialog> | null>(null);
const confirm_rekyou_dialog = ref<InstanceType<typeof ConfirmReKyouDialog> | null>(null);
const kyou_histories_dialog = ref<InstanceType<typeof kyouHistoriesDialog> | null>(null);

const props = defineProps<URLogContextMenuProps>()
const emits = defineEmits<KyouViewEmits>()
defineExpose({ show })

const is_show: Ref<boolean> = ref(false)
const position_x: Ref<Number> = ref(0)
const position_y: Ref<Number> = ref(0)
const context_menu_style = computed(() => `{ position: absolute; left: ${Math.min(document.defaultView!.innerWidth - 130, position_x.value.valueOf())}px; top: ${Math.min(Math.max(50, document.defaultView!.innerHeight - (+ 8 + (48 * (8 + (props.gkill_api.get_saved_last_added_tag() !== "" ? 1 : 0) + (props.application_config.session_is_local ? 2 : 0))))), position_y.value.valueOf())}px; }`)

async function show(e: PointerEvent): Promise<void> {
    position_x.value = e.clientX
    position_y.value = e.clientY
    is_show.value = true
}

async function copy_id(): Promise<void> {
    navigator.clipboard.writeText(props.kyou.id)
    const message = new GkillMessage()
    message.message_code = GkillMessageCodes.copied_urlog_id
    message.message = i18n.global.t("COPIED_ID_MESSAGE")
    const messages = new Array<GkillMessage>()
    messages.push(message)
    emits('received_messages', messages)
}

async function show_edit_urlog_dialog(): Promise<void> {
    edit_urlog_dialog.value?.show()
}

async function show_add_tag_dialog(): Promise<void> {
    add_tag_dialog.value?.show()
}

async function show_add_text_dialog(): Promise<void> {
    add_text_dialog.value?.show()
}

async function show_confirm_delete_kyou_dialog(): Promise<void> {
    confirm_delete_kyou_dialog.value?.show()
}

async function show_add_notification_dialog(): Promise<void> {
    add_notification_dialog.value?.show()
}

async function show_confirm_rekyou_dialog(): Promise<void> {
    confirm_rekyou_dialog.value?.show()
}

async function show_kyou_histories_dialog(): Promise<void> {
    kyou_histories_dialog.value?.show()
}

async function open_folder(): Promise<void> {
    const req = new OpenDirectoryRequest()
    req.target_id = props.kyou.id
    const res = await props.gkill_api.open_directory(req)
    if (res.errors && res.errors.length > 0) {
        emits('received_errors', res.errors)
    }
    if (res.messages && res.messages.length > 0) {
        emits('received_messages', res.messages)
    }
}

async function open_file(): Promise<void> {
    const req = new OpenFileRequest()
    req.target_id = props.kyou.id
    const res = await props.gkill_api.open_file(req)
    if (res.errors && res.errors.length > 0) {
        emits('received_errors', res.errors)
    }
    if (res.messages && res.messages.length > 0) {
        emits('received_messages', res.messages)
    }
}

async function add_last_added_tag(): Promise<void> {
    // UserIDやDevice情報を取得する
    const tag_names = props.gkill_api.get_saved_last_added_tag().split("、")
    for (let i = 0; i < tag_names.length; i++) {
        const tag = tag_names[i]
        // タグ情報を用意する
        const new_tag = new Tag()
        new_tag.tag = tag
        new_tag.id = props.gkill_api.generate_uuid()
        new_tag.is_deleted = false
        new_tag.target_id = props.kyou.id
        new_tag.related_time = new Date(Date.now())
        new_tag.create_app = "gkill"
        new_tag.create_device = props.application_config.device
        new_tag.create_time = new Date(Date.now())
        new_tag.create_user = props.application_config.user_id
        new_tag.update_app = "gkill"
        new_tag.update_device = props.application_config.device
        new_tag.update_time = new Date(Date.now())
        new_tag.update_user = props.application_config.user_id

        // 追加リクエストを飛ばす
        await delete_gkill_kyou_cache(new_tag.id)
        await delete_gkill_kyou_cache(new_tag.target_id)
        const req = new AddTagRequest()
        req.tag = new_tag
        const res = await props.gkill_api.add_tag(req)
        if (res.errors && res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }
        emits('registered_tag', res.added_tag)
        emits('requested_reload_kyou', props.kyou)
    }
    return
}
</script>
