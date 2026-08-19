'use strict'

import type { BrowseZipContentsDialogProps } from '@/pages/dialogs/browse-zip-contents-dialog-props'
import type { KyouDialogEmits } from '@/pages/views/kyou-dialog-emits'
import { type Ref, ref, computed, watch, onMounted, onUnmounted } from 'vue'
import type { ZipEntry } from '@/classes/api/req_res/browse-zip-contents-response'
import { BrowseZipContentsRequest } from '@/classes/api/req_res/browse-zip-contents-request'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { GkillError } from '@/classes/api/gkill-error'
import { i18n } from '@/i18n'
import { useFloatingDialog } from "@/classes/use-floating-dialog"
import { detect_and_decode_text } from '@/classes/decode-text'

interface BreadcrumbItem {
    name: string
    path: string
}
interface SubdirItem {
    name: string
    path: string
}

// テキストビューワーが読む上限。これを超える分は切って表示する
const TEXT_VIEWER_MAX_BYTES = 512 * 1024

export function useBrowseZipContentsDialog(options: {
    props: BrowseZipContentsDialogProps
    emits: KyouDialogEmits
}) {
    const { props, emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog, { onClosed: () => emits('closed') })
    const ui = useFloatingDialog("browse-zip-contents-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    const is_loading: Ref<boolean> = ref(false)
    const all_entries: Ref<ZipEntry[]> = ref([])
    const current_dir: Ref<string> = ref('')
    const enlarged_image_index: Ref<number> = ref(-1)
    // オーバーレイ用ヒストリースタック管理
    const is_enlarged: Ref<boolean> = ref(false)
    const is_text_viewer: Ref<boolean> = ref(false)
    const is_media_viewer: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_enlarged)
    useDialogHistoryStack(is_text_viewer)
    useDialogHistoryStack(is_media_viewer)
    watch(is_enlarged, (v) => { if (!v) enlarged_image_index.value = -1 })
    watch(is_text_viewer, (v) => { if (!v) close_text_viewer() })
    watch(is_media_viewer, (v) => { if (!v) close_media_viewer() })
    // テキストビューワー
    const text_viewer_entry: Ref<ZipEntry | null> = ref(null)
    const text_viewer_content: Ref<string> = ref('')
    const text_viewer_loading: Ref<boolean> = ref(false)
    // メディアビューワー（動画・音声）
    const media_viewer_entry: Ref<ZipEntry | null> = ref(null)
    const media_error: Ref<boolean> = ref(false)
    const breadcrumbs = computed((): BreadcrumbItem[] => {
        if (current_dir.value === '') return []
        const parts = current_dir.value.split('/')
        const crumbs: BreadcrumbItem[] = []
        for (let i = 0; i < parts.length; i++) {
            crumbs.push({
                name: parts[i],
                path: parts.slice(0, i + 1).join('/'),
            })
        }
        return crumbs
    })
    const current_subdirs = computed((): SubdirItem[] => {
        const prefix = current_dir.value === '' ? '' : current_dir.value + '/'
        const dir_set = new Set<string>()
        for (const entry of all_entries.value) {
            if (!entry.path.startsWith(prefix)) continue
            const rest = entry.path.slice(prefix.length)
            if (rest === '') continue
            const slash_idx = rest.indexOf('/')
            if (slash_idx >= 0) {
                dir_set.add(rest.slice(0, slash_idx))
            } else if (entry.is_dir) {
                dir_set.add(rest)
            }
        }
        const dirs: SubdirItem[] = []
        for (const name of Array.from(dir_set).sort()) {
            dirs.push({ name, path: prefix + name })
        }
        return dirs
    })
    const current_files = computed((): ZipEntry[] => {
        const prefix = current_dir.value === '' ? '' : current_dir.value + '/'
        return all_entries.value.filter(entry => {
            if (entry.is_dir) return false
            if (!entry.path.startsWith(prefix)) return false
            const rest = entry.path.slice(prefix.length)
            return rest.indexOf('/') < 0
        })
    })
    const current_image_entries = computed(() => current_files.value.filter(e => e.is_image))
    const current_text_entries = computed(() => current_files.value.filter(e => e.is_text))
    const text_viewer_index = computed((): number => {
        if (text_viewer_entry.value === null) return -1
        return current_text_entries.value.findIndex(e => e.path === text_viewer_entry.value!.path)
    })
    // テンプレートの分岐順（image・text優先）と揃えるため、両者に該当するものは除く（.ts等の拡張子重複対策）
    const current_media_entries = computed(() => current_files.value.filter(e => (e.is_video || e.is_audio) && !e.is_image && !e.is_text))
    const media_viewer_index = computed((): number => {
        if (media_viewer_entry.value === null) return -1
        return current_media_entries.value.findIndex(e => e.path === media_viewer_entry.value!.path)
    })
    function navigate_to(dir: string): void {
        current_dir.value = dir
        enlarged_image_index.value = -1
        close_text_viewer()
        close_media_viewer()
    }
    function navigate_up(): void {
        const last_slash = current_dir.value.lastIndexOf('/')
        current_dir.value = last_slash >= 0 ? current_dir.value.slice(0, last_slash) : ''
        enlarged_image_index.value = -1
        close_text_viewer()
        close_media_viewer()
    }
    function file_name(path: string): string {
        const idx = path.lastIndexOf('/')
        return idx >= 0 ? path.slice(idx + 1) : path
    }
    async function show(): Promise<void> {
        is_show_dialog.value = true
        current_dir.value = ''
        close_text_viewer()
        await load_entries()
    }
    async function hide(): Promise<void> {
        close_enlarged()
        close_text_viewer()
        close_media_viewer()
        close_dialog_via_history(is_show_dialog)
    }
    async function load_entries(): Promise<void> {
        is_loading.value = true
        try {
            const req = new BrowseZipContentsRequest()
            req.target_id = props.kyou.id
            const res = await props.gkill_api.browse_zip_contents(req)
            if (res.errors && res.errors.length > 0) {
                emits('received_errors', res.errors)
                return
            }
            if (res.messages && res.messages.length > 0) {
                emits('received_messages', res.messages)
            }
            all_entries.value = res.entries || []
        } finally {
            is_loading.value = false
        }
    }
    function format_size(bytes: number): string {
        if (bytes < 1024) return bytes + ' B'
        if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
        return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
    }
    function open_enlarged_by_entry(entry: ZipEntry): void {
        const idx = current_image_entries.value.findIndex(e => e.path === entry.path)
        if (idx >= 0) {
            enlarged_image_index.value = idx
            is_enlarged.value = true
        }
    }
    function close_enlarged(): void {
        enlarged_image_index.value = -1
        close_dialog_via_history(is_enlarged)
    }
    function show_prev_image(): void {
        if (enlarged_image_index.value > 0) {
            enlarged_image_index.value--
        }
    }
    function show_next_image(): void {
        if (enlarged_image_index.value < current_image_entries.value.length - 1) {
            enlarged_image_index.value++
        }
    }
    function open_media_viewer(entry: ZipEntry): void {
        media_error.value = false
        media_viewer_entry.value = entry
        is_media_viewer.value = true
    }
    function close_media_viewer(): void {
        media_viewer_entry.value = null
        media_error.value = false
        close_dialog_via_history(is_media_viewer)
    }
    function show_prev_media(): void {
        const idx = media_viewer_index.value
        if (idx > 0) {
            media_error.value = false
            media_viewer_entry.value = current_media_entries.value[idx - 1]
        }
    }
    function show_next_media(): void {
        const idx = media_viewer_index.value
        if (idx >= 0 && idx < current_media_entries.value.length - 1) {
            media_error.value = false
            media_viewer_entry.value = current_media_entries.value[idx + 1]
        }
    }
    function onMediaError(): void {
        media_error.value = true
    }
    function show_prev_text(): void {
        const idx = text_viewer_index.value
        if (idx > 0) open_text_viewer(current_text_entries.value[idx - 1])
    }
    function show_next_text(): void {
        const idx = text_viewer_index.value
        if (idx < current_text_entries.value.length - 1) open_text_viewer(current_text_entries.value[idx + 1])
    }
    function emit_text_error(message: string): void {
        const err = new GkillError()
        err.error_message = message
        emits('received_errors', [err])
        close_text_viewer()
    }
    async function open_text_viewer(entry: ZipEntry): Promise<void> {
        text_viewer_entry.value = entry
        is_text_viewer.value = true
        text_viewer_content.value = ''
        text_viewer_loading.value = true
        try {
            const res = await fetch(entry.file_url, { credentials: 'include' })
            if (!res.ok) {
                emit_text_error(`HTTP ${res.status}`)
                return
            }
            const content_length = res.headers.get('content-length')
            if (content_length && parseInt(content_length) > TEXT_VIEWER_MAX_BYTES) {
                emit_text_error(i18n.global.t('ZIP_TEXT_TOO_LARGE_MESSAGE'))
                return
            }
            const reader = res.body?.getReader()
            if (!reader) {
                emit_text_error(i18n.global.t('ZIP_TEXT_LOAD_FAILED_MESSAGE'))
                return
            }
            const chunks: Uint8Array[] = []
            let total_bytes = 0
            let truncated = false
            while (true) {
                const { done, value } = await reader.read()
                if (done) break
                if (value) {
                    total_bytes += value.byteLength
                    if (total_bytes > TEXT_VIEWER_MAX_BYTES) {
                        const remaining = TEXT_VIEWER_MAX_BYTES - (total_bytes - value.byteLength)
                        chunks.push(value.slice(0, remaining))
                        truncated = true
                        await reader.cancel()
                        break
                    }
                    chunks.push(value)
                }
            }
            const all_bytes = new Uint8Array(chunks.reduce((acc, c) => acc + c.byteLength, 0))
            let offset = 0
            for (const chunk of chunks) {
                all_bytes.set(chunk, offset)
                offset += chunk.byteLength
            }
            const decoded = detect_and_decode_text(all_bytes)
            text_viewer_content.value = decoded + (truncated ? '\n\n' + i18n.global.t('ZIP_TEXT_TRUNCATED_MESSAGE') : '')
        } catch (e) {
            emit_text_error(String(e))
        } finally {
            text_viewer_loading.value = false
        }
    }
    function close_text_viewer(): void {
        text_viewer_entry.value = null
        text_viewer_content.value = ''
        close_dialog_via_history(is_text_viewer)
    }
    function onKeydown(e: KeyboardEvent): void {
        if (media_viewer_entry.value !== null) {
            // 矢印キーはネイティブプレイヤーのシーク・音量操作に譲るため、Escapeだけ扱う
            if (e.key === 'Escape') {
                close_media_viewer()
                e.stopPropagation()
            }
            return
        }
        if (text_viewer_entry.value !== null) {
            if (e.key === 'Escape') {
                close_text_viewer()
                e.stopPropagation()
            } else if (e.key === 'ArrowLeft') {
                show_prev_text()
                e.preventDefault()
            } else if (e.key === 'ArrowRight') {
                show_next_text()
                e.preventDefault()
            }
            return
        }
        if (enlarged_image_index.value < 0) return
        if (e.key === 'Escape') {
            close_enlarged()
            e.stopPropagation()
        } else if (e.key === 'ArrowLeft') {
            show_prev_image()
            e.preventDefault()
        } else if (e.key === 'ArrowRight') {
            show_next_image()
            e.preventDefault()
        }
    }
    onMounted(() => {
        document.addEventListener('keydown', onKeydown)
    })
    onUnmounted(() => {
        document.removeEventListener('keydown', onKeydown)
    })

    return {
        is_show_dialog,
        ui,
        is_loading,
        all_entries,
        current_dir,
        enlarged_image_index,
        text_viewer_entry,
        text_viewer_content,
        text_viewer_loading,
        media_viewer_entry,
        media_error,
        breadcrumbs,
        current_subdirs,
        current_files,
        current_image_entries,
        current_text_entries,
        text_viewer_index,
        current_media_entries,
        media_viewer_index,
        navigate_to,
        navigate_up,
        file_name,
        show,
        hide,
        format_size,
        open_enlarged_by_entry,
        close_enlarged,
        show_prev_image,
        show_next_image,
        open_media_viewer,
        close_media_viewer,
        show_prev_media,
        show_next_media,
        onMediaError,
        show_prev_text,
        show_next_text,
        open_text_viewer,
        close_text_viewer,
    }
}
