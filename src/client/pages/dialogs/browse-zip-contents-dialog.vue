<template>
  <Teleport to="body" v-if="is_show_dialog">
    <div class="gkill-float-scrim" :class="ui.isTransparent.value ? 'is-transparent' : ''" />

    <div :ref="ui.containerRef" :style="ui.fixedStyle.value" class="gkill-floating-dialog"
      :class="ui.isTransparent.value ? 'is-transparent' : ''">
      <div class="gkill-floating-dialog__header pa-0 ma-0" @mousedown="ui.onHeaderPointerDown"
        @touchstart="ui.onHeaderPointerDown">
        <div class="gkill-floating-dialog__title"></div>
        <div class="gkill-floating-dialog__spacer"></div>
        <v-checkbox v-model="ui.isTransparent.value" color="white" size="small" variant="flat"
          :label="i18n.global.t('TRANSPARENT_TITLE')" hide-details />
        <v-btn size="small" class="rounded-sm mx-auto" icon @click.prevent="hide" hide-details :color="'primary'"
          variant="flat">
          <v-icon>mdi-close</v-icon>
        </v-btn>
      </div>

      <div class="gkill-floating-dialog__body">
        <v-card class="pa-2">
          <v-card-title>
            <span>{{ i18n.global.t("BROWSE_ZIP_CONTENTS_TITLE") }}</span>
            <span v-if="all_entries.length > 0" class="text-caption ml-2">({{ all_entries.length }})</span>
          </v-card-title>

          <v-overlay v-model="is_loading" class="align-center justify-center" contained persistent>
            <v-progress-circular indeterminate color="primary" />
          </v-overlay>

          <div v-if="enlarged_image_index >= 0" class="zip-image-overlay" @click="closeEnlarged()">
            <v-btn v-if="enlarged_image_index > 0" icon class="zip-nav-btn zip-nav-prev"
              @click.stop="showPrevImage()" variant="flat" color="primary">
              <v-icon>mdi-chevron-left</v-icon>
            </v-btn>
            <img :src="current_image_entries[enlarged_image_index].file_url" class="zip-enlarged-image" @click.stop />
            <v-btn v-if="enlarged_image_index < current_image_entries.length - 1" icon class="zip-nav-btn zip-nav-next"
              @click.stop="showNextImage()" variant="flat" color="primary">
              <v-icon>mdi-chevron-right</v-icon>
            </v-btn>
            <div class="zip-overlay-top-bar">
              <span class="zip-image-counter">{{ enlarged_image_index + 1 }} / {{ current_image_entries.length }}</span>
              <a :href="current_image_entries[enlarged_image_index].file_url" :download="fileName(current_image_entries[enlarged_image_index].path)" class="zip-text-download-link">
                <v-btn icon variant="flat" color="primary">
                  <v-icon>mdi-download</v-icon>
                </v-btn>
              </a>
              <v-btn icon class="zip-close-btn" @click.stop="closeEnlarged()" variant="flat" color="primary">
                <v-icon>mdi-close</v-icon>
              </v-btn>
            </div>
          </div>

          <!-- テキストビューワーオーバーレイ -->
          <div v-if="text_viewer_entry !== null" class="zip-text-overlay" @click.self="closeTextViewer()">
            <v-btn v-if="text_viewer_index > 0" icon class="zip-nav-btn zip-nav-prev"
              @click.stop="showPrevText()" variant="flat" color="primary">
              <v-icon>mdi-chevron-left</v-icon>
            </v-btn>
            <div class="zip-text-viewer" @click.stop>
              <v-progress-circular v-if="text_viewer_loading" indeterminate color="primary" class="ma-4 align-self-center" />
              <pre v-else class="zip-text-content">{{ text_viewer_content }}</pre>
            </div>
            <v-btn v-if="text_viewer_index < current_text_entries.length - 1" icon class="zip-nav-btn zip-nav-next"
              @click.stop="showNextText()" variant="flat" color="primary">
              <v-icon>mdi-chevron-right</v-icon>
            </v-btn>
            <div class="zip-overlay-top-bar">
              <span class="zip-image-counter">{{ fileName(text_viewer_entry.path) }}<template v-if="current_text_entries.length > 1"> ({{ text_viewer_index + 1 }} / {{ current_text_entries.length }})</template></span>
              <a :href="text_viewer_entry.file_url" :download="fileName(text_viewer_entry.path)" class="zip-text-download-link">
                <v-btn icon variant="flat" color="primary">
                  <v-icon>mdi-download</v-icon>
                </v-btn>
              </a>
              <v-btn icon @click.stop="closeTextViewer()" variant="flat" color="primary">
                <v-icon>mdi-close</v-icon>
              </v-btn>
            </div>
          </div>

          <!-- パンくずナビゲーション -->
          <div class="zip-breadcrumbs pa-2">
            <span class="zip-breadcrumb-item" :class="{ 'zip-breadcrumb-current': current_dir === '' }"
              @click="navigateTo('')">
              <v-icon size="small">mdi-folder-zip</v-icon>
              <span class="ml-1">{{ i18n.global.t("BROWSE_ZIP_CONTENTS_TITLE") }}</span>
            </span>
            <template v-for="(crumb, idx) in breadcrumbs" :key="crumb.path">
              <v-icon size="x-small" class="mx-1">mdi-chevron-right</v-icon>
              <span class="zip-breadcrumb-item"
                :class="{ 'zip-breadcrumb-current': idx === breadcrumbs.length - 1 }"
                @click="navigateTo(crumb.path)">
                {{ crumb.name }}
              </span>
            </template>
          </div>

          <div class="zip-entries-list">
            <!-- 親ディレクトリへ戻る -->
            <div v-if="current_dir !== ''" class="zip-entry-item zip-entry-dir zip-entry-clickable"
              @click="navigateUp()">
              <v-icon size="small" class="mr-1">mdi-arrow-up</v-icon>
              <span class="text-caption">..</span>
            </div>

            <!-- サブディレクトリ -->
            <div v-for="dir in current_subdirs" :key="'d:' + dir.path" class="zip-entry-item zip-entry-dir zip-entry-clickable"
              @click="navigateTo(dir.path)">
              <v-icon size="small" class="mr-1">mdi-folder</v-icon>
              <span class="text-caption">{{ dir.name }}/</span>
            </div>

            <!-- ファイル -->
            <div v-for="entry in current_files" :key="'f:' + entry.path" class="zip-entry-item">
              <template v-if="entry.is_image">
                <div class="zip-image-wrap" @click="openEnlargedByEntry(entry)">
                  <img :src="entry.file_url" loading="lazy" decoding="async" fetchpriority="low"
                    class="zip-thumb-image" />
                </div>
                <span class="text-caption zip-entry-path">{{ fileName(entry.path) }}</span>
              </template>
              <template v-else-if="entry.is_text">
                <v-icon size="small" class="mr-1">mdi-file-document-outline</v-icon>
                <a :href="entry.file_url" class="text-caption" @click.prevent="openTextViewer(entry)">{{ fileName(entry.path) }}</a>
                <span class="text-caption text-grey ml-1">({{ formatSize(entry.size) }})</span>
              </template>
              <template v-else>
                <v-icon size="small" class="mr-1">mdi-file-download-outline</v-icon>
                <a :href="entry.file_url" :download="fileName(entry.path)" class="text-caption">{{ fileName(entry.path) }}</a>
                <span class="text-caption text-grey ml-1">({{ formatSize(entry.size) }})</span>
              </template>
            </div>

            <!-- 空の場合 -->
            <div v-if="current_subdirs.length === 0 && current_files.length === 0 && !is_loading"
              class="zip-entry-item text-caption text-grey">
              {{ i18n.global.t("BROWSE_ZIP_CONTENTS_EMPTY") }}
            </div>
          </div>
        </v-card>
      </div>
    </div>
  </Teleport>
