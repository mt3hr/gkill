'use strict'

import { type Ref, ref } from 'vue'
import { i18n } from '@/i18n'
import type { ShareKyousListViewProps } from '@/pages/views/share-kyou-view-props'
import type { ShareKyousListViewEmits } from '@/pages/views/share-kyou-view-emits'
import { ShareKyousInfo } from '@/classes/datas/share-kyous-info'
import { AddShareKyouListInfoRequest } from '@/classes/api/req_res/add-share-kyou-list-infos-request'

export function useShareKyouView(options: { props: ShareKyousListViewProps, emits: ShareKyousListViewEmits }) {
    const { props, emits } = options

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
        const query = props.find_kyou_query.clone()
        if (view_type.value === "mi") {
            query.use_rep_types = true
            query.rep_types = ["mi"]
        }

        const share_kyou_list_info = new ShareKyousInfo()
        share_kyou_list_info.share_id = props.gkill_api.generate_uuid()
        share_kyou_list_info.user_id = props.application_config.user_id
        share_kyou_list_info.device = props.application_config.device
        share_kyou_list_info.find_query_json = query
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

    return {
        view_types,
        share_title,
        view_type,
        is_share_time_only,
        is_share_with_tags,
        is_share_with_texts,
        is_share_with_timeiss,
        is_share_with_locations,
        share,
    }
}
