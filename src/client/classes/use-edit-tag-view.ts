import { i18n } from '@/i18n'
import { computed, type Ref, ref } from 'vue'
import type { EditTagViewProps } from '@/pages/views/edit-tag-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { UpdateTagRequest } from '@/classes/api/req_res/update-tag-request'
import { GkillError } from '@/classes/api/gkill-error'
import type { Tag } from '@/classes/datas/tag'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { Kyou } from '@/classes/datas/kyou'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

export function useEditTagView(options: {
    props: EditTagViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const is_loading = ref(true)
    const is_requested_submit = ref(false)
    const is_busy = computed(() => is_loading.value || is_requested_submit.value)
    const cloned_kyou: Ref<Kyou> = ref(props.kyou.clone())
    const cloned_tag: Ref<Tag> = ref(props.tag.clone())
    const tag_name: Ref<string> = ref(props.tag.tag)
    const show_kyou: Ref<boolean> = ref(false)

    // ── Business logic ──
    async function load(): Promise<void> {
        try {
            is_loading.value = true
            cloned_kyou.value = props.kyou.clone()
            await cloned_kyou.value.load_typed_datas()
            cloned_tag.value = props.tag.clone()
            tag_name.value = cloned_tag.value.tag
        } finally {
            is_loading.value = false
        }
    }

    async function save(): Promise<void> {
        try {
            is_requested_submit.value = true
            // 値がなかったらエラーメッセージを出力する
            if (tag_name.value === "") {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.tag_is_blank
                error.error_message = i18n.global.t("TAG_IS_BLANK_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 更新がなかったらエラーメッセージを出力する
            if (cloned_tag.value.tag === tag_name.value) {
                const error = new GkillError()
                error.error_code = GkillErrorCodes.tag_is_no_update
                error.error_message = i18n.global.t("TAG_IS_NO_UPDATE_MESSAGE")
                const errors = new Array<GkillError>()
                errors.push(error)
                emits('received_errors', errors)
                return
            }

            // 更新後タグ情報を用意する
            const updated_tag = cloned_tag.value.clone()
            updated_tag.tag = tag_name.value
            updated_tag.update_app = "gkill"
            updated_tag.update_device = props.application_config.device
            updated_tag.update_time = new Date(Date.now())
            updated_tag.update_user = props.application_config.user_id

            // 更新リクエストを飛ばす。
            // target_id側のキャッシュ破棄はGkillAPI.update_tagが応答受領後に行う
            await delete_gkill_kyou_cache(updated_tag.id)
            const req = new UpdateTagRequest()
            req.tag = updated_tag
            const res = await props.gkill_api.update_tag(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length !== 0) {
                emits('received_messages', res.messages)
            }
            emits("updated_tag", res.updated_tag)
            emits('requested_reload_kyou', props.kyou)
            emits('requested_close_dialog')
            return
        } finally {
            is_requested_submit.value = false
        }
    }

    // ── Event relay objects ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

    // ── Init calls ──
    load()

    // ── Return ──
    return {
        // State
        is_loading,
        is_requested_submit,
        is_busy,
        cloned_kyou,
        cloned_tag,
        tag_name,
        show_kyou,

        // Business logic
        save,

        // Event relay objects
        crudRelayHandlers,
    }
}

