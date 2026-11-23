<template>
    <div>
        <!--
            骨格は App.vue と同じにしておくこと。
            - #control-height … 画面の高さの基準（resize_content が querySelector で引く）
            - v-app … v-app-bar / v-navigation-drawer / v-main はレイアウトの中でしか描けない。
              包まないと「[Vuetify] Could not find injected layout」で rykv ごと落ちる
            - VLocaleProvider … Vuetify 自身の文言のロケール
            出力 HTML は router を持たないので、RouterView の代わりに rykv を直接置く。
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
                                <ExportedRykvView v-if="is_ready" :app_content_height="app_content_height"
                                    :app_content_width="app_content_width"
                                    :app_title_bar_height="app_title_bar_height"
                                    :application_config="application_config" :gkill_api="gkill_api"
                                    :is_exported_with_dnote="is_exported_with_dnote"
                                    :is_exported_with_map="is_exported_with_map"
                                    :is_exported_with_sidebar="is_exported_with_sidebar"
                                    :share_title="share_title"
                                    @received_errors="(...errors: unknown[]) => write_errors(errors[0] as Array<GkillError>)"
                                    @received_messages="(...messages: unknown[]) => write_messages(messages[0] as Array<GkillMessage>)" />
                                <!--
                                    焼き込みが読めないとき。ガワだけの template.html を直接開いた場合もここ。
                                    出力 HTML は通信をしないので、サーバへ取りに行くフォールバックは置かない
                                -->
                                <v-container v-else class="exported_rykv_page_empty">
                                    <v-alert type="info" variant="tonal"
                                        :text="i18n.global.t('EXPORTED_RYKV_PAGE_NO_DATA_MESSAGE')" />
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
import { ref, type Ref } from 'vue'
import { VLocaleProvider } from 'vuetify/components'
import { useTheme } from 'vuetify'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import { i18n } from '@/i18n'
import ExportedRykvView from './views/exported-rykv-view.vue'
import { useExportedRykvPage } from '@/classes/use-exported-rykv-page'
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
    application_config,
    gkill_api,

    // Event handlers
    write_errors,
    write_messages,
} = useExportedRykvPage()

// テーマは焼き込んだ設定から決める（App.vue は Cookie から取るが、出力 HTML に Cookie は無い）
const theme = useTheme()
theme.global.name.value = application_config.value.use_dark_theme ? 'gkill_dark_theme' : 'gkill_theme'

const locale: Ref<string> = ref(i18n.global.locale)
</script>
<style scoped>
:root {
    --actual_height: v-bind(actual_height)
}
</style>
