<template>
    <div>
        <div :class="text_class" @contextmenu.prevent="async (e) => show_context_menu(e as PointerEvent)">
            <div class="text_content">{{ text.text }}</div>
        </div>
        <AttachedTextContextMenu :application_config="application_config" :gkill_api="gkill_api" :text="text"
            :kyou="kyou" :last_added_tag="last_added_tag" :highlight_targets="highlight_targets"
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
    </div>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { AttachedTextProps } from './attached-text-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { computed, ref } from 'vue'
import AttachedTextContextMenu from './attached-text-context-menu.vue'

const context_menu = ref<InstanceType<typeof AttachedTextContextMenu> | null>(null);

const props = defineProps<AttachedTextProps>()
const emits = defineEmits<KyouViewEmits>()

const text_class = computed(() => {
    let highlighted = false;
    for (let i = 0; i < props.highlight_targets.length; i++) {
        if (props.highlight_targets[i].id === props.text.id
            && props.highlight_targets[i].create_time.getTime() === props.text.create_time.getTime()
            && props.highlight_targets[i].update_time.getTime() === props.text.update_time.getTime()) {
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
    if (props.enable_context_menu) {
        context_menu.value?.show(e)
    }
}
</script>
<style lang="css" scoped>
.text {
    background-color: var(--v-attached-text-background-base);
    border: dashed 1px;
    margin: 8px;
    padding: 8px;
}

.highlighted_text {
    background-color: rgb(var(--v-theme-highlight));
    border: dashed 1px;
    margin: 8px;
    padding: 8px;
}

.text_content {
    white-space: pre-line;
}
</style>