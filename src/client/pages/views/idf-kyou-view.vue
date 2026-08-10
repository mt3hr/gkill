<template>
    <v-card elevation="0" @contextmenu.prevent="show_context_menu" :width="width" :height="height">
        <!-- Markdownファイル: HTMLに変換してリッチテキスト表示 -->
        <div v-if="kyou.typed_idf_kyou && is_markdown" class="idf_text_wrap">
            <a :href="kyou.typed_idf_kyou.file_url" @click="open_link" class="idf_text_filename">
                {{ kyou.typed_idf_kyou.file_name }}
            </a>
            <v-progress-linear v-if="text_loading" indeterminate color="primary" height="2" />
            <!-- eslint-disable-next-line vue/no-v-html markdown_to_safe_html でDOMPurifyサニタイズ済み -->
            <div v-if="markdown_html" ref="markdown_content"
                :class="['idf_markdown_content', is_image_request_to_thumb_size ? 'idf_markdown_content--list' : '']"
                @click="onMarkdownContentClick" @dblclick="onMarkdownContentDblclick"
                v-html="markdown_html"></div>
        </div>
        <!-- テキストファイル: 内容をインライン表示 -->
        <div v-else-if="kyou.typed_idf_kyou && is_text" class="idf_text_wrap">
            <a :href="kyou.typed_idf_kyou.file_url" @click="open_link" class="idf_text_filename">
                {{ kyou.typed_idf_kyou.file_name }}
            </a>
            <v-progress-linear v-if="text_loading" indeterminate color="primary" height="2" />
            <pre v-if="text_content !== null"
                :class="['idf_text_content', is_image_request_to_thumb_size ? 'idf_text_content--noscroll' : '']"><LinkifiedText :text="text_content" /></pre>
        </div>
        <!-- その他のファイル: リンクのみ -->
        <a v-if="kyou.typed_idf_kyou && !is_text && !is_markdown && !kyou.typed_idf_kyou.is_image && !kyou.typed_idf_kyou.is_video && !kyou.typed_idf_kyou.is_audio"
            :href="kyou.typed_idf_kyou.file_url" @click="open_link">
            {{ kyou.typed_idf_kyou.file_name }}
        </a>
        <img v-if="kyou.typed_idf_kyou && kyou.typed_idf_kyou.is_image"
            :src="build_media_url(kyou.typed_idf_kyou.file_url, false)" loading="lazy" decoding="async"
            fetchpriority="low" class="kyou_image" />
        <video v-if="kyou.typed_idf_kyou && kyou.typed_idf_kyou.is_video" :src="kyou.typed_idf_kyou.file_url"
            preload="none" :poster="build_media_url(kyou.typed_idf_kyou.file_url, true)" class="kyou_video"
            controls></video>
        <audio v-if="kyou.typed_idf_kyou && kyou.typed_idf_kyou.is_audio" :src="kyou.typed_idf_kyou.file_url"
            class="kyou_audio" controls></audio>
        <IDFKyouContextMenu :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" ref="context_menu"
            @deleted_kyou="crudRelayHandlers['deleted_kyou']"
            @deleted_tag="crudRelayHandlers['deleted_tag']"
            @deleted_text="crudRelayHandlers['deleted_text']"
            @deleted_notification="crudRelayHandlers['deleted_notification']"
            @registered_kyou="crudRelayHandlers['registered_kyou']"
            @registered_tag="crudRelayHandlers['registered_tag']"
            @registered_text="crudRelayHandlers['registered_text']"
            @registered_notification="crudRelayHandlers['registered_notification']"
            @updated_kyou="crudRelayHandlers['updated_kyou']"
            @updated_tag="crudRelayHandlers['updated_tag']"
            @updated_text="crudRelayHandlers['updated_text']"
            @updated_notification="crudRelayHandlers['updated_notification']"
            @received_errors="crudRelayHandlers['received_errors']"
            @received_messages="crudRelayHandlers['received_messages']"
            @requested_reload_kyou="crudRelayHandlers['requested_reload_kyou']"
            @requested_reload_list="crudRelayHandlers['requested_reload_list']"
            @requested_update_check_kyous="crudRelayHandlers['requested_update_check_kyous']"
            @requested_open_rykv_dialog="crudRelayHandlers['requested_open_rykv_dialog']" />
    </v-card>
</template>
<script setup lang="ts">
import IDFKyouContextMenu from './idf-kyou-context-menu.vue'
import LinkifiedText from './linkified-text.vue'
import type { IDFKyouProps } from './idf-kyou-props'
import type { KyouViewEmits } from './kyou-view-emits'
import { useIDFKyouView } from '@/classes/use-idf-kyou-view'

const props = defineProps<IDFKyouProps>()
const emits = defineEmits<KyouViewEmits>()

const {
    context_menu,
    markdown_content,
    is_text,
    is_markdown,
    text_content,
    markdown_html,
    text_loading,
    show_context_menu,
    open_link,
    onMarkdownContentClick,
    onMarkdownContentDblclick,
    build_media_url,
    crudRelayHandlers,
} = useIDFKyouView({ props, emits })

defineExpose({ show_context_menu })
</script>

<style scoped>
.idf_text_wrap {
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: hidden;
    min-width: 0;
}

