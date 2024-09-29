<template>
    <span class="tag" @contextmenu.prevent="async (e) => show_context_menu(e as PointerEvent)">
        {{ tag.tag }}
    </span>
    <AttachedTagContextMenu :application_config="application_config" :gkill_api="gkill_api" :tag="tag" :kyou="kyou"
        :last_added_tag="last_added_tag" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)"
        ref="context_menu" />
</template>
<script setup lang="ts">
import type { AttachedTagProps } from './attached-tag-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { type Ref, ref } from 'vue'
import AttachedTagContextMenu from './attached-tag-context-menu.vue'

const context_menu = ref<InstanceType<typeof AttachedTagContextMenu> | null>(null);

const props = defineProps<AttachedTagProps>()
const emits = defineEmits<KyouViewEmits>()

async function show_context_menu(e: PointerEvent): Promise<void> {
    context_menu.value?.show(e)
}
</script>
<style scoped>
.tag {
    border: solid white 2px;
    border-left: 0px;
    color: blue;
    cursor: pointer;
    padding: 0 6px 0 2px;
    font-size: small;
    border-radius: 0 1em 1em 0;
    background: lightgray;
    display: inline-flex;
}

.tag::before {
    content: "・";
    color: white;
}
</style>