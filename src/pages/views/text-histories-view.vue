<template>
    <TextView :application_config="application_config" :gkill_api="gkill_api" :text="text"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
</template>
<script lang="ts" setup>
import { type Ref, computed, ref } from 'vue'
import type { KyouViewEmits } from './kyou-view-emits'
import type { TextHistoriesViewProps } from './text-histories-view-props'
import TextView from './text-view.vue'
import { Text } from '@/classes/datas/text'

const props = defineProps<TextHistoriesViewProps>()
const emits = defineEmits<KyouViewEmits>()
</script>
