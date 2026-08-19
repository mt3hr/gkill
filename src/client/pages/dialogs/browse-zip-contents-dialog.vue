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
        <v-card variant="flat" class="pa-2">
          <v-card-title>
            <span>{{ i18n.global.t("BROWSE_ZIP_CONTENTS_TITLE") }}</span>
            <span v-if="all_entries.length > 0" class="text-caption ml-2">({{ all_entries.length }})</span>
          </v-card-title>

          <v-overlay v-model="is_loading" class="align-center justify-center" contained persistent>
            <v-progress-circular indeterminate color="primary" />
          </v-overlay>

          <div v-if="enlarged_image_index >= 0" class="zip-image-overlay" @click="close_enlarged()">
            <v-btn v-if="enlarged_image_index > 0" icon class="zip-nav-btn zip-nav-prev"
              @click.stop="show_prev_image()" variant="flat" color="primary">
              <v-icon>mdi-chevron-left</v-icon>
            </v-btn>
            <img :src="current_image_entries[enlarged_image_index].file_url" class="zip-enlarged-image" @click.stop />
            <v-btn v-if="enlarged_image_index < current_image_entries.length - 1" icon class="zip-nav-btn zip-nav-next"
              @click.stop="show_next_image()" variant="flat" color="primary">
              <v-icon>mdi-chevron-right</v-icon>
            </v-btn>
            <div class="zip-overlay-top-bar">
              <span class="zip-image-counter">{{ enlarged_image_index + 1 }} / {{ current_image_entries.length }}</span>
              <a :href="current_image_entries[enlarged_image_index].file_url" :download="file_name(current_image_entries[enlarged_image_index].path)" class="zip-text-download-link" :title="i18n.global.t('ZIP_DOWNLOAD_LINK_TITLE')">
                <v-btn icon variant="flat" color="primary">
                  <v-icon>mdi-download</v-icon>
                </v-btn>
              </a>
              <v-btn icon class="zip-close-btn" @click.stop="close_enlarged()" variant="flat" color="primary">
                <v-icon>mdi-close</v-icon>
              </v-btn>
            </div>
          </div>

          <!-- テキストビューワーオーバーレイ -->
          <div v-if="text_viewer_entry !== null" class="zip-text-overlay" @click.self="close_text_viewer()">
            <v-btn v-if="text_viewer_index > 0" icon class="zip-nav-btn zip-nav-prev"
              @click.stop="show_prev_text()" variant="flat" color="primary">
              <v-icon>mdi-chevron-left</v-icon>
            </v-btn>
            <div class="zip-text-viewer" @click.stop>
              <v-progress-circular v-if="text_viewer_loading" indeterminate color="primary" class="ma-4 align-self-center" />
              <pre v-else class="zip-text-content"><LinkifiedText :text="text_viewer_content" /></pre>
            </div>
            <v-btn v-if="text_viewer_index < current_text_entries.length - 1" icon class="zip-nav-btn zip-nav-next"
              @click.stop="show_next_text()" variant="flat" color="primary">
              <v-icon>mdi-chevron-right</v-icon>
            </v-btn>
            <div class="zip-overlay-top-bar">
              <span class="zip-image-counter">{{ file_name(text_viewer_entry.path) }}<template v-if="current_text_entries.length > 1"> ({{ text_viewer_index + 1 }} / {{ current_text_entries.length }})</template></span>
              <a :href="text_viewer_entry.file_url" :download="file_name(text_viewer_entry.path)" class="zip-text-download-link" :title="i18n.global.t('ZIP_DOWNLOAD_LINK_TITLE')">
                <v-btn icon variant="flat" color="primary">
                  <v-icon>mdi-download</v-icon>
                </v-btn>
              </a>
              <v-btn icon @click.stop="close_text_viewer()" variant="flat" color="primary">
                <v-icon>mdi-close</v-icon>
              </v-btn>
            </div>
          </div>

          <!-- メディアビューワーオーバーレイ（動画・音声） -->
          <div v-if="media_viewer_entry !== null" class="zip-media-overlay" @click.self="close_media_viewer()">
            <v-btn v-if="media_viewer_index > 0" icon class="zip-nav-btn zip-nav-prev"
              @click.stop="show_prev_media()" variant="flat" color="primary">
              <v-icon>mdi-chevron-left</v-icon>
            </v-btn>
            <div v-if="media_error" class="zip-media-error" @click.stop>
              <div>{{ i18n.global.t('ZIP_MEDIA_PLAYBACK_FAILED_MESSAGE') }}</div>
              <a :href="media_viewer_entry.file_url" :download="file_name(media_viewer_entry.path)"
                class="zip-text-download-link">
                <v-btn prepend-icon="mdi-download" variant="flat" color="primary">
                  {{ i18n.global.t('ZIP_DOWNLOAD_LINK_TITLE') }}
                </v-btn>
              </a>
            </div>
            <!-- :key でエントリごとに要素を作り直し、autoplayとエラー状態をリセットする -->
            <video v-else-if="media_viewer_entry.is_video" :key="media_viewer_entry.path"
              :src="media_viewer_entry.file_url" class="zip-media-video" controls autoplay playsinline
              @click.stop @error="onMediaError" />
            <audio v-else :key="media_viewer_entry.path" :src="media_viewer_entry.file_url"
              class="zip-media-audio" controls autoplay @click.stop @error="onMediaError" />
            <v-btn v-if="media_viewer_index >= 0 && media_viewer_index < current_media_entries.length - 1" icon
              class="zip-nav-btn zip-nav-next" @click.stop="show_next_media()" variant="flat" color="primary">
              <v-icon>mdi-chevron-right</v-icon>
            </v-btn>
            <div class="zip-overlay-top-bar">
              <span class="zip-image-counter">{{ file_name(media_viewer_entry.path) }}<template v-if="current_media_entries.length > 1"> ({{ media_viewer_index + 1 }} / {{ current_media_entries.length }})</template></span>
              <a :href="media_viewer_entry.file_url" target="_blank" rel="noopener" class="zip-text-download-link"
                :title="i18n.global.t('ZIP_OPEN_IN_NEW_TAB_TITLE')">
                <v-btn icon variant="flat" color="primary">
                  <v-icon>mdi-open-in-new</v-icon>
                </v-btn>
              </a>
              <a :href="media_viewer_entry.file_url" :download="file_name(media_viewer_entry.path)"
                class="zip-text-download-link" :title="i18n.global.t('ZIP_DOWNLOAD_LINK_TITLE')">
                <v-btn icon variant="flat" color="primary">
                  <v-icon>mdi-download</v-icon>
                </v-btn>
              </a>
              <v-btn icon @click.stop="close_media_viewer()" variant="flat" color="primary">
                <v-icon>mdi-close</v-icon>
              </v-btn>
            </div>
          </div>

          <!-- パンくずナビゲーション -->
          <div class="zip-breadcrumbs pa-2">
            <span class="zip-breadcrumb-item" :class="{ 'zip-breadcrumb-current': current_dir === '' }"
              @click="navigate_to('')">
              <v-icon size="small">mdi-folder-zip</v-icon>
              <span class="ml-1">{{ i18n.global.t("BROWSE_ZIP_CONTENTS_TITLE") }}</span>
            </span>
            <template v-for="(crumb, idx) in breadcrumbs" :key="crumb.path">
              <v-icon size="x-small" class="mx-1">mdi-chevron-right</v-icon>
              <span class="zip-breadcrumb-item"
                :class="{ 'zip-breadcrumb-current': idx === breadcrumbs.length - 1 }"
                @click="navigate_to(crumb.path)">
                {{ crumb.name }}
              </span>
            </template>
          </div>

          <div class="zip-entries-list">
            <!-- 親ディレクトリへ戻る -->
            <div v-if="current_dir !== ''" class="zip-entry-item zip-entry-dir zip-entry-clickable"
              @click="navigate_up()">
              <v-icon size="small" class="mr-1">mdi-arrow-up</v-icon>
              <span class="text-caption">..</span>
            </div>

            <!-- サブディレクトリ -->
            <div v-for="dir in current_subdirs" :key="'d:' + dir.path" class="zip-entry-item zip-entry-dir zip-entry-clickable"
              @click="navigate_to(dir.path)">
              <v-icon size="small" class="mr-1">mdi-folder</v-icon>
              <span class="text-caption">{{ dir.name }}/</span>
            </div>

            <!-- ファイル -->
            <div v-for="entry in current_files" :key="'f:' + entry.path" class="zip-entry-item">
              <template v-if="entry.is_image">
                <div class="zip-image-wrap" @click="open_enlarged_by_entry(entry)">
                  <img :src="entry.file_url" loading="lazy" decoding="async" fetchpriority="low"
                    class="zip-thumb-image" />
                </div>
                <span class="text-caption zip-entry-path">{{ file_name(entry.path) }}</span>
              </template>
              <template v-else-if="entry.is_text">
                <v-icon size="small" class="mr-1">mdi-file-document-outline</v-icon>
                <a :href="entry.file_url" class="text-caption" @click.prevent="open_text_viewer(entry)">{{ file_name(entry.path) }}</a>
                <span class="text-caption text-grey ml-1">({{ format_size(entry.size) }})</span>
              </template>
              <template v-else-if="entry.is_video || entry.is_audio">
                <v-icon size="small" class="mr-1">{{ entry.is_video ? 'mdi-movie-outline' : 'mdi-music-note' }}</v-icon>
                <a :href="entry.file_url" class="text-caption" @click.prevent="open_media_viewer(entry)">{{ file_name(entry.path) }}</a>
                <span class="text-caption text-grey ml-1">({{ format_size(entry.size) }})</span>
                <a :href="entry.file_url" target="_blank" rel="noopener" class="zip-entry-action-link"
                  :title="i18n.global.t('ZIP_OPEN_IN_NEW_TAB_TITLE')">
                  <v-icon size="small">mdi-open-in-new</v-icon>
                </a>
                <a :href="entry.file_url" :download="file_name(entry.path)" class="zip-entry-action-link"
                  :title="i18n.global.t('ZIP_DOWNLOAD_LINK_TITLE')">
                  <v-icon size="small">mdi-download</v-icon>
                </a>
              </template>
              <template v-else-if="entry.is_pdf">
                <v-icon size="small" class="mr-1">mdi-file-pdf-box</v-icon>
                <a :href="entry.file_url" target="_blank" rel="noopener" class="text-caption"
                  :title="i18n.global.t('ZIP_OPEN_IN_NEW_TAB_TITLE')">{{ file_name(entry.path) }}</a>
                <span class="text-caption text-grey ml-1">({{ format_size(entry.size) }})</span>
                <a :href="entry.file_url" :download="file_name(entry.path)" class="zip-entry-action-link"
                  :title="i18n.global.t('ZIP_DOWNLOAD_LINK_TITLE')">
                  <v-icon size="small">mdi-download</v-icon>
                </a>
              </template>
              <template v-else>
                <v-icon size="small" class="mr-1">mdi-file-download-outline</v-icon>
                <a :href="entry.file_url" :download="file_name(entry.path)" class="text-caption">{{ file_name(entry.path) }}</a>
                <span class="text-caption text-grey ml-1">({{ format_size(entry.size) }})</span>
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
import type { BrowseZipContentsDialogProps } from './browse-zip-contents-dialog-props'
import type { KyouDialogEmits } from '../views/kyou-dialog-emits'
import { i18n } from '@/i18n'
import LinkifiedText from '../views/linkified-text.vue'
import { useBrowseZipContentsDialog } from '@/classes/use-browse-zip-contents-dialog'

