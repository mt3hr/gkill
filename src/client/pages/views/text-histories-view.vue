<template>
    <TextView class="text_history" v-for="text in cloned_text.attached_histories" :key="text.id"
        :application_config="application_config" :gkill_api="gkill_api" :text="text" :kyou="kyou"
        :highlight_targets="highlight_targets"
         :enable_context_menu="enable_context_menu"
        :enable_dialog="enable_dialog" v-on="crudRelayHandlers" />
</template>
<script lang="ts" setup>
import type { KyouViewEmits } from './kyou-view-emits'
import type { TextHistoriesViewProps } from './text-histories-view-props'
import TextView from './text-view.vue'
import { useTextHistoriesView } from '@/classes/use-text-histories-view'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

const props = defineProps<TextHistoriesViewProps>()
const emits = defineEmits<KyouViewEmits>()

const crudRelayHandlers = build_kyou_view_relay(emits)

const {
    cloned_text,
} = useTextHistoriesView({ props, emits })
</script>
<style lang="css">
.text_history .highlighted_text,
.text_history .text {
    width: 400px;
}
</style>
