<template>
    <v-card>
        <div>
            <KyouView v-for="kyou in cloned_kyou.attached_histories" :application_config="application_config"
                :key="kyou.update_time.getTime()" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
                :is_image_view="false" :kyou="kyou" :last_added_tag="last_added_tag" :show_checkbox="false"
                :show_content_only="false" :show_mi_create_time="true" :show_mi_estimate_end_time="true"
                :show_mi_estimate_start_time="true" :show_mi_limit_time="true" :show_timeis_elapsed_time="false"
                :show_timeis_plaing_end_button="false" :height="'100%'" :width="'100%'"
                :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" :is_readonly_mi_check="true"
                :show_rep_name="true" :force_show_latest_kyou_info="false" :show_attached_timeis="false"
                :show_attached_tags="false" :show_attached_texts="false" :show_attached_notifications="false"
                @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)" :show_update_time="true"
                :show_related_time="false" @deleted_kyou="(...deleted_kyou :any[]) => emits('deleted_kyou', deleted_kyou[0] as Kyou)"
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
                @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
                @requested_reload_list="emits('requested_reload_list')"
                @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)" />
        </div>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { ref } from 'vue';
import type { KyouHistoriesViewProps } from './kyou-histories-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import KyouView from './kyou-view.vue'
import { Kyou } from '@/classes/datas/kyou';
import type { GkillMessage } from '@/classes/api/gkill-message';
import type { GkillError } from '@/classes/api/gkill-error';
import type { Tag } from '@/classes/datas/tag';
import type { Text } from '@/classes/datas/text';
import type { Notification } from '@/classes/datas/notification';

const props = defineProps<KyouHistoriesViewProps>()
const emits = defineEmits<KyouViewEmits>()

const cloned_kyou = ref(new Kyou())
load_cloned_kyou()

async function load_cloned_kyou() {
    const cloned_kyou_value = props.kyou.clone()
    await cloned_kyou_value.load_attached_histories()
    for (let i = 0; i < cloned_kyou.value.attached_histories.length; i++) {
        cloned_kyou_value.attached_histories[i].related_time = cloned_kyou_value.attached_histories[i].update_time
    }
    cloned_kyou.value = cloned_kyou_value
}
</script>
