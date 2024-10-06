<template>
    <TextView v-for="text, index in text.attached_histories" :application_config="application_config"
        :gkill_api="gkill_api" :text="text" :kyou="kyou" :last_added_tag="last_added_tag"
        :highlight_targets="highlight_targets" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
</template>
<script lang="ts" setup>
import { type Ref, computed, ref } from 'vue'
import type { KyouViewEmits } from './kyou-view-emits'
import type { TextHistoriesViewProps } from './text-histories-view-props'
import { Text } from '@/classes/datas/text'
import TextView from './text-view.vue'

const props = defineProps<TextHistoriesViewProps>()
const emits = defineEmits<KyouViewEmits>()
</script>
