<template>
    <TagStructElement />
    <RepStructElement />
    <AddNewRepStructElementDialog />
    <EditRepStructElementDialog />
</template>
<script lang="ts" setup>
import { ref } from 'vue';
import type { EditRepStructViewEmits } from './edit-rep-struct-view-emits';
import type { EditRepStructViewProps } from './edit-rep-struct-view-props';
import RepStructElement from './rep-struct-element.vue';
import TagStructElement from './tag-struct-element.vue';
import AddNewRepStructElementDialog from '../dialogs/add-new-rep-struct-element-dialog.vue';
import EditRepStructElementDialog from '../dialogs/edit-rep-struct-element-dialog.vue';

const props = defineProps<EditRepStructViewProps>();
const emits = defineEmits<EditRepStructViewEmits>();
const cloned_application_config = ref(await props.application_config.clone());
</script>
