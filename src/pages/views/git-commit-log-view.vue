<template>
    <v-card v-if="kyou.typed_git_commit_log" @contextmenu.prevent="show_context_menu" :width="width" :height="height">
        <div>
            <span class="git_commit_addition"> + {{ kyou.typed_git_commit_log.addition }} </span>
            <span class="git_commit_deletion"> - {{ kyou.typed_git_commit_log.deletion }} </span>
        </div>
        <div class="git_commit_log_message">{{ kyou.typed_git_commit_log.commit_message }}</div>
        <GitCommitLogContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
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
            @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
            @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou: Kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="emits('requested_reload_list')"
            @requested_update_check_kyous="(kyou: Array<Kyou>, is_checked: boolean) => emits('requested_update_check_kyous', kyou, is_checked)"
            ref="context_menu" />
    </v-card>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import { ref } from 'vue'
import GitCommitLogContextMenu from './git-commit-log-context-menu.vue'
import type { GitCommitLogViewProps } from './git-commit-log-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import type { Kyou } from '@/classes/datas/kyou'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
const context_menu = ref<InstanceType<typeof GitCommitLogContextMenu> | null>(null);

const props = defineProps<GitCommitLogViewProps>()
const emits = defineEmits<KyouViewEmits>()
defineExpose({ show_context_menu })

async function show_context_menu(e: PointerEvent): Promise<void> {
    if (props.enable_context_menu) {
        context_menu.value?.show(e)
    }
}
</script>
<style lang="css">
.git_commit_log_message {
    white-space: pre-line;
}

.git_commit_addition {
    color: limegreen;
}

.git_commit_deletion {
    color: crimson;
}
</style>