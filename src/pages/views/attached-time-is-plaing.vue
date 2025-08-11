<template>
    <span v-if="timeis_kyou.typed_timeis" :class="plaing_class"
        @contextmenu.prevent="async (e) => show_context_menu(e as PointerEvent)"
        @dblclick.stop.prevent="show_kyou_dialog">
        {{ timeis_kyou.typed_timeis.title }}
    </span>
    <AttachedTimeIsPlaingContextMenu :application_config="application_config" :gkill_api="gkill_api" :target_kyou="kyou"
        v-if="timeis_kyou.typed_timeis" :timeis_kyou="timeis_kyou" :last_added_tag="last_added_tag"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" :highlight_targets="highlight_targets"
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
    <KyouDialog v-if="timeis_kyou" :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="[]" :kyou="timeis_kyou" :last_added_tag="''" :enable_context_menu="enable_context_menu"
        :enable_dialog="enable_dialog" @deleted_kyou="(deleted_kyou) => emits('deleted_kyou', deleted_kyou)"
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
        @requested_reload_kyou="(cloned_kyou) => emits('requested_reload_kyou', cloned_kyou)"
        @requested_reload_list="emits('requested_reload_list')"
        @requested_update_check_kyous="(cloned_kyous, is_checked) => emits('requested_update_check_kyous', cloned_kyous, is_checked)"
        ref="kyou_dialog" />
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { AttachedTimeIsPlaingProps } from './attached-time-is-plaing-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { computed, ref } from 'vue'
import AttachedTimeIsPlaingContextMenu from './attached-timeis-plaing-context-menu.vue'
import KyouDialog from '../dialogs/kyou-dialog.vue';

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
    top: 6px;


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
    top: 6px;

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