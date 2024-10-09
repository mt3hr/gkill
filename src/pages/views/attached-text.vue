<template>
    <div :class="text_class" @contextmenu.prevent="async (e) => show_context_menu(e as PointerEvent)">
        <div class="text_content">{{ text.text }}</div>
        {{ text.text }}
    </div>
    <AttachedTextContextMenu :application_config="application_config" :gkill_api="gkill_api" :text="text" :kyou="kyou"
        :last_added_tag="last_added_tag" :highlight_targets="[text.generate_info_identifer()]"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
        ref="context_menu" />
</template>
<script setup lang="ts">
import type { AttachedTextProps } from './attached-text-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { computed, type Ref, ref } from 'vue'
import AttachedTextContextMenu from './attached-text-context-menu.vue'

const context_menu = ref<InstanceType<typeof AttachedTextContextMenu> | null>(null);

const props = defineProps<AttachedTextProps>()
const emits = defineEmits<KyouViewEmits>()

const text_class = computed(() => {
    let highlighted = false;
    for (let i = 0; i < props.highlight_targets.length; i++) {
        if (props.highlight_targets[i].id === props.text.id
            && props.highlight_targets[i].create_time === props.text.create_time
            && props.highlight_targets[i].update_time === props.text.update_time) {
            highlighted = true
            break
        }
    }
    if (highlighted) {
        return "highlighted_text"
    }
    return "text"
})


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

.highlighted_text {
    background-color: lightgreen;
    border: dashed 1px;
    margin: 8px;
    padding: 8px;
}

.text_content {
    white-space: pre-line;
}
</style>