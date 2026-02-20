<script setup lang="ts">
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue';
import { RouterView } from 'vue-router'
import { VLocaleProvider } from 'vuetify/components';
import { useTheme } from 'vuetify'
import { GkillAPI } from './classes/api/gkill-api';
import SaihateStarsOverlay from './pages/views/saihate-stars-overlay.vue'
import SnowFallOverlay from './pages/views/snow-fall-overlay.vue';

GkillAPI.get_gkill_api().apply_locale()

const theme = useTheme()
const use_dark_theme = GkillAPI.get_gkill_api().get_use_dark_theme()
if (use_dark_theme) {
  theme.global.name.value = 'gkill_dark_theme'
} else {
  theme.global.name.value = 'gkill_theme'
}

const locale: Ref<string> = ref(i18n.global.locale)
</script>


<template>
  <div>
    <SaihateStarsOverlay v-if="theme.global.name.value === 'gkill_dark_theme'" />
    <SnowFallOverlay v-if="theme.global.name.value === 'gkill_theme'" />
    <table>
      <tr>
        <td>
          <div id="control-height"></div>
        </td>
        <td>
          <v-app>
            <VLocaleProvider :locale="locale">
              <RouterView />
            </VLocaleProvider>
          </v-app>
        </td>
      </tr>
    </table>
  </div>
</template>

<style lang="css">
/* =========================================================
   Base / Layout
========================================================= */

#control-height {
  height: 100vh;
  width: 0;
  position: absolute;
  overflow-y: hidden;
}

html {
  overflow-y: hidden !important;
}

.mi_view_table,
.rykv_view_table {
  padding-top: 0px;
}

.mi_view_wrap,
.rykv_view_wrap,
.saihate_view_wrap {
  position: relative;
}

.rykv_dnote_wrap {
  width: fit-content;
}

.kyou_list_calendar_in_share_mi_view {
  width: 416px;
}

.v-navigation-drawer {
  transition: transform 0.3s ease-out !important;
  will-change: transform;
}

.mkfl_view_wrap .kftl_view {
  width: 100vw;
  overflow-y: hidden;
}

.gkill-float-scrim .kftl_view {
  width: 100vw;
  overflow-y: hidden;
}

/* テーブルの隙間埋め */
table,
tr,
td {
  border-spacing: 0 !important;
}

/* =========================================================
   Media (Kyou image/video)
========================================================= */

/* detail view: image/video 共通 */
.kyou_detail_view :is(.kyou_image, .kyou_video) {
  width: -webkit-fill-available !important;
  height: -webkit-fill-available !important;
  max-width: -webkit-fill-available !important;
  max-height: 100vh !important;
  object-fit: contain;
}

/* dialog: image/video 共通 */
.kyou_dialog :is(.kyou_image, .kyou_video) {
  width: unset !important;
  height: unset !important;
  max-width: 80vw !important;
  max-height: 85vh !important;
}

/* =========================================================
   Scroll / Overflow (dedup)
========================================================= */

/*
  NOTE:
  calc(actual_height) は CSS 的に無効なので、
  本来は var(--actual-height) などにするのが安全です。
  ただし「現状のまま」維持するため、ここでは触っていません。
*/
body,
.v-application--wrap,
.v-navigation-drawer--open {
  overflow-y: scroll !important;
  overflow-x: auto !important;
  height: calc(var(--actual_height)) !important;
  min-height: calc(var(--actual_height)) !important;
  max-height: calc(var(--actual_height)) !important;
}

/* 不要なスクロールバーを消す（body自体は隠す） */
body {
  overflow-y: hidden !important;
}

body::-webkit-scrollbar {
  display: none;
}

/* ダイアログ（スクロール） */
.kyou_detail_view,
.kyou_list_view,
.gkill-float-scrim .v-card {
  overflow-y: auto;
}

/* mkfl ダイアログだけ例外 */
.mkfl_dialog.gkill-float-scrim .v-card {
  overflow-y: hidden;
}

.mkfl_dialog.gkill-float-scrim .v-card::-webkit-scrollbar {
  width: 0px;
}

/* =========================================================
   Alerts
========================================================= */

.alert_container>div {
  width: fit-content;
}

.alert_container {
  justify-items: end;
  position: fixed;
  top: 60px;
  right: 10px;
  display: grid;
  grid-gap: 0.5em;
  z-index: 100000000;
}

/* =========================================================
   Scrollbar style (dedup)
========================================================= */

:root {
  --gkill-scrollbar-size: 8px;
  --gkill-scrollbar-thumb-width: 6px;
  --gkill-scrollbar-thumb-radius: 5px;
}

/* スクロールバー（対象の一覧） */
:where(.tag_struct_root,
  .rep_struct_root,
  .rep_type_struct_root,
  .device_struct_root,
  .kftl_template_struct_root,
  .v-navigation-drawer__content,
  .kyou_detail_view,
  .kyou_list_view,
  .kyou_list_view_image,
  .dnote_list_view,
  .kftl_text_area,
  .gkill-float-scrim .v-card .gkill-floating-dialog__body)::-webkit-scrollbar {
  margin-left: 1px;
  width: var(--gkill-scrollbar-size);
  height: var(--gkill-scrollbar-size);
}

:where(.tag_struct_root,
  .rep_struct_root,
  .rep_type_struct_root,
  .device_struct_root,
  .kftl_template_struct_root,
  .v-navigation-drawer__content,
  .kyou_detail_view,
  .ryuu_view,
  .kyou_list_view,
  .kyou_list_view_image,
  .dnote_list_view,
  .kftl_text_area,
  .gkill-float-scrim .v-card .gkill-floating-dialog__body)::-webkit-scrollbar-thumb {
  background: rgb(var(--v-theme-primary));
  width: var(--gkill-scrollbar-thumb-width);
  border-radius: var(--gkill-scrollbar-thumb-radius);
}

