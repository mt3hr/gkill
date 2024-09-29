<template>

</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { AddRepViewEmits } from './add-rep-view-emits'
import type { AddRepViewProps } from './add-rep-view-props'

const props = defineProps<AddRepViewProps>()
const emits = defineEmits<AddRepViewEmits>()
const user_id: Ref<string> = ref("")
const device: Ref<string> = ref("")
const type: Ref<string> = ref("")
const file: Ref<string> = ref("")
const use_to_write: Ref<boolean> = ref(false)
const is_execute_idf_when_reload: Ref<boolean> = ref(false)
const is_enable: Ref<boolean> = ref(true)
</script>
