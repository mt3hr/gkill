<template>
    <v-card elevation="0" @contextmenu.prevent="show_context_menu" :width="width" :height="height" style="overflow: hidden">
        <LantanaFlowersView v-if="kyou.typed_lantana" :application_config="application_config" :gkill_api="gkill_api"
            :editable="false" :mood="kyou.typed_lantana.mood" />
        <LantanaContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" ref="context_menu"
            v-on="crudRelayHandlers"
            />
    </v-card>
</template>
<script setup lang="ts">
import type { LantanaViewProps } from './lantana-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import LantanaContextMenu from './lantana-context-menu.vue'
import LantanaFlowersView from './lantana-flowers-view.vue'
import { useLantanaView } from '@/classes/use-lantana-view'

const props = defineProps<LantanaViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    context_menu,
    show_context_menu,
    // Event relay objects
    crudRelayHandlers,
} = useLantanaView({ props, emits })

defineExpose({ show_context_menu })
</script>
