<template>
    <v-card>
        <v-card-title>
            {{ $t("SHARED_KYOU_LINK_LIST_TITLE") }}
        </v-card-title>
        <v-text-field v-model="share_title" :label="$t('SHARE_KYOU_TITLE_TITLE')" />
        <v-select v-model="view_type" :items="view_types" :label="$t('SHARE_KYOU_VIEW_TYPE')" />
        <v-checkbox v-model="is_share_time_only" :label="$t('SHARE_KYOU_SHARE_TIME_ONLY')" />
        <v-checkbox v-model="is_share_with_tags" :label="$t('SHARE_KYOU_SHARE_WITH_TAGS')" />
        <v-checkbox v-model="is_share_with_texts" :label="$t('SHARE_KYOU_SHARE_WITH_TEXTS')" />
        <v-checkbox v-model="is_share_with_timeiss" :label="$t('SHARE_KYOU_SHARE_WITH_TIMEISS')" />
        <v-checkbox v-model="is_share_with_locations" :label="$t('SHARE_KYOU_SHARE_WITH_LOCATIONS')" />
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
import type { ShareKyousLinkViewEmits } from './share-kyou-link-view-emits'
import type { ShareKyousLinkViewProps } from './share-kyou-link-view-props'
import { ShareKyousInfo } from '@/classes/datas/share-kyous-info';
import { GkillMessage } from '@/classes/api/gkill-message';
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request';
import { UpdateShareKyouListInfoRequest } from '@/classes/api/req_res/update-share-kyou-list-info-request';
import { GkillErrorCodes } from '@/classes/api/message/gkill_error';
import { i18n } from '@/i18n';

const props = defineProps<ShareKyousLinkViewProps>()
const emits = defineEmits<ShareKyousLinkViewEmits>()

const cloned_share_kyou_info: Ref<ShareKyousInfo> = ref(props.share_kyou_list_info.clone())

const view_types = ref([
    {
        title: i18n.global.t("RYKV_APP_NAME"),
        value: "rykv"
    },
    {
        title: i18n.global.t("MI_APP_NAME"),
        value: "mi"
    },
])

const share_title: Ref<string> = ref(cloned_share_kyou_info.value.share_title)
const view_type = ref(cloned_share_kyou_info.value.view_type)
const is_share_time_only: Ref<boolean> = ref(cloned_share_kyou_info.value.is_share_time_only)
const is_share_with_tags: Ref<boolean> = ref(cloned_share_kyou_info.value.is_share_with_tags)
const is_share_with_texts: Ref<boolean> = ref(cloned_share_kyou_info.value.is_share_with_texts)
const is_share_with_timeiss: Ref<boolean> = ref(cloned_share_kyou_info.value.is_share_with_timeiss)
const is_share_with_locations: Ref<boolean> = ref(cloned_share_kyou_info.value.is_share_with_locations)

const lan_share_url = computed(() => {
    return location.protocol + "//" + location.host + "/shared_page?share_id=" + cloned_share_kyou_info.value.share_id
})
const over_lan_share_url = computed(() => {
    return location.protocol + "//" + location.host + "/shared_page?share_id=" + cloned_share_kyou_info.value.share_id
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

    cloned_share_kyou_info.value.user_id = gkill_res.user_id
    cloned_share_kyou_info.value.device = gkill_res.device
    cloned_share_kyou_info.value.share_title = share_title.value
    cloned_share_kyou_info.value.view_type = view_type.value
    cloned_share_kyou_info.value.is_share_time_only = is_share_time_only.value
    cloned_share_kyou_info.value.is_share_with_tags = is_share_with_tags.value
    cloned_share_kyou_info.value.is_share_with_texts = is_share_with_texts.value
    cloned_share_kyou_info.value.is_share_with_timeiss = is_share_with_timeiss.value
    cloned_share_kyou_info.value.is_share_with_locations = is_share_with_locations.value

    const req = new UpdateShareKyouListInfoRequest()
    req.share_kyou_list_info = cloned_share_kyou_info.value
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
