import { i18n } from '@/i18n'
import { computed, ref, type Ref } from 'vue'
import { GkillMessage } from '@/classes/api/gkill-message'
import { OpenDirectoryRequest } from '@/classes/api/req_res/open-directory-request'
import { OpenFileRequest } from '@/classes/api/req_res/open-file-request'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'
import { AddTagRequest } from '@/classes/api/req_res/add-tag-request'
import { Tag } from '@/classes/datas/tag'
import delete_gkill_kyou_cache from '@/classes/delete-gkill-cache'
import type { NlogContextMenuProps } from '@/pages/views/nlog-context-menu-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

export function useNlogContextMenu(options: {
    props: NlogContextMenuProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State refs ──
    const is_show: Ref<boolean> = ref(false)
    const position_x: Ref<Number> = ref(0)
    const position_y: Ref<Number> = ref(0)
    const tag_history: Ref<string[]> = ref([])

    // ── Computed ──
    const context_menu_style = computed(() => `{ position: absolute; left: ${Math.min(document.defaultView!.innerWidth - 130, position_x.value.valueOf())}px; top: ${Math.min(Math.max(50, document.defaultView!.innerHeight - ( + 8 + (48 * (8 + (tag_history.value.length > 0 ? 1 : 0) + (props.application_config.session_is_local ? 2 : 0))))), position_y.value.valueOf())}px; }`)

    // ── Business logic ──
    async function show(e: PointerEvent): Promise<void> {
        tag_history.value = props.gkill_api.get_saved_tag_history()
        position_x.value = e.clientX
        position_y.value = e.clientY
        is_show.value = true
    }

    async function copy_id(): Promise<void> {
        navigator.clipboard.writeText(props.kyou.id)
        const message = new GkillMessage()
        message.message_code = GkillMessageCodes.copied_nlog_id
        message.message = i18n.global.t("COPIED_KYOU_ID_MESSAGE")
        const messages = new Array<GkillMessage>()
        messages.push(message)
        emits('received_messages', messages)
    }

    async function show_edit_nlog_dialog(): Promise<void> {
        emits('requested_open_rykv_dialog', 'edit_nlog', props.kyou)
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

    async function show_confirm_delete_kyou_dialog(): Promise<void> {
        emits('requested_open_rykv_dialog', 'confirm_delete_kyou', props.kyou)
    }

    async function show_confirm_rekyou_dialog(): Promise<void> {
        emits('requested_open_rykv_dialog', 'confirm_re_kyou', props.kyou)
    }

    async function show_kyou_histories_dialog(): Promise<void> {
        emits('requested_open_rykv_dialog', 'kyou_histories', props.kyou)
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

    async function add_tag_from_history(tag_value: string): Promise<void> {
        is_show.value = false
        props.gkill_api.push_tag_to_history(tag_value)
        const tag_names = tag_value.split("\u3001")
        for (let i = 0; i < tag_names.length; i++) {
            const tag = tag_names[i]
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
            emits('requested_reload_kyou', props.kyou)
        }
        return
    }

    // ── Return ──
    return {
        // State
        is_show,
        tag_history,
        context_menu_style,

        // Business logic
        show,
        copy_id,
        show_edit_nlog_dialog,
        show_add_tag_dialog,
        show_add_text_dialog,
        show_add_notification_dialog,
        show_confirm_delete_kyou_dialog,
        show_confirm_rekyou_dialog,
        show_kyou_histories_dialog,
        open_folder,
        open_file,
        add_tag_from_history,
    }
}
