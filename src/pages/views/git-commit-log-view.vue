<template>
    <GitCommitLogContextMenu />
</template>
<script setup lang="ts">
import { ref } from 'vue';
import GitCommitLogContextMenu from './git-commit-log-context-menu.vue';
import type { GitCommitLog } from '@/classes/datas/git-commit-log';
import type { GitCommitLogViewProps } from './git-commit-log-view-props';
import type { KyouViewEmits } from './kyou-view-emits';

const props = defineProps<GitCommitLogViewProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_git_commit_log: GitCommitLog = ref(await props.git_commit_log.clone());
</script>
