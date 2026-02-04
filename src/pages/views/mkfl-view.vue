<template>
    <div>
        <kftlView :app_content_height="app_content_height.valueOf() / 2" :app_content_width="app_content_width"
            :application_config="application_config" :gkill_api="gkill_api"
            @deleted_kyou="(...deleted_kyou: any[]) => { reload_plaing_timeis_view(); emits('deleted_kyou', deleted_kyou[0] as Kyou) }"
            @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
            @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
            @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
            @registered_kyou="(...registered_kyou: any[]) => { reload_plaing_timeis_view(); emits('registered_kyou', registered_kyou[0] as Kyou) }"
            @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
            @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
            @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
            @updated_kyou="(...updated_kyou: any[]) => { reload_plaing_timeis_view(); emits('updated_kyou', updated_kyou[0] as Kyou) }"
            @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
            @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
            @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @saved_kyou_by_kftl="(last_added_request_time: Date) => {
                plaing_timeis_view?.set_last_added_request_time(last_added_request_time)
                reload_plaing_timeis_view()
            }" ref="kftl_view" />
        <PlaingTimeisView :application_config="application_config" :gkill_api="gkill_api"
            :app_content_height="app_content_height.valueOf() / 2" :app_content_width="app_content_width"
            @deleted_kyou="(...deleted_kyou: any[]) => { reload_plaing_timeis_view(); emits('deleted_kyou', deleted_kyou[0] as Kyou) }"
            @deleted_tag="(...deleted_tag: any[]) => emits('deleted_tag', deleted_tag[0] as Tag)"
            @deleted_text="(...deleted_text: any[]) => emits('deleted_text', deleted_text[0] as Text)"
            @deleted_notification="(...deleted_notification: any[]) => emits('deleted_notification', deleted_notification[0] as Notification)"
            @registered_kyou="(...registered_kyou: any[]) => { reload_plaing_timeis_view(); emits('registered_kyou', registered_kyou[0] as Kyou) }"
            @registered_tag="(...registered_tag: any[]) => emits('registered_tag', registered_tag[0] as Tag)"
            @registered_text="(...registered_text: any[]) => emits('registered_text', registered_text[0] as Text)"
            @registered_notification="(...registered_notification: any[]) => emits('registered_notification', registered_notification[0] as Notification)"
            @updated_kyou="(...updated_kyou: any[]) => { reload_plaing_timeis_view(); emits('updated_kyou', updated_kyou[0] as Kyou) }"
            @updated_tag="(...updated_tag: any[]) => emits('updated_tag', updated_tag[0] as Tag)"
            @updated_text="(...updated_text: any[]) => emits('updated_text', updated_text[0] as Text)"
            @updated_notification="(...updated_notification: any[]) => emits('updated_notification', updated_notification[0] as Notification)"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            ref="plaing_timeis_view" />
    </div>
</template>
<script lang="ts" setup>
import { ref } from 'vue';
import kftlView from './kftl-view.vue'
import PlaingTimeisView from './plaing-timeis-view.vue'
import type { MKFLProps } from './mkfl-view-props';
import type { MKFLViewEmits } from './mkfl-view-emits';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';
import type { Kyou } from "@/classes/datas/kyou"
import type { Tag } from "@/classes/datas/tag"
import type { Text } from "@/classes/datas/text"
import type { Notification } from "@/classes/datas/notification"

const plaing_timeis_view = ref<InstanceType<typeof PlaingTimeisView> | null>(null);

defineProps<MKFLProps>()
const emits = defineEmits<MKFLViewEmits>()

async function reload_plaing_timeis_view(): Promise<void> {
    plaing_timeis_view.value?.reload_list(false)
}
</script>