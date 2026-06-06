import { ref, computed, watch } from 'vue'
import type { RykvDialogKind, RykvDialogPayload } from '@/pages/views/rykv-dialog-kind'
import type { IDFKyouProps } from '@/pages/views/idf-kyou-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import type { Kyou } from '@/classes/datas/kyou'
import type { Tag } from '@/classes/datas/tag'
import type { Text } from '@/classes/datas/text'
import type { Notification } from '@/classes/datas/notification'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { ComponentRef } from '@/classes/component-ref'

const TEXT_EXTENSIONS = new Set([
    'txt', 'md', 'markdown', 'csv', 'tsv',
    'json', 'jsonl', 'xml', 'yaml', 'yml', 'toml', 'ini', 'cfg', 'conf', 'env',
    'html', 'htm', 'css', 'js', 'ts', 'jsx', 'tsx', 'vue', 'svelte',
    'py', 'go', 'java', 'c', 'cpp', 'h', 'hpp', 'cs', 'rb', 'php',
    'swift', 'kt', 'rs', 'scala', 'r', 'sh', 'bash', 'zsh', 'bat', 'ps1',
    'sql', 'graphql', 'proto', 'log', 'diff', 'patch',
])

const MAX_TEXT_LENGTH = 10000

function get_extension(filename: string): string {
    const dot = filename.lastIndexOf('.')
    return dot >= 0 ? filename.slice(dot + 1).toLowerCase() : ''
}

export function useIDFKyouView(options: {
    props: IDFKyouProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── Template refs ──
    const context_menu = ref<ComponentRef | null>(null)

    // ── Text file preview ──
    const text_content = ref<string | null>(null)
    const text_loading = ref(false)

    const is_text = computed((): boolean => {
        const fname = props.kyou.typed_idf_kyou?.file_name ?? ''
        const idf = props.kyou.typed_idf_kyou
        if (!idf || idf.is_image || idf.is_video || idf.is_audio || idf.is_zip) return false
        return TEXT_EXTENSIONS.has(get_extension(fname))
    })

    async function load_text_content(): Promise<void> {
        const url = props.kyou.typed_idf_kyou?.file_url
        if (!url || !is_text.value) return
        text_loading.value = true
        text_content.value = null
        try {
            const res = await fetch(url)
            if (!res.ok) return
            const raw = await res.text()
            text_content.value = raw.length > MAX_TEXT_LENGTH
                ? raw.slice(0, MAX_TEXT_LENGTH) + '\n…'
                : raw
        } catch {
            // 取得失敗時はリンク表示にフォールバックするため無視
        } finally {
            text_loading.value = false
        }
    }

    watch(
        () => props.kyou.typed_idf_kyou?.file_url,
        () => { if (is_text.value) load_text_content() },
        { immediate: true },
    )

    // ── Business logic ──
    function show_context_menu(e: PointerEvent): void {
        if (props.enable_context_menu) {
            context_menu.value?.show(e)
        }
    }

    function open_link(): void {
        const url = props.kyou.typed_idf_kyou?.file_url
        if (url) {
            window.open(url, "_blank")
        }
    }

    function buildMediaUrl(fileUrl: string, isVideoThumb: boolean): string {
        let is_added_query = false
        if (isVideoThumb) {
            if (!is_added_query) {
                fileUrl += "?"
            } else {
                fileUrl += "&"
            }
            fileUrl += "is_video=true"
            is_added_query = true

            if (!is_added_query) {
                fileUrl += "?"
            } else {
                fileUrl += "&"
            }
            fileUrl += "thumb=400x400"
            is_added_query = true
            return fileUrl
        }
        if (props.is_image_request_to_thumb_size) {
            if (!is_added_query) {
                fileUrl += "?"
            } else {
                fileUrl += "&"
            }
            fileUrl += "thumb=400x400"
            is_added_query = true
        }
        return fileUrl
    }

    // ── Event relay objects ──
    const crudRelayHandlers = {
        'deleted_kyou': (kyou: Kyou) => emits('deleted_kyou', kyou),
        'deleted_tag': (tag: Tag) => emits('deleted_tag', tag),
        'deleted_text': (text: Text) => emits('deleted_text', text),
        'deleted_notification': (notification: Notification) => emits('deleted_notification', notification),
        'registered_kyou': (kyou: Kyou) => emits('registered_kyou', kyou),
        'registered_tag': (tag: Tag) => emits('registered_tag', tag),
        'registered_text': (text: Text) => emits('registered_text', text),
        'registered_notification': (notification: Notification) => emits('registered_notification', notification),
        'updated_kyou': (kyou: Kyou) => emits('updated_kyou', kyou),
        'updated_tag': (tag: Tag) => emits('updated_tag', tag),
        'updated_text': (text: Text) => emits('updated_text', text),
        'updated_notification': (notification: Notification) => emits('updated_notification', notification),
        'received_errors': (errors: Array<GkillError>) => emits('received_errors', errors),
        'received_messages': (messages: Array<GkillMessage>) => emits('received_messages', messages),
        'requested_reload_kyou': (kyou: Kyou) => emits('requested_reload_kyou', kyou),
        'requested_reload_list': () => emits('requested_reload_list'),
        'requested_update_check_kyous': (kyous: Array<Kyou>, checked: boolean) => emits('requested_update_check_kyous', kyous, checked),
        'requested_open_rykv_dialog': (kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => emits('requested_open_rykv_dialog', kind, kyou, payload),
    }

    // ── Return ──
    return {
        // Template refs
        context_menu,

        // Text preview
        is_text,
        text_content,
        text_loading,

        // Business logic
        show_context_menu,
        open_link,
        buildMediaUrl,

        // Event relay objects
        crudRelayHandlers,
    }
}
