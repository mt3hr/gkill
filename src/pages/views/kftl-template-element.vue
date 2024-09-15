<template>
    <ApplicationConfigStructContextMenu />
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue';
import ApplicationConfigStructContextMenu from './application-config-struct-context-menu.vue';
import type { KFTLTemplateElementEmits } from './kftl-template-element-emits';
import type { KFTLTemplateElementProps } from './kftl-template-element-props';

const props = defineProps<KFTLTemplateElementProps>();
const emits = defineEmits<KFTLTemplateElementEmits>();
const cloned_is_open: Ref<boolean> = ref(props.is_open);
const struct_list: Ref<any> = ref(props.struct_obj);
</script>
