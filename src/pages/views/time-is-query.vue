<template>
    <FoldableStruct />
</template>
<script setup lang="ts">
import type { TimeIsQueryEmits } from './time-is-query-emits';
import type { TimeIsQueryProps } from './time-is-query-props';
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query';
import { type Ref, ref } from 'vue';

import FoldableStruct from './foldable-struct.vue';

const props = defineProps<TimeIsQueryProps>();
const emits = defineEmits<TimeIsQueryEmits>();
const cloned_query: Ref<FindKyouQuery> = ref(await props.query.clone());
</script>
