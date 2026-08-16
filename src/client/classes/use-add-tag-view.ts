import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import { GkillError } from '@/classes/api/gkill-error'
import { GkillErrorCodes } from '@/classes/api/message/gkill_error'
import { useConfirmUnknownTag } from '@/classes/use-confirm-unknown-tag'
import { add_tags_to_target, parse_tag_names } from '@/classes/kyou-tags'
import type { AddTagViewProps } from '@/pages/views/add-tag-view-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { build_kyou_view_relay } from '@/classes/kyou-view-relay'

export function useAddTagView(options: {
    props: AddTagViewProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Confirm unknown tag ──
    const confirm_unknown_tag = useConfirmUnknownTag({ application_config: () => props.application_config })

    // ── State refs ──
    const is_requested_submit = ref(false)
    const show_kyou: Ref<boolean> = ref(false)
    const tag_name: Ref<string> = ref("")

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

            // TagStructに存在しないタグを検出したら、保存する前に確認を取る
            const tag_names = parse_tag_names(tag_name.value)
            const unknown_tags = confirm_unknown_tag.collect_unknown_tags(tag_names)
            if (unknown_tags.length !== 0) {
                confirm_unknown_tag.open_confirm(unknown_tags)
                return
            }

            await execute_save()
        } finally {
            is_requested_submit.value = false
        }
    }

    function cancel_save(): void {
        confirm_unknown_tag.close_confirm()
    }

    async function confirm_save(): Promise<void> {
        confirm_unknown_tag.close_confirm()
        await execute_save()
    }

    async function execute_save(): Promise<void> {
        try {
            is_requested_submit.value = true
            // 確認ダイアログは非モーダルなので、確認中に入力を書き換えられる。取り直す
            const result = await add_tags_to_target(props.gkill_api, props.application_config, props.kyou.id, parse_tag_names(tag_name.value))
            if (result.messages.length !== 0) {
                emits('received_messages', result.messages)
            }
            if (result.errors.length !== 0) {
                emits('received_errors', result.errors)
                return
            }
            result.added_tags.forEach(added_tag => emits('registered_tag', added_tag))
            emits('requested_reload_kyou', props.kyou)
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
        // Template refs
        confirm_unknown_tag_dialog: confirm_unknown_tag.confirm_unknown_tag_dialog,

        // Confirm unknown tag
        unknown_tags: confirm_unknown_tag.unknown_tags,
        cancel_save,
        confirm_save,

        // State
        is_requested_submit,
        show_kyou,
        tag_name,

        // Business logic / template handlers
        save,

        // Event relay objects
        crudRelayHandlers,
    }
}