</template>
<script setup lang="ts">
import type { KyouViewPropsBase } from '../views/kyou-view-props-base'
import type { KyouDialogEmits } from '../views/kyou-dialog-emits'
import { type Ref, ref, computed, watch, onMounted, onUnmounted } from 'vue'
import type { ZipEntry } from '@/classes/api/req_res/browse-zip-contents-response'
import { BrowseZipContentsRequest } from '@/classes/api/req_res/browse-zip-contents-request'

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { GkillError } from '@/classes/api/gkill-error'
import { i18n } from '@/i18n'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

type BrowseZipContentsDialogProps = KyouViewPropsBase

const props = defineProps<BrowseZipContentsDialogProps>()
const emits = defineEmits<KyouDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
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
useDialogHistoryStack(is_enlarged)
useDialogHistoryStack(is_text_viewer)
watch(is_enlarged, (v) => { if (!v) enlarged_image_index.value = -1 })
watch(is_text_viewer, (v) => { if (!v) closeTextViewer() })

// テキストビューワー
const text_viewer_entry: Ref<ZipEntry | null> = ref(null)
const text_viewer_content: Ref<string> = ref('')
const text_viewer_loading: Ref<boolean> = ref(false)
const TEXT_VIEWER_MAX_BYTES = 512 * 1024

interface BreadcrumbItem {
  name: string
  path: string
}

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

interface SubdirItem {
  name: string
  path: string
}

