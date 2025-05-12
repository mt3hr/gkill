<template>
    <v-card>
        <v-card-title>
            {{ $t("SHARED_MI_TASK_LINK_LIST_TITLE") }}
        </v-card-title>
        <v-text-field v-model="cloned_share_kyou_list_info.share_title" :label="$t('SHARE_MI_TASK_TITLE_TITLE')" />
        <v-checkbox v-model="share_time_only" :label="$t('SHARE_MI_TASK_TIME_ONLY_TITLE')" />
        <v-text-field v-model="lan_share_url" :label="$t('IN_LAN_TITLE')" readonly @click="copy_lan_share_kyou_link"
            @focus="$event.target.select()" />
        <v-text-field v-model="over_lan_share_url" :label="$t('OVER_LAN_TITLE')" readonly
            @click="copy_over_lan_share_kyou_link" @focus="$event.target.select()" />
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col class="pa-0 ma-0" cols="auto">
                <v-btn dark color="primary" @click="update()">{{ $t("UPDATE_TITLE") }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import { computed, type Ref, ref, watch } from 'vue';
import type { ShareKyousListLinkViewEmits } from './share-task-list-link-view-emits'
import type { ShareKyousListLinkViewProps } from './share-task-list-link-view-props'
import { ShareKyouListInfo } from '@/classes/datas/share-kyou-list-info';
import { GkillMessage } from '@/classes/api/gkill-message';
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request';
import { UpdateShareKyouListInfoRequest } from '@/classes/api/req_res/update-share-kyou-list-info-request';
import { GkillErrorCodes } from '@/classes/api/message/gkill_error';

const props = defineProps<ShareKyousListLinkViewProps>()
const emits = defineEmits<ShareKyousListLinkViewEmits>()

const cloned_share_kyou_list_info: Ref<ShareKyouListInfo> = ref(props.share_kyou_list_info.clone())
const share_time_only = ref(!props.share_kyou_list_info.is_share_detail)
watch(() => props.share_kyou_list_info, () => {
    cloned_share_kyou_list_info.value = props.share_kyou_list_info.clone()
    share_time_only.value = !cloned_share_kyou_list_info.value.is_share_detail
})

const lan_share_url = computed(() => {
    return location.protocol + "//" + location.host + "/shared_kyou?share_id=" + cloned_share_kyou_list_info.value.share_id
})
const over_lan_share_url = computed(() => {
    return location.protocol + "//" + location.host + "/shared_kyou?share_id=" + cloned_share_kyou_list_info.value.share_id
})

function copy_lan_share_kyou_link(): void {
    navigator.clipboard.writeText(lan_share_url.value);
    const message = new GkillMessage()
    message.message_code = GkillErrorCodes.copied_lan_share_kyou_link
    message.message = "コピーしました"
    emits('received_messages', [message])
}

function copy_over_lan_share_kyou_link(): void {
    navigator.clipboard.writeText(over_lan_share_url.value);
    const message = new GkillMessage()
    message.message_code = GkillErrorCodes.copied_over_lan_share_kyou_link
    message.message = "コピーしました"
    emits('received_messages', [message])
}

async function update(): Promise<void> {
    const gkill_req = new GetGkillInfoRequest()
    const gkill_res = await props.gkill_api.get_gkill_info(gkill_req)
    if (gkill_res.errors && gkill_res.errors.length !== 0) {
        emits('received_errors', gkill_res.errors)
        return
    }

    cloned_share_kyou_list_info.value.is_share_detail = !share_time_only.value
    const req = new UpdateShareKyouListInfoRequest()
    req.share_kyou_list_info = cloned_share_kyou_list_info.value
    const res = await props.gkill_api.update_share_kyou_list_info(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits('updated_share_kyou_list_info', res.share_kyou_list_info)
    emits('requested_close_dialog')
}
</script>
