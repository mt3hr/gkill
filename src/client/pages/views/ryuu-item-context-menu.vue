<template>
    <v-menu v-model="is_show" :style="context_menu_style">
        <v-list class="gkill_context_menu_list">
            <v-list-item @click="show_edit_related_kyou_query_dialog()">
                <v-list-item-title>{{ i18n.global.t("EDIT_RELATED_KYOU_QUERY") }}</v-list-item-title>
            </v-list-item>
            <v-list-item @click="show_confirm_delete_ryuu_item_dialog()">
                <v-list-item-title>{{ i18n.global.t("DELETE_RELATED_KYOU_QUERY") }}</v-list-item-title>
            </v-list-item>
        </v-list>
    </v-menu>
    <EditRyuuItemDialog :application_config="application_config" :gkill_api="gkill_api" v-model="model_value"
        ref="edit_related_kyou_query_dialog" />
    <ConfirmDeleteRyuuItemDialog :application_config="application_config" :gkill_api="gkill_api"
        @requested_delete_related_kyou_query="(id: string) => emits('requested_delete_related_kyou_query', id)"
        @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
        @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
        ref="confirm_delete_ryuu_item_dialog" />
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { ref } from 'vue'
import ConfirmDeleteRyuuItemDialog from '../dialogs/confirm-delete-ryuu-item-dialog.vue'
import EditRyuuItemDialog from '../dialogs/edit-ryuu-item-dialog.vue'
import type { RyuuItemContextMenuProps } from './ryuu-item-context-menu-props'
import type { RyuuItemContextMenuEmits } from './ryuu-item-context-menu-emits'
import RelatedKyouQuery from '@/classes/dnote/related-kyou-query'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { useRyuuItemContextMenu } from '@/classes/use-ryuu-item-context-menu'

const edit_related_kyou_query_dialog = ref<InstanceType<typeof EditRyuuItemDialog> | null>(null);
const confirm_delete_ryuu_item_dialog = ref<InstanceType<typeof ConfirmDeleteRyuuItemDialog> | null>(null);

const model_value = defineModel<RelatedKyouQuery>()
defineProps<RyuuItemContextMenuProps>()
const emits = defineEmits<RyuuItemContextMenuEmits>()

const {
    is_show,
    context_menu_style,
    show,
    hide,
    show_edit_related_kyou_query_dialog,
    show_confirm_delete_ryuu_item_dialog,
} = useRyuuItemContextMenu({ emits, edit_related_kyou_query_dialog, confirm_delete_ryuu_item_dialog, model_value })

defineExpose({ show, hide })
</script>
