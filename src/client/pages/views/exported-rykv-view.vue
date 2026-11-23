<template>
    <!-- HTML へ出力したファイルの中の rykv 画面。**列の中身は通常の rykv と同じ子**
         （rykv-columns-view）をホストし、編集系の持ち物（画面切替メニュー・
         編集ダイアログ群・FAB・ヘルプ・設定）を1つも置かない。
         だから「出力かどうか」を判定する場所が無く、スロットを空にするだけで済む。

         詳細（ダブルクリックで開く KyouDialog）は**使える** —— 出力は
         「読むために渡すファイル」なので、記録を開けないと用途を果たさない。
         開けるのは詳細だけで、残りはコンテキストメニュー由来なので
         enable_context_menu=false で閉じたままになる -->
    <RykvColumnsView :application_config="application_config" :gkill_api="gkill_api"
        :app_title_bar_height="app_title_bar_height" :app_content_height="app_content_height"
        :app_content_width="app_content_width"
        :application_config_load_failed="false /* 設定も焼き込み済みなので待たない */"
        :column_state_instance_key="''" :kyou_change_channel="null /* 単独ファイル。画面間の伝播は無い */"
        :enable_context_menu="false" :enable_add_column="false" :is_readonly_mi_check="true"
        :show_column_footer="false" :show_rep_name="false" :show_timeis_playing_end_button="false"
        :enable_keyboard_shortcuts="false /* 開く相手のダイアログが無い */"
        :is_dnote_available="is_exported_with_dnote" :is_map_available="is_exported_with_map"
        :is_sidebar_available="is_exported_with_sidebar"
        @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
        @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)">
        <template v-slot:app_bar_title>
            <div>{{ share_title }}</div>
        </template>
    </RykvColumnsView>
</template>
<script setup lang="ts">
import RykvColumnsView from './rykv-columns-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { ExportedRykvViewProps } from './exported-rykv-view-props'

defineProps<ExportedRykvViewProps>()
const emits = defineEmits<{
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'received_messages', messages: Array<GkillMessage>): void
}>()
</script>
