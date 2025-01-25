<template>
    <v-menu v-model="is_show" :style="context_menu_style">
        <v-list>
            <v-list-item @click="show_add_tag_dialog()">
                <v-list-item-title>タグ追加</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_add_text_dialog()">
                <v-list-item-title>テキスト追加</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_confirm_rekyou_dialog()">
                <v-list-item-title>リポスト</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_add_notification_dialog()">
                <v-list-item-title>通知追加</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_edit_idf_kyou_dialog()">
                <v-list-item-title>編集</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_kyou_histories_dialog()">
                <v-list-item-title>履歴</v-list-item-title>
            </v-list-item>
            <v-list-item @click="copy_id()">
                <v-list-item-title>IDをコピー</v-list-item-title>
            </v-list-item>
            <v-list-item v-if="application_config.session_is_local" @click="open_folder()">
                <v-list-item-title>フォルダを開く</v-list-item-title>
            </v-list-item>
            <v-list-item v-if="application_config.session_is_local" @click="open_file()">
                <v-list-item-title>ファイルを開く</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_confirm_delete_kyou_dialog()">
                <v-list-item-title>削除</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
    <EditIDFKyouDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
        ref="edit_idf_kyou_dialog" />
    <AddTagDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
        :kyou="kyou" :last_added_tag="last_added_tag" @received_errors="(errors) => emits('received_errors', errors)"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')"
        @received_messages="(messages) => emits('received_messages', messages)" ref="add_tag_dialog" />
    <AddTextDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" ref="add_text_dialog" />
    <ConfirmDeleteKyouDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(message) => emits('received_messages', message)" ref="confirm_delete_kyou_dialog" />
    <AddNotificationDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" ref="add_notification_dialog" />
    <ConfirmReKyouDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" ref="confirm_rekyou_dialog" />
    <KyouHistoriesDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        @received_messages="(messages) => emits('received_messages', messages)"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
        ref="kyou_histories_dialog" />
</template>
<script lang="ts" setup>
import type { IDFKyouContextMenuProps } from './idf-kyou-context-menu-props'
import type { KyouViewEmits } from './kyou-view-emits'
import EditIDFKyouDialog from '../dialogs/edit-idf-kyou-dialog.vue'
import AddTagDialog from '../dialogs/add-tag-dialog.vue'
import AddTextDialog from '../dialogs/add-text-dialog.vue'
import AddNotificationDialog from '../dialogs/add-notification-dialog.vue'
import ConfirmReKyouDialog from '../dialogs/confirm-re-kyou-dialog.vue'
import ConfirmDeleteKyouDialog from '../dialogs/confirm-delete-idf-kyou-dialog.vue'
import { type Ref, computed, ref } from 'vue'
import { GkillMessage } from '@/classes/api/gkill-message'
import KyouHistoriesDialog from '../dialogs/kyou-histories-dialog.vue'
import { OpenDirectoryRequest } from '@/classes/api/req_res/open-directory-request'
import { OpenFileRequest } from '@/classes/api/req_res/open-file-request'

const edit_idf_kyou_dialog = ref<InstanceType<typeof EditIDFKyouDialog> | null>(null);
const add_tag_dialog = ref<InstanceType<typeof AddTagDialog> | null>(null);
const add_text_dialog = ref<InstanceType<typeof AddTextDialog> | null>(null);
const add_notification_dialog = ref<InstanceType<typeof AddNotificationDialog> | null>(null);
const confirm_delete_kyou_dialog = ref<InstanceType<typeof ConfirmDeleteKyouDialog> | null>(null);
const confirm_rekyou_dialog = ref<InstanceType<typeof ConfirmReKyouDialog> | null>(null);
const kyou_histories_dialog = ref<InstanceType<typeof KyouHistoriesDialog> | null>(null);

const props = defineProps<IDFKyouContextMenuProps>()
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
    message.message_code = "//TODO"
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
</script>
