<template>
    <!-- //TODO 実装 -->
    <NewBoardNameDialog :application_config="application_config" :gkill_api="gkill_api"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" ref="foldable_struct"
        @setted_new_board_name="(new_board_name) => board_name = new_board_name" />
</template>
<script lang="ts" setup>
import type { AddMiViewProps } from './add-mi-view-props';
import type { KyouViewEmits } from './kyou-view-emits';
import NewBoardNameDialog from '../dialogs/new-board-name-dialog.vue';
import { ref, type Ref } from 'vue';

const props = defineProps<AddMiViewProps>();
const emits = defineEmits<KyouViewEmits>();

const board_name: Ref<string> = ref("");
</script>
