<template>
    <v-card @contextmenu.prevent="show_context_menu" :width="width" :height="height">
        <div v-if="kyou.typed_kmemo" class="kmemo_text_content">{{ kyou.typed_kmemo.content }}</div>
        <KmemoContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
            v-on="crudRelayHandlers"
            ref="context_menu" />
    </v-card>
</template>
<script setup lang="ts">
import KmemoContextMenu from './kmemo-context-menu.vue'
import type { KmemoViewProps } from './kmemo-view-props.ts'
import type { KyouViewEmits } from './kyou-view-emits'
import { useKmemoView } from '@/classes/use-kmemo-view'

const props = defineProps<KmemoViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    context_menu,
    show_context_menu,
    crudRelayHandlers,
} = useKmemoView({ props, emits })

defineExpose({ show_context_menu })
</script>

<style lang="css" scoped>
.kmemo_text_content {
    white-space: pre-line;
    overflow-wrap: anywhere;
}
</style>
