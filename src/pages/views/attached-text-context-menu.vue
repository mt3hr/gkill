<template>
    <EditTextDialog />
    <ConfirmDeleteTextDialog />
    <TextHistoriesDialog />
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { AttachedTextContextMenuProps } from './attached-text-context-menu-props';
import type { KyouViewEmits } from './kyou-view-emits';
import { Text } from '@/classes/datas/text';
import EditTextDialog from '../dialogs/edit-text-dialog.vue';
import ConfirmDeleteTextDialog from '../dialogs/confirm-delete-text-dialog.vue';
import TextHistoriesDialog from '../dialogs/text-histories-dialog.vue';

const props = defineProps<AttachedTextContextMenuProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_text: Ref<Text> = ref(await props.text.clone());
</script>
