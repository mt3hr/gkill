<template>
    <v-menu v-model="is_show" :style="context_menu_style">
        <v-list>
            <v-list-item @click="show_edit_timeis_dialog()">
                <v-list-item-title>TimeIs編集</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_timeis_histories_dialog()">
                <v-list-item-title>TimeIs履歴</v-list-item-title>
            </v-list-item>
            <v-list-item @click="copy_id()">
                <v-list-item-title>TimeIsIDコピー</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_confirm_delete_timeis_dialog()">
                <v-list-item-title>TimeIs削除</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
    <EditTimeIsDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="[timeis.generate_info_identifer()]" :kyou="kyou" :last_added_tag="last_added_tag"
        :timeis="timeis" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
        ref="edit_timeis_dialog" />
    <ConfirmDeleteKyouDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="[timeis.generate_info_identifer()]" :kyou="kyou" :last_added_tag="last_added_tag"
        :timeis="timeis" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
        ref="confirm_delete_kyou_dialog" />
    <KyouHistoriesDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="[timeis.generate_info_identifer()]" :kyou="kyou" :last_added_tag="last_added_tag"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
        ref="kyou_histories_dialog" />
</template>
<script lang="ts" setup>
import type { AttachedTimeisPlaingContextMenuProps } from './attached-timeis-plaing-context-menu-props'
import type { KyouViewEmits } from './kyou-view-emits'
import type { TimeIs } from '@/classes/datas/time-is'
import { computed, type Ref, ref } from 'vue'
import EditTimeIsDialog from '../dialogs/edit-time-is-dialog.vue'
import ConfirmDeleteKyouDialog from '../dialogs/confirm-delete-kyou-dialog.vue'
import KyouHistoriesDialog from '../dialogs/kyou-histories-dialog.vue'
import { InfoIdentifier } from '@/classes/datas/info-identifier'
import { GkillMessage } from '@/classes/api/gkill-message'

const edit_timeis_dialog = ref<InstanceType<typeof EditTimeIsDialog> | null>(null);
const confirm_delete_kyou_dialog = ref<InstanceType<typeof ConfirmDeleteKyouDialog> | null>(null);
const kyou_histories_dialog = ref<InstanceType<typeof KyouHistoriesDialog> | null>(null);

const props = defineProps<AttachedTimeisPlaingContextMenuProps>()
const emits = defineEmits<KyouViewEmits>()
defineExpose({ show, hide })

const is_show: Ref<boolean> = ref(false)
const position_x: Ref<Number> = ref(0)
const position_y: Ref<Number> = ref(0)
const context_menu_style = computed(() => `{ position: absolute; left: ${position_x.value}px; top: ${position_y.value}px; }`)

async function show(e: PointerEvent): Promise<void> {
    position_x.value = e.clientX
    position_y.value = e.clientY
    is_show.value = true
}

async function hide(): Promise<void> {
    is_show.value = false
}

async function show_edit_timeis_dialog(): Promise<void> {
    edit_timeis_dialog.value?.show()
}

async function show_timeis_histories_dialog(): Promise<void> {
    kyou_histories_dialog.value?.show()
}

async function copy_id(): Promise<void> {
    navigator.clipboard.writeText(props.timeis.id)
    const message = new GkillMessage()
    message.message_code = "//TODO"
    message.message = "TimeIsIDをコピーしました"
    const messages = new Array<GkillMessage>()
    messages.push(message)
    emits('received_messages', messages)
}

async function show_confirm_delete_timeis_dialog(): Promise<void> {
    confirm_delete_kyou_dialog.value?.show()
}
</script>
<style lang="css" scoped></style>
