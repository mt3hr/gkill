<template>
    <v-menu v-model="is_show" :style="context_menu_style">
        <v-list class="gkill_context_menu_list">
            <v-list-item @click="show_edit_text_dialog()">
                <v-list-item-title>{{ i18n.global.t("TEXT_CONTEXTMENU_EDIT_TEXT") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_text_histories_dialog()">
                <v-list-item-title>{{ i18n.global.t("TEXT_CONTEXTMENU_HISTORIES") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="copy_id()">
                <v-list-item-title>{{ i18n.global.t("TEXT_CONTEXTMENU_COPY_ID") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_confirm_delete_text_dialog()">
                <v-list-item-title>{{ i18n.global.t("TEXT_CONTEXTMENU_DELETE") }}</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import type { AttachedTextContextMenuProps } from './attached-text-context-menu-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { computed, type Ref, ref } from 'vue'
import { GkillMessage } from '@/classes/api/gkill-message'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'
import type { GkillError } from '@/classes/api/gkill-error'
import type { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';
const props = defineProps<AttachedTextContextMenuProps>()
const emits = defineEmits<KyouViewEmits>()
defineExpose({ show, hide })

const is_show: Ref<boolean> = ref(false)
const position_x: Ref<Number> = ref(0)
const position_y: Ref<Number> = ref(0)
const context_menu_style = computed(() => `{ position: absolute; left: ${Math.min(document.defaultView!.innerWidth - 130, position_x.value.valueOf())}px; top: ${Math.min(Math.max(50, document.defaultView!.innerHeight - ( + 8 + (48 * 4))), position_y.value.valueOf())}px; }`)

async function show(e: PointerEvent): Promise<void> {
    position_x.value = e.clientX
    position_y.value = e.clientY
    is_show.value = true
}

async function hide(): Promise<void> {
    is_show.value = false
}

async function show_edit_text_dialog(): Promise<void> {
    emits('requested_open_rykv_dialog', 'edit_text', props.kyou, props.text)
}

async function show_text_histories_dialog(): Promise<void> {
    emits('requested_open_rykv_dialog', 'text_histories', props.kyou, props.text)
}

async function copy_id(): Promise<void> {
    navigator.clipboard.writeText(props.text.id)
    const message = new GkillMessage()
    message.message_code = GkillMessageCodes.copied_text_id
    message.message = i18n.global.t("COPIED_ID_MESSAGE")
    const messages = new Array<GkillMessage>()
    messages.push(message)
    emits('received_messages', messages)
}

async function show_confirm_delete_text_dialog(): Promise<void> {
    emits('requested_open_rykv_dialog', 'confirm_delete_text', props.kyou, props.text)
}
</script>

