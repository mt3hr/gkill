<template>
    <AttachedTagContextMenu />
    <ConfirmDeleteTagDialog />
    <EditTagDialog />
</template>
<script setup lang="ts">
import type { AttachedTagProps } from './attached-tag-props';
import type { KyouViewEmits } from './kyou-view-emits';
import type { Tag } from '@/classes/datas/tag';
import { type Ref, ref } from 'vue';
import AttachedTagContextMenu from './attached-tag-context-menu.vue';
import ConfirmDeleteTagDialog from '../dialogs/confirm-delete-tag-dialog.vue';
import EditTagDialog from '../dialogs/edit-tag-dialog.vue';

const props = defineProps<AttachedTagProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_tag: Ref<Tag> = ref(await props.tag.clone());
</script>