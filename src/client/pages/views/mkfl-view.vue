<template>
    <div>
        <kftlView :app_content_height="app_content_height.valueOf() / 2" :app_content_width="app_content_width"
            :application_config="application_config" :gkill_api="gkill_api"
            @deleted_kyou="(deleted_kyou: Kyou) => { reload_plaing_timeis_view(); emits('deleted_kyou', deleted_kyou) }"
            @deleted_tag="(deleted_tag: Tag) => emits('deleted_tag', deleted_tag)"
            @deleted_text="(deleted_text: any) => emits('deleted_text', deleted_text)"
            @deleted_notification="(deleted_notification: any) => emits('deleted_notification', deleted_notification)"
            @registered_kyou="(registered_kyou: Kyou) => { reload_plaing_timeis_view(); emits('registered_kyou', registered_kyou) }"
            @registered_tag="(registered_tag: Tag) => emits('registered_tag', registered_tag)"
            @registered_text="(registered_text: any) => emits('registered_text', registered_text)"
            @registered_notification="(registered_notification: any) => emits('registered_notification', registered_notification)"
            @updated_kyou="(updated_kyou: Kyou) => { reload_plaing_timeis_view(); emits('updated_kyou', updated_kyou) }"
            @updated_tag="(updated_tag: Tag) => emits('updated_tag', updated_tag)"
            @updated_text="(updated_text: any) => emits('updated_text', updated_text)"
            @updated_notification="(updated_notification: any) => emits('updated_notification', updated_notification)"
            @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
            @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
            @saved_kyou_by_kftl="(last_added_request_time: Date) => {
                plaing_timeis_view?.set_last_added_request_time(new Date(Math.max(last_added_request_time.getTime(), Date.now())))
                reload_plaing_timeis_view()
            }" ref="kftl_view" />
        <PlaingTimeisView :application_config="application_config" :gkill_api="gkill_api"
            :app_content_height="(app_content_height.valueOf() / 2) + 4" :app_content_width="app_content_width"
            @deleted_kyou="(deleted_kyou: Kyou) => { reload_plaing_timeis_view(); emits('deleted_kyou', deleted_kyou) }"
            @deleted_tag="(deleted_tag: Tag) => emits('deleted_tag', deleted_tag)"
            @deleted_text="(deleted_text: any) => emits('deleted_text', deleted_text)"
            @deleted_notification="(deleted_notification: any) => emits('deleted_notification', deleted_notification)"
            @registered_kyou="(registered_kyou: Kyou) => { reload_plaing_timeis_view(); emits('registered_kyou', registered_kyou) }"
            @registered_tag="(registered_tag: Tag) => emits('registered_tag', registered_tag)"
            @registered_text="(registered_text: any) => emits('registered_text', registered_text)"
            @registered_notification="(registered_notification: any) => emits('registered_notification', registered_notification)"
            @updated_kyou="(updated_kyou: Kyou) => { reload_plaing_timeis_view(); emits('updated_kyou', updated_kyou) }"
            @updated_tag="(updated_tag: Tag) => emits('updated_tag', updated_tag)"
            @updated_text="(updated_text: any) => emits('updated_text', updated_text)"
            @updated_notification="(updated_notification: any) => emits('updated_notification', updated_notification)"
            @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
            @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
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
import { useMkflView } from '@/classes/use-mkfl-view'

const plaing_timeis_view = ref<InstanceType<typeof PlaingTimeisView> | null>(null);

defineProps<MKFLProps>()
const emits = defineEmits<MKFLViewEmits>()

const {
    reload_plaing_timeis_view,
} = useMkflView({ emits, plaing_timeis_view })
</script>
