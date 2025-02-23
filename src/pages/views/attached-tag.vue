<template>
    <span class="tag_wrap">
        <span :class="tag_class" @contextmenu.prevent="async (e) => show_context_menu(e as PointerEvent)">
            {{ tag.tag }}
        </span>
        <AttachedTagContextMenu :application_config="application_config" :gkill_api="gkill_api" :tag="tag" :kyou="kyou"
            :last_added_tag="last_added_tag" :highlight_targets="highlight_targets"
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
    </span>
</template>
<script setup lang="ts">
import type { AttachedTagProps } from './attached-tag-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { computed, ref } from 'vue'
import AttachedTagContextMenu from './attached-tag-context-menu.vue'

const context_menu = ref<InstanceType<typeof AttachedTagContextMenu> | null>(null);

const props = defineProps<AttachedTagProps>()
const emits = defineEmits<KyouViewEmits>()

const tag_class = computed(() => {
    let highlighted = false;
    for (let i = 0; i < props.highlight_targets.length; i++) {
        if (props.highlight_targets[i].id === props.tag.id
            && props.highlight_targets[i].create_time.getTime() === props.tag.create_time.getTime()
            && props.highlight_targets[i].update_time.getTime() === props.tag.update_time.getTime()) {
            highlighted = true
            break
        }
    }
    if (highlighted) {
        return "highlighted_tag"
    }
    return "tag"
})

async function show_context_menu(e: PointerEvent): Promise<void> {
    if (props.enable_context_menu) {
        context_menu.value?.show(e)
    }
}
</script>
<style lang="css" scoped>
.tag {
    border: solid var(--v-background-base) 2px;
    border-left: 0px;
    color: blue;
    cursor: pointer;
    padding: 0 6px 0 2px;
    font-size: small;
    border-radius: 0 1em 1em 0;
    background: var(--v-background-base);
    display: inline-flex;
}

.tag::before {
    content: "・";
    color: var(--v-background-base);
}

.highlighted_tag {
    border: solid var(--v-background-base) 2px;
    border-left: 0px;
    color: blue;
    cursor: pointer;
    padding: 0 6px 0 2px;
    font-size: small;
    border-radius: 0 1em 1em 0;
    background: var(--v-highlight-base);
    display: inline-flex;
}

.highlighted_tag::before {
    content: "・";
    color: white;
}
</style>