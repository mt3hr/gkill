<template>

</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import { MiCheckState } from '@/classes/api/find_query/mi-check-state'
import type { miExtructCheckStateQueryEmits } from './mi-extruct-check-state-query-emits'
import type { miExtructCheckStateQueryProps } from './mi-extruct-check-state-query-props'
const props = defineProps<miExtructCheckStateQueryProps>()
const emits = defineEmits<miExtructCheckStateQueryEmits>()
const cloned_check_state: Ref<MiCheckState> = ref(MiCheckState.uncheck)
</script>
