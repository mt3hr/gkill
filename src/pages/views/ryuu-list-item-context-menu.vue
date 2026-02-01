<template>
    <v-menu v-model="is_show" :style="context_menu_style">
        <v-list class="gkill_context_menu_list">
            <v-list-item @click="show_edit_related_kyou_query_dialog()">
                <v-list-item-title>{{ i18n.global.t("EDIT_RELATED_KYOU_QUERY") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_confirm_delete_ryuu_list_item_dialog()">
                <v-list-item-title>{{ i18n.global.t("DELETE_RELATED_KYOU_QUERY") }}</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
    <EditRyuuItemDialog :application_config="application_config" :gkill_api="gkill_api" v-model="model_value"
        ref="edit_related_kyou_query_dialog" />
    <ConfirmDeleteRyuuListItemDialog :application_config="application_config" :gkill_api="gkill_api"
        @requested_delete_related_kyou_query="(...id: any[]) => emits('requested_delete_related_kyou_query', id[0] as string)"
        @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
        @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
        ref="confirm_delete_ryuu_list_item_dialog" />
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { computed, type Ref, ref } from 'vue'
import ConfirmDeleteRyuuListItemDialog from '../dialogs/confirm-delete-ryuu-list-item-dialog.vue'
import EditRyuuItemDialog from '../dialogs/edit-ryuu-item-dialog.vue'
import type { RyuuListItemContextMenuProps } from './ryuu-list-item-context-menu-props'
import type { RyuuListItemContextMenuEmits } from './ryuu-list-item-context-menu-emits'
import RelatedKyouQuery from '@/classes/dnote/related-kyou-query'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

const edit_related_kyou_query_dialog = ref<InstanceType<typeof EditRyuuItemDialog> | null>(null);
const confirm_delete_ryuu_list_item_dialog = ref<InstanceType<typeof ConfirmDeleteRyuuListItemDialog> | null>(null);

const model_value = defineModel<RelatedKyouQuery>()
const props = defineProps<RyuuListItemContextMenuProps>()
const emits = defineEmits<RyuuListItemContextMenuEmits>()
defineExpose({ show, hide })

const is_show: Ref<boolean> = ref(false)
const position_x: Ref<Number> = ref(0)
const position_y: Ref<Number> = ref(0)
const context_menu_style = computed(() => `{ position: absolute; left: ${Math.min(document.defaultView!.innerWidth - 130, position_x.value.valueOf())}px; top: ${Math.min(Math.max(50, document.defaultView!.innerHeight - ( + 8 + (48 * 2))), position_y.value.valueOf())}px; }`)

async function show(e: PointerEvent): Promise<void> {
    position_x.value = e.clientX
    position_y.value = e.clientY
    is_show.value = true
}

async function hide(): Promise<void> {
    is_show.value = false
}

async function show_edit_related_kyou_query_dialog(): Promise<void> {
    edit_related_kyou_query_dialog.value?.show()
}

async function show_confirm_delete_ryuu_list_item_dialog(): Promise<void> {
    confirm_delete_ryuu_list_item_dialog.value?.show(model_value.value!)
}
</script>

