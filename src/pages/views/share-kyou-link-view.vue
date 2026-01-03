<template>
    <v-card>
        <v-card-title>
            {{ i18n.global.t("SHARED_KYOU_LINK_LIST_TITLE") }}
        </v-card-title>
        <v-text-field v-model="share_title" :label="i18n.global.t('SHARE_KYOU_TITLE_TITLE')" />
        <v-select v-model="view_type" :items="view_types" :label="i18n.global.t('SHARE_KYOU_VIEW_TYPE')" />
        <v-checkbox v-model="is_share_time_only" :label="i18n.global.t('SHARE_KYOU_SHARE_TIME_ONLY')" />
        <v-checkbox v-model="is_share_with_tags" :label="i18n.global.t('SHARE_KYOU_SHARE_WITH_TAGS')" />
        <v-checkbox v-model="is_share_with_texts" :label="i18n.global.t('SHARE_KYOU_SHARE_WITH_TEXTS')" />
        <v-checkbox v-model="is_share_with_timeiss" :label="i18n.global.t('SHARE_KYOU_SHARE_WITH_TIMEISS')" />
        <v-checkbox v-model="is_share_with_locations" :label="i18n.global.t('SHARE_KYOU_SHARE_WITH_LOCATIONS')" />
        <v-text-field v-model="local_share_url" :label="i18n.global.t('LOCAL_TITLE')" readonly
            @click="copy_local_share_kyou_link" @focus="$event.target.select()" />
        <v-text-field v-model="lan_share_url" :label="i18n.global.t('IN_LAN_TITLE')" readonly
            @click="copy_lan_share_kyou_link" @focus="$event.target.select()" />
        <v-text-field v-model="over_lan_share_url" :label="i18n.global.t('OVER_LAN_TITLE')" readonly
            @click="copy_over_lan_share_kyou_link" @focus="$event.target.select()" />
        <v-row class="pa-0 ma-0">
            <v-spacer />
            <v-col class="pa-0 ma-0" cols="auto">
                <v-btn dark color="primary" @click="update()">{{ i18n.global.t("UPDATE_TITLE") }}</v-btn>
            </v-col>
        </v-row>
    </v-card>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { computed, nextTick, type Ref, ref, watch } from 'vue';
import type { ShareKyousLinkViewEmits } from './share-kyou-link-view-emits'
import type { ShareKyousLinkViewProps } from './share-kyou-link-view-props'
import { ShareKyousInfo } from '@/classes/datas/share-kyous-info';
import { GkillMessage } from '@/classes/api/gkill-message';
import { UpdateShareKyouListInfoRequest } from '@/classes/api/req_res/update-share-kyou-list-info-request';
import { GkillErrorCodes } from '@/classes/api/message/gkill_error';

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

const local_share_url = computed(() => (location.protocol + "//" + location.host + "/shared_page?share_id=" + cloned_share_kyou_info.value.share_id))
const lan_share_url = computed(() => ((props.application_config.private_ip && props.application_config.private_ip !== "") ? location.protocol + "//" + props.application_config.private_ip + "/shared_page?share_id=" + cloned_share_kyou_info.value.share_id : ""))
const over_lan_share_url = computed(() => ((props.application_config.global_ip && props.application_config.global_ip !== "") ? location.protocol + "//" + props.application_config.global_ip + "/shared_page?share_id=" + cloned_share_kyou_info.value.share_id : ""))

async function copy_local_share_kyou_link(): Promise<void> {
    navigator.clipboard.writeText(local_share_url.value);
    const message = new GkillMessage()
    message.message_code = GkillErrorCodes.copied_local_share_kyou_link
    message.message = i18n.global.t("COPIED_LOCAL_SHARE_KYOU_LINK_MESSAGE")
    emits('received_messages', [message])
}

async function copy_lan_share_kyou_link(): Promise<void> {
    navigator.clipboard.writeText(lan_share_url.value);
    const message = new GkillMessage()
    message.message_code = GkillErrorCodes.copied_lan_share_kyou_link
    message.message = i18n.global.t("COPIED_LAN_SHARE_KYOU_LINK_MESSAGE")
    emits('received_messages', [message])
}

async function copy_over_lan_share_kyou_link(): Promise<void> {
    navigator.clipboard.writeText(over_lan_share_url.value);
    const message = new GkillMessage()
    message.message_code = GkillErrorCodes.copied_over_lan_share_kyou_link
    message.message = i18n.global.t("COPIED_LAN_SHARE_KYOU_OVER_LINK_MESSAGE")
    emits('received_messages', [message])
}

async function update(): Promise<void> {
    cloned_share_kyou_info.value.user_id = props.application_config.user_id
    cloned_share_kyou_info.value.device = props.application_config.device
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
