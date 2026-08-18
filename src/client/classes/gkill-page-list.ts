'use strict'

import { computed, type ComputedRef } from 'vue'
import { i18n } from '@/i18n'

export interface GkillPageListItem {
    app_name: string
    page_name: string
}

/**
 * ツールバーのタイトルをクリックすると出る画面切替メニューの一覧。
 *
 * 同じ内容が6つのコンポーザブルへ複製され、さらに kftl-page.vue には
 * テンプレート直書きの7つ目があった。画面が1つ増えるたびに7箇所へ配ることになるので、
 * ここ1箇所に寄せてある。
 *
 * ロケール切り替えに追随させるため computed。i18n.global.t を評価済みの
 * 配列にしてしまうと、言語を変えてもメニューが古い訳のまま残る。
 */
export const gkill_page_list: ComputedRef<Array<GkillPageListItem>> = computed(() => [
    { app_name: i18n.global.t('RYKV_APP_NAME'), page_name: 'rykv' },
    { app_name: i18n.global.t('MI_APP_NAME'), page_name: 'mi' },
    { app_name: i18n.global.t('KFTL_APP_NAME'), page_name: 'kftl' },
    { app_name: i18n.global.t('PLAING_TIMEIS_APP_NAME'), page_name: 'plaing' },
    { app_name: i18n.global.t('MKFL_APP_NAME'), page_name: 'mkfl' },
    { app_name: i18n.global.t('DASHBOARD_APP_NAME'), page_name: 'dashboard' },
    { app_name: i18n.global.t('RUDBECKIA_APP_NAME'), page_name: 'rudbeckia' },
    { app_name: i18n.global.t('SAIHATE_APP_NAME'), page_name: 'saihate' },
])

/**
 * ポートの中でダイアログとして開ける画面だけを抜いた一覧。
 * メモ帳 / 打刻メモ帳 は専用の追加ダイアログがあるのでここには入れない。
 */
export const rudbeckia_page_list: ComputedRef<Array<GkillPageListItem>> = computed(() =>
    gkill_page_list.value.filter(page =>
        page.page_name === 'rykv'
        || page.page_name === 'mi'
        || page.page_name === 'plaing'
        || page.page_name === 'dashboard'))
