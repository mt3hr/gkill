<template>
    <TextView v-for="text in cloned_text.attached_histories" :key="text.id"
        :application_config="application_config" :gkill_api="gkill_api" :text="text" :kyou="kyou"
        :last_added_tag="last_added_tag" :highlight_targets="highlight_targets"
        @received_errors="(errors) => emits('received_errors', errors)" :enable_context_menu="enable_context_menu"
        :enable_dialog="enable_dialog" @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')" />
</template>
<script lang="ts" setup>
import { type Ref, nextTick, ref, watch } from 'vue'
import type { KyouViewEmits } from './kyou-view-emits'
import type { TextHistoriesViewProps } from './text-histories-view-props'
import { Text } from '@/classes/datas/text'
import TextView from './text-view.vue'

const props = defineProps<TextHistoriesViewProps>()
const emits = defineEmits<KyouViewEmits>()

const cloned_text: Ref<Text> = ref(props.text.clone())
watch(() => props.text, () => {
    cloned_text.value = props.text.clone()
    nextTick(() => cloned_text.value.load_attached_histories())
})
nextTick(() => cloned_text.value.load_attached_histories())

</script>
