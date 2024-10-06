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
            <v-list-item @click="show_edit_timeis_dialog()">
                <v-list-item-title>編集</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_kyou_histories_dialog()">
                <v-list-item-title>履歴</v-list-item-title>
            </v-list-item>
            <v-list-item @click="copy_id()">
                <v-list-item-title>IDをコピー</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_confirm_delete_kyou_dialog()">
                <v-list-item-title>削除</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
    <EditTimeIsDialog v-if="kyou.typed_timeis" :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag" :timeis="kyou.typed_timeis"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
        ref="edit_timeis_dialog" />
    <AddTagDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
        :kyou="kyou" :last_added_tag="last_added_tag" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" ref="add_tag_dialog" />
    <AddTextDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" ref="add_text_dialog" />
    <ConfirmDeleteKyouDialog v-if="kyou.typed_timeis" :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(message) => emits('received_messages', message)" ref="confirm_delete_kyou_dialog" />
    <ConfirmReKyouDialog v-if="kyou.typed_timeis" :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" ref="confirm_rekyou_dialog" />
    <KyouHistoriesDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
        ref="kyou_histories_dialog" />
</template>
<script lang="ts" setup>
import type { TimeIsContextMenuProps } from './time-is-context-menu-props'
import type { KyouViewEmits } from './kyou-view-emits'
import EditTimeIsDialog from '../dialogs/edit-time-is-dialog.vue'
import AddTagDialog from '../dialogs/add-tag-dialog.vue'
import AddTextDialog from '../dialogs/add-text-dialog.vue'
import ConfirmReKyouDialog from '../dialogs/confirm-re-kyou-dialog.vue'
import ConfirmDeleteKyouDialog from '../dialogs/confirm-delete-kyou-dialog.vue'
import { InfoIdentifier } from '@/classes/datas/info-identifier'
import { type ComputedRef, type Ref, computed, ref } from 'vue'
import { GkillMessage } from '@/classes/api/gkill-message'
import KyouHistoriesDialog from '../dialogs/kyou-histories-dialog.vue'

const edit_timeis_dialog = ref<InstanceType<typeof EditTimeIsDialog> | null>(null);
const add_tag_dialog = ref<InstanceType<typeof AddTagDialog> | null>(null);
const add_text_dialog = ref<InstanceType<typeof AddTextDialog> | null>(null);
const confirm_delete_kyou_dialog = ref<InstanceType<typeof ConfirmDeleteKyouDialog> | null>(null);
const confirm_rekyou_dialog = ref<InstanceType<typeof ConfirmReKyouDialog> | null>(null);
const kyou_histories_dialog = ref<InstanceType<typeof KyouHistoriesDialog> | null>(null);

const props = defineProps<TimeIsContextMenuProps>()
const emits = defineEmits<KyouViewEmits>()
defineExpose({ show })
const latest_kyou_identifier: ComputedRef<Array<InfoIdentifier>> = computed(() => {
    const identifier = new InfoIdentifier()
    identifier.create_time = props.kyou.create_time
    identifier.id = props.kyou.id
    identifier.update_time = props.kyou.update_time
    return [identifier]
})

const is_show: Ref<boolean> = ref(false)
const position_x: Ref<Number> = ref(0)
const position_y: Ref<Number> = ref(0)
const context_menu_style = computed(() => `{ position: absolute; left: ${position_x.value}px; top: ${position_y.value}px; }`)

async function show(e: PointerEvent): Promise<void> {
    position_x.value = e.clientX
    position_y.value = e.clientY
    is_show.value = true
}

async function copy_id(): Promise<void> {
    navigator.clipboard.writeText(props.kyou.id)
    const message = new GkillMessage()
    message.message_code = "//TODO"
    message.message = "TimeIsIDをコピーしました"
    const messages = new Array<GkillMessage>()
    messages.push(message)
    emits('received_messages', messages)
}

async function show_edit_timeis_dialog(): Promise<void> {
    edit_timeis_dialog.value?.show()
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

async function show_confirm_rekyou_dialog(): Promise<void> {
    confirm_rekyou_dialog.value?.show()
}

async function show_kyou_histories_dialog(): Promise<void> {
    kyou_histories_dialog.value?.show()
}
</script>
