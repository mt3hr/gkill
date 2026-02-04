<template>
    <span v-if="timeis_kyou.typed_timeis" :class="plaing_class"
        @contextmenu.prevent="async (...e: any[]) => show_context_menu(e[0] as PointerEvent)"
        @dblclick.stop.prevent="show_kyou_dialog">
        {{ timeis_kyou.typed_timeis.title }}
    </span>
    <AttachedTimeIsPlaingContextMenu :application_config="application_config" :gkill_api="gkill_api" :target_kyou="kyou"
        v-if="timeis_kyou.typed_timeis" :timeis_kyou="timeis_kyou" :last_added_tag="last_added_tag"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" :highlight_targets="highlight_targets"
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
    <KyouDialog v-if="timeis_kyou" :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="[]" :kyou="timeis_kyou" :last_added_tag="''" :enable_context_menu="enable_context_menu"
        :enable_dialog="enable_dialog"
        @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0])" :is_readonly_mi_check="false"
        :show_timeis_plaing_end_button="false"
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
        @requested_reload_kyou="(...cloned_kyou: any[]) => emits('requested_reload_kyou', cloned_kyou[0] as Kyou)"
        @requested_reload_list="emits('requested_reload_list')"
        @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)"
        ref="kyou_dialog" />
</template>
<script setup lang="ts">
import type { AttachedTimeIsPlaingProps } from './attached-time-is-plaing-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { computed, ref } from 'vue'
import AttachedTimeIsPlaingContextMenu from './attached-timeis-plaing-context-menu.vue'
import KyouDialog from '../dialogs/kyou-dialog.vue';
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';

const context_menu = ref<InstanceType<typeof AttachedTimeIsPlaingContextMenu> | null>(null);
const kyou_dialog = ref<InstanceType<typeof KyouDialog> | null>(null);

const props = defineProps<AttachedTimeIsPlaingProps>()
const emits = defineEmits<KyouViewEmits>()

const plaing_class = computed(() => {
    if (!props.kyou) {
        return ""
    }
    let highlighted = false;
    for (let i = 0; i < props.highlight_targets.length; i++) {
        if (props.highlight_targets[i].id === props.kyou.id
            && props.highlight_targets[i].create_time.getTime() === props.timeis_kyou.create_time.getTime()
            && props.highlight_targets[i].update_time.getTime() === props.timeis_kyou.update_time.getTime()) {
            highlighted = true
            break
        }
    }
    if (highlighted) {
        return "highlighted_plaing"
    }
    return "plaing"
})


async function show_context_menu(e: PointerEvent): Promise<void> {
    if (props.enable_context_menu) {
        context_menu.value?.show(e)
    }
}

function show_kyou_dialog(): void {
    if (props.enable_dialog) {
        kyou_dialog.value?.show()
    }
}

</script>
<style scoped>
.plaing {
    /* タグとの合わせ */
    position: relative;
    display: inline-flex;
    border: solid rgb(var(--v-theme-background)) 2px;
    border-left: 0px;
    color: blue;
    cursor: pointer;
    font-size: small;
    background: lightgray;
}

.plaing::after {
    content: "";
    background: rgb(var(--v-theme-background));
    border-top: 9.5px solid rgb(var(--v-theme-background));
    border-left: 10px solid lightgray;
    border-bottom: 9.5px solid rgb(var(--v-theme-background));
}

.plaing::before {
    content: "";
    background: rgb(var(--v-theme-background));
    border-top: 9.5px solid rgb(var(--v-theme-background));
    border-right: 10px solid lightgray;
    border-bottom: 9.5px solid rgb(var(--v-theme-background));
}

.highlighted_plaing {
    /* タグとの合わせ */
    position: relative;
    display: inline-flex;
    border: solid rgb(var(--v-theme-background)) 2px;
    border-left: 0px;
    color: blue;
    cursor: pointer;
    font-size: small;
    background: lightgray;
}

.highlighted_plaing::after {
    content: "";
    background: rgb(var(--v-theme-background));
    border-top: 9.5px solid rgb(var(--v-theme-background));
    border-left: 10px solid lightgray;
    border-bottom: 9.5px solid rgb(var(--v-theme-background));
}

.highlighted_plaing::before {
    content: "";
    background: rgb(var(--v-theme-background));
    border-top: 9.5px solid rgb(var(--v-theme-background));
    border-right: 10px solid lightgray;
    border-bottom: 9.5px solid rgb(var(--v-theme-background));
}
</style>