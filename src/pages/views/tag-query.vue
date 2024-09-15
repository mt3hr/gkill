<template>
    <FoldableStruct />
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue';
import type { TagQueryEmits } from './tag-query-emits';
import type { TagQueryProps } from './tag-query-props';
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query';
import FoldableStruct from './foldable-struct.vue';

const props = defineProps<TagQueryProps>();
const emits = defineEmits<TagQueryEmits>();
const cloned_query: Ref<FindKyouQuery> = ref(await props.query.clone());
</script>
