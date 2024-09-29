<template>
    <v-menu v-model="is_show" :style="context_menu_style">
        <v-list>
            <v-list-item @click="show_edit_text_dialog()">
                <v-list-item-title>テキスト編集</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_text_histories_dialog()">
                <v-list-item-title>テキスト履歴</v-list-item-title>
            </v-list-item>
            <v-list-item @click="copy_id()">
                <v-list-item-title>テキストIDコピー</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_confirm_delete_text_dialog()">
                <v-list-item-title>テキスト削除</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
    <EditTextDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="[generate_info_identifer_from_text(text)]" :kyou="kyou" :last_added_tag="last_added_tag"
        :text="text" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
        ref="edit_text_dialog" />
    <ConfirmDeleteTextDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="[generate_info_identifer_from_text(text)]" :kyou="kyou" :last_added_tag="last_added_tag"
        :text="text" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
        ref="confirm_delete_text_dialog" />
    <TextHistoriesDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="[generate_info_identifer_from_text(text)]" :kyou="kyou" :last_added_tag="last_added_tag"
        :text="text" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
        ref="text_histories_dialog" />
</template>
<script lang="ts" setup>
import type { AttachedTextContextMenuProps } from './attached-text-context-menu-props'
import type { KyouViewEmits } from './kyou-view-emits'
import type { Text } from '@/classes/datas/text'
import { computed, type Ref, ref } from 'vue'
import EditTextDialog from '../dialogs/edit-text-dialog.vue'
import ConfirmDeleteTextDialog from '../dialogs/confirm-delete-text-dialog.vue'
import TextHistoriesDialog from '../dialogs/text-histories-dialog.vue'
import { InfoIdentifier } from '@/classes/datas/info-identifier'
import { GkillMessage } from '@/classes/api/gkill-message'

const edit_text_dialog = ref<InstanceType<typeof EditTextDialog> | null>(null);
const confirm_delete_text_dialog = ref<InstanceType<typeof ConfirmDeleteTextDialog> | null>(null);
const text_histories_dialog = ref<InstanceType<typeof TextHistoriesDialog> | null>(null);

const props = defineProps<AttachedTextContextMenuProps>()
const emits = defineEmits<KyouViewEmits>()
defineExpose({ show, hide })

const is_show: Ref<boolean> = ref(false)
const position_x: Ref<Number> = ref(0)
const position_y: Ref<Number> = ref(0)
const context_menu_style = computed(() => `{ position: absolute; left: ${position_x.value}px; top: ${position_y.value}px; }`)

function generate_info_identifer_from_text(text: Text): InfoIdentifier {
    const info_identifer = new InfoIdentifier()
    info_identifer.create_time = text.create_time
    info_identifer.id = text.id
    info_identifer.update_time = text.update_time
    return info_identifer
}

async function show(e: PointerEvent): Promise<void> {
    position_x.value = e.clientX
    position_y.value = e.clientY
    is_show.value = true
}

async function hide(): Promise<void> {
    is_show.value = false
}

async function show_edit_text_dialog(): Promise<void> {
    edit_text_dialog.value?.show()
}

async function show_text_histories_dialog(): Promise<void> {
    text_histories_dialog.value?.show()
}

async function copy_id(): Promise<void> {
    navigator.clipboard.writeText(props.text.id)
    const message = new GkillMessage()
    message.message_code = "//TODO"
    message.message = "テキストIDをコピーしました"
    const messages = new Array<GkillMessage>()
    messages.push(message)
    emits('received_messages', messages)
}

async function show_confirm_delete_text_dialog(): Promise<void> {
    confirm_delete_text_dialog.value?.show()
}
</script>
<style lang="css" scoped></style>
