<template>
    <div>
        <div :class="text_class" @contextmenu.prevent="async (e: PointerEvent) => show_context_menu(e)">
            <div class="text_content"><LinkifiedText :text="text.text" /></div>
        </div>
        <AttachedTextContextMenu :application_config="application_config" :gkill_api="gkill_api" :text="text"
            :kyou="kyou" :highlight_targets="highlight_targets"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
            v-on="crudRelayHandlers"
            ref="context_menu" />
    </div>
</template>
<script setup lang="ts">
import type { AttachedTextProps } from './attached-text-props'
import type { KyouViewEmits } from './kyou-view-emits'
import AttachedTextContextMenu from './attached-text-context-menu.vue'
import LinkifiedText from './linkified-text.vue'
import { useAttachedText } from '@/classes/use-attached-text'

const props = defineProps<AttachedTextProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    context_menu,
    text_class,
    show_context_menu,
    // Event relay objects
    crudRelayHandlers,
} = useAttachedText({ props, emits })
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