<template>

</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { MiSortType } from '@/classes/api/find_query/mi-sort-type'
import type { miSortTypeQueryEmits } from './mi-sort-type-query-emits'
import type { miSortTypeQueryProps } from './mi-sort-type-query-props'
const props = defineProps<miSortTypeQueryProps>()
const emits = defineEmits<miSortTypeQueryEmits>()
const cloned_sort_type: Ref<MiSortType> = ref(props.sort_type)
</script>
