<template>
    <v-card>
        <v-card-title>
            {{ $t("SHARE_MI_TASK_TITLE") }}
        </v-card-title>
        <div>{{ $t("SHARE_MI_TASK_MESSAGE") }}</div>
        <v-text-field v-model="share_title" :label="$t('SHARE_MI_TASK_TITLE_TITLE')" />
        <v-checkbox v-model="share_time_only" :label="$t('SHARE_MI_TASK_TIME_ONLY_TITLE')" />
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col class="pa-0 ma-0" cols="auto">
                <v-btn dark color="primary" @click="share()">{{ $t("OK_TITLE") }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { ShareTaskListViewEmits } from './share-task-list-view-emits'
import type { ShareTaskListViewProps } from './share-task-list-view-props'
import { AddShareMiTaskListInfoRequest } from '@/classes/api/req_res/add-share-mi-task-list-info-request';
import { ShareMiTaskListInfo } from '@/classes/datas/share-mi-task-list-info';
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request';
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const props = defineProps<ShareTaskListViewProps>()
const emits = defineEmits<ShareTaskListViewEmits>()

const share_title: Ref<string> = ref("")
const share_time_only: Ref<boolean> = ref(false)

async function share(): Promise<void> {
    const gkill_req = new GetGkillInfoRequest()
    const gkill_res = await props.gkill_api.get_gkill_info(gkill_req)
    if (gkill_res.errors && gkill_res.errors.length !== 0) {
        emits('received_errors', gkill_res.errors)
        return
    }

    const share_mi_task_list_info = new ShareMiTaskListInfo()
    share_mi_task_list_info.share_id = props.gkill_api.generate_uuid()
    share_mi_task_list_info.user_id = gkill_res.user_id
    share_mi_task_list_info.device = gkill_res.device
    share_mi_task_list_info.find_query_json = props.find_kyou_query
    share_mi_task_list_info.is_share_detail = !share_time_only.value
    share_mi_task_list_info.share_title = share_title.value

    const req = new AddShareMiTaskListInfoRequest()
    req.share_mi_task_list_info = share_mi_task_list_info
    const res = await props.gkill_api.add_share_mi_task_list_info(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits('regestered_share_mi_task_list_info', res.share_mi_task_list_info)
    emits('requested_close_dialog')
}
</script>
