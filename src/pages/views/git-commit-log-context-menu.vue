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
            <v-list-item @click="copy_id()">
                <v-list-item-title>IDをコピー</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>

    <AddTagDialog :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
        :kyou="kyou" :last_added_tag="last_added_tag" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)" @requested_reload_list="() => { }"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
    <AddTextDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)" @requested_reload_list="() => { }"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
    <ConfirmReKyouDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)" @requested_reload_list="() => { }"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
</template>
<script lang="ts" setup>
import type { GitCommitLogContextMenuProps } from './git-commit-log-context-menu-props'
import type { KyouViewEmits } from './kyou-view-emits'
import AddTagDialog from '../dialogs/add-tag-dialog.vue'
import AddTextDialog from '../dialogs/add-text-dialog.vue'
import ConfirmReKyouDialog from '../dialogs/confirm-re-kyou-dialog.vue'
import type { Kyou } from '@/classes/datas/kyou'
import { computed, type Ref, ref } from 'vue'
import { GkillMessage } from '@/classes/api/gkill-message'

const add_tag_dialog = ref<InstanceType<typeof AddTagDialog> | null>(null);
const add_text_dialog = ref<InstanceType<typeof AddTextDialog> | null>(null);
const confirm_rekyou_dialog = ref<InstanceType<typeof ConfirmReKyouDialog> | null>(null);

const is_show: Ref<boolean> = ref(false)
const position_x: Ref<Number> = ref(0)
const position_y: Ref<Number> = ref(0)
const context_menu_style = computed(() => `{ position: absolute; left: ${position_x.value}px; top: ${position_y.value}px; }`)

async function show(e: PointerEvent): Promise<void> {
    position_x.value = e.clientX
    position_y.value = e.clientY
    is_show.value = true
}

const props = defineProps<GitCommitLogContextMenuProps>()
const emits = defineEmits<KyouViewEmits>()
defineExpose({ show })

async function copy_id(): Promise<void> {
    navigator.clipboard.writeText(props.kyou.id)
    const message = new GkillMessage()
    message.message_code = "//TODO"
    message.message = "KmemoIDをコピーしました"
    const messages = new Array<GkillMessage>()
    messages.push(message)
    emits('received_messages', messages)
}

async function show_add_tag_dialog(): Promise<void> {
    add_tag_dialog.value?.show()
}

async function show_add_text_dialog(): Promise<void> {
    add_text_dialog.value?.show()
}

async function show_confirm_rekyou_dialog(): Promise<void> {
    confirm_rekyou_dialog.value?.show()
}
</script>
