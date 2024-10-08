<template>
    <v-menu v-model="is_show" :style="context_menu_style">
        <v-list>
            <v-list-item @click="show_edit_tag_dialog()">
                <v-list-item-title>タグ編集</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_tag_histories_dialog()">
                <v-list-item-title>タグ履歴</v-list-item-title>
            </v-list-item>
            <v-list-item @click="copy_id()">
                <v-list-item-title>タグIDコピー</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_confirm_delete_tag_dialog()">
                <v-list-item-title>タグ削除</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
    <EditTagDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        :tag="tag" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
        ref="edit_tag_dialog" />
    <ConfirmDeleteTagDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        :tag="tag" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
        ref="confirm_delete_tag_dialog" />
    <TagHistoriesDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        :tag="tag" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
        ref="tag_histories_dialog" />
</template>
<script lang="ts" setup>
import type { AttachedTagContextMenuProps } from './attached-tag-context-menu-props'
import type { KyouViewEmits } from './kyou-view-emits'
import type { Tag } from '@/classes/datas/tag'
import { computed, type Ref, ref } from 'vue'
import EditTagDialog from '../dialogs/edit-tag-dialog.vue'
import ConfirmDeleteTagDialog from '../dialogs/confirm-delete-tag-dialog.vue'
import TagHistoriesDialog from '../dialogs/tag-histories-dialog.vue'
import { InfoIdentifier } from '@/classes/datas/info-identifier'
import { GkillMessage } from '@/classes/api/gkill-message'

const edit_tag_dialog = ref<InstanceType<typeof EditTagDialog> | null>(null);
const confirm_delete_tag_dialog = ref<InstanceType<typeof ConfirmDeleteTagDialog> | null>(null);
const tag_histories_dialog = ref<InstanceType<typeof TagHistoriesDialog> | null>(null);

const props = defineProps<AttachedTagContextMenuProps>()
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

async function show_edit_tag_dialog(): Promise<void> {
    edit_tag_dialog.value?.show()
}

async function show_tag_histories_dialog(): Promise<void> {
    tag_histories_dialog.value?.show()
}

async function copy_id(): Promise<void> {
    navigator.clipboard.writeText(props.tag.id)
    const message = new GkillMessage()
    message.message_code = "//TODO"
    message.message = "タグIDをコピーしました"
    const messages = new Array<GkillMessage>()
    messages.push(message)
    emits('received_messages', messages)
}

async function show_confirm_delete_tag_dialog(): Promise<void> {
    confirm_delete_tag_dialog.value?.show()
}
</script>
<style lang="css" scoped></style>
