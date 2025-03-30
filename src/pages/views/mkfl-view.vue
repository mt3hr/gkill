<template>
    <div>
        <kftlView :app_content_height="app_content_height.valueOf() / 2" :app_content_width="app_content_width"
            :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" @saved_kyou_by_kftl="(last_added_request_time: Date) => {
                plaing_timeis_view?.set_last_added_request_time(last_added_request_time)
                reload_plaing_timeis_view()
            }" />
        <PlaingTimeisView :application_config="application_config" :gkill_api="gkill_api"
            :app_content_height="app_content_height.valueOf() / 2" :app_content_width="app_content_width"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" ref="plaing_timeis_view" />
    </div>
</template>
<script lang="ts" setup>
import { ref } from 'vue';
import kftlView from './kftl-view.vue'
import PlaingTimeisView from './plaing-timeis-view.vue'
import type { MKFLProps } from './mkfl-view-props';
import type { MKFLViewEmits } from './mkfl-view-emits';
const plaing_timeis_view = ref<InstanceType<typeof PlaingTimeisView> | null>(null);

defineProps<MKFLProps>()
const emits = defineEmits<MKFLViewEmits>()

async function reload_plaing_timeis_view(): Promise<void> {
    plaing_timeis_view.value?.reload_list(false)
}
</script>