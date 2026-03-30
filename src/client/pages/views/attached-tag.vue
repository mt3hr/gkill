<template>
    <span class="tag_wrap">
        <span :class="tag_class" @contextmenu.prevent="async (e: PointerEvent) => show_context_menu(e)">
            {{ tag.tag }}
        </span>
        <AttachedTagContextMenu :application_config="application_config" :gkill_api="gkill_api" :tag="tag" :kyou="kyou"
            :highlight_targets="highlight_targets"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
            @deleted_kyou="crudRelayHandlers['deleted_kyou']"
            @deleted_tag="crudRelayHandlers['deleted_tag']"
            @deleted_text="crudRelayHandlers['deleted_text']"
            @deleted_notification="crudRelayHandlers['deleted_notification']"
            @registered_kyou="crudRelayHandlers['registered_kyou']"
            @registered_tag="crudRelayHandlers['registered_tag']"
            @registered_text="crudRelayHandlers['registered_text']"
            @registered_notification="crudRelayHandlers['registered_notification']"
            @updated_kyou="crudRelayHandlers['updated_kyou']"
            @updated_tag="crudRelayHandlers['updated_tag']"
            @updated_text="crudRelayHandlers['updated_text']"
            @updated_notification="crudRelayHandlers['updated_notification']"
            @received_errors="crudRelayHandlers['received_errors']"
            @received_messages="crudRelayHandlers['received_messages']"
            @requested_reload_kyou="crudRelayHandlers['requested_reload_kyou']"
            @requested_reload_list="crudRelayHandlers['requested_reload_list']"
            @requested_update_check_kyous="crudRelayHandlers['requested_update_check_kyous']"
            @requested_open_rykv_dialog="crudRelayHandlers['requested_open_rykv_dialog']"
            ref="context_menu" />
    </span>
</template>
<script setup lang="ts">
import type { AttachedTagProps } from './attached-tag-props'
import type { KyouViewEmits } from './kyou-view-emits'
import AttachedTagContextMenu from './attached-tag-context-menu.vue'
import { useAttachedTag } from '@/classes/use-attached-tag'

const props = defineProps<AttachedTagProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    context_menu,
    tag_class,
    show_context_menu,
    crudRelayHandlers,
} = useAttachedTag({ props, emits })
</script>
<style lang="css" scoped>
.tag {
    border: solid rgb(var(--v-theme-background)) 2px;
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
    color: rgb(var(--v-theme-background));
}

.highlighted_tag {
    border: solid rgb(var(--v-theme-background)) 2px;
    border-left: 0px;
    color: blue;
    cursor: pointer;
    padding: 0 6px 0 2px;
    font-size: small;
    border-radius: 0 1em 1em 0;
    background: rgb(var(--v-theme-highlight));
    display: inline-flex;
}

.highlighted_tag::before {
    content: "・";
    color: rgb(var(--v-theme-background));
}
</style>