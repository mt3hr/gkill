<template>
    <v-card>
        <v-card-title>
            タスク一覧共有リンク
        </v-card-title>
        <v-text-field v-model="cloned_share_mi_task_list_info.share_title" label="タイトル" />
        <v-checkbox v-model="share_time_only" label="タスク有無と時刻のみ共有" />
        <v-text-field v-model="lan_share_url" label="ローカル" readonly @click="copy_lan_share_mi_link"
            @focus="$event.target.select()" />
        <v-text-field v-model="over_lan_share_url" label="それ以外" readonly @click="copy_over_lan_share_mi_link"
            @focus="$event.target.select()" />
        <v-row>
            <v-spacer />
            <v-col>
                <v-btn @click="update()">更新</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import { computed, type Ref, ref, watch } from 'vue';
import type { ShareTaskListLinkViewEmits } from './share-task-list-link-view-emits'
import type { ShareTaskListLinkViewProps } from './share-task-list-link-view-props'
import { ShareMiTaskListInfo } from '@/classes/datas/share-mi-task-list-info';
import { GkillMessage } from '@/classes/api/gkill-message';
import { GkillAPI } from '@/classes/api/gkill-api';
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request';
import { UpdateShareMiTaskListInfoRequest } from '@/classes/api/req_res/update-share-mi-task-list-info-request';

const props = defineProps<ShareTaskListLinkViewProps>()
const emits = defineEmits<ShareTaskListLinkViewEmits>()

const cloned_share_mi_task_list_info: Ref<ShareMiTaskListInfo> = ref(props.share_mi_task_list_info.clone())
const share_time_only = ref(!props.share_mi_task_list_info.is_share_detail)
watch(() => props.share_mi_task_list_info, () => {
    cloned_share_mi_task_list_info.value = props.share_mi_task_list_info.clone()
    share_time_only.value = !cloned_share_mi_task_list_info.value.is_share_detail
})

const lan_share_url = computed(() => {
    return location.protocol + "//" + location.host + "/shared_mi?share_id=" + cloned_share_mi_task_list_info.value.share_id
})
const over_lan_share_url = computed(() => {
    return location.protocol + "//" + location.host + "/shared_mi?share_id=" + cloned_share_mi_task_list_info.value.share_id
})

function copy_lan_share_mi_link(): void {
    navigator.clipboard.writeText(lan_share_url.value);
    const message = new GkillMessage()
    message.message_code = "//TODO"
    message.message = "コピーしました"
    emits('received_messages', [message])
}

function copy_over_lan_share_mi_link(): void {
    navigator.clipboard.writeText(over_lan_share_url.value);
    const message = new GkillMessage()
    message.message_code = "//TODO"
    message.message = "コピーしました"
    emits('received_messages', [message])
}

async function update(): Promise<void> {
    const gkill_req = new GetGkillInfoRequest()
    gkill_req.session_id = props.gkill_api.get_session_id()
    const gkill_res = await props.gkill_api.get_gkill_info(gkill_req)
    if (gkill_res.errors && gkill_res.errors.length !== 0) {
        emits('received_errors', gkill_res.errors)
        return
    }

    cloned_share_mi_task_list_info.value.is_share_detail = !share_time_only.value
    const req = new UpdateShareMiTaskListInfoRequest()
    req.session_id = props.gkill_api.get_session_id()
    req.share_mi_task_list_info = cloned_share_mi_task_list_info.value
    const res = await props.gkill_api.update_share_mi_task_list_info(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits('updated_share_mi_task_list_info', res.share_mi_task_list_info)
    emits('requested_close_dialog')
}
</script>
