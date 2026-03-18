<template>
    <v-card elevation="0" v-if="kyou.typed_git_commit_log" @contextmenu.prevent="show_context_menu" :width="width" :height="height">
        <div>
            <span class="git_commit_addition"> + {{ kyou.typed_git_commit_log.addition }} </span>
            <span class="git_commit_deletion"> - {{ kyou.typed_git_commit_log.deletion }} </span>
        </div>
        <div class="git_commit_log_message">{{ kyou.typed_git_commit_log.commit_message }}</div>
        <GitCommitLogContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
            v-on="crudRelayHandlers"
            ref="context_menu" />
    </v-card>
</template>
<script setup lang="ts">
import GitCommitLogContextMenu from './git-commit-log-context-menu.vue'
import type { GitCommitLogViewProps } from './git-commit-log-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { useGitCommitLogView } from '@/classes/use-git-commit-log-view'

const props = defineProps<GitCommitLogViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    context_menu,
    show_context_menu,
    crudRelayHandlers,
} = useGitCommitLogView({ props, emits })

defineExpose({ show_context_menu })
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
