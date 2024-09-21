<template>
    <GitCommitLogContextMenu :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="cloned_kyou" :last_added_tag="last_added_tag"
        @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
        @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou: Kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')"
        @requested_update_check_kyous="(kyou: Array<Kyou>, is_checked: boolean) => emits('requested_update_check_kyous', kyou, is_checked)" />
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

const props = defineProps<GitCommitLogViewProps>()
const emits = defineEmits<KyouViewEmits>()
const cloned_kyou: Ref<Kyou> = ref(await props.kyou.clone())
const cloned_git_commit_log: Ref<GitCommitLog> = ref(await props.git_commit_log.clone())
</script>
