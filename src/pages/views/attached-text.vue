<template>
    <div class="text" @contextmenu.prevent="async (e) => show_context_menu(e as PointerEvent)">
        <div class="text_content">{{ text.text }}</div>
        {{ text.text }}
    </div>
    <AttachedTextContextMenu :application_config="application_config" :gkill_api="gkill_api" :text="text" :kyou="kyou"
        :last_added_tag="last_added_tag" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
        ref="context_menu" />
</template>
<script setup lang="ts">
import type { AttachedTextProps } from './attached-text-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { type Ref, ref } from 'vue'
import AttachedTextContextMenu from './attached-text-context-menu.vue'

const context_menu = ref<InstanceType<typeof AttachedTextContextMenu> | null>(null);

const props = defineProps<AttachedTextProps>()
const emits = defineEmits<KyouViewEmits>()

async function show_context_menu(e: PointerEvent): Promise<void> {
    context_menu.value?.show(e)
}
</script>
<style scoped>
.text {
    background-color: #eee;
    border: dashed 1px;
    margin: 8px;
    padding: 8px;
}

.text_content {
    white-space: pre-line;
}
</style>