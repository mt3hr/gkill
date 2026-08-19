<template>
    <v-card elevation="0" @contextmenu.prevent="show_context_menu" :width="width" :height="height">
        <div v-if="kyou.typed_kc">{{ kyou.typed_kc.title }}</div>
        <div v-if="kyou.typed_kc">{{ kyou.typed_kc.num_value }}</div>
        <KCContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
            v-on="crudRelayHandlers"
            ref="context_menu" />
    </v-card>
</template>
<script setup lang="ts">
import KCContextMenu from './kc-context-menu.vue'
import type { KCViewProps } from './kc-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { useKCView } from '@/classes/use-kc-view'

const props = defineProps<KCViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    context_menu,
    show_context_menu,
    // Event relay objects
    crudRelayHandlers,
} = useKCView({ props, emits })

defineExpose({ show_context_menu })
</script>