.idf_text_filename {
    font-size: 0.75rem;
    line-height: 1.4;
    padding: 2px 4px;
    flex-shrink: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

.idf_text_content {
    flex: 1;
    overflow: auto;
    font-size: 0.72rem;
    line-height: 1.5;
    padding: 4px 6px;
    margin: 0;
    white-space: pre-wrap;
    word-break: break-all;
    font-family: 'Consolas', 'Menlo', 'Monaco', monospace;
    background: rgba(0, 0, 0, 0.04);
    border-radius: 4px;
}

.idf_text_content--noscroll {
    overflow: hidden;
}

/* Markdownをv-htmlで描画するため、scoped属性が付かない子要素は :deep() で指定する。
   App.vue のグローバルスタイル (h1,h2 の margin:0、tr/td の padding:0) を打ち消す必要がある。 */
.idf_markdown_content {
    flex: 1;
    overflow: auto;
    font-size: 0.78rem;
    line-height: 1.6;
    padding: 4px 6px;
    min-width: 0;
    max-width: 100%;
    /* word-break: break-word と違い overflow-wrap: anywhere は min-content を縮める。
       長いURLや連続英数字で親の幅を押し広げないために必要。 */
    overflow-wrap: anywhere;
    word-break: break-word;
}

.idf_markdown_content--list {
    overflow: hidden;
    contain: layout;
    pointer-events: none;
}

.idf_markdown_content :deep(h1),
.idf_markdown_content :deep(h2),
.idf_markdown_content :deep(h3),
.idf_markdown_content :deep(h4),
.idf_markdown_content :deep(h5),
.idf_markdown_content :deep(h6) {
    margin: 0.6em 0 0.3em;
    line-height: 1.3;
    font-weight: bold;
}

.idf_markdown_content :deep(h1) {
    font-size: 1.5em;
}

.idf_markdown_content :deep(h2) {
    font-size: 1.3em;
}

.idf_markdown_content :deep(h3) {
    font-size: 1.15em;
}

.idf_markdown_content :deep(h1),
.idf_markdown_content :deep(h2) {
    border-bottom: 1px solid rgba(var(--v-border-color), 0.25);
    padding-bottom: 0.2em;
}

.idf_markdown_content :deep(p),
.idf_markdown_content :deep(ul),
.idf_markdown_content :deep(ol),
.idf_markdown_content :deep(blockquote),
.idf_markdown_content :deep(table),
.idf_markdown_content :deep(pre) {
    margin: 0.5em 0;
}

.idf_markdown_content :deep(ul),
.idf_markdown_content :deep(ol) {
    padding-inline-start: 1.5em;
}

.idf_markdown_content :deep(blockquote) {
    border-left: 3px solid rgba(var(--v-border-color), 0.35);
    padding-left: 0.8em;
    opacity: 0.85;
}

.idf_markdown_content :deep(code) {
    font-family: 'Consolas', 'Menlo', 'Monaco', monospace;
    font-size: 0.92em;
    background: rgba(var(--v-theme-on-surface), 0.06);
    border-radius: 3px;
    padding: 0.1em 0.3em;
}

/* UA既定の white-space: pre のままだと最長コード行が min-content になり、親の幅を突き破る。
   横スクロールは出さず、プレーンテキスト表示 (.idf_text_content) と同じく折り返す。 */
.idf_markdown_content :deep(pre) {
    background: rgba(var(--v-theme-on-surface), 0.06);
    border-radius: 4px;
    padding: 6px 8px;
    overflow-x: hidden;
    white-space: pre-wrap;
    word-break: break-all;
    max-width: 100%;
}

.idf_markdown_content :deep(pre code) {
    background: none;
    padding: 0;
}

/* table-layout: fixed にしないと、横に広い表の列 min-content がそのまま親の幅を押し広げる */
.idf_markdown_content :deep(table) {
    border-collapse: collapse;
    table-layout: fixed;
    width: 100%;
    max-width: 100%;
}

.idf_markdown_content :deep(th),
.idf_markdown_content :deep(td) {
    border: 1px solid rgba(var(--v-border-color), 0.3);
    padding: 3px 6px;
    overflow-wrap: anywhere;
    word-break: break-all;
}

/* 画像は遅れて読み込まれて高さを押し広げる。
   リスト表示は行高固定 (v-virtual-scroll の item-height) なので上限を掛ける。 */
.idf_markdown_content :deep(img) {
    max-width: 100%;
    height: auto;
}

.idf_markdown_content--list :deep(img) {
    max-height: 120px;
}

/* Mermaid: 描画済みのSVGラッパ。描画前・失敗時はプレースホルダの pre のまま表示される。 */
.idf_markdown_content :deep(.gkill_mermaid) {
    margin: 0.5em 0;
    max-width: 100%;
    overflow-x: auto;
    text-align: center;
}

.idf_markdown_content :deep(.gkill_mermaid svg) {
    max-width: 100%;
    height: auto;
}

/* リスト表示は行高固定 (v-virtual-scroll の item-height) なので、画像と同じく上限を掛ける */
.idf_markdown_content--list :deep(.gkill_mermaid svg) {
    max-height: 120px;
}

.idf_markdown_content :deep(hr) {
    border: none;
    border-top: 1px solid rgba(var(--v-border-color), 0.3);
    margin: 0.8em 0;
}
</style>