<template>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { miBoardQueryEmits } from './mi-board-query-emits'
import type { miBoardQueryProps } from './mi-board-query-props'
const props = defineProps<miBoardQueryProps>()
const emits = defineEmits<miBoardQueryEmits>()
const cloned_board_names: Ref<Array<string>> = ref(props.board_names.slice())
</script>
