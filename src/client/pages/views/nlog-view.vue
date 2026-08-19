<template>
    <v-card elevation="0" @contextmenu.prevent="show_context_menu" :width="width" :height="height">
        <p v-if="kyou.typed_nlog" class="ma-0 pa-0">
            {{ (kyou.typed_nlog && kyou.typed_nlog.amount.valueOf() > 0) ? "↑" : "↓" }} {{ kyou.typed_nlog.title }}
        </p>
        <p v-if="kyou.typed_nlog" class="ma-0 pa-0">
            {{ "@".concat(kyou.typed_nlog.shop) }}
        </p>
        <div v-if="kyou.typed_nlog" class="ma-0 pa-0">
            <span
                :class="(kyou.typed_nlog && kyou.typed_nlog.amount.valueOf() > 0) ? 'nlog_amount_plus' : 'nlog_amount_minus'">{{
                    format_number(kyou.typed_nlog.amount.valueOf()) }}</span>
            {{ i18n.global.t("YEN_TITLE") }}
        </div>
    </v-card>
    <NlogContextMenu :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :enable_context_menu="enable_context_menu"
        v-on="crudRelayHandlers"
        :enable_dialog="enable_dialog" ref="context_menu" />
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { NlogViewProps } from './nlog-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import NlogContextMenu from './nlog-context-menu.vue'
import { useNlogView } from '@/classes/use-nlog-view'
import { format_number } from '@/classes/format-date-time'

const props = defineProps<NlogViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    context_menu,
    show_context_menu,
    crudRelayHandlers,
} = useNlogView({ props, emits })

defineExpose({ show_context_menu })
</script>
<style lang="css" scoped>
.nlog_amount_plus {
    color: limegreen;
}

.nlog_amount_minus {
    color: crimson;
}
</style>