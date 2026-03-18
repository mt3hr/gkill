<template>
    <v-card elevation="0" @contextmenu.prevent="show_context_menu" :width="width" :height="height">
        <table class="ma-0 pa-0">
            <tr class="ma-0 pa-0">
                <td v-if="kyou.typed_nlog && kyou.typed_nlog.amount.valueOf() > 0" cols="auto" class="ma-0 pa-0">
                    ↑
                </td>
                <td v-if="kyou.typed_nlog && kyou.typed_nlog.amount.valueOf() <= 0" cols="auto" class="ma-0 pa-0">
                    ↓
                </td>
                <td v-if="kyou.typed_nlog" cols="auto" class="ma-0 pa-0">
                    {{ kyou.typed_nlog.title }}
                </td>
            </tr>
        </table>
        <v-row class="ma-0 pa-0">
            <v-col v-if="kyou.typed_nlog" cols="auto" class="ma-0 pa-0">
                {{ "@".concat(kyou.typed_nlog.shop) }}
            </v-col>
        </v-row>
        <v-row class="ma-0 pa-0">
            <v-col v-if="kyou.typed_nlog && kyou.typed_nlog.amount.valueOf() > 0" class="ma-0 pa-0 nlog_amount_plus"
                cols="auto">
                {{ kyou.typed_nlog.amount }}
            </v-col>
            <v-col v-if="kyou.typed_nlog && kyou.typed_nlog.amount.valueOf() <= 0" class="ma-0 pa-0 nlog_amount_minus"
                cols="auto">
                {{ kyou.typed_nlog.amount }}
            </v-col>
            <v-col v-if="kyou.typed_nlog" class="ma-0 pa-0">
                {{ i18n.global.t("YEN_TITLE") }}
            </v-col>
            <v-col cols="auto" class="ma-0 pa-0"></v-col>
        </v-row>
    </v-card>
    <NlogContextMenu :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" ref="context_menu"
        @deleted_kyou="crudRelayHandlers['deleted_kyou']"
        @deleted_tag="crudRelayHandlers['deleted_tag']"
        @deleted_text="crudRelayHandlers['deleted_text']"
        @deleted_notification="crudRelayHandlers['deleted_notification']"
        @registered_kyou="crudRelayHandlers['registered_kyou']"
        @registered_tag="crudRelayHandlers['registered_tag']"
        @registered_text="crudRelayHandlers['registered_text']"
        @registered_notification="crudRelayHandlers['registered_notification']"
        @updated_kyou="crudRelayHandlers['updated_kyou']"
        @updated_tag="crudRelayHandlers['updated_tag']"
        @updated_text="crudRelayHandlers['updated_text']"
        @updated_notification="crudRelayHandlers['updated_notification']"
        @received_errors="crudRelayHandlers['received_errors']"
        @received_messages="crudRelayHandlers['received_messages']"
        @requested_reload_kyou="crudRelayHandlers['requested_reload_kyou']"
        @requested_reload_list="crudRelayHandlers['requested_reload_list']"
        @requested_update_check_kyous="crudRelayHandlers['requested_update_check_kyous']"
        @requested_open_rykv_dialog="crudRelayHandlers['requested_open_rykv_dialog']" />
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { NlogViewProps } from './nlog-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import NlogContextMenu from './nlog-context-menu.vue'
import { useNlogView } from '@/classes/use-nlog-view'

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