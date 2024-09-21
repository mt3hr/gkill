<template>

</template>
<script lang="ts" setup>
import type { CalendarQueryEmits } from './calendar-query-emits'
import type { CalendarQueryProps } from './calendar-query-props'
import { ref } from 'vue'

const props = defineProps<CalendarQueryProps>()
const emits = defineEmits<CalendarQueryEmits>()
const cloned_query = ref(await props.query.clone())
</script>
