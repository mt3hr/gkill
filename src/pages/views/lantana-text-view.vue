<template>
    <LantanaTextTypeSelectBox />
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import LantanaTextTypeSelectBox from './lantana-text-type-select-box.vue'
import type { LantanaTextViewEmits } from './lantana-text-view-emits'
import { LantanaTextData } from '@/classes/lantana/lantana-text-data'
import { LantanaTextType } from '@/classes/lantana/lantana-text-type'

const emits = defineEmits<LantanaTextViewEmits>()
defineExpose({ update_text_content, update_text_type, get_text_data })

const lantana_text: Ref<LantanaTextData> = ref(new LantanaTextData())
const text_data: Ref<string> = ref("")

async function update_text_content(text_content: string): Promise<void> {
    throw new Error('Not implemented')
}
async function update_text_type(text_type: LantanaTextType): Promise<void> {
    throw new Error('Not implemented')
}
async function get_text_data(): Promise<LantanaTextData> {
    throw new Error('Not implemented')
}
</script>