const current_subdirs = computed((): SubdirItem[] => {
  const prefix = current_dir.value === '' ? '' : current_dir.value + '/'
  const dirSet = new Set<string>()
  for (const entry of all_entries.value) {
    if (!entry.path.startsWith(prefix)) continue
    const rest = entry.path.slice(prefix.length)
    if (rest === '') continue
    const slashIdx = rest.indexOf('/')
    if (slashIdx >= 0) {
      dirSet.add(rest.slice(0, slashIdx))
    } else if (entry.is_dir) {
      dirSet.add(rest)
    }
  }
  const dirs: SubdirItem[] = []
  for (const name of Array.from(dirSet).sort()) {
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

function navigateTo(dir: string): void {
  current_dir.value = dir
  enlarged_image_index.value = -1
  closeTextViewer()
}

function navigateUp(): void {
  const lastSlash = current_dir.value.lastIndexOf('/')
  current_dir.value = lastSlash >= 0 ? current_dir.value.slice(0, lastSlash) : ''
  enlarged_image_index.value = -1
  closeTextViewer()
}

function fileName(path: string): string {
  const idx = path.lastIndexOf('/')
  return idx >= 0 ? path.slice(idx + 1) : path
}

async function show(): Promise<void> {
  is_show_dialog.value = true
  current_dir.value = ''
  closeTextViewer()
  await loadEntries()
}
async function hide(): Promise<void> {
  is_show_dialog.value = false
  enlarged_image_index.value = -1
  closeTextViewer()
  emits('closed')
}

async function loadEntries(): Promise<void> {
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

function formatSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function openEnlargedByEntry(entry: ZipEntry): void {
  const idx = current_image_entries.value.findIndex(e => e.path === entry.path)
  if (idx >= 0) {
    enlarged_image_index.value = idx
    is_enlarged.value = true
  }
}

function closeEnlarged(): void {
  enlarged_image_index.value = -1
  is_enlarged.value = false
}

function showPrevImage(): void {
  if (enlarged_image_index.value > 0) {
    enlarged_image_index.value--
  }
}

function showNextImage(): void {
  if (enlarged_image_index.value < current_image_entries.value.length - 1) {
    enlarged_image_index.value++
  }
}

function showPrevText(): void {
  const idx = text_viewer_index.value
  if (idx > 0) openTextViewer(current_text_entries.value[idx - 1])
}

function showNextText(): void {
  const idx = text_viewer_index.value
  if (idx < current_text_entries.value.length - 1) openTextViewer(current_text_entries.value[idx + 1])
}

function emitTextError(message: string): void {
  const err = new GkillError()
  err.error_message = message
  emits('received_errors', [err])
  closeTextViewer()
}

async function openTextViewer(entry: ZipEntry): Promise<void> {
  text_viewer_entry.value = entry
  is_text_viewer.value = true
  text_viewer_content.value = ''
  text_viewer_loading.value = true
  try {
    const res = await fetch(entry.file_url, { credentials: 'include' })
    if (!res.ok) {
      emitTextError(`HTTP ${res.status}`)
      return
    }
    const contentLength = res.headers.get('content-length')
    if (contentLength && parseInt(contentLength) > TEXT_VIEWER_MAX_BYTES) {
      emitTextError(i18n.global.t('ZIP_TEXT_TOO_LARGE_MESSAGE'))
      return
    }
    const reader = res.body?.getReader()
    if (!reader) {
      emitTextError(i18n.global.t('ZIP_TEXT_LOAD_FAILED_MESSAGE'))
      return
    }
    const chunks: Uint8Array[] = []
    let totalBytes = 0
    let truncated = false
    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      if (value) {
        totalBytes += value.byteLength
        if (totalBytes > TEXT_VIEWER_MAX_BYTES) {
          const remaining = TEXT_VIEWER_MAX_BYTES - (totalBytes - value.byteLength)
          chunks.push(value.slice(0, remaining))
          truncated = true
          await reader.cancel()
          break
        }
        chunks.push(value)
      }
    }
    const allBytes = new Uint8Array(chunks.reduce((acc, c) => acc + c.byteLength, 0))
    let offset = 0
    for (const chunk of chunks) {
      allBytes.set(chunk, offset)
      offset += chunk.byteLength
    }
    const decoded = detectAndDecodeText(allBytes)
    text_viewer_content.value = decoded + (truncated ? '\n\n' + i18n.global.t('ZIP_TEXT_TRUNCATED_MESSAGE') : '')
  } catch (e) {
    emitTextError(String(e))
  } finally {
    text_viewer_loading.value = false
  }
}

function closeTextViewer(): void {
  text_viewer_entry.value = null
  text_viewer_content.value = ''
  is_text_viewer.value = false
}

function detectAndDecodeText(bytes: Uint8Array): string {
  // BOM検出: UTF-8 / UTF-16LE / UTF-16BE
  if (bytes.length >= 3 && bytes[0] === 0xEF && bytes[1] === 0xBB && bytes[2] === 0xBF) {
    return new TextDecoder('utf-8').decode(bytes.slice(3))
  }
  if (bytes.length >= 2 && bytes[0] === 0xFF && bytes[1] === 0xFE) {
    return new TextDecoder('utf-16le').decode(bytes.slice(2))
  }
  if (bytes.length >= 2 && bytes[0] === 0xFE && bytes[1] === 0xFF) {
    return new TextDecoder('utf-16be').decode(bytes.slice(2))
  }
  // UTF-8 厳密デコード
  try {
    return new TextDecoder('utf-8', { fatal: true }).decode(bytes)
  } catch { /* UTF-8 ではない */ }
  // Shift_JIS (CP932)
  try {
    return new TextDecoder('shift_jis', { fatal: true }).decode(bytes)
  } catch { /* Shift_JIS ではない */ }
  // EUC-JP
  try {
    return new TextDecoder('euc-jp', { fatal: true }).decode(bytes)
  } catch { /* EUC-JP ではない */ }
  // フォールバック: 置換文字付き UTF-8
  return new TextDecoder('utf-8').decode(bytes)
}

function onKeydown(e: KeyboardEvent): void {
  if (text_viewer_entry.value !== null) {
    if (e.key === 'Escape') {
      closeTextViewer()
      e.stopPropagation()
    } else if (e.key === 'ArrowLeft') {
      showPrevText()
      e.preventDefault()
    } else if (e.key === 'ArrowRight') {
      showNextText()
      e.preventDefault()
    }
    return
  }
  if (enlarged_image_index.value < 0) return
  if (e.key === 'Escape') {
    closeEnlarged()
    e.stopPropagation()
  } else if (e.key === 'ArrowLeft') {
    showPrevImage()
    e.preventDefault()
  } else if (e.key === 'ArrowRight') {
    showNextImage()
    e.preventDefault()
  }
}

onMounted(() => {
  document.addEventListener('keydown', onKeydown)
})
onUnmounted(() => {
  document.removeEventListener('keydown', onKeydown)
})
</script>
<style lang="css" scoped>
.zip-breadcrumbs {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 2px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.12);
}
.zip-breadcrumb-item {
  cursor: pointer;
  padding: 2px 4px;
  border-radius: 4px;
  font-size: 0.85rem;
  white-space: nowrap;
}
.zip-breadcrumb-item:hover {
  background-color: rgba(0, 0, 0, 0.06);
}
.zip-breadcrumb-current {
  font-weight: bold;
  cursor: default;
}
.zip-breadcrumb-current:hover {
  background-color: transparent;
}
.zip-entries-list {
  max-height: 60vh;
  overflow-y: auto;
  padding: 4px;
}
.zip-entry-item {
  display: flex;
  align-items: center;
  padding: 4px 8px;
  border-bottom: 1px solid rgba(0, 0, 0, 0.08);
}
.zip-entry-dir {
  background-color: rgba(0, 0, 0, 0.02);
}
.zip-entry-clickable {
  cursor: pointer;
}
.zip-entry-clickable:hover {
  background-color: rgba(0, 0, 0, 0.06);
}
.zip-image-wrap {
  cursor: pointer;
  margin-right: 8px;
  flex-shrink: 0;
}
.zip-thumb-image {
  max-width: 200px;
  max-height: 150px;
  object-fit: contain;
  border-radius: 4px;
}
.zip-entry-path {
  margin-left: 8px;
  word-break: break-all;
}
.zip-image-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.85);
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
}
.zip-enlarged-image {
  max-width: 90vw;
  max-height: 90vh;
  object-fit: contain;
}
.zip-overlay-top-bar {
  position: fixed;
  top: 16px;
  right: 16px;
  display: flex;
  align-items: center;
  gap: 8px;
}
.zip-image-counter {
  color: white;
  font-size: 14px;
  user-select: none;
}
.zip-nav-btn {
  position: fixed;
  top: 50%;
  transform: translateY(-50%);
  z-index: 10000;
}
.zip-nav-prev {
  left: 16px;
}
.zip-nav-next {
  right: 16px;
}
/* テキストビューワーオーバーレイ */
.zip-text-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.85);
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  padding-top: 56px;
  box-sizing: border-box;
}
.zip-text-viewer {
  background: #1e1e1e;
  color: #d4d4d4;
  border-radius: 8px;
  width: 90vw;
  max-width: 900px;
  max-height: calc(90vh - 56px);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.zip-text-content {
  flex: 1;
  overflow: auto;
  padding: 16px;
  font-family: 'Consolas', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-all;
  margin: 0;
}
.zip-text-download-link {
  text-decoration: none;
}
</style>
