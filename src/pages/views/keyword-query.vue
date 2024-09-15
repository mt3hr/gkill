<template>

</template>
<script lang="ts" setup>
import type { KeywordQueryEmits } from './keyword-query-emits';
import type { KeywordQueryProps } from './keyword-query-props';
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query';
import { ref } from 'vue';

const props = defineProps<KeywordQueryProps>();
const emits = defineEmits<KeywordQueryEmits>();
const cloned_query: FindKyouQuery = ref(await props.application_config.clone());
</script>
