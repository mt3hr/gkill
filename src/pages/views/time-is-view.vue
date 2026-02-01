<template>
    <v-card @contextmenu.prevent="show_context_menu" :width="width" :height="height">
        <div>{{ kyou.typed_timeis?.title }}</div>
        <div class="duration" v-if="kyou.typed_timeis && kyou.typed_timeis.start_time && show_timeis_elapsed_time">{{
            duration }}</div>
        <br />
        <div v-if="kyou.typed_timeis">{{ i18n.global.t("START_DATE_TIME_TITLE") }}：<span>{{
            format_time(kyou.typed_timeis.start_time) }}</span></div>
        <div v-if="kyou.typed_timeis && kyou.typed_timeis.end_time">{{ i18n.global.t("END_DATE_TIME_TITLE") }}：<span>{{
            format_time(kyou.typed_timeis?.end_time) }}</span></div>
        <div v-if="kyou.typed_timeis && !kyou.typed_timeis.end_time && show_timeis_elapsed_time">{{
            i18n.global.t("PLAING_TITLE")
        }}</div>
        <v-row v-if="show_timeis_plaing_end_button && kyou.typed_timeis && !kyou.typed_timeis.end_time"
            class="pa-0 ma-0">
            <v-spacer cols="auto" />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="show_end_timeis_dialog()">{{ i18n.global.t("END_TITLE") }}</v-btn>
            </v-col>
        </v-row>
        <TimeIsContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag" ref="context_menu"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
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
            @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)" />
        <EndTimeIsPlaingDialog :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
            ref="end_timeis_plaing_dialog" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
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
            @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)" />
    </v-card>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import { computed, ref } from 'vue'
import type { TimeIsViewProps } from './time-is-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import TimeIsContextMenu from './time-is-context-menu.vue'
import moment from 'moment';
import EndTimeIsPlaingDialog from '../dialogs/end-time-is-plaing-dialog.vue';
import { format_duration, format_time } from '@/classes/format-date-time'
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { Kyou } from '@/classes/datas/kyou'

const context_menu = ref<InstanceType<typeof TimeIsContextMenu> | null>(null);
const end_timeis_plaing_dialog = ref<InstanceType<typeof EndTimeIsPlaingDialog> | null>(null);

const props = defineProps<TimeIsViewProps>()
const emits = defineEmits<KyouViewEmits>()
defineExpose({ show_context_menu })

const duration = computed(() => {
    let time1 = props.timeis.start_time
    let time2 = props.timeis.end_time

    time2 = time2 ? time2 : moment().toDate()
    const diff = Math.abs(time2.getTime() - time1.getTime())
    return format_duration(diff).replace("<br>", " ")
})

function show_context_menu(e: PointerEvent): void {
    if (props.enable_context_menu) {
        context_menu.value?.show(e)
    }
}

function show_end_timeis_dialog(): void {
    end_timeis_plaing_dialog.value?.show()
}
</script>