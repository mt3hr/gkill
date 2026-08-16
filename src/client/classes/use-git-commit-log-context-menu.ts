import { i18n } from '@/i18n'
import { ref, type Ref } from 'vue'
import { useContextMenuPosition } from '@/classes/use-context-menu-position'
import { GkillMessage } from '@/classes/api/gkill-message'
import { OpenDirectoryRequest } from '@/classes/api/req_res/open-directory-request'
import { OpenFileRequest } from '@/classes/api/req_res/open-file-request'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'
import { copy_kyou_content } from '@/classes/kyou-content-text'
import { add_tags_to_target, parse_tag_names } from '@/classes/kyou-tags'
import type { GitCommitLogContextMenuProps } from '@/pages/views/git-commit-log-context-menu-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

export function useGitCommitLogContextMenu(options: {
    props: GitCommitLogContextMenuProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const is_requested_submit = ref(false)
    const { is_show, menu_target, open_at } = useContextMenuPosition()
    const tag_history: Ref<string[]> = ref([])

    // ── Business logic ──
    async function show(e: PointerEvent): Promise<void> {
        tag_history.value = props.gkill_api.get_saved_tag_history()
        open_at(e)
    }

    async function copy_content(): Promise<void> {
        const res = await copy_kyou_content(props.kyou, props.gkill_api)
        if (res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }
    }

    async function copy_id(): Promise<void> {
        navigator.clipboard.writeText(props.kyou.id)
        const message = new GkillMessage()
        message.message_code = GkillMessageCodes.copied_git_commit_log_id
        message.message = i18n.global.t("COPIED_ID_MESSAGE")
        const messages = new Array<GkillMessage>()
        messages.push(message)
        emits('received_messages', messages)
    }

    async function show_add_tag_dialog(): Promise<void> {
        emits('requested_open_rykv_dialog', 'add_tag', props.kyou)
    }

    async function show_add_text_dialog(): Promise<void> {
        emits('requested_open_rykv_dialog', 'add_text', props.kyou)
    }

    async function show_add_notification_dialog(): Promise<void> {
        emits('requested_open_rykv_dialog', 'add_notification', props.kyou)
    }

    async function show_confirm_rekyou_dialog(): Promise<void> {
        emits('requested_open_rykv_dialog', 'confirm_re_kyou', props.kyou)
    }

    async function show_add_mi_re_kyou_dialog(): Promise<void> {
        emits('requested_open_rykv_dialog', 'add_mi_re_kyou', props.kyou)
    }

    async function open_folder(): Promise<void> {
        const req = new OpenDirectoryRequest()
        req.target_id = props.kyou.id
        const res = await props.gkill_api.open_directory(req)
        if (res.errors && res.errors.length > 0) {
            emits('received_errors', res.errors)
        }
        if (res.messages && res.messages.length > 0) {
            emits('received_messages', res.messages)
        }
    }

    async function open_file(): Promise<void> {
        const req = new OpenFileRequest()
        req.target_id = props.kyou.id
        const res = await props.gkill_api.open_file(req)
        if (res.errors && res.errors.length > 0) {
            emits('received_errors', res.errors)
        }
        if (res.messages && res.messages.length > 0) {
            emits('received_messages', res.messages)
        }
    }

    // タグ履歴からの付与はそのままサーバ更新に繋がる。連打で同じタグが重なるのを防ぐ
    async function add_tag_from_history(tag_value: string): Promise<void> {
        if (is_requested_submit.value) {
            return
        }
        is_requested_submit.value = true
        try {
            await execute_add_tag_from_history(tag_value)
        } finally {
            is_requested_submit.value = false
        }
    }

    async function execute_add_tag_from_history(tag_value: string): Promise<void> {
        // 追加の手順（id採番・キャッシュ破棄・履歴への積み直し）は classes/kyou-tags.ts に集約してある
        const result = await add_tags_to_target(props.gkill_api, props.application_config, props.kyou.id, parse_tag_names(tag_value))
        if (result.messages.length !== 0) {
            emits('received_messages', result.messages)
        }
        if (result.errors.length !== 0) {
            emits('received_errors', result.errors)
            return
        }
        result.added_tags.forEach(added_tag => emits('registered_tag', added_tag))
        emits('requested_reload_kyou', props.kyou)
    }

    // ── Return ──
    return {
        // State
        is_show,
        is_requested_submit,
        tag_history,
        menu_target,

        // Business logic
        show,
        copy_content,
        copy_id,
        show_add_tag_dialog,
        show_add_text_dialog,
        show_add_notification_dialog,
        show_confirm_rekyou_dialog,
        show_add_mi_re_kyou_dialog,
        open_folder,
        open_file,
        add_tag_from_history,
    }
}
