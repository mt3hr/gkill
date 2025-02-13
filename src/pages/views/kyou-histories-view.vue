<template>
    <v-card>
        <div>
            <KyouView v-for="kyou in kyou.attached_histories" :application_config="application_config"
                :key="kyou.update_time.getTime()" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                :is_image_view="false" :kyou="kyou" :last_added_tag="last_added_tag" :show_checkbox="false"
                :show_content_only="false" :show_mi_create_time="true" :show_mi_estimate_end_time="true"
                :show_mi_estimate_start_time="true" :show_mi_limit_time="true" :show_timeis_elapsed_time="false"
                :show_timeis_plaing_end_button="false" :height="'100%'" :width="'100%'"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" :is_readonly_mi_check="true"
                :show_attached_timeis="false" @received_errors="(errors) => emits('received_errors', errors)"
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
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
        </div>
    </v-card>
</template>
<script lang="ts" setup>
import type { KyouHistoriesViewProps } from './kyou-histories-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import KyouView from './kyou-view.vue'

defineProps<KyouHistoriesViewProps>()
const emits = defineEmits<KyouViewEmits>()
</script>
