<template>
    <v-card>
        <v-card-title>
            {{ $t("SHARE_KYOU_TITLE") }}
        </v-card-title>
        <div>{{ $t("SHARE_KYOU_MESSAGE") }}</div>
        <v-text-field v-model="share_title" :label="$t('SHARE_KYOU_TITLE_TITLE')" item-title="title"
            item-value="value" />
        <v-select v-model="view_type" :items="view_types" :label="$t('SHARE_KYOU_VIEW_TYPE')" />
        <v-checkbox v-model="is_share_time_only" :label="$t('SHARE_KYOU_SHARE_TIME_ONLY')" />
        <v-checkbox v-model="is_share_with_tags" :label="$t('SHARE_KYOU_SHARE_WITH_TAGS')" />
        <v-checkbox v-model="is_share_with_texts" :label="$t('SHARE_KYOU_SHARE_WITH_TEXTS')" />
        <v-checkbox v-model="is_share_with_timeiss" :label="$t('SHARE_KYOU_SHARE_WITH_TIMEISS')" />
        <v-checkbox v-model="is_share_with_locations" :label="$t('SHARE_KYOU_SHARE_WITH_LOCATIONS')" />
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
import type { ShareKyousListViewEmits } from './share-kyou-view-emits'
import type { ShareKyousListViewProps } from './share-kyou-view-props'
import { AddShareKyouListInfoRequest } from '@/classes/api/req_res/add-share-kyou-list-infos-request';
import { ShareKyousInfo } from '@/classes/datas/share-kyous-info';
import { GetGkillInfoRequest } from '@/classes/api/req_res/get-gkill-info-request';
import { i18n } from '@/i18n';

const props = defineProps<ShareKyousListViewProps>()
const emits = defineEmits<ShareKyousListViewEmits>()

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

const share_title: Ref<string> = ref("")
const view_type = ref(view_types.value[0].value)
const is_share_time_only: Ref<boolean> = ref(false)
const is_share_with_tags: Ref<boolean> = ref(false)
const is_share_with_texts: Ref<boolean> = ref(false)
const is_share_with_timeiss: Ref<boolean> = ref(false)
const is_share_with_locations: Ref<boolean> = ref(false)

async function share(): Promise<void> {
    const gkill_req = new GetGkillInfoRequest()
    const gkill_res = await props.gkill_api.get_gkill_info(gkill_req)
    if (gkill_res.errors && gkill_res.errors.length !== 0) {
        emits('received_errors', gkill_res.errors)
        return
    }

    const share_kyou_list_info = new ShareKyousInfo()
    share_kyou_list_info.share_id = props.gkill_api.generate_uuid()
    share_kyou_list_info.user_id = gkill_res.user_id
    share_kyou_list_info.device = gkill_res.device
    share_kyou_list_info.find_query_json = props.find_kyou_query
    share_kyou_list_info.share_title = share_title.value
    share_kyou_list_info.view_type = view_type.value
    share_kyou_list_info.is_share_time_only = is_share_time_only.value
    share_kyou_list_info.is_share_with_tags = is_share_with_tags.value
    share_kyou_list_info.is_share_with_texts = is_share_with_texts.value
    share_kyou_list_info.is_share_with_timeiss = is_share_with_timeiss.value
    share_kyou_list_info.is_share_with_locations = is_share_with_locations.value

    const req = new AddShareKyouListInfoRequest()
    req.share_kyou_list_info = share_kyou_list_info
    const res = await props.gkill_api.add_share_kyou_list_info(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }
    emits('regestered_share_kyou_list_info', res.share_kyou_list_info)
    emits('requested_close_dialog')
}
</script>
