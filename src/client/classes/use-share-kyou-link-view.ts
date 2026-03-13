'use strict'

import { computed, ref, type Ref } from 'vue'
import { i18n } from '@/i18n'
import type { ShareKyousLinkViewEmits } from '@/pages/views/share-kyou-link-view-emits'
import type { ShareKyousLinkViewProps } from '@/pages/views/share-kyou-link-view-props'
import { ShareKyousInfo } from '@/classes/datas/share-kyous-info'
import { GkillMessage } from '@/classes/api/gkill-message'
import { UpdateShareKyouListInfoRequest } from '@/classes/api/req_res/update-share-kyou-list-info-request'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'

export function useShareKyouLinkView(options: {
    props: ShareKyousLinkViewProps
    emits: ShareKyousLinkViewEmits
}) {
    const { props, emits } = options

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

    return {
        cloned_share_kyou_info,
        view_types,
        share_title,
        view_type,
        is_share_time_only,
        is_share_with_tags,
        is_share_with_texts,
        is_share_with_timeiss,
        is_share_with_locations,
        local_share_url,
        lan_share_url,
        over_lan_share_url,
        copy_local_share_kyou_link,
        copy_lan_share_kyou_link,
        copy_over_lan_share_kyou_link,
        update,
    }
}
