<template>

</template>
<script setup lang="ts">

// ˅
'use strict';

import { GitCommitLogContextMenu } from './git-commit-log-context-menu.vue';
import { GitCommitLogViewProps } from './git-commit-log-view-props';
import { KyouViewEmits } from './kyou-view-emits';

// ˄

export class GitCommitLogView {
    // ˅
    
    // ˄

    private cloned_git_commit_log: GitCommitLog;

    private props: GitCommitLogViewProps;

    private contextmenu: GitCommitLogContextMenu;

    private emits: KyouViewEmits;

    // ˅
    
    // ˄
}

// ˅

// ˄

</script>
<style scoped></style>