<template>
    <v-menu v-model="is_show" :style="context_menu_style">
        <v-list>
            <v-list-item v-if="gkill_api.get_saved_last_added_tag() !== ''" @click="add_last_added_tag()">
                <v-list-item-title>{{ $t("ADD_TAG_TITLE") }} 「{{ gkill_api.get_saved_last_added_tag()
                    }}」</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_add_tag_dialog()">
                <v-list-item-title>{{ $t("ADD_TAG_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_add_text_dialog()">
                <v-list-item-title>{{ $t("ADD_TEXT_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_confirm_rekyou_dialog()">
                <v-list-item-title>{{ $t("REKYOU_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_add_notification_dialog()">
                <v-list-item-title>{{ $t("ADD_NOTIFICATION_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_edit_idf_kyou_dialog()">
                <v-list-item-title>{{ $t("EDIT_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_kyou_histories_dialog()">
                <v-list-item-title>{{ $t("KYOU_HISTORIES_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="copy_id()">
                <v-list-item-title>{{ $t("COPY_ID_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item v-if="application_config.session_is_local" @click="open_folder()">
                <v-list-item-title>{{ $t("OPEN_FOLDER_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item v-if="application_config.session_is_local" @click="open_file()">
                <v-list-item-title>{{ $t("OPEN_FILE_TITLE") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_confirm_delete_kyou_dialog()">
                <v-list-item-title>{{ $t("DELETE_TITLE") }}</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
    <EditReKyouDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
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
        ref="edit_idf_kyou_dialog" />
    <AddTagDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
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
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')" :kyou="kyou" :last_added_tag="last_added_tag"
        @received_errors="(errors) => emits('received_errors', errors)" :enable_context_menu="enable_context_menu"
        :enable_dialog="enable_dialog" @received_messages="(messages) => emits('received_messages', messages)"
        ref="add_tag_dialog" />
    <AddTextDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
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
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" ref="add_text_dialog" />
    <AddNotificationDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
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
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" ref="add_notification_dialog" />
    <ConfirmDeleteKyouDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
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
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(message) => emits('received_messages', message)" ref="confirm_delete_kyou_dialog" />
    <ConfirmReKyouDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
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
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" ref="confirm_rekyou_dialog" />
    <KyouHistoriesDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
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
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
        ref="kyou_histories_dialog" />
</template>
<script lang="ts" setup>
import type { KyouViewEmits } from './kyou-view-emits'
import EditReKyouDialog from '../dialogs/edit-re-kyou-dialog.vue'
import AddTagDialog from '../dialogs/add-tag-dialog.vue'
import AddTextDialog from '../dialogs/add-text-dialog.vue'
import AddNotificationDialog from '../dialogs/add-notification-dialog.vue'
import ConfirmReKyouDialog from '../dialogs/confirm-re-kyou-dialog.vue'
import ConfirmDeleteKyouDialog from '../dialogs/confirm-delete-idf-kyou-dialog.vue'
import { type Ref, computed, ref } from 'vue'
import { GkillMessage } from '@/classes/api/gkill-message'
import KyouHistoriesDialog from '../dialogs/kyou-histories-dialog.vue'
import type { ReKyouContextMenuProps } from './re-kyou-context-menu-props'
import { OpenDirectoryRequest } from '@/classes/api/req_res/open-directory-request'
import { OpenFileRequest } from '@/classes/api/req_res/open-file-request'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'
import { AddTagRequest } from '@/classes/api/req_res/add-tag-request'
import { Tag } from '@/classes/datas/tag'
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const edit_idf_kyou_dialog = ref<InstanceType<typeof EditReKyouDialog> | null>(null);
const add_tag_dialog = ref<InstanceType<typeof AddTagDialog> | null>(null);
const add_text_dialog = ref<InstanceType<typeof AddTextDialog> | null>(null);
const add_notification_dialog = ref<InstanceType<typeof AddNotificationDialog> | null>(null);
const confirm_delete_kyou_dialog = ref<InstanceType<typeof ConfirmDeleteKyouDialog> | null>(null);
const confirm_rekyou_dialog = ref<InstanceType<typeof ConfirmReKyouDialog> | null>(null);
const kyou_histories_dialog = ref<InstanceType<typeof KyouHistoriesDialog> | null>(null);

const props = defineProps<ReKyouContextMenuProps>()
const emits = defineEmits<KyouViewEmits>()
defineExpose({ show })

const is_show: Ref<boolean> = ref(false)
const position_x: Ref<Number> = ref(0)
const position_y: Ref<Number> = ref(0)
const context_menu_style = computed(() => `{ position: absolute; left: ${Math.min(document.defaultView!.innerWidth - 130, position_x.value.valueOf())}px; top: ${Math.min(document.defaultView!.innerHeight - (props.application_config.session_is_local ? 500 : 400), position_y.value.valueOf())}px; }`)

async function show(e: PointerEvent): Promise<void> {
    position_x.value = e.clientX
    position_y.value = e.clientY
    is_show.value = true
}

async function copy_id(): Promise<void> {
    navigator.clipboard.writeText(props.kyou.id)
    const message = new GkillMessage()
    message.message_code = GkillMessageCodes.copied_rekyou_id
    message.message = "KyouIDをコピーしました"
    const messages = new Array<GkillMessage>()
    messages.push(message)
    emits('received_messages', messages)
}

async function show_edit_idf_kyou_dialog(): Promise<void> {
    edit_idf_kyou_dialog.value?.show()
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
    const get_gkill_req = new GetGkillInfoRequest()
    const gkill_info_res = await props.gkill_api.get_gkill_info(get_gkill_req)
    if (gkill_info_res.errors && gkill_info_res.errors.length !== 0) {
        emits('received_errors', gkill_info_res.errors)
        return
    }

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
        new_tag.create_device = gkill_info_res.device
        new_tag.create_time = new Date(Date.now())
        new_tag.create_user = gkill_info_res.user_id
        new_tag.update_app = "gkill"
        new_tag.update_device = gkill_info_res.device
        new_tag.update_time = new Date(Date.now())
        new_tag.update_user = gkill_info_res.user_id

        // 追加リクエストを飛ばす
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
