<template>
    <NewBoardNameDialog />
</template>
<script lang="ts" setup>
import type { AddMiViewProps } from './add-mi-view-props';
import type { KyouViewEmits } from './kyou-view-emits';
import NewBoardNameDialog from '../dialogs/new-board-name-dialog.vue';

const props = defineProps<AddMiViewProps>();
const emits = defineEmits<KyouViewEmits>();
</script>