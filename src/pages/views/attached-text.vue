<template>
    <AttachedTextContextMenu />
    <ConfirmDeleteTextDialog />
    <EditTextDialog />
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue';
import type { AttachedTextProps } from './attached-text-props';
import type { KyouViewEmits } from './kyou-view-emits';
import AttachedTextContextMenu from './attached-text-context-menu.vue';
import ConfirmDeleteTextDialog from '../dialogs/confirm-delete-text-dialog.vue';
import EditTextDialog from '../dialogs/edit-text-dialog.vue';

const props = defineProps<AttachedTextProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_text: Ref<Text> = ref(await props.text.clone());
</script>
