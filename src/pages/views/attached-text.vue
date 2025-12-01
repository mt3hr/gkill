<template>
    <div>
        <div :class="text_class" @contextmenu.prevent="async (...e: any[]) => show_context_menu(e[0] as PointerEvent)">
            <div class="text_content">{{ text.text }}</div>
        </div>
        <AttachedTextContextMenu :application_config="application_config" :gkill_api="gkill_api" :text="text"
            :kyou="kyou" :last_added_tag="last_added_tag" :highlight_targets="highlight_targets"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
            @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0])"
            @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
            @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0])"
            @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
            @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
            @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
            @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
            @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
            @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
            @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
            @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
            @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
            @requested_reload_list="() => emits('requested_reload_list')"
            @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)"
            ref="context_menu" />
    </div>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { AttachedTextProps } from './attached-text-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { computed, ref } from 'vue'
import AttachedTextContextMenu from './attached-text-context-menu.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';

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