<template>
    <TagStructElement />
    <AddNewTagStructElementDialog />
    <EditTagStructElementDialog />
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { EditTagStructViewEmits } from './edit-tag-struct-view-emits';
import type { EditTagStructViewProps } from './edit-tag-struct-view-props';
import TagStructElement from './tag-struct-element.vue';
import { TagStruct } from '@/classes/datas/config/tag-struct';
import AddNewTagStructElementDialog from '../dialogs/add-new-tag-struct-element-dialog.vue';
import EditTagStructElementDialog from '../dialogs/edit-tag-struct-element-dialog.vue';

const props = defineProps<EditTagStructViewProps>();
const emits = defineEmits<EditTagStructViewEmits>();
const cloned_application_config = ref(await props.application_config.clone());
const tag_struct: Ref<TagStruct> = ref(await props.tag_struct.clone());
const tag_struct_root: Ref<TagStruct> = ref(await props.tag_struct_root.clone());
</script>