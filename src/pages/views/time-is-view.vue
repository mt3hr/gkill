<template>
    <v-card @contextmenu.prevent="show_context_menu" :width="width" :height="height">
        <div>{{ kyou.typed_timeis?.title }}</div>
        <div v-if="kyou.typed_timeis && kyou.typed_timeis.start_time && show_timeis_elapsed_time">{{ duration }}</div>
        <br />
        <div v-if="kyou.typed_timeis">開始日時：<span>{{ format_time(kyou.typed_timeis.start_time) }}</span></div>
        <div v-if="kyou.typed_timeis && kyou.typed_timeis.end_time">終了日時：<span>{{
            format_time(kyou.typed_timeis?.end_time) }}</span></div>
        <div v-if="kyou.typed_timeis && !kyou.typed_timeis.end_time && show_timeis_elapsed_time">実行中</div>
        <v-row v-if="show_timeis_plaing_end_button && kyou.typed_timeis && !kyou.typed_timeis.end_time"
            class="pa-0 ma-0">
            <v-spacer cols="auto" />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn color="primary" @click="show_end_timeis_dialog()">終了</v-btn>
            </v-col>
        </v-row>
        <TimeIsContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag" ref="context_menu"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        <EndTimeIsPlaingDialog :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
            ref="end_timeis_plaing_dialog" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="emits('requested_reload_list')"
            @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
    </v-card>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue'
import type { TimeIsViewProps } from './time-is-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import TimeIsContextMenu from './time-is-context-menu.vue'
import moment from 'moment';
import EndTimeIsPlaingDialog from '../dialogs/end-time-is-plaing-dialog.vue';

const context_menu = ref<InstanceType<typeof TimeIsContextMenu> | null>(null);
const end_timeis_plaing_dialog = ref<InstanceType<typeof EndTimeIsPlaingDialog> | null>(null);

const props = defineProps<TimeIsViewProps>()
const emits = defineEmits<KyouViewEmits>()
defineExpose({ show_context_menu })

const duration = computed(() => {
    let time1 = props.timeis.start_time
    let time2 = props.timeis.end_time

    let diff_str = ""
    time2 = time2 ? time2 : moment().toDate()
    const offset_in_locale_milli_second = new Date().getTimezoneOffset().valueOf() * 60000
    const diff = Math.abs(time2.getTime() - time1.getTime())
    const diff_date = moment(diff + offset_in_locale_milli_second).toDate()
    if (diff_date.getFullYear() - 1970 !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += diff_date.getFullYear() - 1970 + "年"
    }
    if (diff_date.getMonth() !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += (diff_date.getMonth() + 1) + "ヶ月"
    }
    if ((diff_date.getDate() - 1) !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += (diff_date.getDate() - 1) + "日"
    }
    if (diff_date.getHours() !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += (diff_date.getHours()) + "時間"
    }
    if (diff_date.getMinutes() !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += diff_date.getMinutes() + "分"
    }
    if (diff_str === "") {
        diff_str += diff_date.getSeconds() + "秒"
    }
    return diff_str
})

function format_time(time: Date): string {
    return moment(time).format("yyyy-MM-DD HH:mm:ss")
}

function show_context_menu(e: PointerEvent): void {
    if (props.enable_context_menu) {
        context_menu.value?.show(e)
    }
}

function show_end_timeis_dialog(): void {
    end_timeis_plaing_dialog.value?.show()
}
</script>
