<template>

</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { KyouViewEmits } from './kyou-view-emits'
import type { TextViewProps } from './text-view-props'
import { Text } from '@/classes/datas/text'

const props = defineProps<TextViewProps>()
const emits = defineEmits<KyouViewEmits>()
const cloned_text: Ref<Text> = ref(await props.text.clone())
</script>
