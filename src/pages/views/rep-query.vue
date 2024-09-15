<template>
    <FoldableStruct />
    <FoldableStruct />
    <FoldableStruct />
    <FoldableStruct />
</template>
<script setup lang="ts">
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query';
import { type Ref, ref } from 'vue';
import FoldableStruct from './foldable-struct.vue';
import type { RepQueryEmits } from './rep-query-emits';
import type { RepQueryProps } from './rep-query-props';

const props = defineProps<RepQueryProps>();
const emits = defineEmits<RepQueryEmits>();
const cloned_query: Ref<FindKyouQuery> = ref(await props.query.clone());
</script>
<style scoped></style>