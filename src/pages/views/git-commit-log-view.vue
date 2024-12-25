<template>
    <v-card v-if="kyou.typed_git_commit_log" @contextmenu.prevent="show_context_menu" :width="width" :height="height">
        <div>
            <span class="git_commit_addition"> + {{ kyou.typed_git_commit_log.addition }} </span>
            <span class="git_commit_deletion"> - {{ kyou.typed_git_commit_log.deletion }} </span>
        </div>
        <div class="git_commit_log_message">{{ kyou.typed_git_commit_log.commit_message }}</div>
        <GitCommitLogContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
            @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
            @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou: Kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="emits('requested_reload_list')"
            @requested_update_check_kyous="(kyou: Array<Kyou>, is_checked: boolean) => emits('requested_update_check_kyous', kyou, is_checked)"
            ref="context_menu" />
    </v-card>
</template>
<script setup lang="ts">
import { ref, type Ref } from 'vue'
import GitCommitLogContextMenu from './git-commit-log-context-menu.vue'
import type { GitCommitLog } from '@/classes/datas/git-commit-log'
import type { GitCommitLogViewProps } from './git-commit-log-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import type { Kyou } from '@/classes/datas/kyou'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
const context_menu = ref<InstanceType<typeof GitCommitLogContextMenu> | null>(null);

const props = defineProps<GitCommitLogViewProps>()
const emits = defineEmits<KyouViewEmits>()

async function show_context_menu(e: PointerEvent): Promise<void> {
    context_menu.value?.show(e)
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