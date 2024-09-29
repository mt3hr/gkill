<template>

</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { ConfirmDeleteRepViewEmits } from './confirm-delete-rep-view-emits'
import type { ConfirmDeleteRepViewProps } from './confirm-delete-rep-view-props'

const props = defineProps<ConfirmDeleteRepViewProps>()
const emits = defineEmits<ConfirmDeleteRepViewEmits>()
</script>
