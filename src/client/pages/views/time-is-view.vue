<template>
    <v-card elevation="0" @contextmenu.prevent="show_context_menu" :width="width" :height="height">
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
            <v-col cols="auto" class="pa-0 ma-0 pa-0">
                <v-btn dark color="primary" @click="show_end_timeis_dialog()">{{ i18n.global.t("END_TITLE") }}</v-btn>
            </v-col>
        </v-row>
        <TimeIsContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou" ref="context_menu"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
            v-on="crudRelayHandlers" />
        <EndTimeIsPlaingDialog :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou"
            ref="end_timeis_plaing_dialog" :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
            v-on="crudRelayHandlers" />
    </v-card>
</template>
<script setup lang="ts">
import { i18n } from '@/i18n'
import { ref } from 'vue'
import type { TimeIsViewProps } from './time-is-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import TimeIsContextMenu from './time-is-context-menu.vue'
import EndTimeIsPlaingDialog from '../dialogs/end-time-is-plaing-dialog.vue'
import { format_time } from '@/classes/format-date-time'
import { useTimeIsView } from '@/classes/use-time-is-view'

const context_menu = ref<InstanceType<typeof TimeIsContextMenu> | null>(null)
const end_timeis_plaing_dialog = ref<InstanceType<typeof EndTimeIsPlaingDialog> | null>(null)

const props = defineProps<TimeIsViewProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    duration,
    show_context_menu,
    show_end_timeis_dialog,
    crudRelayHandlers,
} = useTimeIsView({ props, emits, context_menu, end_timeis_plaing_dialog })

defineExpose({ show_context_menu })
</script>
