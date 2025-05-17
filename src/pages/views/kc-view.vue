<template>
    <v-card @contextmenu.prevent="show_context_menu" :width="width" :height="height">
        <div v-if="kyou.typed_kc">{{ kyou.typed_kc.title }}</div>
        <div v-if="kyou.typed_kc">{{ kyou.typed_kc.num_value }}</div>
        <KCContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
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
            @requested_reload_list="() => emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
            ref="context_menu" />
    </v-card>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import { ref } from 'vue'
import KCContextMenu from './kc-context-menu.vue'
import type { KCViewProps } from './kc-view-props'
import type { KyouViewEmits } from './kyou-view-emits'

const props = defineProps<KCViewProps>()
const emits = defineEmits<KyouViewEmits>()
defineExpose({ show_context_menu })

const context_menu = ref<InstanceType<typeof KCContextMenu> | null>(null);

async function show_context_menu(e: PointerEvent): Promise<void> {
    if (props.enable_context_menu) {
        context_menu.value?.show(e)
    }
}

</script>