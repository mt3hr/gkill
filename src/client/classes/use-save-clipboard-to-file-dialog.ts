'use strict'

import { nextTick, onBeforeUnmount, onMounted, type Ref, ref } from 'vue'
import type { SaveClipboardToFileDialogProps } from '@/pages/dialogs/save-clipboard-to-file-dialog-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import { FileData } from '@/classes/api/file-data'
import { UploadFilesRequest } from '@/classes/api/req_res/upload-files-request'
import { FileUploadConflictBehavior } from '@/classes/api/req_res/file-upload-conflict-behavior'
import { GetRepositoriesRequest } from '@/classes/api/req_res/get-repositories-request'
import type { Repository } from '@/classes/datas/config/repository'

const MIME_TO_EXT: Record<string, string> = {
    'image/png': 'png',
    'image/jpeg': 'jpg',
    'image/gif': 'gif',
    'image/webp': 'webp',
    'image/svg+xml': 'svg',
    'image/bmp': 'bmp',
    'application/pdf': 'pdf',
    'text/html': 'html',
    'text/plain': 'txt',
    'text/rtf': 'rtf',
    'text/uri-list': 'url',
}

const TYPE_PRIORITY = [
    'image/png', 'image/jpeg', 'image/gif', 'image/webp',
    'image/svg+xml', 'image/bmp', 'application/pdf',
    'text/html', 'text/plain', 'text/rtf', 'text/uri-list',
]

const CLIPBOARD_LAST_REP_KEY = 'gkill_clipboard_save_last_rep_name'

function get_ext_from_filename(name: string): string {
    const dot = name.lastIndexOf('.')
    return dot >= 0 ? name.slice(dot + 1).toLowerCase() : 'bin'
}

function mime_from_ext(ext: string): string {
    const map: Record<string, string> = {
        png: 'image/png', jpg: 'image/jpeg', jpeg: 'image/jpeg',
        gif: 'image/gif', webp: 'image/webp', svg: 'image/svg+xml',
        bmp: 'image/bmp', pdf: 'application/pdf',
        htm: 'text/html', html: 'text/html',
        txt: 'text/plain', rtf: 'text/rtf',
    }
    return map[ext] ?? 'application/octet-stream'
}