const props = defineProps<BrowseZipContentsDialogProps>()
const emits = defineEmits<KyouDialogEmits>()
const { is_show_dialog, ui, is_loading, all_entries, current_dir, enlarged_image_index, text_viewer_entry, text_viewer_content, text_viewer_loading, media_viewer_entry, media_error, breadcrumbs, current_subdirs, current_files, current_image_entries, current_text_entries, text_viewer_index, current_media_entries, media_viewer_index, navigate_to, navigate_up, file_name, show, hide, format_size, open_enlarged_by_entry, close_enlarged, show_prev_image, show_next_image, open_media_viewer, close_media_viewer, show_prev_media, show_next_media, onMediaError, show_prev_text, show_next_text, open_text_viewer, close_text_viewer } = useBrowseZipContentsDialog({ props, emits })
defineExpose({ show, hide })
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
/* メディアビューワーオーバーレイ */
.zip-media-overlay {
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
.zip-media-video {
  max-width: 90vw;
  max-height: calc(90vh - 56px);
}
.zip-media-audio {
  width: min(90vw, 500px);
}
.zip-media-error {
  background: #1e1e1e;
  color: #d4d4d4;
  border-radius: 8px;
  padding: 24px;
  max-width: 90vw;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}
.zip-entry-action-link {
  margin-left: 8px;
  color: inherit;
  text-decoration: none;
  display: inline-flex;
  align-items: center;
}
</style>
