<template>
    <v-card @contextmenu.prevent="show_context_menu" :width="width" :height="height" style="overflow: unset">
        <LantanaFlowersView v-if="kyou.typed_lantana" :application_config="application_config" :gkill_api="gkill_api"
            :editable="false" :mood="kyou.typed_lantana.mood" />
        <LantanaContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" ref="context_menu"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
    </v-card>
</template>
<script setup lang="ts">
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
