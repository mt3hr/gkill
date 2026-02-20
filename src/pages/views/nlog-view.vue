<template>
    <v-card @contextmenu.prevent="show_context_menu" :width="width" :height="height">
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
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" ref="context_menu"
        @deleted_kyou="(...deleted_kyou :any[]) => emits('deleted_kyou', deleted_kyou[0] as Kyou)"
        @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
        @deleted_text="(...deleted_text :any[]) => emits('deleted_text', deleted_text[0] as Text)"
        @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
        @registered_kyou="(...registered_kyou: any[]) => emits('registered_kyou', registered_kyou[0] as Kyou)"
        @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
        @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
        @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
        @updated_kyou="(...updated_kyou: any[]) => emits('updated_kyou', updated_kyou[0] as Kyou)"
        @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
        @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
        @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
        @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
        @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
        @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
        @requested_reload_list="emits('requested_reload_list')"
        @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)"
            @requested_open_rykv_dialog="(...params: any[]) => emits('requested_open_rykv_dialog', params[0], params[1], params[2])" />
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import type { NlogViewProps } from './nlog-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { ref } from 'vue'
import NlogContextMenu from './nlog-context-menu.vue'
import type { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

const context_menu = ref<InstanceType<typeof NlogContextMenu> | null>(null);

const props = defineProps<NlogViewProps>()
const emits = defineEmits<KyouViewEmits>()
defineExpose({ show_context_menu })

function show_context_menu(e: PointerEvent): void {
    if (props.enable_context_menu) {
        context_menu.value?.show(e)
    }
}
</script>
<style lang="css" scoped>
.nlog_amount_plus {
    color: limegreen;
}

.nlog_amount_minus {
    color: crimson;
}
</style>
