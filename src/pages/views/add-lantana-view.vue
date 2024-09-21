<template>
    <!-- //TODO 実装 -->
    <LantanaFlowersView :application_config="application_config" :gkill_api="gkill_api" :editable="true" :mood="0"
        @updated_mood="(new_mood_value) => mood = new_mood_value" />
</template>
<script lang="ts" setup>
import type { GkillError } from '@/classes/api/gkill-error'
import { LantanaTextData } from '@/classes/lantana/lantana-text-data'
import { ref, type Ref } from 'vue'

import type { LantanaProps } from './lantana-props'
import type { AddLantanaViewEmits } from './add-lantana-view-emits'
import LantanaFlowersView from './lantana-flowers-view.vue'

const text_datas: Ref<Array<LantanaTextData>> = ref(new Array<LantanaTextData>())
const mood: Ref<Number> = ref(0)
const text_datas_dummy: Ref<Array<LantanaTextData>> = ref(new Array<LantanaTextData>())
const props = defineProps<LantanaProps>()
const emits = defineEmits<AddLantanaViewEmits>()

async function update_mood(mood: Number): Promise<void> {
    throw new Error('Not implemented')
}
async function clear(): Promise<void> {
    throw new Error('Not implemented')
}
async function submit(): Promise<Array<GkillError>> {
    throw new Error('Not implemented')
}
async function update_text_data(text_data: LantanaTextData): Promise<void> {
    throw new Error('Not implemented')
}
async function add_text_data(): Promise<void> {
    throw new Error('Not implemented')
}
async function delete_text_data(text_data: LantanaTextData): Promise<boolean> {
    throw new Error('Not implemented')
}
</script>
