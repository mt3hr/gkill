<template>
    <AddNewKftlTemplateStructElementDialog />
    <EditKftlTemplateStructElementDialog />
    <TagStructElement />
    <KFTLTemplateElement />
</template>
<script lang="ts" setup>
import type { EditKFTLTemplateViewEmits } from './edit-kftl-template-view-emits';
import type { EditKFTLTemplateViewProps } from './edit-kftl-template-view-props';
import KFTLTemplateElement from './kftl-template-element.vue';
import TagStructElement from './tag-struct-element.vue';
import AddNewKftlTemplateStructElementDialog from '../dialogs/add-new-kftl-template-struct-element-dialog.vue';
import EditKftlTemplateStructElementDialog from '../dialogs/edit-kftl-template-struct-element-dialog.vue';
import { ref } from 'vue';

const props = defineProps<EditKFTLTemplateViewProps>();
const emits = defineEmits<EditKFTLTemplateViewEmits>();
const cloned_application_config = ref(await props.application_config.clone())
</script>