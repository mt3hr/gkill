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
            @deleted_kyou="(...deleted_kyou: any[]) => emits('deleted_kyou', deleted_kyou[0] as Kyou)"
            @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
            @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
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
            @requested_reload_list="emits('requested_reload_list')"
            @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)"
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
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';
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