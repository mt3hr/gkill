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
            <span v-if="entries.length > 0" class="text-caption ml-2">({{ entries.length }})</span>
          </v-card-title>

          <v-overlay v-model="is_loading" class="align-center justify-center" contained persistent>
            <v-progress-circular indeterminate color="primary" />
          </v-overlay>

          <div v-if="enlarged_image_index >= 0" class="zip-image-overlay" @click="closeEnlarged()">
            <v-btn v-if="enlarged_image_index > 0" icon class="zip-nav-btn zip-nav-prev"
              @click.stop="showPrevImage()" variant="flat" color="primary">
              <v-icon>mdi-chevron-left</v-icon>
            </v-btn>
            <img :src="image_entries[enlarged_image_index].file_url" class="zip-enlarged-image" @click.stop />
            <v-btn v-if="enlarged_image_index < image_entries.length - 1" icon class="zip-nav-btn zip-nav-next"
              @click.stop="showNextImage()" variant="flat" color="primary">
              <v-icon>mdi-chevron-right</v-icon>
            </v-btn>
            <div class="zip-overlay-top-bar">
              <span class="zip-image-counter">{{ enlarged_image_index + 1 }} / {{ image_entries.length }}</span>
              <v-btn icon class="zip-close-btn" @click.stop="closeEnlarged()" variant="flat" color="primary">
                <v-icon>mdi-close</v-icon>
              </v-btn>
            </div>
          </div>

          <div class="zip-entries-list">
            <div v-for="entry in entries" :key="entry.path" class="zip-entry-item"
              :class="{ 'zip-entry-dir': entry.is_dir }">
              <template v-if="entry.is_dir">
                <v-icon size="small" class="mr-1">mdi-folder</v-icon>
                <span class="text-caption">{{ entry.path }}/</span>
              </template>
              <template v-else-if="entry.is_image">
                <div class="zip-image-wrap" @click="openEnlargedByEntry(entry)">
                  <img :src="entry.file_url" loading="lazy" decoding="async" fetchpriority="low"
                    class="zip-thumb-image" />
                </div>
                <span class="text-caption zip-entry-path">{{ entry.path }}</span>
              </template>
              <template v-else>
                <v-icon size="small" class="mr-1">mdi-file</v-icon>
                <a :href="entry.file_url" class="text-caption" @click.prevent="openFileLink(entry.file_url)">{{ entry.path }}</a>
                <span class="text-caption text-grey ml-1">({{ formatSize(entry.size) }})</span>
              </template>
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
import { type Ref, ref, computed, onMounted, onUnmounted } from 'vue'
import type { ZipEntry } from '@/classes/api/req_res/browse-zip-contents-response'
import { BrowseZipContentsRequest } from '@/classes/api/req_res/browse-zip-contents-request'

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { i18n } from '@/i18n'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

interface BrowseZipContentsDialogProps extends KyouViewPropsBase {
}

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
const entries: Ref<ZipEntry[]> = ref([])
const enlarged_image_index: Ref<number> = ref(-1)

const image_entries = computed(() => entries.value.filter(e => e.is_image))

async function show(): Promise<void> {
  is_show_dialog.value = true
  await loadEntries()
}
async function hide(): Promise<void> {
  is_show_dialog.value = false
  enlarged_image_index.value = -1
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
    entries.value = res.entries || []
  } finally {
    is_loading.value = false
  }
}

function openFileLink(url: string): void {
  window.open(url, "_blank")
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function openEnlargedByEntry(entry: ZipEntry): void {
  const idx = image_entries.value.findIndex(e => e.path === entry.path)
  if (idx >= 0) enlarged_image_index.value = idx
}

function closeEnlarged(): void {
  enlarged_image_index.value = -1
}

function showPrevImage(): void {
  if (enlarged_image_index.value > 0) {
    enlarged_image_index.value--
  }
}

function showNextImage(): void {
  if (enlarged_image_index.value < image_entries.value.length - 1) {
    enlarged_image_index.value++
  }
}

function onKeydown(e: KeyboardEvent): void {
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
</style>
