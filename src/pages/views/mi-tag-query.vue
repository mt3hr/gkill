<template>
    <FoldableStruct />
</template>
<script setup lang="ts">
import FoldableStruct from './foldable-struct.vue';
import type { miTagQueryEmits } from './mi-tag-query-emits';
import type { miTagQueryProps } from './mi-tag-query-props';

const props = defineProps<miTagQueryProps>();
const emits = defineEmits<miTagQueryEmits>();
</script>