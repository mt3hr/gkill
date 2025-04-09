<template>
    <v-card @contextmenu.prevent="show_context_menu" :width="width" :height="height">
        <div>{{ kyou.typed_timeis?.title }}</div>
        <div v-if="kyou.typed_timeis && kyou.typed_timeis.start_time && show_timeis_elapsed_time">{{ duration }}</div>
        <br />
        <div v-if="kyou.typed_timeis">{{ $t("START_DATE_TIME_TITLE") }}：<span>{{
            format_time(kyou.typed_timeis.start_time) }}</span></div>
        <div v-if="kyou.typed_timeis && kyou.typed_timeis.end_time">{{ $t("END_DATE_TIME_TITLE") }}：<span>{{
            format_time(kyou.typed_timeis?.end_time) }}</span></div>
        <div v-if="kyou.typed_timeis && !kyou.typed_timeis.end_time && show_timeis_elapsed_time">{{ $t("PLAING_TITLE")
            }}</div>
        <v-row v-if="show_timeis_plaing_end_button && kyou.typed_timeis && !kyou.typed_timeis.end_time"
            class="pa-0 ma-0">
            <v-spacer cols="auto" />
            <v-col cols="auto" class="pa-0 ma-0">
                <v-btn dark color="primary" @click="show_end_timeis_dialog()">{{ $t("END_TITLE") }}</v-btn>
            </v-col>
        </v-row>
        <TimeIsContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag" ref="context_menu"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
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
        <EndTimeIsPlaingDialog :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
            ref="end_timeis_plaing_dialog" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
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
    </v-card>
</template>
<script setup lang="ts">
import { computed, ref } from 'vue'
import type { TimeIsViewProps } from './time-is-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import TimeIsContextMenu from './time-is-context-menu.vue'
import moment from 'moment';
import EndTimeIsPlaingDialog from '../dialogs/end-time-is-plaing-dialog.vue';
import { useI18n } from 'vue-i18n'

import { i18n } from '@/i18n'

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
        diff_str += diff_date.getFullYear() - 1970 + i18n.global.t("YEAR_SUFFIX")
    }
    if (diff_date.getMonth() !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += (diff_date.getMonth() + 1) + i18n.global.t("MONTH_SUFFIX")
    }
    if ((diff_date.getDate() - 1) !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += (diff_date.getDate() - 1) + i18n.global.t("DAY_SUFFIX")
    }
    if (diff_date.getHours() !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += (diff_date.getHours()) + i18n.global.t("HOUR_SUFFIX")
    }
    if (diff_date.getMinutes() !== 0) {
        if (diff_str !== "") {
            diff_str += " "
        }
        diff_str += diff_date.getMinutes() + i18n.global.t("MINUTE_SUFFIX")
    }
    if (diff_str === "") {
        diff_str += diff_date.getSeconds() + i18n.global.t("SECOND_SUFFIX")
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
