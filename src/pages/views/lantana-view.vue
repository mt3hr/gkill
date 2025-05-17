<template>
    <v-card @contextmenu.prevent="show_context_menu" :width="width" :height="height" style="overflow: hidden">
        <LantanaFlowersView v-if="kyou.typed_lantana" :application_config="application_config" :gkill_api="gkill_api"
            :editable="false" :mood="kyou.typed_lantana.mood" />
        <LantanaContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" ref="context_menu"
            @deleted_kyou="(deleted_kyou) => emits('deleted_kyou', deleted_kyou)"
            @deleted_tag="(deleted_tag) => emits('deleted_tag', deleted_tag)"
            @deleted_text="(deleted_text) => emits('deleted_text', deleted_text)"
            @deleted_notification="(deleted_notification) => emits('deleted_notification', deleted_notification)"
            @registered_kyou="(registered_kyou) => emits('registered_kyou', registered_kyou)"
            @registered_tag="(registered_tag) => emits('registered_tag', registered_tag)"
            @registered_text="(registered_text) => emits('registered_text', registered_text)"
            @registered_notification="(registered_notification) => emits('registered_notification', registered_notification)"
            @updated_kyou="(updated_kyou) => emits('updated_kyou', updated_kyou)"
            @updated_tag="(updated_tag) => emits('updated_tag', updated_tag)"
            @updated_text="(updated_text) => emits('updated_text', updated_text)"
            @updated_notification="(updated_notification) => emits('updated_notification', updated_notification)"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
    </v-card>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { LantanaViewProps } from './lantana-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { ref } from 'vue'
import LantanaContextMenu from './lantana-context-menu.vue'
import LantanaFlowersView from './lantana-flowers-view.vue'

const context_menu = ref<InstanceType<typeof LantanaContextMenu> | null>(null);

const props = defineProps<LantanaViewProps>()
const emits = defineEmits<KyouViewEmits>()
defineExpose({ show_context_menu })

function show_context_menu(e: PointerEvent): void {
    if (props.enable_context_menu) {
        context_menu.value?.show(e)
    }
}
</script>
