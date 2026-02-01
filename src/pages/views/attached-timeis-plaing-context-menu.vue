<template>
    <v-menu v-model="is_show" :style="context_menu_style">
        <v-list class="gkill_context_menu_list">
            <v-list-item @click="show_edit_timeis_dialog()">
                <v-list-item-title>{{ i18n.global.t("TIMEIS_CONTEXTMENU_EDIT_TIMEIS") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_timeis_histories_dialog()">
                <v-list-item-title>{{ i18n.global.t("TIMEIS_CONTEXTMENU_HISTORIES") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="copy_id()">
                <v-list-item-title>{{ i18n.global.t("TIMEIS_CONTEXTMENU_COPY_ID") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_confirm_delete_timeis_dialog()">
                <v-list-item-title>{{ i18n.global.t("TIMEIS_CONTEXTMENU_DELETE") }}</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
    <EditTimeIsDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="cloned_timeis_kyou" :last_added_tag="last_added_tag"
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
        ref="edit_timeis_dialog" />
    <ConfirmDeleteKyouDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="cloned_timeis_kyou" :last_added_tag="last_added_tag"
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
        ref="confirm_delete_kyou_dialog" />
    <KyouHistoriesDialog :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="cloned_timeis_kyou" :last_added_tag="last_added_tag"
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
        ref="kyou_histories_dialog" />
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { AttachedTimeisPlaingContextMenuProps } from './attached-timeis-plaing-context-menu-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { computed, type Ref, ref, watch } from 'vue'
import EditTimeIsDialog from '../dialogs/edit-time-is-dialog.vue'
import ConfirmDeleteKyouDialog from '../dialogs/confirm-delete-idf-kyou-dialog.vue'
import KyouHistoriesDialog from '../dialogs/kyou-histories-dialog.vue'
import { GkillMessage } from '@/classes/api/gkill-message'
import type { Kyou } from '@/classes/datas/kyou'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'
import type { GkillError } from '@/classes/api/gkill-error'
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';

const edit_timeis_dialog = ref<InstanceType<typeof EditTimeIsDialog> | null>(null);
const confirm_delete_kyou_dialog = ref<InstanceType<typeof ConfirmDeleteKyouDialog> | null>(null);
const kyou_histories_dialog = ref<InstanceType<typeof KyouHistoriesDialog> | null>(null);

const props = defineProps<AttachedTimeisPlaingContextMenuProps>()
const emits = defineEmits<KyouViewEmits>()
defineExpose({ show, hide })

const cloned_timeis_kyou: Ref<Kyou> = ref(props.timeis_kyou.clone())

watch(() => props.timeis_kyou, () => {
    reload_cloned_timeis_kyou()
})

reload_cloned_timeis_kyou()

const is_show: Ref<boolean> = ref(false)
const position_x: Ref<Number> = ref(0)
const position_y: Ref<Number> = ref(0)
const context_menu_style = computed(() => `{ position: absolute; left: ${Math.min(document.defaultView!.innerWidth - 130, position_x.value.valueOf())}px; top: ${Math.min(Math.max(50, document.defaultView!.innerHeight - ( + 8 + (48 * 4))), position_y.value.valueOf())}px; }`)

function reload_cloned_timeis_kyou(): void {
    cloned_timeis_kyou.value = props.timeis_kyou.clone()
    cloned_timeis_kyou.value.load_typed_datas()
    cloned_timeis_kyou.value.load_attached_histories()
}

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
    navigator.clipboard.writeText(props.timeis_kyou.id)
    const message = new GkillMessage()
    message.message_code = GkillMessageCodes.copied_timeis_id
    message.message = i18n.global.t("COPIED_ID_MESSAGE")
    const messages = new Array<GkillMessage>()
    messages.push(message)
    emits('received_messages', messages)
}

async function show_confirm_delete_timeis_dialog(): Promise<void> {
    confirm_delete_kyou_dialog.value?.show()
}
</script>