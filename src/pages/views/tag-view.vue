<template>
    <v-row class="pa-0 ma-0">
        <v-col cols="auto" class="pa-0 ma-0">
            <AttachedTag :tag="tag" :application_config="application_config" :gkill_api="gkill_api" :kyou="kyou"
                :highlight_targets="highlight_targets" :last_added_tag="last_added_tag"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="() => emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        </v-col>
        <v-spacer />
        <v-col cols="auto" class="pa-0 ma-0">
            <span class="update_time">
                {{ format_time(tag.update_time) }}
            </span>
        </v-col>
        <v-spacer />
        <v-col cols="auto" class="pa-0 ma-0">
            <span class="update_device">
                {{ tag.update_device }}
            </span>
        </v-col>
    </v-row>
</template>
<script lang="ts" setup>
import type { KyouViewEmits } from './kyou-view-emits'
import type { TagViewProps } from './tag-view-props'
import moment from 'moment';
import AttachedTag from './attached-tag.vue';

const props = defineProps<TagViewProps>()
const emits = defineEmits<KyouViewEmits>()

function format_time(time: Date): string {
    return moment(time).format("yyyy-MM-DD HH:mm:ss")
}
</script>