export function useSaveClipboardToFileDialog(options: {
    props: SaveClipboardToFileDialogProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    // ── State ──
    const is_show_dialog = ref(false)
    const is_loading = ref(false)
    const clipboard_blob: Ref<Blob | null> = ref(null)
    const selected_mime_type = ref('')
    const filename = ref('')
    const preview_url = ref('')
    const text_preview = ref('')
    const error_message_key = ref('')
    const target_rep_names: Ref<Array<string>> = ref([])
    const target_rep_name = ref(localStorage.getItem(CLIPBOARD_LAST_REP_KEY) ?? '')
    const conflict_behavior: Ref<FileUploadConflictBehavior> = ref(FileUploadConflictBehavior.rename)
    const show_filename_editor = ref(false)
    const show_already_saved_confirm = ref(false)
    const show_saved_snackbar = ref(false)
    const last_saved_hash = ref('')
    const save_btn = ref<HTMLElement | null>(null)
    const dialog_el = ref<HTMLElement | null>(null)
    const is_last_clicked_dialog = ref(false)

    function set_dialog_el(el: HTMLElement | null): void {
        dialog_el.value = el
    }

    function on_mousedown(e: MouseEvent): void {
        const target = e.target as Element | null
        if (!target) return
        const clicked_dialog = target.closest('.gkill-floating-dialog')
        if (!clicked_dialog) return  // ダイアログ外クリックは状態を変えない
        if (dialog_el.value && clicked_dialog === dialog_el.value) {
            is_last_clicked_dialog.value = true
        } else {
            is_last_clicked_dialog.value = false
        }
    }

    // ── Helpers ──
    function is_image_type(): boolean {
        return selected_mime_type.value.startsWith('image/')
    }

    function is_text_type(): boolean {
        return selected_mime_type.value.startsWith('text/')
    }

    function type_display_name(): string {
        const m = selected_mime_type.value
        if (!m) return ''
        return (MIME_TO_EXT[m] ?? m).toUpperCase()
    }

    function file_size_display(): string {
        const b = clipboard_blob.value
        if (!b) return ''
        if (b.size < 1024) return `${b.size} B`
        if (b.size < 1024 * 1024) return `${(b.size / 1024).toFixed(1)} KB`
        return `${(b.size / 1024 / 1024).toFixed(1)} MB`
    }

    // ── Repository loading ──
    async function load_target_rep_names(): Promise<void> {
        const req = new GetRepositoriesRequest()
        const res = await props.gkill_api.get_repositories(req)
        if (res.errors && res.errors.length > 0) {
            emits('received_errors', res.errors)
            return
        }
        const names: string[] = []
        res.repositories?.forEach((rep: Repository) => {
            if (rep.type === 'directory' && rep.is_enable && rep.use_to_write) {
                names.push(rep.rep_name)
            }
        })
        target_rep_names.value = names
        // localStorage の値が有効なら保持、無効なら最初の候補に切り替える
        if (!target_rep_name.value || !names.includes(target_rep_name.value)) {
            target_rep_name.value = names[0] ?? ''
        }
    }

    // ── Clipboard ──
    function select_best_type(items: ClipboardItem[]): { item: ClipboardItem; type: string } | null {
        for (const type of TYPE_PRIORITY) {
            for (const item of items) {
                if (item.types.includes(type)) return { item, type }
            }
        }
        if (items.length > 0 && items[0].types.length > 0) {
            return { item: items[0], type: items[0].types[0] }
        }
        return null
    }

    function generate_filename(mimeType: string, originalName?: string): string {
        if (originalName) return originalName
        const now = new Date()
        const pad = (n: number) => n.toString().padStart(2, '0')
        const ts = `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}_${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`
        const ext = MIME_TO_EXT[mimeType] ?? 'bin'
        return `clipboard_${ts}.${ext}`
    }

    async function set_clipboard_data(blob: Blob, mimeType: string, originalName?: string): Promise<void> {
        if (preview_url.value) {
            URL.revokeObjectURL(preview_url.value)
            preview_url.value = ''
        }
        text_preview.value = ''
        clipboard_blob.value = blob
        selected_mime_type.value = mimeType
        filename.value = generate_filename(mimeType, originalName)
        show_already_saved_confirm.value = false
        error_message_key.value = ''

        if (mimeType.startsWith('image/')) {
            preview_url.value = URL.createObjectURL(blob)
        } else if (mimeType.startsWith('text/')) {
            const text = await blob.text()
            text_preview.value = text.split('\n').slice(0, 5).join('\n')
        }
    }

    async function load_clipboard(): Promise<void> {
        if (!navigator.clipboard?.read) {
            error_message_key.value = 'CLIPBOARD_NOT_SUPPORTED_MESSAGE'
            return
        }
        is_loading.value = true
        error_message_key.value = ''
        try {
            const items = await navigator.clipboard.read()
            if (items.length === 0) {
                error_message_key.value = 'CLIPBOARD_EMPTY_MESSAGE'
                return
            }
            const best = select_best_type(items)
            if (!best) {
                error_message_key.value = 'CLIPBOARD_EMPTY_MESSAGE'
                return
            }
            const blob = await best.item.getType(best.type)
            await set_clipboard_data(blob, best.type)
        } catch (e: unknown) {
            if (e instanceof Error && e.name === 'NotAllowedError') {
                error_message_key.value = 'CLIPBOARD_PERMISSION_DENIED_MESSAGE'
            } else {
                error_message_key.value = 'CLIPBOARD_PASTE_HERE_MESSAGE'
            }
        } finally {
            is_loading.value = false
        }
    }

    // ── Paste event ──
    async function on_paste(e: ClipboardEvent): Promise<void> {
        if (!is_show_dialog.value) return
        // 最後にクリックされたダイアログがこのダイアログでない場合はスキップ
        if (!is_last_clicked_dialog.value) return
        e.preventDefault()
        e.stopPropagation()

        const dt = e.clipboardData
        if (!dt) return

        // Files from Windows file copy (Ctrl+C on file in Explorer)
        if (dt.files && dt.files.length > 0) {
            const file = dt.files[0]
            const ext = get_ext_from_filename(file.name)
            const mimeType = file.type || mime_from_ext(ext)
            await set_clipboard_data(file, mimeType, file.name)
            await save_or_confirm()
            return
        }

        // DataTransferItems — try file kind first (images from web, etc.)
        if (dt.items && dt.items.length > 0) {
            for (const type of TYPE_PRIORITY) {
                for (let i = 0; i < dt.items.length; i++) {
                    const item = dt.items[i]
                    if (item.kind === 'file' && item.type === type) {
                        const file = item.getAsFile()
                        if (file) {
                            await set_clipboard_data(file, type, file.name || undefined)
                            await save_or_confirm()
                            return
                        }
                    }
                }
            }
            // String kinds
            for (let i = 0; i < dt.items.length; i++) {
                const item = dt.items[i]
                if (item.kind === 'string') {
                    item.getAsString(async (text: string) => {
                        const mime = item.type || 'text/plain'
                        const blob = new Blob([text], { type: mime })
                        await set_clipboard_data(blob, mime)
                        await save_or_confirm()
                    })
                    return
                }
            }
        }
    }

    // ── Hash ──
    async function compute_blob_hash(blob: Blob): Promise<string> {
        const buffer = await blob.arrayBuffer()
        const hashBuffer = await crypto.subtle.digest('SHA-256', buffer)
        return Array.from(new Uint8Array(hashBuffer))
            .map(b => b.toString(16).padStart(2, '0'))
            .join('')
    }

    // ── Save ──
    async function blob_to_base64(blob: Blob): Promise<string> {
        return new Promise((resolve, reject) => {
            const reader = new FileReader()
            reader.readAsDataURL(blob)
            reader.onload = () => resolve(reader.result as string)
            reader.onerror = reject
        })
    }

    async function do_save(): Promise<void> {
        if (!clipboard_blob.value) return
        // リポジトリが未ロード、または選択値が無効なら最初の候補に切り替える
        if (!target_rep_name.value || !target_rep_names.value.includes(target_rep_name.value)) {
            target_rep_name.value = target_rep_names.value[0] ?? ''
        }
        if (!target_rep_name.value) return

        const filedata = new FileData()
        filedata.file_name = filename.value
        filedata.data_base64 = await blob_to_base64(clipboard_blob.value)
        filedata.last_modified = new Date()

        localStorage.setItem(CLIPBOARD_LAST_REP_KEY, target_rep_name.value)

        const req = new UploadFilesRequest()
        req.files = [filedata]
        req.target_rep_name = target_rep_name.value
        req.conflict_behavior = conflict_behavior.value

        const res = await props.gkill_api.upload_files(req)
        if (res.errors && res.errors.length > 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages && res.messages.length > 0) {
            emits('received_messages', res.messages)
        }

        last_saved_hash.value = await compute_blob_hash(clipboard_blob.value)
        show_already_saved_confirm.value = false
        show_saved_snackbar.value = true

        for (const kyou of res.uploaded_kyous ?? []) {
            await kyou.reload(false, true)
            emits('registered_kyou', kyou)
        }
        // Keep dialog open for continuous saving; restore focus to save button
        await nextTick()
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        ;((save_btn.value as any)?.$el as HTMLElement | undefined)?.focus()
    }

    async function save_or_confirm(): Promise<void> {
        if (!clipboard_blob.value) return
        if (last_saved_hash.value) {
            const hash = await compute_blob_hash(clipboard_blob.value)
            if (hash === last_saved_hash.value) {
                show_already_saved_confirm.value = true
                return
            }
        }
        await do_save()
    }

    async function force_save(): Promise<void> {
        last_saved_hash.value = ''
        show_already_saved_confirm.value = false
        await do_save()
    }

    // ── Dialog control ──
    async function show(): Promise<void> {
        is_show_dialog.value = true
        is_last_clicked_dialog.value = true  // 開いた直後は「最後にクリックされた」扱い
        error_message_key.value = ''
        show_already_saved_confirm.value = false

        await load_target_rep_names()
        await load_clipboard()

        await nextTick()
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        ;((save_btn.value as any)?.$el as HTMLElement | undefined)?.focus()
    }

    function hide(): void {
        is_show_dialog.value = false
        is_last_clicked_dialog.value = false
        if (preview_url.value) {
            URL.revokeObjectURL(preview_url.value)
            preview_url.value = ''
        }
    }

    // ── Enter key → save (dialog-scoped) ──
    function on_keydown(e: KeyboardEvent): void {
        if (!is_show_dialog.value) return
        if (e.key !== 'Enter') return
        if (e.isComposing || e.repeat) return
        // テキスト入力中は発火しない
        const target = e.target as Element | null
        const active = document.activeElement
        const isInput = (el: Element | null) => {
            if (!el) return false
            const tag = (el as HTMLElement).tagName?.toLowerCase()
            return (el as HTMLElement).isContentEditable || tag === 'input' || tag === 'textarea' || tag === 'select'
        }
        if (isInput(target) || isInput(active)) return
        e.preventDefault()
        e.stopPropagation()
        save_or_confirm()
    }

    // ── Lifecycle ──
    onMounted(() => {
        document.addEventListener('paste', on_paste)
        window.addEventListener('keydown', on_keydown, { capture: true })
        document.addEventListener('mousedown', on_mousedown, { capture: true })
    })

    onBeforeUnmount(() => {
        document.removeEventListener('paste', on_paste)
        window.removeEventListener('keydown', on_keydown, { capture: true } as EventListenerOptions)
        document.removeEventListener('mousedown', on_mousedown, { capture: true } as EventListenerOptions)
        if (preview_url.value) URL.revokeObjectURL(preview_url.value)
    })

    return {
        is_show_dialog,
        is_loading,
        clipboard_blob,
        selected_mime_type,
        filename,
        preview_url,
        text_preview,
        error_message_key,
        target_rep_names,
        target_rep_name,
        conflict_behavior,
        show_filename_editor,
        show_already_saved_confirm,
        show_saved_snackbar,
        save_btn,
        dialog_el,
        set_dialog_el,
        is_image_type,
        is_text_type,
        type_display_name,
        file_size_display,
        load_clipboard,
        save_or_confirm,
        force_save,
        show,
        hide,
    }
}
