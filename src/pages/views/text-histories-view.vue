<template>
    <TextView class="text_history" v-for="text in cloned_text.attached_histories" :key="text.id"
        :application_config="application_config" :gkill_api="gkill_api" :text="text" :kyou="kyou"
        :last_added_tag="last_added_tag" :highlight_targets="highlight_targets"
        @deleted_kyou="(deleted_kyou) => emits('deleted_kyou', deleted_kyou)"
        @deleted_tag="(deleted_tag) => emits('deleted_tag', deleted_tag)"
        @deleted_text="(deleted_text) => emits('deleted_text', deleted_text)"
        @deleted_notification="(deleted_notification) => emits('deleted_notification', deleted_notification)"
        @registered_kyou="(registered_kyou) => emits('registered_kyou', registered_kyou)"
        @registered_tag="(registered_tag) => emits('registered_tag', registered_tag)"
        @registered_text="(registered_text) => emits('registered_text', registered_text)"
        @registered_notification="(registered_notification) => emits('registered_notification', registered_notification)"
        @updated_kyou="(updated_kyou) => emits('updated_kyou', updated_kyou)"
        @updated_tag="(updated_tag) => emits('updated_tag', updated_tag)"
        @updated_text="(updated_text) => emits('updated_text', updated_text)"
        @updated_notification="(updated_notification) => emits('updated_notification', updated_notification)"
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
<style lang="css">
.text_history .highlighted_text,
.text_history .text {
    width: 400px;
}
</style>