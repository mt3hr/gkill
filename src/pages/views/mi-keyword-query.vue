<template>

</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { FindMiQuery } from '@/classes/api/find_query/find-mi-query';
import type { miKeywordQueryEmits } from './mi-keyword-query-emits';
import type { miKeywordQueryProps } from './mi-keyword-query-props';
const props = defineProps<miKeywordQueryProps>();
const emits = defineEmits<miKeywordQueryEmits>();
const cloned_query: Ref<FindMiQuery> = ref(await props.query.clone());
</script>