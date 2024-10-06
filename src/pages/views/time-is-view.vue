<template>
    <v-card @contextmenu.prevent="show_context_menu">
        <div>{{ kyou.typed_timeis?.title }}</div>
        <div v-if="kyou.typed_timeis && kyou.typed_timeis.start_time">{{
            format_duration(kyou.typed_timeis?.start_time, kyou.typed_timeis?.end_time) }}</div>
        <br />
        <div v-if="kyou.typed_timeis">開始日時：<span>{{ format_time(kyou.typed_timeis.start_time) }}</span></div>
        <div v-if="kyou.typed_timeis && kyou.typed_timeis.end_time">終了日時：<span>{{
            format_time(kyou.typed_timeis?.end_time) }}</span></div>
        <div v-if="kyou.typed_timeis && !kyou.typed_timeis.end_time">実行中</div>
    </v-card>
    <TimeIsContextMenu :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets.concat([kyou.generate_info_identifer()])" :kyou="kyou"
        :last_added_tag="last_added_tag" ref="context_menu"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="() => emits('requested_reload_list')"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import type { TimeIsViewProps } from './time-is-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import TimeIsContextMenu from './time-is-context-menu.vue'
import moment from 'moment';

const context_menu = ref<InstanceType<typeof TimeIsContextMenu> | null>(null);

const props = defineProps<TimeIsViewProps>()
const emits = defineEmits<KyouViewEmits>()

function format_duration(time1: Date, time2: Date | null): string {
    let diff_str = ""
    time2 = time2 ? time2 : new Date(Date.now())
    const diff = time2?.getTime() - time1.getTime()
    const diff_date = new Date(diff)
    if ((diff_date.getFullYear() - (new Date(0).getFullYear() as number)) !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += diff_date.getFullYear() + "年"
    }
    if (diff_date.getMonth() !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += (diff_date.getMonth() + 1) + "ヶ月"
    }
    if (diff_date.getDate() !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += diff_date.getDate() + "日"
    }
    if (diff_date.getHours() !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += diff_date.getHours() + "時間"
    }
    if (diff_date.getMinutes() !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += diff_date.getMinutes() + "分"
    }
    if (diff_date.getSeconds() !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += diff_date.getSeconds() + "秒"
    }
    return diff_str
}

function format_time(time: Date): string {
    return moment(time).format("yyyy-MM-DD HH:mm:ss")
}

function show_context_menu(e: PointerEvent): void {
    context_menu.value?.show(e)
}
</script>

