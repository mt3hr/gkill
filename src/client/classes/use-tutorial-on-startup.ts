'use strict'

import { nextTick, watch, type Ref } from 'vue'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'

/** チュートリアルダイアログのうち、ここが使う面だけ */
export interface TutorialDialogRef {
    show: () => void | Promise<void>
}

/**
 * 起動時に一度だけチュートリアルを出す。
 *
 * **`application_config` は ref ごと差し替えられる。**
 * `use-config-struct-sync.ts` は未知のタグ / リポジトリ / 端末を見つけるたびに
 * `application_config.value = application_config.value.clone()` を代入する
 * （深い変更だけでは `watch(() => props.application_config, ...)` が発火しないため、
 * 意図的にそうしている）。`clone()` は `is_loaded` も `show_tutorial_on_startup` も
 * 引き継ぐので、素朴に `watch(application_config, ...)` で出すと
 * **タグを1つ足しただけでチュートリアルが開き直す**。
 * 追加/編集ダイアログからタグを打てるようになって以降、これは日常的に踏む。
 *
 * 出してよいのは「起動時」＝ **`is_loaded` が false → true になった瞬間**の1回だけで、
 * 以降は同じページに居るあいだ二度と出さない。
 * 立ち上がりエッジで見るので、設定画面であとから ON にされても
 * その場では被さらない（次の起動から出る）。
 *
 * **`show_tutorial_on_startup` をクライアントから倒して解決してはいけない。**
 * これはサーバ設定で、倒すと利用者が「次から表示しない」を押していないのに
 * 設定そのものが書き換わる（倒してよいのは `tutorial-dialog.vue` の
 * 「次から表示しない」経路だけ）。
 */
export function useTutorialOnStartup(
    application_config: Ref<ApplicationConfig>,
    tutorial_dialog: Ref<TutorialDialogRef | null>,
): void {
    let is_shown = false
    watch(application_config, (config, previous_config) => {
        if (is_shown) {
            return
        }
        if (!config.is_loaded || !config.show_tutorial_on_startup) {
            return
        }
        // すでにロード済みの設定が差し替わっただけ＝起動時ではない
        if (previous_config.is_loaded) {
            return
        }
        is_shown = true
        nextTick(() => {
            void tutorial_dialog.value?.show()
        })
    })
}
