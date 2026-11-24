<template>
    <div>
        <!--
            骨格は App.vue と同じにしておくこと。
            - #control-height … 画面の高さの基準（resize_content が querySelector で引く）
            - v-app … v-app-bar / v-navigation-drawer / v-main はレイアウトの中でしか描けない。
              包まないと「[Vuetify] Could not find injected layout」で画面ごと落ちる
            - VLocaleProvider … Vuetify 自身の文言のロケール
            出力 HTML は router を持たないので、RouterView の代わりにビューを直接置く。
        -->
        <table>
            <tbody>
                <tr>
                    <td>
                        <div id="control-height"></div>
                    </td>
                    <td>
                        <v-app>
                            <VLocaleProvider :locale="locale">
                                <!--
                                    どの画面としてマウントするかは**ここで1回だけ**決める。
                                    焼き込みの印（payload.screen）を見るのはこのページだけで、
                                    ビューの中へモードのフラグを戻さない（ADR-0418 / ADR-0420）。
                                    先例は shared-page.vue の view_type の出し分け
                                -->
                                <ExportedMiView v-if="is_ready && exported_screen === 'mi'"
                                    :app_content_height="app_content_height"
                                    :app_content_width="app_content_width"
                                    :app_title_bar_height="app_title_bar_height"
                                    :application_config="application_config" :gkill_api="gkill_api"
                                    :is_exported_with_sidebar="is_exported_with_sidebar"
                                    :share_title="share_title"
                                    @received_errors="write_errors"
                                    @received_messages="write_messages" />
                                <ExportedRykvView v-else-if="is_ready" :app_content_height="app_content_height"
                                    :app_content_width="app_content_width"
                                    :app_title_bar_height="app_title_bar_height"
                                    :application_config="application_config" :gkill_api="gkill_api"
                                    :is_exported_with_dnote="is_exported_with_dnote"
                                    :is_exported_with_map="is_exported_with_map"
                                    :is_exported_with_sidebar="is_exported_with_sidebar"
                                    :share_title="share_title"
                                    @received_errors="write_errors"
                                    @received_messages="write_messages" />
                                <!--
                                    焼き込みが読めないとき。ガワだけの template.html を直接開いた場合もここ。
                                    出力 HTML は通信をしないので、サーバへ取りに行くフォールバックは置かない
                                -->
                                <v-container v-else class="exported_kyou_page_empty">
                                    <v-alert type="info" variant="tonal"
                                        :text="i18n.global.t('EXPORTED_KYOU_PAGE_NO_DATA_MESSAGE')" />
                                </v-container>
                            </VLocaleProvider>
                        </v-app>
                    </td>
                </tr>
            </tbody>
        </table>
        <GkillMessageFeedView />
    </div>
</template>

<script lang="ts" setup>
import { VLocaleProvider } from 'vuetify/components'
import { i18n } from '@/i18n'
import ExportedRykvView from './views/exported-rykv-view.vue'
import ExportedMiView from './views/exported-mi-view.vue'
import { useExportedKyouPage } from '@/classes/use-exported-kyou-page'
import GkillMessageFeedView from './views/gkill-message-feed-view.vue'

const {
    // State
    actual_height,
    app_title_bar_height,
    app_content_height,
    app_content_width,
    is_ready,
    share_title,
    is_exported_with_dnote,
    is_exported_with_map,
    is_exported_with_sidebar,
    exported_screen,
    application_config,
    gkill_api,
    locale,

    // Event handlers
    write_errors,
    write_messages,
} = useExportedKyouPage()
</script>
<style scoped>
:root {
    --actual_height: v-bind(actual_height)
}
</style>
