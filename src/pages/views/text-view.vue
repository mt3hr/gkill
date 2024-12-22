<template>
    <v-row class="pa-0 ma-0">
        <v-col cols="auto" class="pa-0 ma-0">
            <AttachedText :text="text" :application_config="application_config" :gkill_api="gkill_api" :kyou="kyou"
                :last_added_tag="last_added_tag" :highlight_targets="highlight_targets"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="() => emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        </v-col>
        <v-spacer />
        <v-col cols="auto" class="pa-0 ma-0">
            <span class="update_time">
                {{ format_time(text.update_time) }}
            </span>
        </v-col>
        <v-spacer />
        <v-col cols="auto" class="pa-0 ma-0">
            <span class="update_device">
                {{ text.update_device }}
            </span>
        </v-col>
    </v-row>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { KyouViewEmits } from './kyou-view-emits'
import type { TextViewProps } from './text-view-props'
import { Text } from '@/classes/datas/text'
import AttachedText from './attached-text.vue';
import moment from 'moment';

const props = defineProps<TextViewProps>()
const emits = defineEmits<KyouViewEmits>()

function format_time(time: Date): string {
    return moment(time).format("yyyy-MM-DD HH:mm:ss")
}
</script>
