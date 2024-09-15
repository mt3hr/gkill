<template>
    <EditTagDialog />
    <ConfirmDeleteTagDialog />
    <TagHistoriesDialog />
</template>
<script lang="ts" setup>
import type { AttachedTagContextMenuProps } from './attached-tag-context-menu-props';
import type { KyouViewEmits } from './kyou-view-emits';
import type { Tag } from '@/classes/datas/tag';
import { type Ref, ref } from 'vue';
import EditTagDialog from '../dialogs/edit-tag-dialog.vue';
import ConfirmDeleteTagDialog from '../dialogs/confirm-delete-tag-dialog.vue';
import TagHistoriesDialog from '../dialogs/tag-histories-dialog.vue';

const props = defineProps<AttachedTagContextMenuProps>();
const emits = defineEmits<KyouViewEmits>();
const cloned_tag: Ref<Tag> = ref(await props.tag.clone());
</script>