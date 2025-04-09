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
                {{ $t("YEN_TITLE") }}
            </v-col>
            <v-col cols="auto" class="ma-0 pa-0"></v-col>
        </v-row>
    </v-card>
    <NlogContextMenu :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" ref="context_menu"
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
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
</template>
<script setup lang="ts">
import type { NlogViewProps } from './nlog-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { ref } from 'vue'
import NlogContextMenu from './nlog-context-menu.vue'
import { useI18n } from 'vue-i18n'

import { i18n } from '@/i18n'
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