/* =========================================================
   Dnote / Misc
========================================================= */

.dnote_progress_overlay {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.git_commit_log_message {
  white-space: pre-line;
}

.plus_value {
  color: limegreen;
}

.minus_value {
  color: crimson;
}

.dnote_view {
  position: relative;
  width: 625px;
  min-width: 625px;
}

/* Date picker header hidden */
div.v-sheet.v-picker.v-date-picker.v-date-picker--year>div:nth-child(1),
div.v-sheet.v-picker.v-date-picker.v-date-picker--month>div:nth-child(1),
div.v-sheet.v-picker.v-date-picker.v-date-picker--months>div:nth-child(1) {
  display: none;
}

.v-date-picker-month {
  padding-left: 0px !important;
}

/* Context menu */
.gkill_context_menu_list {
  max-height: 70vh;
  overflow-y: scroll;
}

/* =========================================================
   Lantana icons (dedup across related list & dnote)
========================================================= */

:root {
  --lantana-size: 20px;
  --lantana-half-width: 10px;
  --lantana-tr-width: calc(var(--lantana-size) * 5);
}

/* 対象コンテナ（related_kyou_list_item と dnote_view） */
:where(.related_kyou_list_item, .dnote_view) .lantana_icon {
  position: relative;
  width: var(--lantana-size) !important;
  height: var(--lantana-size) !important;
  max-width: var(--lantana-size) !important;
  min-width: var(--lantana-size) !important;
  max-height: var(--lantana-size) !important;
  min-height: var(--lantana-size) !important;
  display: inline-block;
}

:where(.related_kyou_list_item, .dnote_view) .lantana_icon_fill,
:where(.related_kyou_list_item, .dnote_view) .lantana_icon_none,
:where(.related_kyou_list_item, .dnote_view) .lantana_icon_td {
  width: var(--lantana-size) !important;
  height: var(--lantana-size) !important;
  max-width: var(--lantana-size) !important;
  min-width: var(--lantana-size) !important;
  max-height: var(--lantana-size) !important;
  min-height: var(--lantana-size) !important;
  display: inline-block;
}

:where(.related_kyou_list_item, .dnote_view) .lantana_icon_fill,
:where(.related_kyou_list_item, .dnote_view) .lantana_icon_none {
  z-index: 10;
}

:where(.related_kyou_list_item, .dnote_view) .lantana_icon_harf_left {
  position: absolute;
  left: 0px;
  width: var(--lantana-half-width) !important;
  height: var(--lantana-size) !important;
  max-width: var(--lantana-half-width) !important;
  min-width: var(--lantana-half-width) !important;
  max-height: var(--lantana-size) !important;
  min-height: var(--lantana-size) !important;
  object-fit: cover;
  object-position: 0 0;
  display: inline-block;
  z-index: 10;
}

:where(.related_kyou_list_item, .dnote_view) .lantana_icon_harf_right {
  position: absolute;
  left: 0px;
  width: var(--lantana-size) !important;
  height: var(--lantana-size) !important;
  max-width: var(--lantana-size) !important;
  min-width: var(--lantana-size) !important;
  max-height: var(--lantana-size) !important;
  min-height: var(--lantana-size) !important;
  display: inline-block;
  z-index: 9;
}

:where(.related_kyou_list_item, .dnote_view) .gray {
  filter: grayscale(100%);
}

:where(.related_kyou_list_item, .dnote_view) .lantana_icon_tr {
  width: var(--lantana-tr-width);
  max-width: var(--lantana-tr-width);
  min-width: var(--lantana-tr-width);
}

.kyou_list_view video::-webkit-media-controls,
.kyou_list_view_image video::-webkit-media-controls {
  display: none !important;
}

.gkill-float-scrim {
  position: fixed;
  inset: 0;
  pointer-events: none;
  z-index: 2999;
}

.gkill-floating-dialog {
  border-radius: 12px;
  max-width: 85vw;
  box-shadow: 0 0 8px silver;
}

.gkill-floating-dialog__body .plaing_timeis_view_wrap {
  width: unset;
}

.plaing_timeis_view_wrap {
  width: 100vw;
}

.v-virtual-scroll {
  max-width: unset;
}

.kyou_list_view .kyou_in_list {
  width: max-content;
}

.gkill-floating-dialog.is-transparent {
  opacity: 0.75;
}

.gkill-floating-dialog__header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  cursor: grab;
  user-select: none;
  touch-action: none;
  background: rgb(var(--v-theme-primary));
  color:white;
}

.gkill-floating-dialog__header:active {
  cursor: grabbing;
  background: rgb(var(--v-theme-primary));
}

.gkill-floating-dialog__title {
  font-size: 14px;
  color: rgba(255, 255, 255, 0.95);
}

.gkill-floating-dialog__spacer {
  flex: 1;
}

.gkill-floating-dialog__toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  touch-action: manipulation;
}

.gkill-floating-dialog__btn {
  font-size: 12px;
  padding: 4px 8px;
  border-radius: 8px;
  color: rgba(255, 255, 255, 0.92);
  min-height: 32px;
  min-width: 32px;
  touch-action: manipulation;
}

.gkill-floating-dialog__body {
  box-sizing: border-box;
  width: fit-content;
  max-width: 85vw;
  max-height: 92vh;
  overflow: auto;
  overflow-x: hidden;
  background-color: rgba(var(--v-theme-background));
}
</style>
