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
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { KyouViewEmits } from './kyou-view-emits'
import type { URLogContextMenuProps } from './ur-log-context-menu-props'
import { computed, ref, type Ref } from 'vue'
import { GkillMessage } from '@/classes/api/gkill-message'
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
    emits('requested_open_rykv_dialog', 'edit_urlog', props.kyou)
}

async function show_add_tag_dialog(): Promise<void> {
    emits('requested_open_rykv_dialog', 'add_tag', props.kyou)
}

async function show_add_text_dialog(): Promise<void> {
    emits('requested_open_rykv_dialog', 'add_text', props.kyou)
}

async function show_confirm_delete_kyou_dialog(): Promise<void> {
    emits('requested_open_rykv_dialog', 'confirm_delete_kyou', props.kyou)
}

async function show_add_notification_dialog(): Promise<void> {
    emits('requested_open_rykv_dialog', 'add_notification', props.kyou)
}

async function show_confirm_rekyou_dialog(): Promise<void> {
    emits('requested_open_rykv_dialog', 'confirm_re_kyou', props.kyou)
}

async function show_kyou_histories_dialog(): Promise<void> {
    emits('requested_open_rykv_dialog', 'kyou_histories', props.kyou)
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
