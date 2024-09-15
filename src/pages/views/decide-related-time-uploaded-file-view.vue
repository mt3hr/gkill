<template>
    <EditIDFKyouView />
    <KyouListView />
</template>
<script setup lang="ts">
import type { DecideRelatedTimeUploadedFileViewProps } from './decide-related-time-uploaded-file-view-props';
import type { KyouViewEmits } from './kyou-view-emits';
import EditIDFKyouView from './edit-idf-kyou-view.vue';
import KyouListView from './kyou-list-view.vue';

const props = defineProps<DecideRelatedTimeUploadedFileViewProps>();
const emits = defineEmits<KyouViewEmits>();
</script>
