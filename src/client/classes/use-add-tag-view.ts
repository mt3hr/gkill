import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import { Tag } from '@/classes/datas/tag'
import { tag_exists_in_tag_struct } from '@/classes/tag-struct'
import { AddTagRequest } from '@/classes/api/req_res/add-tag-request'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import { useFloatingDialog } from '@/classes/use-floating-dialog'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import type { AddTagViewProps } from '@/pages/views/add-tag-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

export function useAddTagView(options: {
    props: AddTagViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const is_requested_submit = ref(false)
    const show_kyou: Ref<boolean> = ref(false)
    const tag_name: Ref<string> = ref("")
    const show_confirm_unknown_tag_dialog: Ref<boolean> = ref(false)
    const unknown_tags: Ref<string[]> = ref([])

    // ── Dialog UI ──
    useDialogHistoryStack(show_confirm_unknown_tag_dialog)
    const confirm_dialog_ui = useFloatingDialog("confirm-unknown-tag-dialog", {
        centerMode: "always",
    })

    // ── Business logic ──
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

            // TagStructに存在しないタグを検出
            const tag_names = tag_name.value.split("、")
            const not_found = tag_names.filter(t => !tag_exists_in_tag_struct(t, props.application_config.tag_struct))
            if (not_found.length > 0) {
                unknown_tags.value = not_found
                show_confirm_unknown_tag_dialog.value = true
                return
            }

            await execute_save()
        } finally {
            is_requested_submit.value = false
        }
    }

    function cancel_save(): void {
        close_dialog_via_history(show_confirm_unknown_tag_dialog)
        unknown_tags.value = []
    }

    async function confirm_save(): Promise<void> {
        close_dialog_via_history(show_confirm_unknown_tag_dialog)
        unknown_tags.value = []
        await execute_save()
    }

    async function execute_save(): Promise<void> {
        try {
            is_requested_submit.value = true
            const tag_names = tag_name.value.split("、")
            for (let i = 0; i < tag_names.length; i++) {
                const tag = tag_names[i]
                // タグ情報を用意する
                const new_tag = new Tag()
                new_tag.tag = tag
                new_tag.id = props.gkill_api.generate_uuid()
                new_tag.is_deleted = false
                new_tag.target_id = props.kyou.id
                new_tag.related_time = new Date(Date.now())
                new_tag.create_app = "gkill"
                new_tag.create_device = props.application_config.device
                new_tag.create_time = new Date(Date.now())
                new_tag.create_user = props.application_config.user_id
                new_tag.update_app = "gkill"
                new_tag.update_device = props.application_config.device
                new_tag.update_time = new Date(Date.now())
                new_tag.update_user = props.application_config.user_id

                // 追加リクエストを飛ばす
                await delete_gkill_kyou_cache(new_tag.id)
                await delete_gkill_kyou_cache(new_tag.target_id)
                const req = new AddTagRequest()
                req.tag = new_tag
                const res = await props.gkill_api.add_tag(req)
                if (res.errors && res.errors.length !== 0) {
                    emits('received_errors', res.errors)
                    return
                }
                if (res.messages && res.messages.length !== 0) {
                    emits('received_messages', res.messages)
                }
                emits('registered_tag', res.added_tag)
            }
            emits('requested_reload_kyou', props.kyou)
            props.gkill_api.set_saved_last_added_tag(tag_name.value)
            props.gkill_api.push_tag_to_history(tag_name.value)
            emits('requested_close_dialog')
            return
        } finally {
            is_requested_submit.value = false
        }
    }

    // ── CRUD relay handlers ──
    const crudRelayHandlers = build_kyou_view_relay(emits)

    // ── Return ──
    return {
        // State
        is_requested_submit,
        show_kyou,
        tag_name,
        show_confirm_unknown_tag_dialog,
        unknown_tags,

        // Dialog UI
        confirm_dialog_ui,

        // Business logic / template handlers
        save,
        cancel_save,
        confirm_save,

        // Event relay objects
        crudRelayHandlers,
    }
}

